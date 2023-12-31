package eh

import (
	"ComicCrawler/client"
	"ComicCrawler/utils"
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/spf13/cast"
	"log"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	imageInOnepage = 40
)

type GalleryInfo struct {
	URL        string              `json:"gallery_url"`
	Title      string              `json:"gallery_title"`
	TotalImage int                 `json:"total_image"`
	TagList    map[string][]string `json:"tag_list"`
}

func getGalleryInfo(galleryUrl string) GalleryInfo {
	c := colly.NewCollector(
		//模拟浏览器
		colly.UserAgent(client.ChromeUserAgent),
	)
	var galleryInfo GalleryInfo
	galleryInfo.TagList = make(map[string][]string)
	galleryInfo.URL = galleryUrl

	//找到<h1 id="gn">标签,即为文章标题
	c.OnHTML("h1#gn", func(e *colly.HTMLElement) {
		galleryInfo.Title = e.Text
	})

	//找到<td class="gdt2">标签
	reMaxPage := regexp.MustCompile(`(\d+) pages`)
	c.OnHTML("td.gdt2", func(e *colly.HTMLElement) {
		if reMaxPage.MatchString(e.Text) {
			//转换为int
			galleryInfo.TotalImage, _ = cast.ToIntE(reMaxPage.FindStringSubmatch(e.Text)[1])
		}
	})

	// 找到<div id="taglist">标签
	c.OnHTML("div#taglist", func(e *colly.HTMLElement) {
		// 查找<div id="taglist">标签下的<table>元素
		e.ForEach("table", func(_ int, el *colly.HTMLElement) {
			// 在每个<table>元素中查找<tr>元素
			el.ForEach("tr", func(_ int, el *colly.HTMLElement) {
				// 获取<tr>元素的<td class="tc">标签
				key := strings.TrimSpace(el.ChildText("td.tc"))
				localKey := strings.ReplaceAll(key, ":", "")
				//fmt.Printf("key=%s: \n", localKey)
				el.ForEach("td div", func(_ int, el *colly.HTMLElement) {
					//fmt.Println(el.Text)
					value := strings.TrimSpace(el.Text)
					galleryInfo.TagList[localKey] = append(galleryInfo.TagList[localKey], value)
				})
				//fmt.Println()
			})
		})
	})

	err := c.Visit(galleryUrl)
	if err != nil {
		log.Fatal(err)
		return galleryInfo
	}
	//fmt.Println(galleryInfo.TagList)
	return galleryInfo
}

func generateIndexURL(urlStr string, page int) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return ""
	}

	if page == 0 {
		return u.String()
	}

	q := u.Query()
	q.Set("p", cast.ToString(page))
	u.RawQuery = q.Encode()

	return u.String()
}

// getImagePageUrlList 获取图片页面的url
func getImagePageUrlList(c *colly.Collector, indexUrl string) []string {
	var imagePageUrls []string
	c.OnHTML("div[id='gdt']", func(e *colly.HTMLElement) {
		//找到其下所有<div class="gdtm">标签
		e.ForEach("div.gdtm", func(_ int, el *colly.HTMLElement) {
			//找到<a href="xxx">标签，只有一个
			imgUrl := el.ChildAttr("a", "href")
			imagePageUrls = append(imagePageUrls, imgUrl)
		})
	})
	err := c.Visit(indexUrl)
	utils.ErrorCheck(err)
	return imagePageUrls
}

func getImageUrl(c *colly.Collector, imagePageUrl string) string {
	//id="img"的src属性
	var imageUrl string
	c.OnHTML("img[id='img']", func(e *colly.HTMLElement) {
		imageUrl = e.Attr("src")
	})
	err := c.Visit(imagePageUrl)
	utils.ErrorCheck(err)
	return imageUrl
}

func buildJPEGRequestHeaders() http.Header {
	headers := http.Header{
		"Accept":             {"image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8"},
		"Accept-Encoding":    {"gzip, deflate, br"},
		"Accept-Language":    {"zh-CN,zh;q=0.9"},
		"Connection":         {"keep-alive"},
		"Dnt":                {"1"},
		"Host":               {"dqoaprm.qgankvrkxxiw.hath.network"},
		"Referer":            {"https://e-hentai.org/"},
		"Sec-Ch-Ua":          {"\"Not/A)Brand\";v=\"99\", \"Google Chrome\";v=\"115\", \"Chromium\";v=\"115\""},
		"Sec-Ch-Ua-Mobile":   {"?0"},
		"Sec-Ch-Ua-Platform": {"\"Windows\""},
		"Sec-Fetch-Dest":     {"image"},
		"Sec-Fetch-Mode":     {"no-cors"},
		"Sec-Fetch-Site":     {"cross-site"},
		"Sec-Gpc":            {"1"},
		"User-Agent":         {client.ChromeUserAgent},
	}

	//for key, values := range headers {
	//	fmt.Printf("%s: %s\n", key, values)
	//}
	return headers
}

func getImageInfoFromPage(c *colly.Collector, imagePageUrl string) (string, string) {
	imageIndex := imagePageUrl[strings.LastIndex(imagePageUrl, "-")+1:]
	imageUrl := getImageUrl(c, imagePageUrl)
	imageSuffix := imageUrl[strings.LastIndex(imageUrl, "."):]
	imageTitle := fmt.Sprintf("%s%s", imageIndex, imageSuffix)
	return imageTitle, imageUrl
}

func DownloadGallery(infoJsonPath string, galleryUrl string, onlyInfo bool) error {
	//目录号
	beginIndex := 0
	//余数
	remainder := 0

	//获取画廊信息
	galleryInfo := getGalleryInfo(galleryUrl)
	fmt.Println("Total Image:", galleryInfo.TotalImage)
	safeTitle := utils.ToSafeFilename(galleryInfo.Title)
	fmt.Println(safeTitle)

	if utils.FileExists(filepath.Join(safeTitle, infoJsonPath)) {
		fmt.Println("发现下载记录")
		//获取已经下载的图片数量
		downloadedImageCount := utils.GetFileTotal(safeTitle, []string{".jpg", ".png"})
		fmt.Println("Downloaded image count:", downloadedImageCount)
		//计算剩余图片数量
		remainImageCount := galleryInfo.TotalImage - downloadedImageCount
		if remainImageCount == 0 {
			fmt.Println("本gallery已经下载完毕")
			return nil
		} else if remainImageCount < 0 {
			return fmt.Errorf("下载记录有误！")
		} else {
			fmt.Println("剩余图片数量:", remainImageCount)
			beginIndex = int(math.Floor(float64(downloadedImageCount) / float64(imageInOnepage)))
			remainder = downloadedImageCount - imageInOnepage*beginIndex
		}
	} else {
		//生成缓存文件
		err := utils.BuildCache(safeTitle, infoJsonPath, galleryInfo)
		if err != nil {
			return err
		}
	}

	if onlyInfo {
		fmt.Println("画廊信息获取完毕，程序自动退出。")
		return nil
	}
	//重新初始化Collector
	collector := client.InitJPEGCollector(buildJPEGRequestHeaders())

	sumPage := int(math.Ceil(float64(galleryInfo.TotalImage) / float64(imageInOnepage)))
	for i := beginIndex; i < sumPage; i++ {
		fmt.Println("\nCurrent index:", i)
		indexUrl := generateIndexURL(galleryUrl, i)
		fmt.Println(indexUrl)
		var imagePageUrlList []string
		imagePageUrlList = getImagePageUrlList(collector, indexUrl)
		if i == beginIndex {
			//如果是第一次处理目录，需要去掉前面的余数
			imagePageUrlList = imagePageUrlList[remainder:]
		}

		var imageInfoList []map[string]string
		//根据imagePageUrls获取imageDataList
		for _, imagePageUrl := range imagePageUrlList {
			imageTitle, imageUrl := getImageInfoFromPage(collector, imagePageUrl)
			imageInfoList = append(imageInfoList, map[string]string{
				"imageTitle": imageTitle,
				"imageUrl":   imageUrl,
			})
		}
		//防止被ban，每处理一篇目录就sleep 5-10 seconds
		sleepTime := client.TrueRandFloat(5, 10)
		log.Println("Sleep ", cast.ToString(sleepTime), " seconds...")
		time.Sleep(time.Duration(sleepTime) * time.Second)

		// 进行本次处理目录中所有图片的批量保存
		utils.SaveImages(collector, imageInfoList, safeTitle)

		//防止被ban，每保存一篇目录中的所有图片就sleep 5-15 seconds
		sleepTime = client.TrueRandFloat(5, 15)
		log.Println("Sleep ", cast.ToString(sleepTime), " seconds...")
		time.Sleep(time.Duration(sleepTime) * time.Second)
	}
	return nil
}

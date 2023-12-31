package dmzj

import (
	"ComicCrawler/client"
	"ComicCrawler/utils"
	"ComicCrawler/utils/stack"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	cookiesPath = `dmzj_cookies.json`
	otherDir    = `其他系列`
)

type GalleryInfo struct {
	URL            string              `json:"gallery_url"`
	Title          string              `json:"gallery_title"`
	LastChapter    string              `json:"last_chapter"`
	LastUpdateTime string              `json:"last_update_time"`
	TagList        map[string][]string `json:"tag_list"`
}

// getGalleryInfo 从主目录页获取画廊信息
func getGalleryInfo(doc *goquery.Document, galleryUrl string) GalleryInfo {
	var galleryInfo GalleryInfo
	galleryInfo.TagList = make(map[string][]string)
	galleryInfo.URL = galleryUrl

	//找到<h1>标签,即为文章标题
	galleryInfo.Title = strings.TrimSpace(doc.Find("h1").Text())
	//fmt.Println(galleryInfo.Title)

	//找到<div class="anim-main_list">，即为tagList
	doc.Find(".anim-main_list table tbody tr").Each(func(index int, row *goquery.Selection) {
		key := strings.TrimSpace(row.Find("th").Text())
		localKey := strings.ReplaceAll(key, "：", "")
		row.Find("td").Each(func(index int, cell *goquery.Selection) {
			cell.Find("a").Each(func(index int, a *goquery.Selection) {
				galleryInfo.TagList[localKey] = append(galleryInfo.TagList[localKey], strings.TrimSpace(a.Text()))
			})
			//找到最后更新时间
			cell.Find("span").Each(func(index int, span *goquery.Selection) {
				galleryInfo.LastUpdateTime = strings.TrimSpace(span.Text())
			})
		})
	})

	if galleryInfo.TagList["最新收录"] != nil {
		galleryInfo.LastChapter = galleryInfo.TagList["最新收录"][0]
	} else {
		galleryInfo.LastChapter = "未知"
	}
	return galleryInfo
}

// getImagePageInfoListBySelector 从主目录页获取所有`selector`图片页地址
// selector的值为`div.cartoon_online_border`或`div.cartoon_online_border_other`，
// 返回2个切片，元素均为map[int]string
// imageOtherPageInfoList key为图片页序号，value为图片页地址
// indexToNameMap key为图片页序号，value为图片页名字
func getImagePageInfoListBySelector(selector string, doc *goquery.Document) (imagePageInfoList []map[int]string, indexToTitleMapList []map[int]string) {
	imageInfoStack := stack.Stack{}
	// 找到<div class="cartoon_online_border">
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		s.Find("a").Each(func(j int, a *goquery.Selection) {
			href, exists := a.Attr("href")
			if exists {
				imagePageTitle := strings.TrimSpace(a.Text())
				imagePageInfo := map[string]string{
					imagePageTitle: "https://manhua.dmzj.com" + href,
				}
				imageInfoStack.Push(imagePageInfo)
			}
		})
	})

	index := 1
	//直接处理得到的是逆序序列，通过栈转换为正序
	for !imageInfoStack.IsEmpty() {
		item := imageInfoStack.Pop()
		imageInfo := item.(map[string]string)
		for imagePageTitle, imagePageUrl := range imageInfo {
			imagePageInfo := map[int]string{
				index: imagePageUrl,
			}
			imagePageInfoList = append(imagePageInfoList, imagePageInfo)
			indexToTitleMapList = append(indexToTitleMapList, map[int]string{index: imagePageTitle})
			index++
		}
	}
	return imagePageInfoList, indexToTitleMapList
}

// getImageUrlListFromPage 从单个图片页获取图片地址
func getImageUrlListFromPage(doc *goquery.Document) []string {
	var imageUrlList []string
	//找到<div class="scrollbar-demo-item"
	doc.Find("div.scrollbar-demo-item").Each(func(i int, s *goquery.Selection) {
		s.Find("img").Each(func(j int, img *goquery.Selection) {
			src, exists := img.Attr("src")
			if exists {
				imageUrlList = append(imageUrlList, src)
			}
		})
	})
	return imageUrlList
}

func buildJPEGRequestHeaders() http.Header {
	headers := http.Header{
		"authority": []string{"images.idmzj.com"},
		"method":    []string{"GET"},
		//"path": []string{
		//	"/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC02%E8%AF%9D_1597930984%2F41.jpg",
		//},
		"scheme": []string{"https"},
		"Accept": []string{
			"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		},
		"Accept-Encoding":           []string{"gzip, deflate, br"},
		"Accept-Language":           []string{"zh-CN,zh;q=0.9"},
		"Cache-Control":             []string{"no-cache"},
		"Dnt":                       []string{"1"},
		"Pragma":                    []string{"no-cache"},
		"Sec-Ch-Ua":                 []string{"\"Chromium\";v=\"116\", \"Not)A;Brand\";v=\"24\", \"Google Chrome\";v=\"116\""},
		"Sec-Ch-Ua-Mobile":          []string{"?0"},
		"Sec-Ch-Ua-Platform":        []string{"\"Windows\""},
		"Sec-Fetch-Dest":            []string{"document"},
		"Sec-Fetch-Mode":            []string{"navigate"},
		"Sec-Fetch-Site":            []string{"none"},
		"Sec-Fetch-User":            []string{"?1"},
		"Sec-Gpc":                   []string{"1"},
		"Upgrade-Insecure-Requests": []string{"1"},
		"User-Agent":                []string{client.ChromeUserAgent},
	}

	return headers
}

func DownloadGallery(infoJsonPath string, galleryUrl string, onlyInfo bool) error {
	mainBeginIndex := 0
	otherBeginIndex := 0
	needUpdate := false

	cookies := client.ReadCookiesFromFile(cookiesPath)
	cookiesParam := client.ConvertCookies(cookies)
	// 初始化 Chromedp 上下文
	chromeCtx, cancel := client.InitChromedpContext(false)
	menuDoc := client.GetHtmlDoc(client.GetWaitVisibleRenderedPage(chromeCtx, cookiesParam, galleryUrl, `#__nuxt > div > div:nth-child(2) > div.wrap > div.left > div > div.anim_intro > div.week_mend > div > div.anim-main_list`))
	cancel()

	//获取画廊信息
	galleryInfo := getGalleryInfo(menuDoc, galleryUrl)
	safeTitle := utils.ToSafeFilename(galleryInfo.Title)
	fmt.Println(safeTitle)

	if utils.FileExists(filepath.Join(safeTitle, infoJsonPath)) {
		fmt.Println("发现下载记录")
		//读取缓存文件
		var lastGalleryInfo GalleryInfo
		err := utils.LoadCache(filepath.Join(safeTitle, infoJsonPath), &lastGalleryInfo)
		if err != nil {
			return err
		}

		needUpdate = utils.CheckUpdate(lastGalleryInfo.LastUpdateTime, galleryInfo.LastUpdateTime)
		if needUpdate {
			fmt.Println("发现新章节，更新下载记录")
			err := utils.BuildCache(safeTitle, infoJsonPath, galleryInfo)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("无需更新下载记录")
		}
		mainImagePath, err := filepath.Abs(safeTitle)
		if err != nil {
			return err
		}
		mainBeginIndex = utils.GetBeginIndex(mainImagePath, []string{".jpg", ".png"})

		otherImagePath, err := filepath.Abs(filepath.Join(safeTitle, otherDir))
		if err != nil {
			return err
		}
		otherBeginIndex = utils.GetBeginIndex(otherImagePath, []string{".jpg", ".png"})
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
	fmt.Println("mainBeginIndex=", mainBeginIndex)
	fmt.Println("otherBeginIndex=", otherBeginIndex)

	//主线剧情
	imagePageInfoList, indexToTitleMapList := getImagePageInfoListBySelector("div.cartoon_online_border", menuDoc)
	imagePageInfoList = imagePageInfoList[mainBeginIndex:]
	//其他系列
	otherImagePageInfoList, otherIndexToTitleMapList := getImagePageInfoListBySelector("div.cartoon_online_border_other", menuDoc)
	otherImagePageInfoList = otherImagePageInfoList[otherBeginIndex:]

	err := utils.BuildCache(safeTitle, "menu.json", indexToTitleMapList)
	if err != nil {
		return err
	}
	otherPath := filepath.Join(safeTitle, otherDir)
	if otherImagePageInfoList != nil {
		err = utils.BuildCache(otherPath, "menu.json", otherIndexToTitleMapList)
		if err != nil {
			return err
		}
	}
	fmt.Println("正在下载主线剧情...")
	utils.BatchDownloadImage(getImageUrlListFromPage, buildJPEGRequestHeaders, client.GetRenderedPage, cookiesParam, imagePageInfoList, safeTitle)
	fmt.Println("主线剧情下载完毕")
	fmt.Println("正在下载其他系列...")
	utils.BatchDownloadImage(getImageUrlListFromPage, buildJPEGRequestHeaders, client.GetRenderedPage, cookiesParam, otherImagePageInfoList, otherPath)
	fmt.Println("其他系列下载完毕")
	return nil
}

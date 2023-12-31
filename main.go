package main

import (
	"ComicCrawler/client"
	"ComicCrawler/comicSites/dmzj"
	"ComicCrawler/comicSites/eh"
	"ComicCrawler/comicSites/happymh"
	"ComicCrawler/utils"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cast"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const infoJsonPath = "galleryInfo.json"

var (
	onlyUpdate bool
	onlyInfo   bool
	galleryUrl string
	listFile   string
)

type GalleryInfo struct {
	URL   string `json:"gallery_url"`
	Title string `json:"gallery_title"`
}

type GalleryDownloader struct{}

func (gd GalleryDownloader) Download(infoJson string, url string, onlyInfo bool) error {
	// 根据正则表达式判断是哪个软件包的gallery，并调用相应的下载函数
	if matched, _ := regexp.MatchString(`^https://e-hentai.org/g/[a-z0-9]*/[a-z0-9]{10}/$`, url); matched {
		err := eh.DownloadGallery(infoJson, url, onlyInfo)
		if err != nil {
			return err
		}
	} else if matched, _ := regexp.MatchString(`^https://manhua.dmzj.com/[a-z0-9]*/$`, url); matched {
		err := dmzj.DownloadGallery(infoJson, url, onlyInfo)
		if err != nil {
			return err
		}
	} else if matched, _ := regexp.MatchString(`^https://m.happymh.com/manga/[a-zA-z0-9]*$`, url); matched {
		//因为cloudflare的反爬机制比较严格，所以这里需要设置DebugMode为1，使其不使用headless模式
		client.DebugMode = "1"
		err := happymh.DownloadGallery(infoJson, url, onlyInfo)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("未知的url格式：%s", url)
	}
	return nil
}

func getDownLoadedGalleryUrl() []string {
	galleryInfo := GalleryInfo{}
	// 将下载过的画廊地址添加到列表中
	var downloadedGalleryUrlList []string

	currentDir, err := os.Getwd()
	if err != nil {
		log.Println("获取当前目录时出错：", err)
		return downloadedGalleryUrlList
	}

	// 递归遍历目录
	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("获取当前目录时出错：", err)
			return err
		}

		// 检查是否是文件夹并且文件名是 galleryInfo.json
		if !info.IsDir() && info.Name() == "galleryInfo.json" {
			// 解析 JSON 数据
			err = utils.LoadCache(path, &galleryInfo)
			if err != nil {
				return err
			}

			downloadedGalleryUrlList = append(downloadedGalleryUrlList, galleryInfo.URL)
			//log.Println(galleryInfo)
		}

		return nil
	})

	if err != nil {
		fmt.Println("遍历目录时出错：", err)
	}
	return downloadedGalleryUrlList
}

func getExecutionTime(startTime time.Time, endTime time.Time) string {
	//按时:分:秒格式输出
	duration := endTime.Sub(startTime)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d时%d分%d秒", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%d分%d秒", minutes, seconds)
	} else {
		return fmt.Sprintf("%d秒", seconds)
	}
}

func main() {
	//设置输出颜色
	successColor := color.New(color.Bold, color.FgGreen).FprintlnFunc()
	failColor := color.New(color.Bold, color.FgRed).FprintlnFunc()
	errCount := 0

	app := &cli.App{
		Name:      "ComicCrawler",
		Usage:     "支持e-hentai.org,m.happymh.com,manhua.dmzj.com的漫画下载器\nGithub Link: https://github.com/Gungnir762/ComicCrawler",
		UsageText: "eg:\n	./ComicCrawler -u https://xxxxx/yyyy (-i)\neg:\n	./ComicCrawler.exe -l gallery_list.txt",
		Version:   "0.9.0",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "updateComics", Aliases: []string{"update"}, Destination: &onlyUpdate, Value: false, Usage: "更新当前文件夹下所有已下载的漫画，不能与其他任何参数一起使用"},
			&cli.BoolFlag{Name: "info", Aliases: []string{"i"}, Destination: &onlyInfo, Value: false, Usage: "只下载画廊信息"},
			&cli.StringFlag{Name: "url", Aliases: []string{"u"}, Destination: &galleryUrl, Value: "", Usage: "待下载的画廊地址（必填）"},
			&cli.StringFlag{Name: "list", Aliases: []string{"l"}, Destination: &listFile, Value: "", Usage: "待下载的画廊地址列表文件，每行一个url。(不能与参数-url同时使用)"},
		},
		HideHelpCommand: true,
		Action: func(c *cli.Context) error {
			var galleryUrlList []string
			switch {
			case onlyUpdate:
				galleryUrlList = getDownLoadedGalleryUrl()
			case galleryUrl == "" && listFile == "":
				return fmt.Errorf("本程序为命令行程序，请在命令行中运行参数-h以查看帮助")
			case galleryUrl != "" && listFile != "":
				return fmt.Errorf("参数错误，请在命令行中运行参数-h以查看帮助")
			case listFile != "":
				UrlList, err := utils.ReadListFile(listFile)
				if err != nil {
					return err
				}
				//UrlList... 使用了展开操作符（...），将 UrlList 切片中的所有元素一个一个地展开，作为参数传递给 append 函数
				galleryUrlList = append(galleryUrlList, UrlList...)
			case galleryUrl != "":
				galleryUrlList = append(galleryUrlList, galleryUrl)
			default:
				return fmt.Errorf("未知错误")
			}

			//记录开始时间
			startTime := time.Now()

			//创建下载器
			downloader := GalleryDownloader{}
			for _, url := range galleryUrlList {
				successColor(os.Stdout, "开始下载gallery:", url)
				err := downloader.Download(infoJsonPath, url, onlyInfo)
				if err != nil {
					failColor(os.Stderr, "下载失败:", err, "\n")
					errCount++
					continue
				}
				successColor(os.Stdout, "gallery下载完毕:", url, "\n")
			}

			//记录结束时间
			endTime := time.Now()
			//计算执行时间，单位为秒
			successColor(os.Stdout, "所有gallery下载完毕，共耗时:", getExecutionTime(startTime, endTime))
			if errCount > 0 {
				return fmt.Errorf("有" + cast.ToString(errCount) + "个下载失败")
			}

			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		failColor(os.Stderr, err)
	}

}

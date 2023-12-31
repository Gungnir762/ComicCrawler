package dmzj

import (
	"ComicCrawler/client"
	"ComicCrawler/utils"
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/smallnest/chanx"
	"reflect"
	"testing"
)

// 主函数和测试函数调用路径的区别
const localCookiesPath = "../../dmzj_cookies.json"

var (
	cookies      = client.ReadCookiesFromFile(localCookiesPath)
	cookiesParam = client.ConvertCookies(cookies)
)

func Test_getGalleryInfo(t *testing.T) {
	// 初始化 Chromedp 上下文
	chromeCtx, cancel := client.InitChromedpContext(false)
	defer cancel()
	tests := []struct {
		name       string
		galleryUrl string
		want       GalleryInfo
	}{
		{
			name:       "先下手为强",
			galleryUrl: "https://manhua.dmzj.com/xianxiashouweiqiang/",
			want: GalleryInfo{
				URL:            "https://manhua.dmzj.com/xianxiashouweiqiang/",
				Title:          "先下手为强",
				LastChapter:    "第14话",
				LastUpdateTime: "2020-05-19",
				TagList: map[string][]string{
					"作者":   {"たっくる"},
					"分类":   {"少年漫画"},
					"地域":   {"日本"},
					"状态":   {"连载中"},
					"题材":   {"爱情", "校园"},
					"最新收录": {"第14话"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menuDoc := client.GetHtmlDoc(client.GetRenderedPage(chromeCtx, cookiesParam, tt.galleryUrl))
			got := getGalleryInfo(menuDoc, tt.galleryUrl)
			//fmt.Println(got)
			if !reflect.DeepEqual(got, tt.want) {
				if got.Title != tt.want.Title {
					t.Errorf("title got: %v, want: %v", got.Title, tt.want.Title)
				}
				if got.LastChapter != tt.want.LastChapter {
					t.Errorf("lastChapter got: %v, want: %v", got.LastChapter, tt.want.LastChapter)
				}
				if got.LastUpdateTime != tt.want.LastUpdateTime {
					t.Errorf("lastUpdateTime got: %v, want: %v", got.LastUpdateTime, tt.want.LastUpdateTime)
				}
				if !reflect.DeepEqual(got.TagList, tt.want.TagList) {
					for k, v := range got.TagList {
						if !reflect.DeepEqual(v, tt.want.TagList[k]) {
							t.Errorf("tagList got: %v, want: %v", v, tt.want.TagList[k])
							for i, j := range v {
								if j != tt.want.TagList[k][i] {
									t.Errorf("tag got: %v, want: %v", j, tt.want.TagList[k][i])
								}
							}
						}
					}
				}
			}
		})
	}
}

func Test_getImagePageInfoListBySelector(t *testing.T) {
	// 初始化 Chromedp 上下文
	chromeCtx, cancel := client.InitChromedpContext(false)
	defer cancel()
	type args struct {
		selector string
		doc      *goquery.Document
	}
	tests := []struct {
		name                       string
		args                       args
		wantImageOtherPageInfoList []map[int]string
		wantIndexToNameMap         []map[int]string
	}{
		{
			name: "无其他页",
			args: args{
				selector: "div.cartoon_online_border_other",
				doc:      client.GetHtmlDoc(client.GetRenderedPage(chromeCtx, cookiesParam, "https://manhua.dmzj.com/xianxiashouweiqiang/")),
			},
			wantImageOtherPageInfoList: []map[int]string{},
			wantIndexToNameMap:         []map[int]string{},
		},
		{
			name: "有其他页",
			args: args{
				selector: "div.cartoon_online_border_other",
				doc:      client.GetHtmlDoc(client.GetRenderedPage(chromeCtx, cookiesParam, "https://manhua.dmzj.com/rangwoxinshendangyangdehuainvren/")),
			},
			wantImageOtherPageInfoList: []map[int]string{
				{1: "https://manhua.dmzj.com/rangwoxinshendangyangdehuainvren/118153.shtml#1"},
				{2: "https://manhua.dmzj.com/rangwoxinshendangyangdehuainvren/118154.shtml#1"},
				{3: "https://manhua.dmzj.com/rangwoxinshendangyangdehuainvren/118559.shtml#1"},
				{4: "https://manhua.dmzj.com/rangwoxinshendangyangdehuainvren/118781.shtml#1"},
				{5: "https://manhua.dmzj.com/rangwoxinshendangyangdehuainvren/119128.shtml#1"},
				{6: "https://manhua.dmzj.com/rangwoxinshendangyangdehuainvren/119129.shtml#1"},
			},
			wantIndexToNameMap: []map[int]string{
				{1: "短篇01"},
				{2: "短篇02"},
				{3: "短篇03"},
				{4: "短篇04"},
				{5: "短篇05"},
				{6: "短篇06"},
			},
		},
		{
			name: "主页",
			args: args{
				selector: "div.cartoon_online_border",
				doc:      client.GetHtmlDoc(client.GetRenderedPage(chromeCtx, cookiesParam, "https://manhua.dmzj.com/xianxiashouweiqiang/")),
			},
			wantImageOtherPageInfoList: []map[int]string{
				{1: "https://manhua.dmzj.com/xianxiashouweiqiang/96289.shtml#1"},
				{2: "https://manhua.dmzj.com/xianxiashouweiqiang/96311.shtml#1"},
				{3: "https://manhua.dmzj.com/xianxiashouweiqiang/97810.shtml#1"},
				{4: "https://manhua.dmzj.com/xianxiashouweiqiang/97874.shtml#1"},
				{5: "https://manhua.dmzj.com/xianxiashouweiqiang/98838.shtml#1"},
				{6: "https://manhua.dmzj.com/xianxiashouweiqiang/98938.shtml#1"},
			},
			wantIndexToNameMap: []map[int]string{
				{1: "第01话"},
				{2: "第02话"},
				{3: "第03话"},
				{4: "第04话"},
				{5: "第05话"},
				{6: "第06话"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotImagePageInfoList, gotIndexToTitleMapList := getImagePageInfoListBySelector(tt.args.selector, tt.args.doc)
			if gotImagePageInfoList != nil {
				gotImagePageInfoList = gotImagePageInfoList[0:6]
			}
			//fmt.Println(gotImagePageInfoList)
			if gotIndexToTitleMapList != nil {
				gotIndexToTitleMapList = gotIndexToTitleMapList[0:6]
			}
			if !reflect.DeepEqual(gotImagePageInfoList, tt.wantImageOtherPageInfoList) {
				for i, j := range gotImagePageInfoList {
					if !reflect.DeepEqual(j, tt.wantImageOtherPageInfoList[i]) {
						t.Errorf("gotImagePageInfoList = %v, want %v", j, tt.wantImageOtherPageInfoList[i])
					}
				}
			}
			if !reflect.DeepEqual(gotIndexToTitleMapList, tt.wantIndexToNameMap) {
				for i, j := range gotIndexToTitleMapList {
					if !reflect.DeepEqual(j, tt.wantIndexToNameMap[i]) {
						t.Errorf("gotIndexToTitleMapList = %v, want %v", j, tt.wantIndexToNameMap[i])
					}
				}
			}
		})
	}
}

func Test_getImageUrlListFromPage(t *testing.T) {
	// 初始化 Chromedp 上下文
	chromeCtx, cancel := client.InitChromedpContext(false)
	defer cancel()
	type args struct {
		doc *goquery.Document
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "先下手为强 03话",
			args: args{
				doc: client.GetHtmlDoc(client.GetRenderedPage(chromeCtx, cookiesParam, "https://manhua.dmzj.com/xianxiashouweiqiang/97810.shtml#1")),
			},
			want: []string{
				`https://images.idmzj.com/x%2F%E5%85%88%E4%B8%8B%E6%89%8B%E4%B8%BA%E5%BC%BA%2F%E7%AC%AC03%E8%AF%9D_1578815283%2F73404787_p0_master1200.jpg`,
				`https://images.idmzj.com/x%2F%E5%85%88%E4%B8%8B%E6%89%8B%E4%B8%BA%E5%BC%BA%2F%E7%AC%AC03%E8%AF%9D_1578815283%2F73404787_p1_master1200.jpg`,
				`https://images.idmzj.com/x%2F%E5%85%88%E4%B8%8B%E6%89%8B%E4%B8%BA%E5%BC%BA%2F%E7%AC%AC03%E8%AF%9D_1578815283%2F73404787_p2_master1200.jpg`,
				`https://images.idmzj.com/x%2F%E5%85%88%E4%B8%8B%E6%89%8B%E4%B8%BA%E5%BC%BA%2F%E7%AC%AC03%E8%AF%9D_1578815283%2F73404787_p3_master1200.jpg`,
				`https://images.idmzj.com/x%2F%E5%85%88%E4%B8%8B%E6%89%8B%E4%B8%BA%E5%BC%BA%2F%E7%AC%AC03%E8%AF%9D_1578815283%2F73404787_p4_master1200.jpg`,
			},
		},
		{
			name: "FS社主人公in艾尔登法环 01话",
			args: args{
				doc: client.GetHtmlDoc(client.GetRenderedPage(chromeCtx, cookiesParam, "https://manhua.dmzj.com/fsshezhurengonginaierdengfahuan/128361.shtml#1")),
			},
			want: []string{
				`https://images.idmzj.com/f%2FFS%E7%A4%BE%E4%B8%BB%E4%BA%BA%E5%85%ACin%E8%89%BE%E5%B0%94%E7%99%BB%E6%B3%95%E7%8E%AF%2F01%2F01.jpg`,
				`https://images.idmzj.com/f%2FFS%E7%A4%BE%E4%B8%BB%E4%BA%BA%E5%85%ACin%E8%89%BE%E5%B0%94%E7%99%BB%E6%B3%95%E7%8E%AF%2F01%2F02.jpg`,
				`https://images.idmzj.com/f%2FFS%E7%A4%BE%E4%B8%BB%E4%BA%BA%E5%85%ACin%E8%89%BE%E5%B0%94%E7%99%BB%E6%B3%95%E7%8E%AF%2F01%2F03.jpg`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getImageUrlListFromPage(tt.args.doc)
			if !reflect.DeepEqual(got, tt.want) {
				for i, j := range got {
					if j != tt.want[i] {
						t.Errorf("getImageUrlListFromPage() = %v, want %v", j, tt.want[i])
					}
				}
			}
		})
	}
}

// 由于syncParsePage函数需要传入一个本包的getImageUrlFromPage函数，所以放在本包测试
func Test_syncParsePage(t *testing.T) {
	type args struct {
		taskData             []map[int]string
		numWorkers           int
		tasks                chan map[int]string                     //此处与原代码不同，原代码为<-chan map[int]string，但是这样会导致无法读取channel
		imageInfoListChannel *chanx.UnboundedChan[map[string]string] //此处与原代码不同，原代码为chan<- map[string]string，但是这样会导致无法输入channel
		cookiesParam         []*network.CookieParam
	}
	tests := []struct {
		name string
		args args
		want []map[string]string
	}{
		{
			name: "成为夺心魔的必要",
			args: args{
				tasks:                make(chan map[int]string, utils.PageParallelism),
				imageInfoListChannel: chanx.NewUnboundedChan[map[string]string](utils.PageParallelism),
				cookiesParam:         cookiesParam,
				taskData: []map[int]string{
					{2: "https://manhua.dmzj.com/chengweiduoxinmodebiyao/102022.shtml#1"},
					{137: "https://manhua.dmzj.com/chengweiduoxinmodebiyao/135075.shtml#1"},
					{53: "https://manhua.dmzj.com/chengweiduoxinmodebiyao/109008.shtml#1"},
				},
			},
			want: []map[string]string{
				{
					"imageTitle": "2_0.jpg",
					"imageUrl":   `https://images.idmzj.com/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC02%E8%AF%9D_1597930984%2F41.jpg`,
				},
				{
					"imageTitle": "2_1.jpg",
					"imageUrl":   `https://images.idmzj.com/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC02%E8%AF%9D_1597930984%2F42.jpg`,
				},
				{
					"imageTitle": "2_2.jpg",
					"imageUrl":   `https://images.idmzj.com/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC02%E8%AF%9D_1597930984%2F999.jpg`,
				},
				{
					"imageTitle": "137_0.jpg",
					"imageUrl":   `https://images.idmzj.com/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC137%E8%AF%9D%2F137%E7%A0%94%E7%A9%B6%E6%9D%90%E6%96%99%20%E6%8B%B7%E8%B4%9D.jpg`,
				},
				{
					"imageTitle": "137_1.jpg",
					"imageUrl":   `https://images.idmzj.com/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC137%E8%AF%9D%2F336527817_147870357908547_2342450812458862125_n.jpg`,
				},
				{
					"imageTitle": "53_0.jpg",
					"imageUrl":   `https://images.idmzj.com/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC53%E8%AF%9D%2F53%E6%95%99%E8%AE%AD.jpg`,
				},
				{
					"imageTitle": "53_1.jpg",
					"imageUrl":   `https://images.idmzj.com/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC53%E8%AF%9D%2F%E5%B0%BE%E9%A1%B5.jpg`,
				},
				{
					"imageTitle": "53_2.jpg",
					"imageUrl":   `https://images.idmzj.com/c%2F%E6%88%90%E4%B8%BA%E5%A4%BA%E5%BF%83%E9%AD%94%E7%9A%84%E5%BF%85%E8%A6%81%2F%E7%AC%AC53%E8%AF%9D%2F%E8%B4%BA%E5%9B%BE.jpg`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 发送任务数据到tasks通道
			for _, task := range tt.args.taskData {
				tt.args.tasks <- task
			}
			close(tt.args.tasks)
			chromeCtxChannel := make(chan context.Context, utils.PageParallelism)
			var cancelList []context.CancelFunc
			for i := 0; i < utils.PageParallelism; i++ {
				// 初始化 Chromedp 上下文
				chromeCtx, cancel := client.InitChromedpContext(true)
				chromeCtxChannel <- chromeCtx
				cancelList = append(cancelList, cancel)
			}
			// 启动并发执行
			utils.SyncParsePage(getImageUrlListFromPage, client.GetRenderedPage,
				chromeCtxChannel, tt.args.tasks, tt.args.imageInfoListChannel, tt.args.cookiesParam)
			for _, cancel := range cancelList {
				cancel()
			}
			// 接收所有发送到imageInfoChannel通道的数据
			var got []map[string]string
			for i := 0; i < len(tt.want); i++ {
				imageInfo := <-tt.args.imageInfoListChannel.Out
				got = append(got, imageInfo)
			}
			close(tt.args.imageInfoListChannel.In)

			for _, imageInfo := range got {
				//如果got的元素不在want中
				if !utils.ElementInSlice(imageInfo, tt.want) {
					t.Errorf("syncParsePage() got = %v,not in want", imageInfo)
				}
			}
		})
	}
}

func Test_getBeginIndex(t *testing.T) {
	type args struct {
		dirPath      string
		fileSuffixes []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "成为夺心魔的必要",
			args: args{
				dirPath:      `D:\Games\comic\DMZJ\成为夺心魔的必要\`,
				fileSuffixes: []string{".jpg", ".png"},
			},
			want: 150,
		},
		{
			name: "先下手为强",
			args: args{
				dirPath:      `D:\Games\comic\DMZJ\先下手为强\`,
				fileSuffixes: []string{".jpg", ".png"},
			},
			want: 14,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.GetBeginIndex(tt.args.dirPath, tt.args.fileSuffixes); got != tt.want {
				t.Errorf("getBeginIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

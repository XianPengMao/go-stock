package data

import (
	"context"
	"go-stock/backend/logger"
	"time"

	"github.com/chromedp/chromedp"
)

// @Author spark
// @Date 2025/2/13 9:25
// @Desc
// -----------------------------------------------------------------------------------

type CrawlerApi struct {
	crawlerCtx      context.Context
	crawlerBaseInfo CrawlerBaseInfo
}

func (c *CrawlerApi) NewTimeOutCrawler(timeout int, crawlerBaseInfo CrawlerBaseInfo) CrawlerApi {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	return c.NewCrawler(ctx, crawlerBaseInfo)
}
func (c *CrawlerApi) NewCrawler(ctx context.Context, crawlerBaseInfo CrawlerBaseInfo) CrawlerApi {
	return CrawlerApi{
		crawlerCtx:      ctx,
		crawlerBaseInfo: crawlerBaseInfo,
	}
}

func (c *CrawlerApi) GetHtml(url, waitVisible string, headless bool) (string, bool) {
	// 添加重试机制
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		html, err := c.getHtmlWithRetry(url, waitVisible, headless)
		if err == nil {
			return html, true
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return "", false
}

func (c *CrawlerApi) getHtmlWithRetry(url, waitVisible string, headless bool) (string, error) {
	// 添加超时控制
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	htmlContent := ""
	path, e := checkBrowserOnWindows()
	logger.SugaredLogger.Infof("GetHtml path:%s", path)
	if e {
		pctx, pcancel := chromedp.NewExecAllocator(
			ctx,
			chromedp.ExecPath(path),
			chromedp.Flag("headless", headless),
			chromedp.Flag("blink-settings", "imagesEnabled=false"),
			chromedp.Flag("disable-javascript", false),
			chromedp.Flag("disable-gpu", true),
			chromedp.UserAgent(c.crawlerBaseInfo.Headers["User-Agent"]),
			chromedp.Flag("disable-background-networking", true),
			chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
			chromedp.Flag("disable-background-timer-throttling", true),
			chromedp.Flag("disable-backgrounding-occluded-windows", true),
			chromedp.Flag("disable-breakpad", true),
			chromedp.Flag("disable-client-side-phishing-detection", true),
			chromedp.Flag("disable-default-apps", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
			chromedp.Flag("disable-hang-monitor", true),
			chromedp.Flag("disable-ipc-flooding-protection", true),
			chromedp.Flag("disable-popup-blocking", true),
			chromedp.Flag("disable-prompt-on-repost", true),
			chromedp.Flag("disable-renderer-backgrounding", true),
			chromedp.Flag("disable-sync", true),
			chromedp.Flag("force-color-profile", "srgb"),
			chromedp.Flag("metrics-recording-only", true),
			chromedp.Flag("safebrowsing-disable-auto-update", true),
			chromedp.Flag("enable-automation", true),
			chromedp.Flag("password-store", "basic"),
			chromedp.Flag("use-mock-keychain", true),
		)
		defer pcancel()
		ctx, cancel := chromedp.NewContext(pctx, chromedp.WithLogf(logger.SugaredLogger.Infof))
		defer cancel()
		err := chromedp.Run(ctx, chromedp.Navigate(url),
			chromedp.WaitVisible(waitVisible, chromedp.ByQuery), // 确保  元素可见
			chromedp.WaitReady(waitVisible, chromedp.ByQuery),   // 确保  元素准备好
			chromedp.InnerHTML("body", &htmlContent),
		)
		if err != nil {
			logger.SugaredLogger.Error(err.Error())
			return "", err
		}
	} else {
		ctx, cancel := chromedp.NewContext(ctx, chromedp.WithLogf(logger.SugaredLogger.Infof))
		defer cancel()
		err := chromedp.Run(ctx, chromedp.Navigate(url), chromedp.WaitVisible("body"), chromedp.InnerHTML("body", &htmlContent))
		if err != nil {
			logger.SugaredLogger.Error(err.Error())
			return "", err
		}
	}
	return htmlContent, nil
}

func (c *CrawlerApi) GetHtmlWithNoCancel(url, waitVisible string, headless bool) (html string, ok bool, parent context.CancelFunc, child context.CancelFunc) {
	htmlContent := ""
	path, e := checkBrowserOnWindows()
	logger.SugaredLogger.Infof("GetHtml path:%s", path)
	var parentCancel context.CancelFunc
	var childCancel context.CancelFunc
	var pctx context.Context
	var cctx context.Context

	if e {
		pctx, parentCancel = chromedp.NewExecAllocator(
			c.crawlerCtx,
			chromedp.ExecPath(path),
			chromedp.Flag("headless", headless),
			chromedp.Flag("blink-settings", "imagesEnabled=false"),
			chromedp.Flag("disable-javascript", false),
			chromedp.Flag("disable-gpu", true),
			chromedp.UserAgent(c.crawlerBaseInfo.Headers["User-Agent"]),
			chromedp.Flag("disable-background-networking", true),
			chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
			chromedp.Flag("disable-background-timer-throttling", true),
			chromedp.Flag("disable-backgrounding-occluded-windows", true),
			chromedp.Flag("disable-breakpad", true),
			chromedp.Flag("disable-client-side-phishing-detection", true),
			chromedp.Flag("disable-default-apps", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
			chromedp.Flag("disable-hang-monitor", true),
			chromedp.Flag("disable-ipc-flooding-protection", true),
			chromedp.Flag("disable-popup-blocking", true),
			chromedp.Flag("disable-prompt-on-repost", true),
			chromedp.Flag("disable-renderer-backgrounding", true),
			chromedp.Flag("disable-sync", true),
			chromedp.Flag("force-color-profile", "srgb"),
			chromedp.Flag("metrics-recording-only", true),
			chromedp.Flag("safebrowsing-disable-auto-update", true),
			chromedp.Flag("enable-automation", true),
			chromedp.Flag("password-store", "basic"),
			chromedp.Flag("use-mock-keychain", true),
		)
		//defer pcancel()
		cctx, childCancel = chromedp.NewContext(pctx, chromedp.WithLogf(logger.SugaredLogger.Infof))
		//defer cancel()
		err := chromedp.Run(cctx, chromedp.Navigate(url),
			chromedp.WaitVisible(waitVisible, chromedp.ByQuery), // 确保  元素可见
			chromedp.WaitReady(waitVisible, chromedp.ByQuery),   // 确保  元素准备好
			chromedp.InnerHTML("body", &htmlContent),
		)
		if err != nil {
			logger.SugaredLogger.Error(err.Error())
			return "", false, parentCancel, childCancel
		}
	} else {
		cctx, childCancel = chromedp.NewContext(c.crawlerCtx, chromedp.WithLogf(logger.SugaredLogger.Infof))
		//defer cancel()
		err := chromedp.Run(cctx, chromedp.Navigate(url), chromedp.WaitVisible("body"), chromedp.InnerHTML("body", &htmlContent))
		if err != nil {
			logger.SugaredLogger.Error(err.Error())
			return "", false, parentCancel, childCancel
		}
	}
	return htmlContent, true, parentCancel, childCancel

}

func (c *CrawlerApi) GetHtmlWithActions(actions *[]chromedp.Action, headless bool) (string, bool) {
	htmlContent := ""
	*actions = append(*actions, chromedp.InnerHTML("body", &htmlContent))

	path, e := checkBrowserOnWindows()
	logger.SugaredLogger.Infof("GetHtmlWithActions path:%s", path)
	if e {
		pctx, pcancel := chromedp.NewExecAllocator(
			c.crawlerCtx,
			chromedp.ExecPath(path),
			chromedp.Flag("headless", headless),
			chromedp.Flag("blink-settings", "imagesEnabled=false"),
			chromedp.Flag("disable-javascript", false),
			chromedp.Flag("disable-gpu", true),
			chromedp.UserAgent(c.crawlerBaseInfo.Headers["User-Agent"]),
			chromedp.Flag("disable-background-networking", true),
			chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
			chromedp.Flag("disable-background-timer-throttling", true),
			chromedp.Flag("disable-backgrounding-occluded-windows", true),
			chromedp.Flag("disable-breakpad", true),
			chromedp.Flag("disable-client-side-phishing-detection", true),
			chromedp.Flag("disable-default-apps", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
			chromedp.Flag("disable-hang-monitor", true),
			chromedp.Flag("disable-ipc-flooding-protection", true),
			chromedp.Flag("disable-popup-blocking", true),
			chromedp.Flag("disable-prompt-on-repost", true),
			chromedp.Flag("disable-renderer-backgrounding", true),
			chromedp.Flag("disable-sync", true),
			chromedp.Flag("force-color-profile", "srgb"),
			chromedp.Flag("metrics-recording-only", true),
			chromedp.Flag("safebrowsing-disable-auto-update", true),
			chromedp.Flag("enable-automation", true),
			chromedp.Flag("password-store", "basic"),
			chromedp.Flag("use-mock-keychain", true),
		)
		defer pcancel()
		ctx, cancel := chromedp.NewContext(pctx, chromedp.WithLogf(logger.SugaredLogger.Infof))
		defer cancel()

		err := chromedp.Run(ctx, *actions...)
		if err != nil {
			logger.SugaredLogger.Error(err.Error())
			return "", false
		}
	} else {
		ctx, cancel := chromedp.NewContext(c.crawlerCtx, chromedp.WithLogf(logger.SugaredLogger.Infof))
		defer cancel()

		err := chromedp.Run(ctx, *actions...)
		if err != nil {
			logger.SugaredLogger.Error(err.Error())
			return "", false
		}
	}

	return htmlContent, true
}

type CrawlerBaseInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	BaseUrl     string            `json:"base_url"`
	Headers     map[string]string `json:"headers"`
}

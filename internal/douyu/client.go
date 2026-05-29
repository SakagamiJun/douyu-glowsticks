package douyu

import (
	"errors"
	"log/slog"

	"douyu-glowsticks/internal/config"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Client 封装了请求斗鱼接口的基础能力
type Client struct {
	http           tls_client.HttpClient
	cookies        []config.Cookie
	onCookieUpdate func([]config.Cookie)
}

// NewClient 初始化一个带有 Cookie 的 HTTP 客户端
func NewClient(cookies []config.Cookie, onCookieUpdate func([]config.Cookie)) (*Client, error) {
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(15),
		tls_client.WithClientProfile(profiles.Chrome_120),
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, err
	}

	return &Client{
		http:           c,
		cookies:        cookies,
		onCookieUpdate: onCookieUpdate,
	}, nil
}

// UpdateCookies 运行时更新 Client 的 Cookie
func (c *Client) UpdateCookies(newCookies []config.Cookie) {
	c.cookies = newCookies
}

// Do 封装原生的 fhttp Do 请求，自动携带 Headers 和处理 Set-Cookie
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("nil HTTP request")
	}

	// 设置全局伪装 Header
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", browserUserAgent)
	}
	if req.Header.Get("Referer") == "" {
		req.Header.Set("Referer", "https://www.douyu.com/")
	}
	if req.Header.Get("Origin") == "" {
		req.Header.Set("Origin", "https://www.douyu.com")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	}
	if req.Header.Get("Sec-Fetch-Site") == "" {
		req.Header.Set("Sec-Fetch-Site", "same-origin")
	}
	if req.Header.Get("Sec-Fetch-Mode") == "" {
		req.Header.Set("Sec-Fetch-Mode", "cors")
	}
	if req.Header.Get("Sec-Fetch-Dest") == "" {
		req.Header.Set("Sec-Fetch-Dest", "empty")
	}
	if req.Header.Get("Connection") == "" {
		req.Header.Set("Connection", "keep-alive")
	}

	host := ""
	path := "/"
	if req.URL != nil {
		host = req.URL.Hostname()
		if req.URL.EscapedPath() != "" {
			path = req.URL.EscapedPath()
		}
	}

	// 注入 Cookie
	if cookieHeader := cookiesToHeader(c.cookies, host, path); cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	} else {
		req.Header.Del("Cookie")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	// 统一响应拦截：自动合并新下发的 Cookie
	httpCookies := resp.Cookies()
	if len(httpCookies) > 0 {
		host := "www.douyu.com"
		if req.URL != nil && req.URL.Hostname() != "" {
			host = req.URL.Hostname()
		}

		mergedCookies, changed := MergeHttpCookies(c.cookies, httpCookies, host)
		if changed {
			slog.Info("API 响应拦截：检测到 Cookie 更新，触发合并...")
			c.UpdateCookies(mergedCookies)
			if c.onCookieUpdate != nil {
				c.onCookieUpdate(mergedCookies)
			}
		}
	}

	return resp, nil
}

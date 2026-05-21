package douyu

import (
	"github.com/go-resty/resty/v2"
)

// Client 封装了请求斗鱼接口的基础能力
type Client struct {
	http   *resty.Client
	cookie string
}

// NewClient 初始化一个带有 Cookie 和标准 Header 的 HTTP 客户端
func NewClient(cookie string) *Client {
	c := resty.New()
	// 全套伪装 Headers
	c.SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	c.SetHeader("Referer", "https://www.douyu.com/")
	c.SetHeader("Origin", "https://www.douyu.com")
	c.SetHeader("Accept", "application/json, text/plain, */*")
	c.SetHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	c.SetHeader("Sec-Fetch-Site", "same-origin")
	c.SetHeader("Sec-Fetch-Mode", "cors")
	c.SetHeader("Sec-Fetch-Dest", "empty")
	c.SetHeader("Connection", "keep-alive")
	c.SetHeader("Cookie", cookie)

	return &Client{
		http:   c,
		cookie: cookie,
	}
}

// UpdateCookie 运行时更新 Client 的 Cookie
func (c *Client) UpdateCookie(newCookie string) {
	c.cookie = newCookie
	c.http.SetHeader("Cookie", newCookie)
}

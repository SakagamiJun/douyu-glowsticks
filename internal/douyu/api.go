package douyu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type Room struct {
	RoomID     int
	AnchorName string
	ExpNeed    int
}

// CheckLogin 检查 Cookie 是否有效
func (c *Client) CheckLogin() bool {
	resp, err := c.http.R().Get("https://www.douyu.com/wgapi/livenc/liveweb/follow/list")
	if err != nil {
		slog.Error("登录检查请求失败", "error", err)
		return false
	}

	var result struct {
		Error int `json:"error"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return false
	}

	if result.Error == 0 {
		slog.Info("Cookie有效, 登陆成功")
		return true
	}
	slog.Error("登陆失败, 请检查Cookie有效性")
	return false
}

// ClaimGifts 模拟浏览器访问获取每日荧光棒，并提取最新的 Cookie 实现续命
func (c *Client) ClaimGifts() (string, error) {
	slog.Info("------正在获取荧光棒并尝试刷新Cookie------")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"), // 核心防风控：抹除 WebDriver 痕迹
		chromedp.WindowSize(1920, 1080),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var newCookies []*network.Cookie

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, cookie := range strings.Split(c.cookie, ";") {
				parts := strings.SplitN(strings.TrimSpace(cookie), "=", 2)
				if len(parts) == 2 {
					network.SetCookie(parts[0], parts[1]).WithDomain(".douyu.com").Do(ctx)
				}
			}
			return nil
		}),
		chromedp.Navigate("https://www.douyu.com/1"),
		chromedp.Sleep(5*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			newCookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)

	if err != nil {
		return "", fmt.Errorf("无头浏览器访问失败: %w", err)
	}

	// 将提取出的 Cookie 对象重新组装为长字符串
	var newCookieParts []string
	for _, cookie := range newCookies {
		newCookieParts = append(newCookieParts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}

	newCookieString := strings.Join(newCookieParts, "; ")
	return newCookieString, nil
}

// GetOwnedGifts 获取当前背包中的荧光棒数量
func (c *Client) GetOwnedGifts() int {
	slog.Info("------背包检查开始------")
	resp, err := c.http.R().Get("https://www.douyu.com/japi/prop/backpack/web/v1?rid=12306")
	if err != nil {
		slog.Error("获取背包失败", "error", err)
		return 0
	}

	var result struct {
		Error int `json:"error"`
		Data  struct {
			List []struct {
				ID    int `json:"id"`
				Count int `json:"count"`
			} `json:"list"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil || result.Error != 0 {
		slog.Error("解析背包失败", "error", result.Error)
		return 0
	}

	for _, item := range result.Data.List {
		if item.ID == 268 {
			slog.Info(fmt.Sprintf("当前拥有荧光棒 %d 个, 给你喜欢的主播进行赠送吧", item.Count))
			slog.Info("------背包检查结束------")
			return item.Count
		}
	}
	slog.Warn("当前背包中没有任何道具")
	slog.Info("------背包检查结束------")
	return 0
}

// GetRoomList 获取粉丝勋章房间列表
func (c *Client) GetRoomList() ([]Room, error) {
	resp, err := c.http.R().Get("https://www.douyu.com/member/cp/getFansBadgeList")
	if err != nil {
		return nil, fmt.Errorf("获取粉丝勋章页面失败: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 失败: %w", err)
	}

	var rooms []Room
	doc.Find(".fans-badge-list > tbody > tr").Each(func(i int, s *goquery.Selection) {
		roomIDStr, exists := s.Attr("data-fans-room")
		if !exists {
			return
		}
		roomID, _ := strconv.Atoi(roomIDStr)

		anchorName := strings.TrimSpace(s.Find(".anchor--name").Text())

		// 解析经验值，HTML里的格式可能类似于: 1234 / 5000 或者带空格
		expText := s.Find("td").Eq(2).Text()
		expText = strings.ReplaceAll(expText, " ", "") // 丢弃所有空格
		parts := strings.Split(expText, "/")
		expNeed := 0
		if len(parts) == 2 {
			expNow, _ := strconv.ParseFloat(parts[0], 64)
			expUp, _ := strconv.ParseFloat(parts[1], 64)
			expNeed = int(expUp - expNow)
		}

		rooms = append(rooms, Room{
			RoomID:     roomID,
			AnchorName: anchorName,
			ExpNeed:    expNeed,
		})
	})

	return rooms, nil
}

// Donate 赠送荧光棒
func (c *Client) Donate(count int, roomID int) bool {
	donateUrl := "https://www.douyu.com/japi/prop/donate/mainsite/v2"
	data := fmt.Sprintf(`propId=268&propCount=%d&roomId=%d&bizExt={"yzxq":{}}`, count, roomID)

	resp, err := c.http.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(data).
		Post(donateUrl)

	if err != nil {
		slog.Error("赠送请求失败", "roomId", roomID, "error", err)
		return false
	}

	var result struct {
		Error int    `json:"error"`
		Msg   string `json:"msg"`
	}
	json.Unmarshal(resp.Body(), &result)

	if result.Error == 0 {
		slog.Info(fmt.Sprintf("向房间号 %d 赠送荧光棒 %d 个成功", roomID, count))
		return true
	} else {
		slog.Error(fmt.Sprintf("向房间号 %d 赠送荧光棒失败, 原因: %s", roomID, result.Msg))
		return false
	}
}

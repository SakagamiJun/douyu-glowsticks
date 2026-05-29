package douyu

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SakagamiJun/douyu-glowsticks/internal/config"

	"github.com/PuerkitoBio/goquery"
	http "github.com/bogdanfinn/fhttp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Room describes a room with a fan badge that can receive glowsticks.
type Room struct {
	RoomID     int
	AnchorName string
	ExpNeed    int
}

// CheckLogin 检查 Cookie 是否有效
func (c *Client) CheckLogin() bool {
	req, err := http.NewRequest("GET", "https://www.douyu.com/wgapi/livenc/liveweb/follow/list", nil)
	if err != nil {
		slog.Error("创建登录检查请求失败", "error", err)
		return false
	}
	resp, err := c.Do(req)
	if err != nil {
		slog.Error("登录检查请求失败", "error", err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("读取登录检查响应失败", "error", err)
		return false
	}
	var result struct {
		Error int `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("解析登录检查响应失败", "error", err)
		return false
	}

	if result.Error == 0 {
		slog.Info("Cookie有效, 登陆成功")
		return true
	}
	slog.Error("登陆失败, 请检查Cookie有效性")
	return false
}

// InteractiveLogin opens a visible browser and captures cookies after login.
func (c *Client) InteractiveLogin() ([]config.Cookie, error) {
	slog.Info("启动可视化浏览器，请在弹出的窗口中扫码或输入密码登录斗鱼...")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.UserAgent(browserUserAgent),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var newCookies []*network.Cookie

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return network.ClearBrowserCookies().Do(ctx)
		}),
		chromedp.Navigate("https://passport.douyu.com/index/login"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Info("【注意】请在弹出的浏览器窗口中手动完成登录。")
			slog.Info(">>> 程序会自动检测登录 Cookie；登录成功后也可以按【回车键 (Enter)】立即检查。 <<<")

			enterPressed := make(chan struct{}, 1)
			go waitForEnter(enterPressed)

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				var err error
				newCookies, err = network.GetCookies().Do(ctx)
				if err != nil {
					return err
				}
				if hasLoginCookie(newCookies) {
					slog.Info("检测到登录 Cookie，开始保存浏览器 Cookie。")
					return nil
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-enterPressed:
					newCookies, err = network.GetCookies().Do(ctx)
					if err != nil {
						return err
					}
					if hasLoginCookie(newCookies) {
						return nil
					}
					return fmt.Errorf("未检测到登录 Cookie acf_uid，请确认已完成登录")
				case <-ticker.C:
				}
			}
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("可视化登录失败: %w", err)
	}

	return networkCookiesToConfig(newCookies), nil
}

// ClaimGifts 模拟浏览器访问获取每日荧光棒，并提取最新的 Cookie 实现续命
func (c *Client) ClaimGifts() ([]config.Cookie, error) {
	slog.Info("------正在获取荧光棒并尝试刷新Cookie------")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(browserUserAgent),
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
			if len(c.cookies) == 0 {
				return nil
			}
			for _, cookie := range c.cookies {
				domain := cookie.Domain
				path := cookie.Path
				if path == "" {
					path = "/"
				}
				if err := network.SetCookie(cookie.Name, cookie.Value).WithDomain(domain).WithPath(path).Do(ctx); err != nil {
					return fmt.Errorf("设置浏览器 Cookie %s 失败: %w", cookie.Name, err)
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
		return nil, fmt.Errorf("无头浏览器访问失败: %w", err)
	}

	return networkCookiesToConfig(newCookies), nil
}

// GetOwnedGifts 获取当前背包中的荧光棒数量
func (c *Client) GetOwnedGifts() int {
	slog.Info("------背包检查开始------")
	req, err := http.NewRequest("GET", "https://www.douyu.com/japi/prop/backpack/web/v1?rid=12306", nil)
	if err != nil {
		slog.Error("创建背包请求失败", "error", err)
		return 0
	}
	resp, err := c.Do(req)
	if err != nil {
		slog.Error("获取背包失败", "error", err)
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("读取背包响应失败", "error", err)
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

	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("解析背包响应失败", "error", err)
		return 0
	}
	if result.Error != 0 {
		slog.Error("背包接口返回错误", "error", result.Error)
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
	req, err := http.NewRequest("GET", "https://www.douyu.com/member/cp/getFansBadgeList", nil)
	if err != nil {
		return nil, fmt.Errorf("创建粉丝勋章请求失败: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取粉丝勋章页面失败: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 失败: %w", err)
	}

	var rooms []Room
	doc.Find(".fans-badge-list > tbody > tr").Each(func(i int, s *goquery.Selection) {
		roomIDStr, exists := s.Attr("data-fans-room")
		if !exists {
			return
		}
		roomID, err := strconv.Atoi(roomIDStr)
		if err != nil {
			slog.Warn("解析房间号失败", "room", roomIDStr, "error", err)
			return
		}

		anchorName := strings.TrimSpace(s.Find(".anchor--name").Text())

		// 解析经验值，HTML里的格式可能类似于: 1234 / 5000 或者带空格
		expText := s.Find("td").Eq(2).Text()
		expText = strings.ReplaceAll(expText, " ", "") // 丢弃所有空格
		parts := strings.Split(expText, "/")
		expNeed := 0
		if len(parts) == 2 {
			expNow, errNow := strconv.ParseFloat(parts[0], 64)
			expUp, errUp := strconv.ParseFloat(parts[1], 64)
			if errNow == nil && errUp == nil {
				expNeed = int(expUp - expNow)
			}
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

	req, err := http.NewRequest("POST", donateUrl, strings.NewReader(data))
	if err != nil {
		slog.Error("创建赠送请求失败", "roomId", roomID, "error", err)
		return false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.Do(req)
	if err != nil {
		slog.Error("赠送请求失败", "roomId", roomID, "error", err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("读取赠送响应失败", "roomId", roomID, "error", err)
		return false
	}
	var result struct {
		Error int    `json:"error"`
		Msg   string `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("解析赠送响应失败", "roomId", roomID, "error", err)
		return false
	}

	if result.Error == 0 {
		slog.Info(fmt.Sprintf("向房间号 %d 赠送荧光棒 %d 个成功", roomID, count))
		return true
	} else {
		slog.Error(fmt.Sprintf("向房间号 %d 赠送荧光棒失败, 原因: %s", roomID, result.Msg))
		return false
	}
}

func waitForEnter(done chan<- struct{}) {
	if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
		if err != io.EOF {
			slog.Warn("读取终端输入失败，将继续自动检测登录状态", "error", err)
		}
		return
	}

	select {
	case done <- struct{}{}:
	default:
	}
}

func hasLoginCookie(cookies []*network.Cookie) bool {
	for _, c := range cookies {
		if c.Name == "acf_uid" {
			return true
		}
	}
	return false
}

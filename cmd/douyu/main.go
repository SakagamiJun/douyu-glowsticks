package main

import (
	"log/slog"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/SakagamiJun/douyu-glowsticks/internal/config"
	"github.com/SakagamiJun/douyu-glowsticks/internal/douyu"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("开始启动斗鱼荧光棒赠送程序...")

	// 网络检查逻辑
	if !waitForNetwork() {
		slog.Error("网络检查失败，重试次数耗尽，程序退出")
		os.Exit(1)
	}

	cfg, err := config.Load("config.json")
	if err != nil {
		slog.Error("配置加载失败", "error", err)
		os.Exit(1)
	}

	client, err := douyu.NewClient(cfg.Cookies, func(newCookies []config.Cookie) {
		cfg.Cookies = newCookies
		if err := cfg.Save("config.json"); err != nil {
			slog.Error("自动保存新 Cookie 失败", "error", err)
		} else {
			slog.Info("自动保存新 Cookie 成功！")
		}
	})
	if err != nil {
		slog.Error("初始化斗鱼客户端失败", "error", err)
		os.Exit(1)
	}

	slog.Info("------登录检查开始------")
	var isLogin bool
	var loginErr error
	maxRetries := 3
	for i := 0; i <= maxRetries; i++ {
		isLogin, loginErr = client.CheckLogin()
		if loginErr == nil {
			break // 网络请求成功（无论是否登录过期），跳出重试循环
		}
		if i < maxRetries {
			slog.Warn("登录检查网络请求失败，等待重试", "retry", i+1, "maxRetries", maxRetries, "wait", "5m")
			time.Sleep(5 * time.Minute)
		}
	}
	slog.Info("------登录检查结束------")

	if loginErr != nil {
		slog.Error("登录检查经过多次重试依然失败，疑似网络环境异常，任务中止", "error", loginErr)
		os.Exit(1)
	}

	if !isLogin {
		slog.Warn("未登录或 Cookie 已失效，准备唤起浏览器进行可视化扫码登录...")
		newCookies, err := client.InteractiveLogin()
		if err != nil || len(newCookies) == 0 {
			slog.Error("可视化登录失败，任务终止", "error", err)
			os.Exit(1)
		}

		mergedCookies, changed := douyu.MergeRawCookies(cfg.Cookies, newCookies)
		client.UpdateCookies(mergedCookies)
		if changed {
			slog.Info("检测到可视化登录 Cookie 生成，正在持久化到本地...")
			cfg.Cookies = mergedCookies
			if err := cfg.Save("config.json"); err != nil {
				slog.Error("保存新 Cookie 失败", "error", err)
			} else {
				slog.Info("新 Cookie 已成功保存，恢复正常执行流程！")
			}
		}
	}

	// 1. 获取每日荧光棒，并捕获刷新的 Cookie
	newCookies, err := client.ClaimGifts()
	if err != nil {
		slog.Error("获取每日荧光棒失败", "error", err)
	} else if len(newCookies) > 0 {
		mergedCookies, changed := douyu.MergeRawCookies(cfg.Cookies, newCookies)
		if changed {
			slog.Info("检测到 Cookie 发生了实质性刷新，正在持久化到本地...")
			cfg.Cookies = mergedCookies
			if err := cfg.Save("config.json"); err != nil {
				slog.Error("保存新 Cookie 失败", "error", err)
			} else {
				slog.Info("新 Cookie 已成功保存，实现自动续命！")
				client.UpdateCookies(mergedCookies) // 让后续请求也用新Cookie
			}
		}
	}

	// 2. 检查背包
	own := client.GetOwnedGifts()
	if own == 0 {
		slog.Warn("背包中没有荧光棒,无法执行赠送,任务即将结束")
		return
	}

	slog.Info("当前选择模式为: 平均分配模式")

	// 3. 获取房间列表
	rooms, err := client.GetRoomList()
	if err != nil {
		slog.Error("获取房间列表失败", "error", err)
		return
	}

	if len(rooms) == 0 {
		slog.Warn("没有找到任何带有粉丝勋章的房间")
		return
	}

	// 4. 计算分配并赠送 (修复原版的数学分配漏洞)
	everyGive := own / len(rooms)
	remainder := own % len(rooms)

	slog.Info("------开始捐赠荧光棒------")
	attemptedDonations := 0
	failedRooms := make([]int, 0)
	for i, room := range rooms {
		countToGive := everyGive
		// 将余数依次分配给前面的房间，确保总数绝对精确且不会出现负数
		if i < remainder {
			countToGive++
		}

		if countToGive > 0 {
			attemptedDonations++
			if !client.Donate(countToGive, room.RoomID) {
				failedRooms = append(failedRooms, room.RoomID)
			}

			// 防风控：随机休眠 2~5 秒，模拟真人操作（如果不是最后一个需要送礼的房间）
			if i < len(rooms)-1 {
				sleepTime := time.Duration(rand.Intn(4)+2) * time.Second
				slog.Info("伪装休眠中...", "duration", sleepTime)
				time.Sleep(sleepTime)
			}
		}
	}
	slog.Info("------荧光棒捐赠结束------")
	if len(failedRooms) > 0 {
		slog.Error("部分房间赠送失败，任务以失败状态结束", "failed", len(failedRooms), "attempted", attemptedDonations, "rooms", failedRooms)
		os.Exit(1)
	}

	// 5. 打印升级所需经验
	for _, room := range rooms {
		slog.Info("升级经验状态", "room", room.RoomID, "anchor", room.AnchorName, "need_exp", room.ExpNeed)
	}
}

// waitForNetwork 检查网络连接，失败则重试
func waitForNetwork() bool {
	for i := 1; i <= 5; i++ {
		// 尝试连接斗鱼首页，超时时间 5s
		conn, err := net.DialTimeout("tcp", "www.douyu.com:443", 5*time.Second)
		if err == nil {
			conn.Close()
			slog.Info("网络检查通过")
			return true
		}

		slog.Warn("网络连接异常，将在 120s 后重试", "attempt", i, "max_attempts", 5, "error", err)
		if i < 5 {
			time.Sleep(120 * time.Second)
		}
	}
	return false
}

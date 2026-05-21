package main

import (
	"log/slog"
	"math/rand"
	"os"
	"time"

	"douyu-glowsticks/internal/config"
	"douyu-glowsticks/internal/douyu"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("开始启动斗鱼荧光棒赠送程序...")

	cfg, err := config.Load("config.json")
	if err != nil {
		slog.Error("配置加载失败", "error", err)
		os.Exit(1)
	}

	client := douyu.NewClient(cfg.Cookie)

	slog.Info("------登录检查开始------")
	isLogin := client.CheckLogin()
	slog.Info("------登录检查结束------")

	if !isLogin {
		slog.Error("未登录，任务终止")
		os.Exit(1)
	}

	// 1. 获取每日荧光棒，并捕获刷新的 Cookie
	newCookie, err := client.ClaimGifts()
	if err != nil {
		slog.Error("获取每日荧光棒失败", "error", err)
	} else if newCookie != "" && newCookie != cfg.Cookie {
		slog.Info("检测到 Cookie 发生了刷新，正在持久化到本地...")
		cfg.Cookie = newCookie
		if err := cfg.Save("config.json"); err != nil {
			slog.Error("保存新 Cookie 失败", "error", err)
		} else {
			slog.Info("新 Cookie 已成功保存，实现自动续命！")
			client.UpdateCookie(newCookie) // 让后续请求也用新Cookie
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
	for i, room := range rooms {
		countToGive := everyGive
		// 将余数依次分配给前面的房间，确保总数绝对精确且不会出现负数
		if i < remainder {
			countToGive++
		}

		if countToGive > 0 {
			client.Donate(countToGive, room.RoomID)

			// 防风控：随机休眠 2~5 秒，模拟真人操作（如果不是最后一个需要送礼的房间）
			if i < len(rooms)-1 {
				sleepTime := time.Duration(rand.Intn(4)+2) * time.Second
				slog.Info("伪装休眠中...", "duration", sleepTime)
				time.Sleep(sleepTime)
			}
		}
	}
	slog.Info("------荧光棒捐赠结束------")

	// 5. 打印升级所需经验
	for _, room := range rooms {
		slog.Info("升级经验状态", "room", room.RoomID, "anchor", room.AnchorName, "need_exp", room.ExpNeed)
	}
}

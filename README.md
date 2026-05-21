# 斗鱼荧光棒自动赠送工具 (Douyu Glowsticks)

![Go Version](https://img.shields.io/github/go-mod/go-version/SakagamiJun/douyu-glowsticks)
![License](https://img.shields.io/github/license/SakagamiJun/douyu-glowsticks)
![Release](https://img.shields.io/github/v/release/SakagamiJun/douyu-glowsticks)

这是一个基于 Go 语言开发的斗鱼直播平台荧光棒自动赠送工具。它可以帮助你自动领取每日荧光棒，并平均分配给你拥有粉丝勋章的直播间，实现粉丝勋章的自动续命。

## 功能特性

- **自动领取**：每日自动领取背包荧光棒。
- **自动续命**：检测到 Cookie 刷新时自动更新本地配置，实现长期自动运行。
- **智能分配**：将荧光棒平均分配给所有已加入粉丝团的房间。
- **配置自建**：首次运行自动生成配置文件模板。
- **跨平台支持**：支持 Windows, macOS, Linux。
- **定时任务**：支持通过 macOS `launchd` 或其他系统的 Cron 实现定时运行。

## 快速开始

### 1. 下载

在 [Releases](https://github.com/SakagamiJun/douyu-glowsticks/releases) 页面下载对应系统的二进制文件。

### 2. 运行

首次运行程序：

```bash
./douyu-gift
```

程序会提示 `config.json` 不存在并自动创建一个模板。

### 3. 配置

编辑生成的 `config.json` 文件：

```json
{
  "cookie": "你的真实斗鱼 Cookie",
  "push_key": "可选：微信推送 Key （未开发）"
}
```

> **如何获取 Cookie？**
> 1. 在浏览器登录斗鱼。
> 2. 按 F12 打开开发者工具。
> 3. 在“网络 (Network)”标签下随便找一个请求，在请求头中找到 `cookie` 字段并复制。

### 4. 再次运行

配置完成后再次运行，程序将开始执行任务。

## 自动化运行 (macOS)

本项目附带了 `com.douyu.glowsticks.plist` 模板，可用于设置每天 00:05 自动运行。

1. 修改 `.plist` 文件中的路径为你的实际路径。
2. 复制到 `~/Library/LaunchAgents/`。
3. 运行 `launchctl load ~/Library/LaunchAgents/com.douyu.glowsticks.plist`
4. 运行 `launchctl start com.douyu.glowsticks` 立即测试。

## 开发

如果你想自行编译：

```bash
# 克隆仓库
git clone git@github.com:SakagamiJun/douyu-glowsticks.git
cd douyu-glowsticks

# 编译
go build -o douyu-gift ./cmd/douyu/main.go
```

## 免责声明

本工具仅供学习交流使用，请勿用于非法用途。作者不对因使用本工具导致的任何账号问题负责。

## 开源协议

基于 [MIT License](LICENSE) 开源。

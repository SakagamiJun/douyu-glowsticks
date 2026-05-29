# 斗鱼荧光棒自动赠送工具 (Douyu Glowsticks)

![Go Version](https://img.shields.io/github/go-mod/go-version/SakagamiJun/douyu-glowsticks)
[![Go Report Card](https://goreportcard.com/badge/github.com/SakagamiJun/douyu-glowsticks)](https://goreportcard.com/report/github.com/SakagamiJun/douyu-glowsticks)
![License](https://img.shields.io/github/license/SakagamiJun/douyu-glowsticks)
![Release](https://img.shields.io/github/v/release/SakagamiJun/douyu-glowsticks)

这是一个基于 Go 语言开发的斗鱼直播平台荧光棒自动赠送工具。它可以帮助你自动领取每日荧光棒，并平均分配给你拥有粉丝勋章的直播间，实现粉丝勋章的自动续命。

## 功能特性

- **可视化登录**：检测到未登录时自动唤起 Chrome 浏览器，扫码即可完成登录并自动抓取全量 Cookie。
- **底层抗风控**：内置 `tls-client` 模拟真实浏览器指纹，有效绕过斗鱼对 JA3/TLS 指纹的监测。
- **结构化管理**：Cookie 采用结构化 JSON 存储，支持多域名（如关联登录）及路径精确匹配，彻底解决鉴权丢失问题。
- **自动续命**：API 请求级别拦截并合并 `Set-Cookie` 响应，实现长期免维护自动运行。
- **智能分配**：将荧光棒精确平均分配给所有已加入粉丝团的房间。

## 快速开始

### 1. 下载

在 [Releases](https://github.com/SakagamiJun/douyu-glowsticks/releases) 页面下载对应系统的二进制文件。

### 2. 运行

首次运行程序：

```bash
./douyu-gift
```

程序会提示 `config.json` 不存在并自动创建一个模板。

### 3. 配置与登录

程序支持两种登录方式：

**方式 A：交互式登录（推荐）**
1. 直接运行程序，若检测到未登录，程序会尝试唤起浏览器。
2. 在弹出的窗口中手动完成扫码或账号登录。
3. 登录成功后，程序会自动捕获全量 Cookie 并保存到 `config.json`，无需手动抓包。

**方式 B：手动编辑配置**
编辑生成的 `config.json` 文件：

```json
{
  "cookies": [
    {
      "name": "acf_uid",
      "value": "你的值",
      "domain": ".douyu.com",
      "path": "/"
    }
  ],
  "push_key": "可选：微信推送 Key"
}
```

> **提示**：程序也兼容旧版的单字符串格式（如 `"cookie": "acf_uid=xxx; ..."`），首次运行时会自动迁移为最新的结构化格式。

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

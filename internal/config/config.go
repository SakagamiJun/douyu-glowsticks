package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Cookie stores one structured browser cookie entry.
type Cookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

// Config stores the local runtime configuration.
type Config struct {
	Cookie  string   `json:"cookie,omitempty"`
	Cookies []Cookie `json:"cookies"`
	PushKey string   `json:"push_key"`
}

// Load reads configuration from path and migrates legacy Cookie strings.
func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，创建一个范例配置文件
			template := &Config{
				Cookie:  "请在这里填入你的真实斗鱼 Cookie",
				PushKey: "可选：填入微信推送 Key",
			}
			if saveErr := template.Save(path); saveErr != nil {
				return nil, fmt.Errorf("配置文件不存在，且创建范例文件失败: %w", saveErr)
			}
			return nil, fmt.Errorf("配置文件 %s 不存在，已为您创建范例文件，请填写后再运行", path)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 兼容旧版本的单字符串 Cookie 配置
	if len(cfg.Cookies) == 0 && cfg.Cookie != "" && cfg.Cookie != "请在这里填入你的真实斗鱼 Cookie" {
		cfg.Cookies = ParseRawCookie(cfg.Cookie)
		cfg.Cookie = "" // 清空旧的字段
		// 自动保存为新格式
		if err := cfg.Save(path); err != nil {
			return nil, fmt.Errorf("迁移配置为结构化 Cookie 后保存失败: %w", err)
		}
	}

	return &cfg, nil
}

// ParseRawCookie parses a legacy Cookie header string into structured cookies.
func ParseRawCookie(raw string) []Cookie {
	var cookies []Cookie
	for _, part := range strings.Split(raw, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			cookies = append(cookies, Cookie{
				Name:   kv[0],
				Value:  kv[1],
				Domain: ".douyu.com", // 默认作用域
				Path:   "/",
			})
		}
	}
	return cookies
}

// Save 将当前配置安全地写回到本地 JSON 文件
func (cfg *Config) Save(path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

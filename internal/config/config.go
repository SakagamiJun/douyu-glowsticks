package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Cookie  string `json:"cookie"`
	PushKey string `json:"push_key"`
}

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

	if cfg.Cookie == "" || cfg.Cookie == "请在这里填入你的真实斗鱼 Cookie" {
		return nil, fmt.Errorf("请先在 config.json 中填入真实的 Cookie")
	}

	return &cfg, nil
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

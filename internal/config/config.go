package config

import (
	"encoding/json"
	"os"
)

type TokenConfig struct {
	Address string  `json:"address"`
	ChainId string  `json:"chainId,omitempty"` // 允许为空, 默认 56
	Name    string  `json:"name,omitempty"`    // 允许为空, 默认取地址后四位
	Min     float64 `json:"min,omitempty"`     // 允许为空, 默认 0.1
	Max     float64 `json:"max,omitempty"`     // 允许为空, 默认不限
}

type WebhookConfig struct {
	Wecom          string `json:"wecom,omitempty"`            // 允许为空
	Lark           string `json:"lark,omitempty"`             // 允许为空
	TelegramToken  string `json:"telegram_token,omitempty"`   // 允许为空
	TelegramChatId string `json:"telegram_chat_id,omitempty"` // 允许为空
}

type AppConfig struct {
	Webhook  WebhookConfig `json:"webhook"`
	Interval int           `json:"interval,omitempty"` // 允许为空, 默认 30s
	Tokens   []TokenConfig `json:"tokens"`
}

// 读取 config.json 文件
func LoadConfig() (*AppConfig, error) {
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		// 文件不存在，写入示例文件
		err := WriteConfig()
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	// 文件存在，读取并解析配置
	data, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	var config AppConfig
	// 解析 JSON 数据
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// 写入 config.json 示例文件
func WriteConfig() error {
	configStr := `{
  "webhook": {
    "wecom": "",
    "lark": "",
    "telegram_token": "",
    "telegram_chat_id": ""
  },
  "interval": 30,
  "tokens": [
    {
      "address": "0x1234567890abcdef1234567890abcdef12345678",
      "chainId": "56",
      "name": "MyWallet01",
      "min": 0.1,
      "max": 1000
    },
    {
      "address": "0x1234567890abcdef1234567890abcdef12345678",
      "chainId": "56",
      "name": "MyWallet01",
      "min": 0.1,
      "max": 1000
    }
  ]
}`
	return os.WriteFile("config.json", []byte(configStr), 0644)
}

package config

import (
	"encoding/json"
	"os"
	"sync"
	"time"
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

type HealthCheckConfig struct {
	Interval  int `json:"interval,omitempty"`  // 允许为空, 默认 10s
	WarnCount int `json:"warnCount,omitempty"` // 警告次数 允许为空, 默认 3 次
}

type VolumeMonitorConfig struct {
	NotifyCount int                     `json:"NotifyCount,omitempty"` // 通知次数, 允许为空, 默认 3 次
	Platform    []VolumeMonitorPlatform `json:"platform"`              // 交易所列表
}

type VolumeMonitorPlatform struct {
	Platform     string  `json:"platform"`               // 交易所
	ThresholdUSD float64 `json:"thresholdUSD,omitempty"` // 24h交易量阈值，单位美元，小于该值告警 默认50w
}

type AppConfig struct {
	Webhook       WebhookConfig       `json:"webhook"`
	Interval      int                 `json:"interval,omitempty"` // 允许为空, 默认 30s
	Tokens        []TokenConfig       `json:"tokens"`
	HealthCheck   HealthCheckConfig   `json:"healthCheck"`
	VolumeMonitor VolumeMonitorConfig `json:"volumeMonitor"` // 交易量监控配置
}

// 缓存config, 5秒刷新一次
var configCache *AppConfig
var configMutex sync.RWMutex
var lastLoadTime int64

// 读取 config.json 文件
func LoadConfig() (*AppConfig, error) {
	configMutex.RLock()
	if configCache != nil && (time.Now().Unix()-lastLoadTime) < 5 {
		defer configMutex.RUnlock()
		return configCache, nil
	}
	configMutex.RUnlock()

	// 升级为写锁
	configMutex.Lock()
	defer configMutex.Unlock()

	// 双重检查，避免其他 goroutine 已经更新了配置
	if configCache != nil && (time.Now().Unix()-lastLoadTime) < 5 {
		return configCache, nil
	}

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
	if config.Interval == 0 {
		config.Interval = 30 // 默认 30 秒
	}
	if config.HealthCheck.Interval == 0 {
		config.HealthCheck.Interval = 10 // 默认 10 秒
	}
	if config.HealthCheck.WarnCount == 0 {
		config.HealthCheck.WarnCount = 3 // 默认 3 次
	}

	// 设置Token默认值
	for i := range config.Tokens {
		if config.Tokens[i].ChainId == "" {
			config.Tokens[i].ChainId = "56" // 默认BSC链
		}
		if config.Tokens[i].Min <= 0 {
			config.Tokens[i].Min = 0.1 // 默认最小值
		}
	}

	configCache = &config
	lastLoadTime = time.Now().Unix()

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
  ],
  "volumeMonitor": [
    {
      "platform": "gate",
      "thresholdUSD": 1000000
    },
    {
      "platform": "binance",
      "thresholdUSD": 10000000
    }
  ]
}`
	return os.WriteFile("config.json", []byte(configStr), 0644)
}

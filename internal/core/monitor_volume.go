package core

import (
	"strings"

	"github.com/fuxingjun/balance-bot/internal/config"
	"github.com/fuxingjun/balance-bot/internal/utils"
	"github.com/fuxingjun/balance-bot/pkg"
)

// --- 交易量监控相关 ---

type PerpTicker struct {
	symbol    string
	volume24h string
}

// 定义交易所检查函数的类型
type volumeChecker func([]string) ([]PerpTicker, error)

// 交易所方法配置映射
var volumeCheckers = map[string]volumeChecker{
	"gate":    checkGateVolume,
	"binance": checkBinanceVolume,
}

func checkVolumeMonitor(exchange string, symbols []string) {
	checker, exists := volumeCheckers[strings.ToLower(exchange)]
	if !exists {
		pkg.GetLogger().Debug("Unsupported exchange for volume monitor", "exchange", exchange)
		return
	}

	tickers, err := checker(symbols)
	if err != nil {
		pkg.GetLogger().Debug("Failed to get tickers", "exchange", exchange, "error", err)
		return
	}
	if len(tickers) == 0 {
		pkg.GetLogger().Debug("No matching tickers found", "exchange", exchange)
		return
	}

	var msgParts []string
	notifyCount := getNotifyCount()

	for _, ticker := range tickers {
		// 增加前缀区分不同监控类型的缓存
		cacheKey := "vol:" + strings.ToLower(exchange) + "_" + ticker.symbol
		count := 0
		if val, exists := notifyCache.Get(cacheKey); exists {
			count = val.(int)
		}

		if count >= notifyCount {
			pkg.GetLogger().Debug("Skipping notification for", "symbol", ticker.symbol, "count", count)
			continue
		}

		count++
		notifyCache.Set(cacheKey, count)
		msgParts = append(msgParts, "symbol: "+ticker.symbol+", 24h volume: "+ticker.volume24h)
	}

	if len(msgParts) > 0 {
		msg := "Volume too low on " + exchange + ":\n" + strings.Join(msgParts, "\n")
		pkg.GetLogger().Info("Sending volume alert", "exchange", exchange, "message", msg)
		utils.SendMessage(msg)
	}
}

func getNotifyCount() int {
	cfg, err := config.LoadConfig()
	if err != nil || cfg == nil {
		return 3
	}
	return cfg.VolumeMonitor.NotifyCount
}

// 寻找交易所的监控配置
func findVolumeMonitorConfig(exchange string) *config.VolumeMonitorPlatform {
	cfg, err := config.LoadConfig()
	if err != nil || cfg == nil {
		return nil
	}
	for _, monitor := range cfg.VolumeMonitor.Platform {
		if strings.EqualFold(monitor.Platform, exchange) {
			return &monitor
		}
	}
	return nil
}

type VolumeResponse struct {
	Symbol string `json:"contract"`
	Volume string `json:"volume_24h_settle"`
}

// 查询gate交易所的symbol 交易所数据
func checkGateVolume(symbols []string) ([]PerpTicker, error) {
	url := "https://api.gateio.ws/api/v4/futures/usdt/tickers"
	resp, err := pkg.SendGetRequestMarshal[[]VolumeResponse](pkg.GetHTTPClient(), url, nil, nil)
	if err != nil {
		pkg.GetLogger().Error("Failed to request gate symbols", "error", err)
		return nil, err
	}
	// 筛选出成交量比 ThresholdUSD 低的symbol
	symbolSet := make(map[string]struct{})
	for _, symbol := range symbols {
		symbolSet[symbol] = struct{}{}
	}
	var result []PerpTicker
	// 找到gate的交易量监控配置
	var thresholdUSD float64 = 500000 // 默认50w
	monitorCfg := findVolumeMonitorConfig("gate")
	if monitorCfg != nil && monitorCfg.ThresholdUSD > 0 {
		thresholdUSD = monitorCfg.ThresholdUSD
	}
	for _, ticker := range resp {
		if _, exists := symbolSet[ticker.Symbol]; exists {
			if vol := pkg.StringToFloat(ticker.Volume); vol < thresholdUSD {
				result = append(result, PerpTicker{
					symbol:    ticker.Symbol,
					volume24h: ticker.Volume,
				})
			}
		}
	}
	return result, nil
}

func checkBinanceVolume(symbols []string) ([]PerpTicker, error) {
	url := "https://fapi.binance.com/fapi/v1/ticker/24hr"
	resp, err := pkg.SendGetRequestMarshal[[]VolumeResponse](pkg.GetHTTPClient(), url, nil, nil)
	if err != nil {
		pkg.GetLogger().Error("Failed to request binance symbols", "error", err)
		return nil, err
	}
	// 筛选出成交量比 ThresholdUSD 低的symbol
	symbolSet := make(map[string]struct{})
	for _, symbol := range symbols {
		symbolSet[symbol] = struct{}{}
	}
	var result []PerpTicker
	// 找到binance的交易量监控配置
	var thresholdUSD float64 = 5000000 // 默认500w
	monitorCfg := findVolumeMonitorConfig("binance")
	if monitorCfg != nil && monitorCfg.ThresholdUSD > 0 {
		thresholdUSD = monitorCfg.ThresholdUSD
	}
	for _, ticker := range resp {
		if _, exists := symbolSet[ticker.Symbol]; exists {
			if vol := pkg.StringToFloat(ticker.Volume); vol < thresholdUSD {
				result = append(result, PerpTicker{
					symbol:    ticker.Symbol,
					volume24h: ticker.Volume,
				})
			}
		}
	}
	return result, nil
}

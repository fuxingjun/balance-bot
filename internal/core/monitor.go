package core

import (
	"encoding/json"
	"strings"

	"github.com/fuxingjun/balance-bot/internal/config"
	"github.com/fuxingjun/balance-bot/internal/utils"
	"github.com/fuxingjun/balance-bot/pkg"
	"github.com/gofiber/fiber/v2"
)

type SymbolInfo struct {
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
	Type     string `json:"type"`
}

type PairInfo struct {
	ID string     `json:"id"`
	TS int64      `json:"ts"`
	A  SymbolInfo `json:"a"`
	B  SymbolInfo `json:"b"`
}

func PairsMonitor(c *fiber.Ctx) error {
	var pairs []PairInfo
	// 先打印一下原始字符串
	pkg.GetLogger().Debug("Raw request body", "body", string(c.Body()))
	if err := c.BodyParser(&pairs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid payload",
		})
	}
	pkg.GetLogger().Debug("Pairs monitor received", "pairs", pairs)
	// 收集交易所对应的symbol, symbol 注意去重
	symbolsByExchange := make(map[string][]string)
	for _, pair := range pairs {
		exchange := pair.A.Exchange
		symbol := pair.A.Symbol
		symbolsByExchange[exchange] = append(symbolsByExchange[exchange], symbol)

		exchange = pair.B.Exchange
		symbol = pair.B.Symbol
		symbolsByExchange[exchange] = append(symbolsByExchange[exchange], symbol)
	}
	// 去重, 并且请求各个交易所的数据
	for exchange, symbols := range symbolsByExchange {
		symbolsByExchange[exchange] = utils.RemoveDuplicates(symbols)
		go checkExchangeSymbol(exchange, symbolsByExchange[exchange])
	}
	return c.JSON(fiber.Map{
		"status": "ok",
		"data":   pairs,
	})
}

func checkExchangeSymbol(exchange string, symbols []string) {
	pkg.GetLogger().Debug("Requesting exchange data", "exchange", exchange, "symbols", symbols)
	if strings.ToLower(exchange) == "gate" {
		go func() {
			tickers, err := checkGateSymbols(symbols)
			if err != nil {
				pkg.GetLogger().Debug("Failed to get gate tickers", "error", err)
				return
			}
			if len(tickers) == 0 {
				pkg.GetLogger().Debug("No matching tickers found on gate")
				return
			}
			msg := "Volume too low on gate:\n"
			for _, ticker := range tickers {
				msg += "symbol: " + ticker.symbol + ", 24h volume: " + ticker.volume24h + "\n"
			}
			pkg.GetLogger().Info("Sending volume alert", "message", msg)
			utils.SendMessage(msg)
		}()
	} else if strings.ToLower(exchange) == "binance" {
		go func() {
			tickers, err := checkBinanceSymbols(symbols)
			if err != nil {
				pkg.GetLogger().Debug("Failed to get binance tickers", "error", err)
				return
			}
			if len(tickers) == 0 {
				pkg.GetLogger().Debug("No matching tickers found on binance")
				return
			}
			msg := "Volume too low on binance:\n"
			for _, ticker := range tickers {
				msg += "symbol: " + ticker.symbol + ", 24h volume: " + ticker.volume24h + "\n"
			}
			pkg.GetLogger().Info("Sending volume alert", "message", msg)
			utils.SendMessage(msg)
		}()
	}
}

type PerpTicker struct {
	symbol    string
	volume24h string
}

// 寻找交易所的监控配置
func findVolumeMonitorConfig(exchange string) *config.VolumeMonitorConfig {
	cfg, err := config.LoadConfig()
	if err != nil || cfg == nil {
		return nil
	}
	for _, monitor := range cfg.VolumeMonitor {
		if strings.EqualFold(monitor.Platform, exchange) {
			return &monitor
		}
	}
	return nil
}

// 查询gate交易所的symbol 交易所数据
func checkGateSymbols(symbols []string) ([]PerpTicker, error) {
	url := "https://api.gateio.ws/api/v4/futures/usdt/tickers"
	resp, err := pkg.GetHTTPClient().SendGetRequest(url, nil, nil)
	if err != nil {
		pkg.GetLogger().Error("Failed to request gate symbols", "error", err)
		return nil, err
	}
	var tickers []struct {
		Symbol string `json:"contract"`
		Volume string `json:"volume_24h_settle"`
	}
	err = json.Unmarshal(resp, &tickers)
	if err != nil {
		pkg.GetLogger().Error("Failed to unmarshal gate response", "error", err)
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
	for _, ticker := range tickers {
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

func checkBinanceSymbols(symbols []string) ([]PerpTicker, error) {
	url := "https://fapi.binance.com/fapi/v1/ticker/24hr"
	resp, err := pkg.GetHTTPClient().SendGetRequest(url, nil, nil)
	if err != nil {
		pkg.GetLogger().Error("Failed to request binance symbols", "error", err)
		return nil, err
	}
	var tickers []struct {
		Symbol string `json:"symbol"`
		Volume string `json:"volume"`
	}
	err = json.Unmarshal(resp, &tickers)
	if err != nil {
		pkg.GetLogger().Error("Failed to unmarshal binance response", "error", err)
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
	for _, ticker := range tickers {
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

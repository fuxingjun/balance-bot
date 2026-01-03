package core

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fuxingjun/balance-bot/internal/utils"
	"github.com/fuxingjun/balance-bot/pkg"
)

// --- 合约指数成份监控相关 ---

func checkIndexComponentMonitor(exchange string, symbols []string) {
	checker, exists := indexComponentCheckers[exchange]
	if !exists {
		pkg.GetLogger().Debug("Unsupported exchange for index component monitor", "exchange", exchange)
		return
	}
	checker(symbols)
}

// 交易所方法配置映射
var indexComponentCheckers = map[string]func([]string){
	// 预留接口：后期实现各交易所的成份检查函数
	"binance": checkBinanceIndexComponents,
	"gate":    checkGateIndexComponents,
}

// 指数成份不限时缓存
var indexCache = pkg.NewSimpleCache(nil)

type BinanceIndexResponse struct {
	Symbol       string `json:"symbol"`
	Time         int64  `json:"time"`
	Constituents []struct {
		Exchange string `json:"exchange"`
		Symbol   string `json:"symbol"`
		Price    string `json:"price"`
		Weight   string `json:"weight"`
	} `json:"constituents"`
}

func formatBinanceConstituents(c BinanceIndexResponse) string {
	// 创建副本以避免修改原始数据的顺序
	constituents := make([]struct {
		Exchange string `json:"exchange"`
		Symbol   string `json:"symbol"`
		Price    string `json:"price"`
		Weight   string `json:"weight"`
	}, len(c.Constituents))
	copy(constituents, c.Constituents)

	// 按权重降序排序
	sort.Slice(constituents, func(i, j int) bool {
		w1 := pkg.StringToFloat(constituents[i].Weight)
		w2 := pkg.StringToFloat(constituents[j].Weight)
		return w1 > w2
	})

	var parts []string
	for _, constituent := range constituents {
		// 使用换行和缩进，使每个成分独占一行，格式更清晰
		parts = append(parts, fmt.Sprintf("  - %s: %s (Weight: %s)", constituent.Exchange, constituent.Symbol, constituent.Weight))
	}
	return strings.Join(parts, "\n")
}

func checkBinanceIndexComponents(symbols []string) {
	pkg.GetLogger().Debug("Checking Binance index components", "symbols", symbols)
	for _, symbol := range symbols {
		url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/constituents?symbol=%s", symbol)
		resp, err := pkg.SendGetRequestMarshal[BinanceIndexResponse](pkg.GetHTTPClient(), url, nil, nil)
		if err != nil {
			pkg.GetLogger().Error("Failed to request binance symbols", "error", err)
			continue
		}
		// 如果缓存已经存在且不同，则说明成份有变化
		cacheKey := "binance_index_" + symbol
		if cached, exists := indexCache.Get(cacheKey); exists {
			cachedResp := cached.(BinanceIndexResponse)

			// 使用格式化后的字符串进行比较，可以忽略原始列表的顺序差异
			oldStr := formatBinanceConstituents(cachedResp)
			newStr := formatBinanceConstituents(resp)

			if oldStr != newStr {
				// 调整消息格式以适应多行显示
				msg := fmt.Sprintf("Binance index constituents changed for %s:\n[Old]:\n%s\n\n[New]:\n%s", symbol, oldStr, newStr)
				pkg.GetLogger().Warn(msg)
				// 发送报警通知
				utils.SendMessage(msg)
			} else {
				pkg.GetLogger().Debug("No change in Binance index constituents", "symbol", symbol)
			}
		}
		// 更新缓存
		indexCache.Set(cacheKey, resp)
		// 防止接口限速(公共接口1200r/分钟)
		time.Sleep(60 * time.Millisecond)
	}
}

//	type GateIndexResponse struct {
//		Index        string `json:"index"`
//		Constituents []struct {
//			Exchange string   `json:"exchange"`
//			Symbols  []string `json:"symbols"`
//		} `json:"constituents"`
//	}
type GateIndexResponse struct {
	Method  string `json:"method"`
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    struct {
		Index        string `json:"index"`
		Constituents []struct {
			Symbol      string `json:"symbol"`
			Exchange    string `json:"exchange"`
			SourcePrice string `json:"source_price"`
			Weight      string `json:"weight"`
			Price       string `json:"price"`
		} `json:"constituents"`
	}
}

func formatGateConstituents(c GateIndexResponse) string {
	// 创建副本以避免修改原始数据的顺序
	constituents := make([]struct {
		Symbol      string `json:"symbol"`
		Exchange    string `json:"exchange"`
		SourcePrice string `json:"source_price"`
		Weight      string `json:"weight"`
		Price       string `json:"price"`
	}, len(c.Data.Constituents))
	copy(constituents, c.Data.Constituents)

	// 按权重降序排序
	sort.Slice(constituents, func(i, j int) bool {
		w1 := pkg.StringToFloat(constituents[i].Weight)
		w2 := pkg.StringToFloat(constituents[j].Weight)
		return w1 > w2
	})

	var parts []string
	for _, constituent := range constituents {
		// 使用换行和缩进，使每个成分独占一行，格式更清晰
		parts = append(parts, fmt.Sprintf("  - %s: %s (Weight: %s)", constituent.Exchange, constituent.Symbol, constituent.Weight))
	}
	return strings.Join(parts, "\n")
}

// checkGateIndexComponents 检查 Gate 指数成份是否有变化
func checkGateIndexComponents(symbols []string) {
	pkg.GetLogger().Debug("Checking Gate index components", "symbols", symbols)
	for _, symbol := range symbols {
		// url := fmt.Sprintf("https://api.gateio.ws/api/v4/futures/usdt/index_constituents/%s", symbol)
		// api没有成份占比信息，改用网页接口
		url := fmt.Sprintf("https://www.gate.com/apiw/v2/futures/common/index/breakdown?index=%s", symbol)
		resp, err := pkg.SendGetRequestMarshal[GateIndexResponse](pkg.GetHTTPClient(), url, nil, nil)
		if err != nil {
			pkg.GetLogger().Error("Failed to request gate symbols", "error", err)
		}
		// 如果缓存已经存在且不同，则说明成份有变化
		cacheKey := "gate_index_" + symbol
		if cached, exists := indexCache.Get(cacheKey); exists {
			cachedResp := cached.(GateIndexResponse)
			// 使用格式化后的字符串进行比较，可以忽略原始列表的顺序差异
			oldStr := formatGateConstituents(cachedResp)
			newStr := formatGateConstituents(resp)

			if oldStr != newStr {
				// 调整消息格式以适应多行显示
				msg := fmt.Sprintf("Gate index constituents changed for %s:\n[Old]:\n%s\n\n[New]:\n%s", symbol, oldStr, newStr)
				pkg.GetLogger().Warn(msg)
				// 发送报警通知
				utils.SendMessage(msg)
			} else {
				pkg.GetLogger().Debug("No change in Gate index constituents", "symbol", symbol)
			}
		}
		// 更新缓存
		indexCache.Set(cacheKey, resp)
		// 防止接口限速(公共接口单个接口 200r/10s)
		time.Sleep(60 * time.Millisecond)
	}
}

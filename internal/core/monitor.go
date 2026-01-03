package core

import (
	"strings"
	"sync"
	"time"

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

// 缓存symbols, 循环检测的时候直接取这个
var symbolsCache = pkg.NewSimpleCache(nil)

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
		// 增加非空校验和统一转小写
		if pair.A.Exchange != "" && pair.A.Symbol != "" {
			exchange := strings.ToLower(pair.A.Exchange)
			symbolsByExchange[exchange] = append(symbolsByExchange[exchange], pair.A.Symbol)
		}

		if pair.B.Exchange != "" && pair.B.Symbol != "" {
			exchange := strings.ToLower(pair.B.Exchange)
			symbolsByExchange[exchange] = append(symbolsByExchange[exchange], pair.B.Symbol)
		}
	}

	// 去重, 并且请求各个交易所的数据
	for exchange, symbols := range symbolsByExchange {
		// 这里的 symbols 是局部变量，直接去重即可
		uniqueSymbols := utils.RemoveDuplicates(symbols)
		// 缓存symbols, 后台循环监控使用
		symbolsCache.Set(exchange, uniqueSymbols)
		// 启动协程处理
		go checkExchangeSymbol(exchange, uniqueSymbols)
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"data":   pairs,
	})
}

// 24小时缓存
// 注意：这个变量在同一个包(core)下的 monitor_volume.go 中也会被用到
var notifyCache = pkg.NewTTLCache(24 * 3600 * 1e9)

// checkExchangeSymbol 统一入口，分发到各个具体的监控项
func checkExchangeSymbol(exchange string, symbols []string) {
	pkg.GetLogger().Debug("Requesting exchange data", "exchange", exchange, "symbols", symbols)
	// 1. 交易量监控 (实现位于 monitor_volume.go)
	go checkVolumeMonitor(exchange, symbols)
}

// 后台持续监控指数成份
func StartIndexMonitor() {
	pkg.GetLogger().Info("Starting index monitor...")
	// 所有交易所并行, 等待所有交易所完成3S之后再开始下一轮
	for {
		var wg sync.WaitGroup
		cacheKeyList := symbolsCache.GetAllKeys()
		for exchange, symList := range cacheKeyList {
			wg.Add(1)
			go func(exch string, symbols []string) {
				defer wg.Done()
				checkIndexComponentMonitor(exch, symbols)
			}(exchange, symList)
		}
		wg.Wait()
		time.Sleep(3 * time.Second)
	}
}

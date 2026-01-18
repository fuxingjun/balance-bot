package main

import (
	"fmt"

	"github.com/fuxingjun/balance-bot/internal/config"
	"github.com/fuxingjun/balance-bot/internal/core"
	"github.com/fuxingjun/balance-bot/internal/utils"
	"github.com/fuxingjun/balance-bot/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var (
	version = "dev"
	date    = "unknown"
)

func main() {
	fmt.Printf("version: %s, build time: %s\n", version, date)
	pkg.InitLoggerDefault(utils.GetArgs().Debug)
	appConfig, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	if appConfig == nil {
		println("配置文件不存在, 已生成示例文件 config.json, 请根据需要修改后重新运行程序。")
		return
	}
	println("配置文件加载成功,", "gas检测间隔:", appConfig.Interval, "秒")
	for _, token := range appConfig.Tokens {
		// 地址只显示开始和结尾, 浮点数显示4位小数
		println("代币地址:", token.Address[:6], "...", token.Address[len(token.Address)-4:], "链ID:", token.ChainId, "名称:", token.Name, "最小值:", fmt.Sprintf("%.4f", token.Min), "最大值:", fmt.Sprintf("%.4f", token.Max))
	}
	core.CheckBalance()

	// 启动后台指数监控任务
	if appConfig.IndexComponentMonitor {
		go core.StartIndexMonitor()
		println("合约指数成份监控已启用。")
	} else {
		println("合约指数成份监控未启用。")
	}

	// 健康检测信息
	println("健康检测间隔:", appConfig.HealthCheck.Interval, "告警次数:", appConfig.HealthCheck.WarnCount)

	// 交易量检测配置
	if len(appConfig.VolumeMonitor.Platform) > 0 {
		println("交易量监控配置:")
		for _, platform := range appConfig.VolumeMonitor.Platform {
			println("交易所:", platform.Platform, "24h交易量阈值(美元):", fmt.Sprintf("%.2f", platform.ThresholdUSD))
		}
	} else {
		println("未配置交易量监控。")
	}

	// 启动一个http服务
	args := utils.GetArgs()
	app := fiber.New()
	if args.Debug {
		app.Use(logger.New())
	}

	// 跨域
	app.Use(cors.New())
	// 或者扩展你的配置以进行自定义
	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://gofiber.io, https://taoli.tools",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	app.Post("/health", core.HealthCheck)
	app.Post("/monitor", core.PairsMonitor)

	addr := fmt.Sprintf("%s:%d", args.Host, args.Port)
	// 启动服务器在 指定 端口
	fmt.Printf("Listening on %s\n", addr)
	if err := app.Listen(addr); err != nil {
		fmt.Printf("listen failed: %v\n", err)
	}
}

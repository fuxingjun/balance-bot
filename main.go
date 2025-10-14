package main

import (
	"balance-bot/internal/config"
	"balance-bot/internal/core"
	"balance-bot/internal/utils"
	"fmt"
	"time"
)

func sendMessage(msg string) error {
	appConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if appConfig == nil {
		return fmt.Errorf("config is nil")
	}
	if appConfig.Webhook.TelegramToken != "" && appConfig.Webhook.TelegramChatId != "" {
		err := utils.SendTelegramMessage(msg, appConfig.Webhook.TelegramChatId, appConfig.Webhook.TelegramToken)
		if err != nil {
			return err
		}
	}
	if appConfig.Webhook.Wecom != "" {
		err := utils.SendWecomMessage(msg, appConfig.Webhook.Wecom)
		if err != nil {
			return err
		}
	}
	if appConfig.Webhook.Lark != "" {
		err := utils.SendLarkMessage(msg, appConfig.Webhook.Lark)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkBalanceItem(item *config.TokenConfig) error {
	if item.Address == "" {
		panic("address cannot be empty")
	}
	if item.ChainId == "" {
		item.ChainId = "56" // 默认 BSC
	}
	if item.Min == 0 {
		item.Min = 0.1 // 默认 0.1
	}
	if item.Max == 0 {
		item.Max = 1e18 // 默认不限
	}
	resp, err := core.GetEVMBalance(item.Address, item.ChainId)
	// 地址只显示开始和结尾
	address := item.Address
	if len(address) > 10 {
		address = address[:6] + "**" + address[len(address)-4:]
	}
	// 有name的话在地址后面显示
	if item.Name != "" {
		address = address + "(" + item.Name + ")"
	}
	if err != nil {
		utils.GetLogger("").Error(fmt.Sprintf("Get balance error for %s on chain %s: %v", address, item.ChainId, err))
		return err
	}
	utils.GetLogger("").Info(fmt.Sprintf("Balance for %s on chain %s: %f", address, item.ChainId, resp))
	msg := ""
	if resp < item.Min {
		msg = fmt.Sprintf("⚠️ Balance for %s on chain %s is below minimum %f: %f", address, item.ChainId, item.Min, resp)
		utils.GetLogger("").Warn(msg)
	} else if resp > item.Max {
		msg = fmt.Sprintf("⚠️ Balance for %s on chain %s is above maximum %f: %f", address, item.ChainId, item.Max, resp)
		utils.GetLogger("").Warn(msg)
	}
	if msg != "" {
		// 发送通知
		err := sendMessage(msg)
		if err != nil {
			utils.GetLogger("").Error(fmt.Sprintf("Send message error: %v\n", err))
		}
	}
	return nil
}

func checkBalance() {
	appConfig, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	for _, item := range appConfig.Tokens {
		go func(it config.TokenConfig) {
			err := checkBalanceItem(&it)
			if err != nil {
				utils.GetLogger("").Error(fmt.Sprintf("Check balance error: %v\n", err))
			}
		}(item)
	}
	// 设置定时器
	interval := appConfig.Interval
	if interval <= 0 {
		interval = 30 // 默认 30 秒
	}
	utils.GetLogger("").Info(fmt.Sprintf("Next check in %d seconds...\n", interval))
	go func() {
		time.Sleep(time.Duration(interval) * time.Second)
		checkBalance()
	}()
}

var (
	version = "dev"
	date    = "unknown"
)

func main() {
	fmt.Printf("version: %s, build time: %s\n", version, date)
	utils.GetLogger("")
	appConfig, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	if appConfig == nil {
		println("配置文件不存在, 已生成示例文件 config.json, 请根据需要修改后重新运行程序。")
		return
	}
	println("配置文件加载成功,", "检测间隔:", appConfig.Interval, "秒")
	for _, token := range appConfig.Tokens {
		println("代币地址:", token.Address, "链ID:", token.ChainId, "名称:", token.Name, "最小值:", token.Min, "最大值:", token.Max)
	}
	checkBalance()
	select {}
}

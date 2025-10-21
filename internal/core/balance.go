package core

import (
	"fmt"
	"time"

	"github.com/fuxingjun/balance-bot/internal/config"
	"github.com/fuxingjun/balance-bot/internal/utils"
	"github.com/fuxingjun/balance-bot/pkg"
)

func checkBalanceItem(item *config.TokenConfig) error {
	if item.Address == "" {
		panic("address cannot be empty")
	}
	resp, err := GetEVMBalance(item.Address, item.ChainId)
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
		pkg.GetLogger().Error(fmt.Sprintf("Get balance error for %s on chain %s: %v", address, item.ChainId, err))
		return err
	}
	pkg.GetLogger().Info(fmt.Sprintf("Balance for %s on chain %s: %f", address, item.ChainId, resp))
	msg := ""
	if resp < item.Min {
		msg = fmt.Sprintf("⚠️ Balance for %s on chain %s is below minimum %f: %f", address, item.ChainId, item.Min, resp)
		pkg.GetLogger().Warn(msg)
	} else if resp > item.Max {
		msg = fmt.Sprintf("⚠️ Balance for %s on chain %s is above maximum %f: %f", address, item.ChainId, item.Max, resp)
		pkg.GetLogger().Warn(msg)
	}
	if msg != "" {
		// 发送通知
		err := utils.SendMessage(msg)
		if err != nil {
			pkg.GetLogger().Error(fmt.Sprintf("Send message error: %v\n", err))
		}
	}
	return nil
}

func CheckBalance() {
	appConfig, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	for _, item := range appConfig.Tokens {
		go func(it config.TokenConfig) {
			err := checkBalanceItem(&it)
			if err != nil {
				pkg.GetLogger().Error(fmt.Sprintf("Check balance error: %v\n", err))
			}
		}(item)
	}
	// 设置定时器
	interval := appConfig.Interval
	if interval <= 0 {
		interval = 30 // 默认 30 秒
	}
	pkg.GetLogger().Info(fmt.Sprintf("Next check in %d seconds...\n", interval))
	go func() {
		time.Sleep(time.Duration(interval) * time.Second)
		CheckBalance()
	}()
}

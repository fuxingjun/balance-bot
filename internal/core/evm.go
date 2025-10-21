package core

import (
	"balance-bot/pkg"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

var BSC_RPC = []string{
	"https://bsc-dataseed.bnbchain.org",
	"https://bsc-dataseed.nariox.org",
	"https://bsc-dataseed.defibit.io",
	"https://bsc-dataseed.ninicoin.io",
	"https://bsc.nodereal.io",
}

var (
	bscPoint = 0
	bscMutex sync.Mutex
)

func GetRPC(chainId string) string {
	if chainId == "56" {
		bscMutex.Lock()
		defer bscMutex.Unlock()

		if bscPoint >= len(BSC_RPC) {
			bscPoint = 0
		}
		rpc := BSC_RPC[bscPoint]
		bscPoint++
		return rpc
	}
	return ""
}

// 获取钱包地址在目标链上的原生代币余额（比如 BNB 于 BSC,ETH 于 Ethereum）
func GetEVMBalance(walletAddress, chainId string) (float64, error) {
	params := map[string]any{
		"jsonrpc": "2.0",
		"method":  "eth_getBalance",
		"params":  []any{walletAddress, "latest"},
		"id":      pkg.GetSimpleId(),
	}

	return sendBalanceRequest(params, 18, chainId)
}

// 定义RPC响应结构体，注意Error字段是一个对象
type RPCResponseT[T any] struct {
	Id     string `json:"id"`
	Result T      `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func sendBalanceRequest(params map[string]any, decimals uint64, chainId string) (float64, error) {
	resp, err := pkg.GetHTTPClient().SendPostRequest(GetRPC(chainId), params, nil, nil)
	if err != nil {
		return 0, err
	}
	var data RPCResponseT[*string]
	if err := json.Unmarshal(resp, &data); err != nil {
		return 0, fmt.Errorf("JSON unmarshal failed: %v", err)
	}
	// === 1. 验证 id ===
	if !reflect.DeepEqual(data.Id, params["id"]) {
		return 0, fmt.Errorf("id mismatch: expected %v, got %v", params["id"], data.Id)
	}
	// === 2. 检查 error 字段 ===
	if data.Error != nil {
		return 0, fmt.Errorf("RPC error: %s", data.Error.Message)
	}
	if data.Result == nil {
		return 0, fmt.Errorf("balance not found")
	}
	balance, err := pkg.ConvertBigIntToAmount(pkg.HexToBigInt(*data.Result), decimals)
	if err == nil {
		return balance, nil
	}
	// 如果解析失败，返回错误
	return 0, fmt.Errorf("parse hex error")
}

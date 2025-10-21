package pkg

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
)

var (
	simpleIDCounter int64 // 使用 int64 避免溢出, 原子操作
)

// GetSimpleId 生成应用内唯一的递增整数ID, 返回字符串
func GetSimpleId() string {
	return strconv.FormatInt(atomic.AddInt64(&simpleIDCounter, 1), 10)
}

// A query string with key=value pairs joined by &.
func ToQueryStrWithoutEncode(params map[string]any) string {
	if params == nil {
		return ""
	}
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 字典序排序
	var parts []string

	for _, key := range keys {
		value := params[key]
		v := reflect.ValueOf(value)

		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				parts = append(parts, fmt.Sprintf("%s=%v", key, v.Index(i).Interface()))
			}
		default:
			parts = append(parts, fmt.Sprintf("%s=%v", key, value))
		}
	}
	res := strings.Join(parts, "&")
	return res
}

func ConvertBigIntToAmount(amount *big.Int, decimals uint64) (float64, error) {
	if amount == nil {
		return StringToFloat64("0"), fmt.Errorf("amount cannot be nil")
	}
	// 1. 获取 10^decimals
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(0).SetUint64(uint64(decimals)), nil)
	// 2. 分离整数部分和小数部分
	intPart := new(big.Int).Div(amount, divisor)  // 整数部分
	fracPart := new(big.Int).Mod(amount, divisor) // 小数部分（余数）
	// 3. 将整数部分转为字符串
	intStr := intPart.String()
	// 4. 将小数部分格式化为固定长度（补前导零）
	fracStr := fracPart.String()
	fracLen := len(fracStr)
	requiredLen := int(decimals)
	if fracLen < requiredLen {
		// 补前导零，例如 123 -> 000123（如果 decimals=6）
		fracStr = fmt.Sprintf("%0*s", requiredLen, fracStr)
	} else if fracLen > requiredLen {
		// 理论上不会发生（因为 mod 10^decimals），但保险起见
		fracStr = fracStr[fracLen-requiredLen:] // 取后 requiredLen 位
	}
	// 5. 去除小数部分末尾的无意义零
	fracStr = strings.TrimRight(fracStr, "0")
	if fracStr == "" {
		// 全是零，就显示为整数
		return StringToFloat64(intStr), nil
	}
	// 6. 组合结果
	return StringToFloat64(intStr + "." + fracStr), nil
}

// 字符串转 float64
func StringToFloat64(str string) float64 {
	if str == "" {
		return 0
	}
	// 使用 strconv.ParseUint 解析字符串为
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}
	return f
}

func MultiplyBigIntByFloat(amount *big.Int, multiplier float64) *big.Int {
	// 1. 将 big.Int 转为 big.Float
	a := new(big.Float).SetInt(amount)
	// 2. 创建 multiplier 的 big.Float
	b := new(big.Float).SetFloat64(multiplier)
	// 3. 相乘
	resultFloat := new(big.Float).Mul(a, b)
	// 4. 转回 big.Int（截断小数部分，向下取整）
	resultInt, _ := resultFloat.Int(nil) // 忽略精度错误
	return resultInt
}

// 解析十六进制结果为整数
func HexToBigInt(hexStr string) *big.Int {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	value := new(big.Int)
	_, success := value.SetString(hexStr, 16)
	if !success {
		return big.NewInt(0) // 或 panic/error
	}
	return value
}

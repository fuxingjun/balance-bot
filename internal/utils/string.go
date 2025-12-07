package utils

import "strings"

// ContainsIgnoreCase 检查字符串 s 是否包含子字符串 substr（忽略大小写）
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func RemoveDuplicates(strings []string) []string {
	uniqueMap := make(map[string]struct{}) // 使用 map 存储唯一值
	var result []string

	for _, str := range strings {
		if _, exists := uniqueMap[str]; !exists { // 如果 map 中不存在该字符串
			uniqueMap[str] = struct{}{}  // 添加到 map
			result = append(result, str) // 添加到结果切片
		}
	}

	return result
}

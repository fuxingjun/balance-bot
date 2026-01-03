package pkg

import (
	"sync"
	"time"
)

// TTLCacheItem 缓存条目
type TTLCacheItem struct {
	value      any
	expireTime time.Time
}

// IsExpired 判断是否过期
func (item *TTLCacheItem) IsExpired() bool {
	return time.Now().After(item.expireTime)
}

// TTLCache 主结构
type TTLCache struct {
	items    map[string]*TTLCacheItem
	duration time.Duration
	mutex    sync.RWMutex
}

// NewTTLCache 创建新缓存, duration 为 TTL
func NewTTLCache(defaultTTL time.Duration) *TTLCache {
	return &TTLCache{
		items:    make(map[string]*TTLCacheItem),
		duration: defaultTTL,
	}
}

// Set 添加缓存项, 使用默认 TTL
func (c *TTLCache) Set(key string, value any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &TTLCacheItem{
		value:      value,
		expireTime: time.Now().Add(c.duration),
	}
}

// Get 获取缓存项, 若过期则返回 nil, false
func (c *TTLCache) Get(key string) (any, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.items[key]
	if !exists || item.IsExpired() {
		return nil, false
	}
	return item.value, true
}

// Delete 删除指定项
func (c *TTLCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.items, key)
}

// Clean 清理所有过期项（可选调用）
func (c *TTLCache) Clean() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for k, v := range c.items {
		if now.After(v.expireTime) {
			delete(c.items, k)
		}
	}
}

// SimpleCache 一个不过期的并发安全缓存
type SimpleCache struct {
	items map[string]any
	mu    sync.RWMutex
}

// NewSimpleCache 创建一个永不过期的缓存
func NewSimpleCache(init *map[string]any) *SimpleCache {
	if init == nil {
		init = &map[string]any{}
	}
	return &SimpleCache{
		items: *init,
	}
}

// Set 添加值（永不过期）
func (c *SimpleCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

// Get 获取值，第二个返回值表示是否存在
func (c *SimpleCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.items[key]
	return value, exists
}

// Delete 删除键
func (c *SimpleCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear 清空所有键
func (c *SimpleCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]any)
}

// Len 返回缓存中的条目数量
func (c *SimpleCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// GetAllKeys 返回所有键的切片
func (c *SimpleCache) GetAllKeys() map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string][]string)
	for k, v := range c.items {
		if symList, ok := v.([]string); ok {
			result[k] = symList
		}
	}
	return result
}

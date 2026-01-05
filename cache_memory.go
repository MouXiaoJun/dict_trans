package dict

import (
	"sync"
	"time"
)

// memoryCache 内存缓存实现
type memoryCache struct {
	data       map[string]cacheItem
	mutex      sync.RWMutex
	maxEntries int
}

type cacheItem struct {
	value     string
	expiresAt *time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxEntries int) Cache {
	return &memoryCache{
		data:       make(map[string]cacheItem),
		maxEntries: maxEntries,
	}
}

func (c *memoryCache) Get(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, ok := c.data[key]
	if !ok {
		return "", false
	}

	// 检查是否过期
	if item.expiresAt != nil && time.Now().After(*item.expiresAt) {
		// 异步删除过期项
		go c.Delete(key)
		return "", false
	}

	return item.value, true
}

func (c *memoryCache) Set(key string, value string, ttl int) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 检查最大条目数
	if len(c.data) >= c.maxEntries {
		// 简单的LRU策略：删除第一个（实际应该用更复杂的LRU）
		for k := range c.data {
			delete(c.data, k)
			break
		}
	}

	var expiresAt *time.Time
	if ttl > 0 {
		exp := time.Now().Add(time.Duration(ttl) * time.Second)
		expiresAt = &exp
	}

	c.data[key] = cacheItem{
		value:     value,
		expiresAt: expiresAt,
	}

	return nil
}

func (c *memoryCache) Delete(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, key)
	return nil
}

func (c *memoryCache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make(map[string]cacheItem)
	return nil
}

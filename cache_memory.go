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
	item, ok := c.data[key]
	c.mutex.RUnlock()
	if !ok {
		return "", false
	}

	// 检查是否过期
	if item.expiresAt != nil && time.Now().After(*item.expiresAt) {
		// 同步删除过期项：升级为写锁后双重检查，避免误删并发 Set 写入的新值
		c.mutex.Lock()
		if cur, ok := c.data[key]; ok && cur.expiresAt != nil && time.Now().After(*cur.expiresAt) {
			delete(c.data, key)
		}
		c.mutex.Unlock()
		return "", false
	}

	return item.value, true
}

func (c *memoryCache) Set(key string, value string, ttl int) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 检查最大条目数（maxEntries <= 0 表示不限制）
	// ponytail: 满时随机淘汰一个，需要命中率时再换 LRU
	if _, exists := c.data[key]; !exists && c.maxEntries > 0 && len(c.data) >= c.maxEntries {
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

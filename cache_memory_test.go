package dict

import (
	"fmt"
	"testing"
	"time"
)

// 过期条目 Get 应返回 miss，且同步从 map 中删除
func TestMemoryCacheGetExpiredDeletesSync(t *testing.T) {
	c := NewMemoryCache(10).(*memoryCache)
	past := time.Now().Add(-time.Second)
	c.data["k"] = cacheItem{value: "v", expiresAt: &past}

	if _, ok := c.Get("k"); ok {
		t.Fatal("过期条目不应命中")
	}
	c.mutex.RLock()
	_, exists := c.data["k"]
	c.mutex.RUnlock()
	if exists {
		t.Fatal("过期条目应在 Get 返回前被删除")
	}
}

// maxEntries <= 0 表示不限制
func TestMemoryCacheUnlimited(t *testing.T) {
	c := NewMemoryCache(0)
	for i := 0; i < 100; i++ {
		_ = c.Set(fmt.Sprintf("k%d", i), "v", 0)
	}
	for i := 0; i < 100; i++ {
		if _, ok := c.Get(fmt.Sprintf("k%d", i)); !ok {
			t.Fatalf("maxEntries=0 时 k%d 不应被淘汰", i)
		}
	}
}

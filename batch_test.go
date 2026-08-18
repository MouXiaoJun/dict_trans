package dict

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestBatchTranslate(t *testing.T) {
	// 注册字典
	RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	// 创建大量数据
	items := make([]Item, 100)
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			items[i] = Item{Status: "1"}
		} else {
			items[i] = Item{Status: "0"}
		}
	}

	// 测试顺序处理
	err := BatchTranslate(&items, false)
	if err != nil {
		t.Fatalf("BatchTranslate failed: %v", err)
	}

	// 验证结果
	if items[0].StatusName != "启用" {
		t.Errorf("Expected '启用', got '%s'", items[0].StatusName)
	}
	if items[1].StatusName != "禁用" {
		t.Errorf("Expected '禁用', got '%s'", items[1].StatusName)
	}
}

func TestBatchTranslateParallel(t *testing.T) {
	// 注册字典
	RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	// 创建大量数据
	items := make([]Item, 1000)
	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			items[i] = Item{Status: "1"}
		} else {
			items[i] = Item{Status: "0"}
		}
	}

	// 测试并行处理
	start := time.Now()
	err := BatchTranslate(&items, true)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("BatchTranslate failed: %v", err)
	}

	// 验证结果
	if items[0].StatusName != "启用" {
		t.Errorf("Expected '启用', got '%s'", items[0].StatusName)
	}
	if items[1].StatusName != "禁用" {
		t.Errorf("Expected '禁用', got '%s'", items[1].StatusName)
	}
	if items[999].StatusName == "" {
		t.Errorf("Last item should be translated")
	}

	t.Logf("Parallel translation took: %v", duration)
}

func TestBatchTranslateSmall(t *testing.T) {
	// 注册字典
	RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	// 小批量数据（应该使用顺序处理）
	items := []Item{
		{Status: "1"},
		{Status: "0"},
	}

	err := BatchTranslate(&items, true)
	if err != nil {
		t.Fatalf("BatchTranslate failed: %v", err)
	}

	if items[0].StatusName != "启用" {
		t.Errorf("Expected '启用', got '%s'", items[0].StatusName)
	}
}

// TestBatchTranslateParallelStopsOnError 并行翻译中某个 worker 出错后，其他 worker 应尽快停止。
// 只有第 0 个元素立即失败，其余元素慢速成功：若没有 done 信号，其他 9 个 worker 会把各自的 10 个元素全部翻完（≈90 次）；
// 有 done 信号则每个 worker 最多再翻 1 个就退出。
// 包级变量：字段配置会被缓存，-count>1 时复用首次注册的翻译器闭包
var (
	slowOkTranslated int64
	errFirstItem     = errors.New("first item fails")
)

func TestBatchTranslateParallelStopsOnError(t *testing.T) {
	atomic.StoreInt64(&slowOkTranslated, 0)
	RegisterTranslator("firstFails", TranslatorFunc(func(value any, fieldName, tagValue string) (string, error) {
		if value == "fail" {
			return "", errFirstItem
		}
		atomic.AddInt64(&slowOkTranslated, 1)
		time.Sleep(2 * time.Millisecond) // 让其他 worker 还在跑时 done 已关闭
		return "ok", nil
	}))

	type Item struct {
		Code string `translate:"firstFails" dictField:"Name"`
		Name string
	}

	items := make([]Item, 100)
	for i := range items {
		items[i].Code = "x"
	}
	items[0].Code = "fail" // 落在 worker 0 的 chunk 首位

	err := BatchTranslate(&items, true)
	if !errors.Is(err, errFirstItem) {
		t.Fatalf("expected %v, got %v", errFirstItem, err)
	}
	// 没有首错即停时其余 worker 会翻完 ≈99 个；有的话每个 worker 最多再翻 1~2 个
	if n := atomic.LoadInt64(&slowOkTranslated); n >= 50 {
		t.Errorf("other workers should stop after first error, but translated %d items", n)
	}
}

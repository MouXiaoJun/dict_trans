package dict

import (
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

package dict

import (
	"testing"
)

func TestDictTableTranslator(t *testing.T) {
	// 模拟字典表数据
	mockDictTable := map[string]map[string]string{
		"sex": {
			"1": "男",
			"2": "女",
		},
		"status": {
			"1": "启用",
			"0": "禁用",
		},
	}

	// 注册字典表翻译器
	RegisterDictTableTranslator(DictTableTranslatorFunc(func(dictType, dictKey string) (string, error) {
		if dictData, ok := mockDictTable[dictType]; ok {
			if value, ok := dictData[dictKey]; ok {
				return value, nil
			}
		}
		return "", nil
	}))

	type User struct {
		Sex     string `dictTable:"sex" dictField:"SexName"`
		SexName string
	}

	user := User{Sex: "1"}
	err := Translate(&user)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if user.SexName != "男" {
		t.Errorf("Expected '男', got '%s'", user.SexName)
	}

	// 测试缓存
	user2 := User{Sex: "1"}
	err = Translate(&user2)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if user2.SexName != "男" {
		t.Errorf("Expected '男', got '%s'", user2.SexName)
	}
}

func TestDictTableCache(t *testing.T) {
	queryCount := 0

	// 注册字典表翻译器，记录查询次数
	RegisterDictTableTranslator(DictTableTranslatorFunc(func(dictType, dictKey string) (string, error) {
		queryCount++
		return "结果", nil
	}))

	type Item struct {
		Status     string `dictTable:"status" dictField:"StatusName"`
		StatusName string
	}

	// 第一次翻译，应该查询
	item1 := Item{Status: "1"}
	err := Translate(&item1)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if queryCount != 1 {
		t.Errorf("Expected 1 query, got %d", queryCount)
	}

	// 第二次翻译相同值，应该使用缓存
	item2 := Item{Status: "1"}
	err = Translate(&item2)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if queryCount != 1 {
		t.Errorf("Expected 1 query (cached), got %d", queryCount)
	}

	// 禁用缓存后，应该再次查询
	EnableDictTableCache(false)
	item3 := Item{Status: "1"}
	err = Translate(&item3)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if queryCount != 2 {
		t.Errorf("Expected 2 queries (cache disabled), got %d", queryCount)
	}

	// 重新启用缓存
	EnableDictTableCache(true)
}

package dict

import (
	"testing"
)

func TestDictTableTwoTranslator(t *testing.T) {
	// 模拟双表字典数据
	mockDictType := map[string]bool{
		"sex":    true,
		"status": true,
	}
	mockDictData := map[string]map[string]string{
		"sex": {
			"1": "男",
			"2": "女",
		},
		"status": {
			"1": "启用",
			"0": "禁用",
		},
	}

	// 注册双表字典翻译器
	RegisterDictTableTwoTranslator(DictTableTwoTranslatorFunc(func(dictTypeCode, dictKey string) (string, error) {
		// 先检查字典类型是否存在
		if !mockDictType[dictTypeCode] {
			return "", nil
		}
		// 查询字典数据
		if dictData, ok := mockDictData[dictTypeCode]; ok {
			if value, ok := dictData[dictKey]; ok {
				return value, nil
			}
		}
		return "", nil
	}))

	type User struct {
		Sex     string `dictTableTwo:"sex" dictField:"SexName"`
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

func TestDictTableTwoCache(t *testing.T) {
	queryCount := 0

	// 注册双表字典翻译器，记录查询次数
	RegisterDictTableTwoTranslator(DictTableTwoTranslatorFunc(func(dictTypeCode, dictKey string) (string, error) {
		queryCount++
		return "结果", nil
	}))

	type Item struct {
		Status     string `dictTableTwo:"status" dictField:"StatusName"`
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
	EnableDictTableTwoCache(false)
	item3 := Item{Status: "1"}
	err = Translate(&item3)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if queryCount != 2 {
		t.Errorf("Expected 2 queries (cache disabled), got %d", queryCount)
	}

	// 重新启用缓存
	EnableDictTableTwoCache(true)
}

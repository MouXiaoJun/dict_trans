package dict

import (
	"testing"
)

func TestDBTranslator(t *testing.T) {
	// 模拟数据库数据
	mockData := map[string]map[string]string{
		"user": {
			"1": "张三",
			"2": "李四",
		},
	}

	// 注册数据库翻译器
	RegisterDBTranslator(DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
		// 模拟数据库查询
		keyStr := ""
		switch v := key.(type) {
		case string:
			keyStr = v
		case int:
			keyStr = string(rune(v))
		case int64:
			keyStr = string(rune(v))
		default:
			keyStr = ""
		}

		if tableData, ok := mockData[table]; ok {
			if value, ok := tableData[keyStr]; ok {
				return value, nil
			}
		}
		return "", nil
	}))

	type User struct {
		UserID   string `db:"user:id:name" dictField:"UserName"`
		UserName string
	}

	user := User{UserID: "1"}
	err := Translate(&user)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if user.UserName != "张三" {
		t.Errorf("Expected '张三', got '%s'", user.UserName)
	}

	// 测试缓存
	user2 := User{UserID: "1"}
	err = Translate(&user2)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if user2.UserName != "张三" {
		t.Errorf("Expected '张三', got '%s'", user2.UserName)
	}
}

func TestDBTranslatorFullFormat(t *testing.T) {
	// 模拟数据库数据
	mockData := map[string]map[string]string{
		"device": {
			"1": "设备1",
			"2": "设备2",
		},
	}

	// 注册数据库翻译器
	RegisterDBTranslator(DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
		keyStr := ""
		switch v := key.(type) {
		case string:
			keyStr = v
		case int:
			keyStr = string(rune(v))
		case int64:
			keyStr = string(rune(v))
		default:
			keyStr = ""
		}

		if tableData, ok := mockData[table]; ok {
			if value, ok := tableData[keyStr]; ok {
				return value, nil
			}
		}
		return "", nil
	}))

	type Device struct {
		DeviceID   string `db:"table=device,key=id,value=name" dictField:"DeviceName"`
		DeviceName string
	}

	device := Device{DeviceID: "1"}
	err := Translate(&device)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if device.DeviceName != "设备1" {
		t.Errorf("Expected '设备1', got '%s'", device.DeviceName)
	}
}

func TestDBCache(t *testing.T) {
	queryCount := 0

	// 注册数据库翻译器，记录查询次数
	RegisterDBTranslator(DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
		queryCount++
		return "结果", nil
	}))

	type Item struct {
		ID   string `db:"test:id:name" dictField:"Name"`
		Name string
	}

	// 第一次翻译，应该查询数据库
	item1 := Item{ID: "1"}
	err := Translate(&item1)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if queryCount != 1 {
		t.Errorf("Expected 1 query, got %d", queryCount)
	}

	// 第二次翻译相同值，应该使用缓存
	item2 := Item{ID: "1"}
	err = Translate(&item2)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if queryCount != 1 {
		t.Errorf("Expected 1 query (cached), got %d", queryCount)
	}

	// 禁用缓存后，应该再次查询
	EnableDBCache(false)
	item3 := Item{ID: "1"}
	err = Translate(&item3)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if queryCount != 2 {
		t.Errorf("Expected 2 queries (cache disabled), got %d", queryCount)
	}
}

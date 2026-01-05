package main

import (
	"fmt"

	"github.com/mouxiaojun/dict-trans"
)

func main() {
	fmt.Println("=== dict-trans 完整示例 ===\n")

	// ========== 示例1: 字典翻译 ==========
	fmt.Println("【示例1】字典翻译")
	example1_Dict()
	fmt.Println()

	// ========== 示例2: 枚举转换 ==========
	fmt.Println("【示例2】枚举转换")
	example2_Enum()
	fmt.Println()

	// ========== 示例3: 嵌套翻译 ==========
	fmt.Println("【示例3】嵌套翻译")
	example3_Nested()
	fmt.Println()

	// ========== 示例4: 切片翻译 ==========
	fmt.Println("【示例4】切片翻译")
	example4_Slice()
	fmt.Println()

	// ========== 示例5: 包装类型 ==========
	fmt.Println("【示例5】包装类型支持")
	example5_Wrapper()
	fmt.Println()

	// ========== 示例6: 自定义翻译器 ==========
	fmt.Println("【示例6】自定义翻译器")
	example6_CustomTranslator()
	fmt.Println()

	// ========== 示例7: 数据库翻译 ==========
	fmt.Println("【示例7】数据库翻译（MySQL）")
	example7_Database()
	fmt.Println()

	// ========== 示例8: 字典表翻译 ==========
	fmt.Println("【示例8】字典表翻译（从数据库字典表读取）")
	example8_DictTable()
	fmt.Println()

	// ========== 示例9: 批量并行翻译 ==========
	fmt.Println("【示例9】批量并行翻译")
	example9_Batch()
	fmt.Println()

	// ========== 示例10: 混合使用 ==========
	fmt.Println("【示例10】混合使用（字典+枚举+数据库+字典表）")
	example10_Mixed()
	fmt.Println()
}

// 示例1: 字典翻译
func example1_Dict() {
	// 注册字典
	dict.RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	type User struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
	}

	user := User{Sex: "1"}
	dict.Translate(&user)
	fmt.Printf("  用户性别: %s -> %s\n", user.Sex, user.SexName)

	user2 := User{Sex: "2"}
	dict.Translate(&user2)
	fmt.Printf("  用户性别: %s -> %s\n", user2.Sex, user2.SexName)
}

// 示例2: 枚举转换
func example2_Enum() {
	// 注册枚举
	dict.RegisterEnum("deviceStatus", map[string]string{
		"1": "未使用",
		"2": "试运行",
		"3": "运行中",
		"4": "已停用",
	})

	type Device struct {
		Status     string `enum:"deviceStatus" dictField:"StatusName"`
		StatusName string
	}

	device := Device{Status: "1"}
	dict.Translate(&device)
	fmt.Printf("  设备状态: %s -> %s\n", device.Status, device.StatusName)

	device2 := Device{Status: "3"}
	dict.Translate(&device2)
	fmt.Printf("  设备状态: %s -> %s\n", device2.Status, device2.StatusName)
}

// 示例3: 嵌套翻译
func example3_Nested() {
	dict.RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	dict.RegisterEnum("deviceStatus", map[string]string{
		"1": "未使用",
		"2": "试运行",
	})

	type Device struct {
		Status     string `enum:"deviceStatus" dictField:"StatusName"`
		StatusName string
	}

	type User struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
		Device  Device
	}

	user := User{
		Sex:    "1",
		Device: Device{Status: "2"},
	}
	dict.Translate(&user)
	fmt.Printf("  用户: 性别=%s, 设备状态=%s\n", user.SexName, user.Device.StatusName)
}

// 示例4: 切片翻译
func example4_Slice() {
	dict.RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	items := []Item{
		{Status: "1"},
		{Status: "0"},
		{Status: "1"},
	}
	dict.Translate(&items)

	for i, item := range items {
		fmt.Printf("  项目%d: %s -> %s\n", i+1, item.Status, item.StatusName)
	}
}

// 示例5: 包装类型支持
func example5_Wrapper() {
	dict.RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	type Page struct {
		Data []Item `json:"data"`
	}

	// 注册解包器
	dict.RegisterUnWrapper(dict.UnWrapperFunc(func(value interface{}) (interface{}, error) {
		if page, ok := value.(*Page); ok {
			return &page.Data, nil
		}
		return nil, nil
	}))

	page := &Page{
		Data: []Item{
			{Status: "1"},
			{Status: "0"},
		},
	}
	dict.Translate(page)

	for i, item := range page.Data {
		fmt.Printf("  数据%d: %s -> %s\n", i+1, item.Status, item.StatusName)
	}
}

// 示例6: 自定义翻译器
func example6_CustomTranslator() {
	// 注册自定义翻译器（可以连接 Redis、外部 API 等）
	dict.RegisterTranslator("redis", dict.TranslatorFunc(func(value interface{}, fieldName string, tagValue string) (string, error) {
		// 模拟从 Redis 获取数据
		key := fmt.Sprintf("%v", value)
		// 实际项目中: return redis.Get(key)
		return fmt.Sprintf("Redis值_%s", key), nil
	}))

	type Cache struct {
		Key   string `translate:"redis" dictField:"Value"`
		Value string
	}

	cache := Cache{Key: "user:123"}
	dict.Translate(&cache)
	fmt.Printf("  缓存: %s -> %s\n", cache.Key, cache.Value)
}

// 示例7: 数据库翻译（MySQL）
func example7_Database() {
	// 注意：需要先运行 examples/db/setup.sql 创建测试数据
	// 这里使用模拟数据，实际项目中连接真实数据库
	mockDB := map[string]map[string]string{
		"user": {
			"1": "张三",
			"2": "李四",
			"3": "王五",
		},
		"department": {
			"1": "技术部",
			"2": "产品部",
			"3": "运营部",
		},
	}

	// 注册数据库翻译器
	dict.RegisterDBTranslator(dict.DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
		keyStr := fmt.Sprintf("%v", key)
		if tableData, ok := mockDB[table]; ok {
			if value, ok := tableData[keyStr]; ok {
				return value, nil
			}
		}
		return "", nil
	}))

	// 使用简化格式
	type User struct {
		UserID   string `db:"user:id:name" dictField:"UserName"`
		UserName string
	}

	user := User{UserID: "1"}
	dict.Translate(&user)
	fmt.Printf("  用户ID %s -> 用户名: %s\n", user.UserID, user.UserName)

	// 使用完整格式
	type Employee struct {
		DeptID   string `db:"table=department,key=id,value=name" dictField:"DeptName"`
		DeptName string
	}

	emp := Employee{DeptID: "1"}
	dict.Translate(&emp)
	fmt.Printf("  部门ID %s -> 部门名: %s\n", emp.DeptID, emp.DeptName)

	// 测试缓存
	user2 := User{UserID: "1"}
	dict.Translate(&user2)
	fmt.Printf("  缓存测试 - 用户ID %s -> 用户名: %s (使用缓存)\n", user2.UserID, user2.UserName)
}

// 示例8: 字典表翻译
func example8_DictTable() {
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

	// 注册字典表翻译器（实际项目中从数据库读取）
	dict.RegisterDictTableTranslator(dict.DictTableTranslatorFunc(func(dictType, dictKey string) (string, error) {
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
	dict.Translate(&user)
	fmt.Printf("  性别: %s -> %s (从字典表查询)\n", user.Sex, user.SexName)

	type Item struct {
		Status     string `dictTable:"status" dictField:"StatusName"`
		StatusName string
	}

	item := Item{Status: "1"}
	dict.Translate(&item)
	fmt.Printf("  状态: %s -> %s (从字典表查询)\n", item.Status, item.StatusName)

	// 测试缓存
	user2 := User{Sex: "1"}
	dict.Translate(&user2)
	fmt.Printf("  缓存测试: %s -> %s (使用缓存)\n", user2.Sex, user2.SexName)
}

// 示例9: 批量并行翻译
func example9_Batch() {
	dict.RegisterDict("status", map[string]string{
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

	// 顺序处理
	dict.BatchTranslate(&items, false)
	fmt.Printf("  顺序处理: 已翻译 %d 条数据\n", len(items))
	fmt.Printf("  示例: 第1条 %s -> %s\n", items[0].Status, items[0].StatusName)

	// 并行处理（大批量数据）
	items2 := make([]Item, 1000)
	for i := 0; i < 1000; i++ {
		items2[i] = Item{Status: "1"}
	}
	dict.BatchTranslate(&items2, true)
	fmt.Printf("  并行处理: 已翻译 %d 条数据\n", len(items2))
	fmt.Printf("  示例: 第1条 %s -> %s\n", items2[0].Status, items2[0].StatusName)
}

// 示例10: 混合使用
func example10_Mixed() {
	// 注册字典（内存字典）
	dict.RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	// 注册枚举
	dict.RegisterEnum("priority", map[string]string{
		"1": "低",
		"2": "中",
		"3": "高",
	})

	// 注册数据库翻译器（模拟）
	mockDB := map[string]map[string]string{
		"department": {
			"1": "技术部",
			"2": "产品部",
		},
	}
	dict.RegisterDBTranslator(dict.DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
		keyStr := fmt.Sprintf("%v", key)
		if tableData, ok := mockDB[table]; ok {
			if value, ok := tableData[keyStr]; ok {
				return value, nil
			}
		}
		return "", nil
	}))

	// 注册字典表翻译器（模拟）
	mockDictTable := map[string]map[string]string{
		"status": {
			"1": "启用",
			"0": "禁用",
		},
	}
	dict.RegisterDictTableTranslator(dict.DictTableTranslatorFunc(func(dictType, dictKey string) (string, error) {
		if dictData, ok := mockDictTable[dictType]; ok {
			if value, ok := dictData[dictKey]; ok {
				return value, nil
			}
		}
		return "", nil
	}))

	type Employee struct {
		Sex          string `dict:"sex" dictField:"SexName"` // 内存字典
		SexName      string
		DeptID       string `db:"department:id:name" dictField:"DeptName"` // 数据库表
		DeptName     string
		Priority     string `enum:"priority" dictField:"PriorityName"` // 枚举
		PriorityName string
		Status       string `dictTable:"status" dictField:"StatusName"` // 字典表
		StatusName   string
	}

	emp := Employee{
		Sex:      "1",
		DeptID:   "1",
		Priority: "3",
		Status:   "1",
	}
	dict.Translate(&emp)

	fmt.Printf("  员工信息:\n")
	fmt.Printf("    性别: %s -> %s (内存字典)\n", emp.Sex, emp.SexName)
	fmt.Printf("    部门: %s -> %s (数据库表)\n", emp.DeptID, emp.DeptName)
	fmt.Printf("    优先级: %s -> %s (枚举)\n", emp.Priority, emp.PriorityName)
	fmt.Printf("    状态: %s -> %s (字典表)\n", emp.Status, emp.StatusName)
}

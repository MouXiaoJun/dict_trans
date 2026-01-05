package main

import (
	"fmt"

	"github.com/mouxiaojun/dict-trans"
)

func main() {
	// 模拟数据库数据
	mockDB := map[string]map[string]string{
		"user": {
			"1": "张三",
			"2": "李四",
			"3": "王五",
		},
		"device": {
			"1": "设备A",
			"2": "设备B",
		},
	}

	// 注册数据库翻译器（实际项目中可以连接真实的数据库）
	dict.RegisterDBTranslator(dict.DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
		// 将 key 转换为字符串
		keyStr := fmt.Sprintf("%v", key)

		// 模拟数据库查询
		if tableData, ok := mockDB[table]; ok {
			if value, ok := tableData[keyStr]; ok {
				return value, nil
			}
		}
		return "", nil
	}))

	// 示例1: 使用简化格式 db:"table:key:value"
	type User struct {
		UserID   string `db:"user:id:name" dictField:"UserName"`
		UserName string
	}

	user := User{UserID: "1"}
	dict.Translate(&user)
	fmt.Printf("示例1 - 数据库翻译(简化格式): %+v\n", user)

	// 示例2: 使用完整格式 db:"table=user,key=id,value=name"
	type Device struct {
		DeviceID   string `db:"table=device,key=id,value=name" dictField:"DeviceName"`
		DeviceName string
	}

	device := Device{DeviceID: "1"}
	dict.Translate(&device)
	fmt.Printf("示例2 - 数据库翻译(完整格式): %+v\n", device)

	// 示例3: 测试缓存（第二次查询相同值会使用缓存）
	user2 := User{UserID: "1"}
	dict.Translate(&user2)
	fmt.Printf("示例3 - 缓存测试: %+v (应该使用缓存)\n", user2)

	// 示例4: 混合使用（字典 + 数据库）
	dict.RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type UserWithStatus struct {
		UserID     string `db:"user:id:name" dictField:"UserName"`
		UserName   string
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	userWithStatus := UserWithStatus{
		UserID: "2",
		Status: "1",
	}
	dict.Translate(&userWithStatus)
	fmt.Printf("示例4 - 混合使用: %+v\n", userWithStatus)
}

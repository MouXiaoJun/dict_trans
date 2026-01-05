package main

import (
	"fmt"

	"github.com/mouxiaojun/dict-trans"
)

func main() {
	// 注册字典
	dict.RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	// 注册枚举
	dict.RegisterEnum("deviceStatus", map[string]string{
		"1": "未使用",
		"2": "试运行",
		"3": "运行中",
	})

	// 注册自定义翻译器
	dict.RegisterTranslator("custom", dict.TranslatorFunc(func(value interface{}, fieldName string, tagValue string) (string, error) {
		// 这里可以实现从 Redis、数据库等获取翻译
		return fmt.Sprintf("自定义翻译: %v", value), nil
	}))

	// 示例1: 枚举转换
	type Device struct {
		Status     string `enum:"deviceStatus" dictField:"StatusName"`
		StatusName string
	}

	device := Device{Status: "1"}
	dict.Translate(&device)
	fmt.Printf("示例1 - 枚举转换: %+v\n", device)

	// 示例2: 自定义翻译器
	type User struct {
		ID   string `translate:"custom" dictField:"Name"`
		Name string
	}

	user := User{ID: "123"}
	dict.Translate(&user)
	fmt.Printf("示例2 - 自定义翻译器: %+v\n", user)

	// 示例3: 混合使用
	type UserDevice struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
		Device  Device
	}

	userDevice := UserDevice{
		Sex:    "1",
		Device: Device{Status: "2"},
	}
	dict.Translate(&userDevice)
	fmt.Printf("示例3 - 混合使用: %+v\n", userDevice)
}

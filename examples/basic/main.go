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

	dict.RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	// 示例1: 基本翻译
	type User struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
	}

	user := User{Sex: "1"}
	dict.Translate(&user)
	fmt.Printf("示例1 - 基本翻译: %+v\n", user)

	// 示例2: 切片翻译
	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	items := []Item{
		{Status: "1"},
		{Status: "0"},
	}
	dict.Translate(&items)
	fmt.Printf("示例2 - 切片翻译: %+v\n", items)

	// 示例3: 嵌套翻译
	type Device struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	type UserWithDevice struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
		Device  Device
	}

	userWithDevice := UserWithDevice{
		Sex:    "2",
		Device: Device{Status: "1"},
	}
	dict.Translate(&userWithDevice)
	fmt.Printf("示例3 - 嵌套翻译: %+v\n", userWithDevice)
}

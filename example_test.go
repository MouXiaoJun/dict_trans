package dict_test

import (
	"fmt"

	"github.com/MouXiaoJun/dict_trans"
)

// 示例：基本字典翻译
func ExampleTranslate() {
	// 定义结构体
	type User struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
	}

	// 注册字典
	dict.RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	// 翻译
	user := User{Sex: "1"}
	_ = dict.Translate(&user)
	fmt.Println(user.SexName)
	// Output: 男
}

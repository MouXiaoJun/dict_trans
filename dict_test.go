package dict

import (
	"testing"
)

func TestTranslate(t *testing.T) {
	// 注册字典
	RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	type User struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
	}

	// 测试翻译
	user := User{Sex: "1"}
	err := Translate(&user)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if user.SexName != "男" {
		t.Errorf("Expected '男', got '%s'", user.SexName)
	}

	// 测试翻译 "2"
	user2 := User{Sex: "2"}
	err = Translate(&user2)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if user2.SexName != "女" {
		t.Errorf("Expected '女', got '%s'", user2.SexName)
	}
}

func TestTranslateSlice(t *testing.T) {
	// 注册字典
	RegisterDict("status", map[string]string{
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
	}

	err := Translate(&items)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if items[0].StatusName != "启用" {
		t.Errorf("Expected '启用', got '%s'", items[0].StatusName)
	}

	if items[1].StatusName != "禁用" {
		t.Errorf("Expected '禁用', got '%s'", items[1].StatusName)
	}
}

func TestTranslateNested(t *testing.T) {
	// 注册字典
	RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type Device struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	type User struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
		Device  Device
	}

	user := User{
		Sex:    "1",
		Device: Device{Status: "1"},
	}

	err := Translate(&user)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if user.SexName != "男" {
		t.Errorf("Expected '男', got '%s'", user.SexName)
	}

	if user.Device.StatusName != "启用" {
		t.Errorf("Expected '启用', got '%s'", user.Device.StatusName)
	}
}

func TestTranslatePointer(t *testing.T) {
	// 注册字典
	RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	type User struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
	}

	user := &User{Sex: "1"}
	err := Translate(user)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if user.SexName != "男" {
		t.Errorf("Expected '男', got '%s'", user.SexName)
	}
}

func TestTranslateError(t *testing.T) {
	// 测试非指针类型
	user := struct {
		Name string
	}{Name: "test"}

	err := Translate(user)
	if err != ErrNotPointer {
		t.Errorf("Expected ErrNotPointer, got %v", err)
	}

	// 测试非结构体类型
	var str string = "test"
	err = Translate(&str)
	if err != ErrNotStruct {
		t.Errorf("Expected ErrNotStruct, got %v", err)
	}
}

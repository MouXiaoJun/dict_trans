package dict

import (
	"testing"
)

func TestEnumTranslate(t *testing.T) {
	// 注册枚举
	RegisterEnum("deviceStatus", map[string]string{
		"1": "未使用",
		"2": "试运行",
		"3": "运行中",
	})

	type Device struct {
		Status     string `enum:"deviceStatus" dictField:"StatusName"`
		StatusName string
	}

	device := Device{Status: "1"}
	err := Translate(&device)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if device.StatusName != "未使用" {
		t.Errorf("Expected '未使用', got '%s'", device.StatusName)
	}

	device2 := Device{Status: "2"}
	err = Translate(&device2)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if device2.StatusName != "试运行" {
		t.Errorf("Expected '试运行', got '%s'", device2.StatusName)
	}
}

func TestEnumTranslateInt(t *testing.T) {
	// 注册枚举
	RegisterEnum("priority", map[string]string{
		"1": "低",
		"2": "中",
		"3": "高",
	})

	type Task struct {
		Priority     int `enum:"priority" dictField:"PriorityName"`
		PriorityName string
	}

	task := Task{Priority: 2}
	err := Translate(&task)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if task.PriorityName != "中" {
		t.Errorf("Expected '中', got '%s'", task.PriorityName)
	}
}

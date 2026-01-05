package dict

import (
	"testing"
)

func TestUnWrapper(t *testing.T) {
	// 注册字典
	RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	// 定义包装类型
	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	type Page struct {
		Data []Item `json:"data"`
	}

	// 注册解包器
	RegisterUnWrapper(UnWrapperFunc(func(value interface{}) (interface{}, error) {
		// 尝试解包 Page 类型
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

	err := Translate(page)
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}

	if page.Data[0].StatusName != "启用" {
		t.Errorf("Expected '启用', got '%s'", page.Data[0].StatusName)
	}

	if page.Data[1].StatusName != "禁用" {
		t.Errorf("Expected '禁用', got '%s'", page.Data[1].StatusName)
	}
}

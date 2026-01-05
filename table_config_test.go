package dict

import (
	"testing"
)

func TestTableConfig_BuildQuery(t *testing.T) {
	config := DefaultTableConfig("test_table")
	query, args := config.BuildQuery("sex")

	expectedQuery := "SELECT dict_value FROM test_table WHERE dict_type = ? AND status = ?"
	if query != expectedQuery {
		t.Errorf("Expected query '%s', got '%s'", expectedQuery, query)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if args[0] != "sex" {
		t.Errorf("Expected first arg 'sex', got '%v'", args[0])
	}

	if args[1] != "1" {
		t.Errorf("Expected second arg '1', got '%v'", args[1])
	}
}

func TestTableConfig_BuildQueryWithKey(t *testing.T) {
	config := DefaultTableConfig("test_table")
	query, args := config.BuildQueryWithKey("sex", "1")

	expectedQuery := "SELECT dict_value FROM test_table WHERE dict_type = ? AND dict_key = ? AND status = ?"
	if query != expectedQuery {
		t.Errorf("Expected query '%s', got '%s'", expectedQuery, query)
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
}

func TestTableConfig_CustomFields(t *testing.T) {
	config := &TableConfig{
		TableName: "custom_dict",
		Fields: TableFields{
			TypeField:  "type_code",
			KeyField:   "code",
			ValueField: "label",
		},
		StatusField: &StatusFieldConfig{
			FieldName:     "is_active",
			EnabledValue:  "Y",
			DisabledValue: "N",
		},
	}

	query, args := config.BuildQueryWithKey("product_type", "001")
	expectedQuery := "SELECT label FROM custom_dict WHERE type_code = ? AND code = ? AND is_active = ?"
	if query != expectedQuery {
		t.Errorf("Expected query '%s', got '%s'", expectedQuery, query)
	}

	if args[2] != "Y" {
		t.Errorf("Expected status value 'Y', got '%v'", args[2])
	}
}

func TestTableConfig_NoStatusField(t *testing.T) {
	config := &TableConfig{
		TableName: "simple_dict",
		Fields: TableFields{
			TypeField:  "type",
			KeyField:   "key",
			ValueField: "value",
		},
		StatusField: nil, // 无状态字段
	}

	query, args := config.BuildQueryWithKey("sex", "1")
	expectedQuery := "SELECT value FROM simple_dict WHERE type = ? AND key = ?"
	if query != expectedQuery {
		t.Errorf("Expected query '%s', got '%s'", expectedQuery, query)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args (no status), got %d", len(args))
	}
}

package dict

// TableConfig 表结构配置
// 用于自定义字典表的表结构
type TableConfig struct {
	// 表名
	TableName string

	// 字段映射
	Fields TableFields

	// 状态字段配置（可选）
	StatusField *StatusFieldConfig

	// 排序字段（可选）
	SortField string
}

// TableFields 表字段映射
type TableFields struct {
	// 字典类型字段名（单表字典使用）
	TypeField string

	// 字典键字段名
	KeyField string

	// 字典值字段名
	ValueField string
}

// StatusFieldConfig 状态字段配置
type StatusFieldConfig struct {
	// 状态字段名
	FieldName string

	// 启用状态的值（如 "1", "Y", "true"）
	EnabledValue string

	// 禁用状态的值（如 "0", "N", "false"）
	DisabledValue string
}

// DefaultTableConfig 默认表结构配置（单表字典）
func DefaultTableConfig(tableName string) *TableConfig {
	return &TableConfig{
		TableName: tableName,
		Fields: TableFields{
			TypeField:  "dict_type",
			KeyField:   "dict_key",
			ValueField: "dict_value",
		},
		StatusField: &StatusFieldConfig{
			FieldName:     "status",
			EnabledValue:  "1",
			DisabledValue: "0",
		},
	}
}

// DefaultDictTypeTableConfig 默认字典类型表配置（双表字典）
func DefaultDictTypeTableConfig(tableName string) *TableConfig {
	return &TableConfig{
		TableName: tableName,
		Fields: TableFields{
			TypeField:  "dict_type_code",
			KeyField:   "", // 类型表不需要 key
			ValueField: "dict_type_name",
		},
		StatusField: &StatusFieldConfig{
			FieldName:     "status",
			EnabledValue:  "1",
			DisabledValue: "0",
		},
	}
}

// DefaultDictDataTableConfig 默认字典数据表配置（双表字典）
func DefaultDictDataTableConfig(tableName string) *TableConfig {
	return &TableConfig{
		TableName: tableName,
		Fields: TableFields{
			TypeField:  "dict_type_code",
			KeyField:   "dict_key",
			ValueField: "dict_value",
		},
		StatusField: &StatusFieldConfig{
			FieldName:     "status",
			EnabledValue:  "1",
			DisabledValue: "0",
		},
	}
}

// BuildQuery 构建查询 SQL
func (tc *TableConfig) BuildQuery(dictType string) (string, []interface{}) {
	query := "SELECT " + tc.Fields.ValueField + " FROM " + tc.TableName + " WHERE "
	args := []interface{}{}

	// 添加类型条件（如果有）
	if tc.Fields.TypeField != "" {
		query += tc.Fields.TypeField + " = ?"
		args = append(args, dictType)
	}

	// 添加状态条件（如果配置了状态字段）
	if tc.StatusField != nil {
		if len(args) > 0 {
			query += " AND "
		}
		query += tc.StatusField.FieldName + " = ?"
		args = append(args, tc.StatusField.EnabledValue)
	}

	return query, args
}

// BuildQueryWithKey 构建带键的查询 SQL
func (tc *TableConfig) BuildQueryWithKey(dictType, dictKey string) (string, []interface{}) {
	query := "SELECT " + tc.Fields.ValueField + " FROM " + tc.TableName + " WHERE "
	args := []interface{}{}

	// 添加类型条件（如果有）
	if tc.Fields.TypeField != "" {
		query += tc.Fields.TypeField + " = ?"
		args = append(args, dictType)
	}

	// 添加键条件
	if tc.Fields.KeyField != "" {
		if len(args) > 0 {
			query += " AND "
		}
		query += tc.Fields.KeyField + " = ?"
		args = append(args, dictKey)
	}

	// 添加状态条件（如果配置了状态字段）
	if tc.StatusField != nil {
		if len(args) > 0 {
			query += " AND "
		}
		query += tc.StatusField.FieldName + " = ?"
		args = append(args, tc.StatusField.EnabledValue)
	}

	return query, args
}

// BuildTypeCheckQuery 构建类型检查查询（用于双表字典）
func (tc *TableConfig) BuildTypeCheckQuery(dictTypeCode string) (string, []interface{}) {
	query := "SELECT COUNT(1) FROM " + tc.TableName + " WHERE "
	args := []interface{}{}

	// 类型字段（通常是 dict_type_code）
	if tc.Fields.TypeField != "" {
		query += tc.Fields.TypeField + " = ?"
		args = append(args, dictTypeCode)
	}

	// 添加状态条件（如果配置了状态字段）
	if tc.StatusField != nil {
		if len(args) > 0 {
			query += " AND "
		}
		query += tc.StatusField.FieldName + " = ?"
		args = append(args, tc.StatusField.EnabledValue)
	}

	return query, args
}

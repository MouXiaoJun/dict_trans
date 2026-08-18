package dict

// DBTranslator 数据库翻译器接口：按 table / keyField / valueField 查一个 key 对应的 value。
// 可选实现 DBContextTranslator（ctx）和 DBBatchTranslator（批量），批量翻译时会自动使用。
type DBTranslator interface {
	// Query 查询数据库
	// table: 表名, keyField: 键字段, valueField: 值字段, key: 键值
	Query(table, keyField, valueField string, key any) (string, error)
}

// DBTranslatorFunc 数据库翻译器函数类型
type DBTranslatorFunc func(table, keyField, valueField string, key any) (string, error)

// Query 实现 DBTranslator 接口
func (f DBTranslatorFunc) Query(table, keyField, valueField string, key any) (string, error) {
	return f(table, keyField, valueField, key)
}

package dict

// Translator 翻译器接口
type Translator interface {
	// Translate 翻译方法
	// value: 源字段的值
	// fieldName: 字段名
	// tagValue: 标签的值
	// 返回: 翻译后的值
	Translate(value interface{}, fieldName string, tagValue string) (string, error)
}

// TranslatorFunc 翻译器函数类型
type TranslatorFunc func(value interface{}, fieldName string, tagValue string) (string, error)

// Translate 实现 Translator 接口
func (f TranslatorFunc) Translate(value interface{}, fieldName string, tagValue string) (string, error) {
	return f(value, fieldName, tagValue)
}

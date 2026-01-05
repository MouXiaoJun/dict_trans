package dict

import (
	"fmt"
	"reflect"
)

// EnumTranslator 枚举翻译器
type EnumTranslator struct {
	enumMap map[string]map[string]string // enumName -> {key: value}
}

var defaultEnumTranslator = &EnumTranslator{
	enumMap: make(map[string]map[string]string),
}

// RegisterEnum 注册枚举
func RegisterEnum(name string, enum map[string]string) {
	defaultEnumTranslator.enumMap[name] = enum
}

// GetEnum 获取枚举
func GetEnum(name string) map[string]string {
	return defaultEnumTranslator.enumMap[name]
}

// Translate 实现 Translator 接口
func (e *EnumTranslator) Translate(value interface{}, fieldName string, tagValue string) (string, error) {
	// tagValue 是枚举名称
	enum := e.enumMap[tagValue]
	if enum == nil {
		return "", fmt.Errorf("enum '%s' not found", tagValue)
	}

	// 将 value 转换为字符串
	var key string
	switch v := value.(type) {
	case string:
		key = v
	case int, int8, int16, int32, int64:
		key = fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		key = fmt.Sprintf("%d", v)
	default:
		// 尝试使用反射
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.String {
			key = rv.String()
		} else if rv.CanInt() {
			key = fmt.Sprintf("%d", rv.Int())
		} else if rv.CanUint() {
			key = fmt.Sprintf("%d", rv.Uint())
		} else {
			return "", fmt.Errorf("unsupported enum value type: %T", value)
		}
	}

	result := enum[key]
	return result, nil
}

// DefaultEnumTranslator 获取默认枚举翻译器
func DefaultEnumTranslator() *EnumTranslator {
	return defaultEnumTranslator
}

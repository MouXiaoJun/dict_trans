package dict

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// EnumTranslator 枚举翻译器。枚举表读多写少，与 DictManager 的注册表一样用写时复制：
// 读一次原子 Load 无锁，写在 mu 下拷贝后 Store。
type EnumTranslator struct {
	enums atomic.Pointer[map[string]map[string]string] // enumName -> {key: value}
	mu    sync.Mutex                                   // 串行化写者
}

var defaultEnumTranslator = &EnumTranslator{}

func (e *EnumTranslator) load() map[string]map[string]string {
	if m := e.enums.Load(); m != nil {
		return *m
	}
	return nil
}

// Register 注册枚举（并发安全）
func (e *EnumTranslator) Register(name string, enum map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	old := e.load()
	next := make(map[string]map[string]string, len(old)+1)
	for k, v := range old {
		next[k] = v
	}
	next[name] = enum
	e.enums.Store(&next)
}

// Get 获取枚举
func (e *EnumTranslator) Get(name string) map[string]string {
	return e.load()[name]
}

// RegisterEnum 注册枚举
func RegisterEnum(name string, enum map[string]string) {
	defaultEnumTranslator.Register(name, enum)
}

// GetEnum 获取枚举
func GetEnum(name string) map[string]string {
	return defaultEnumTranslator.Get(name)
}

// Translate 实现 Translator 接口
func (e *EnumTranslator) Translate(value any, fieldName string, tagValue string) (string, error) {
	// tagValue 是枚举名称
	enum := e.load()[tagValue]
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

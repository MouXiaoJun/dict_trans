package dict

import "reflect"

// UnWrapper 包装类型解包器接口
// 用于处理 Page、Result 等包装类型，提取其中的实际数据
type UnWrapper interface {
	// UnWrap 解包方法
	// value: 包装类型的值
	// 返回: 解包后的实际数据（通常是切片或结构体）
	UnWrap(value interface{}) (interface{}, error)
}

// UnWrapperFunc 解包器函数类型
type UnWrapperFunc func(value interface{}) (interface{}, error)

// UnWrap 实现 UnWrapper 接口
func (f UnWrapperFunc) UnWrap(value interface{}) (interface{}, error) {
	return f(value)
}

// RegisterUnWrapper 注册解包器
func RegisterUnWrapper(unwrapper UnWrapper) {
	defaultManager.unwrappers = append(defaultManager.unwrappers, unwrapper)
}

// tryUnwrap 尝试解包
func (dm *DictManager) tryUnwrap(v interface{}) (interface{}, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	// 尝试所有解包器
	for _, unwrapper := range dm.unwrappers {
		if result, err := unwrapper.UnWrap(v); err == nil && result != nil {
			return result, true
		}
	}

	return nil, false
}

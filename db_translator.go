package dict

import (
	"fmt"
	"sync"
)

// DBTranslator 数据库翻译器接口
// 用于从数据库查询翻译数据（类似 Easy Trans 的自动查表功能）
type DBTranslator interface {
	// Query 查询数据库获取翻译
	// table: 表名
	// keyField: 主键字段名（如 "id"）
	// valueField: 值字段名（如 "name"）
	// key: 要查询的键值
	// 返回: 翻译后的值
	Query(table, keyField, valueField string, key interface{}) (string, error)
}

// DBTranslatorFunc 数据库翻译器函数类型
type DBTranslatorFunc func(table, keyField, valueField string, key interface{}) (string, error)

// Query 实现 DBTranslator 接口
func (f DBTranslatorFunc) Query(table, keyField, valueField string, key interface{}) (string, error) {
	return f(table, keyField, valueField, key)
}

// dbTranslatorManager 数据库翻译器管理器
type dbTranslatorManager struct {
	translator DBTranslator
	cache      map[string]string // 结果缓存: "table:keyField:valueField:key" -> value
	cacheMutex sync.RWMutex
	enabled    bool // 是否启用缓存
}

var defaultDBTranslatorManager = &dbTranslatorManager{
	cache:   make(map[string]string),
	enabled: true,
}

// RegisterDBTranslator 注册数据库翻译器
func RegisterDBTranslator(translator DBTranslator) {
	defaultDBTranslatorManager.translator = translator
}

// EnableDBCache 启用/禁用数据库翻译缓存
func EnableDBCache(enabled bool) {
	defaultDBTranslatorManager.enabled = enabled
}

// ClearDBCache 清空数据库翻译缓存
func ClearDBCache() {
	defaultDBTranslatorManager.cacheMutex.Lock()
	defer defaultDBTranslatorManager.cacheMutex.Unlock()
	defaultDBTranslatorManager.cache = make(map[string]string)
}

// createDBTranslator 创建数据库翻译器实例
func createDBTranslator(table, keyField, valueField string) Translator {
	return TranslatorFunc(func(value interface{}, fieldName string, tagValue string) (string, error) {
		manager := defaultDBTranslatorManager

		// 构建缓存键
		cacheKey := fmt.Sprintf("%s:%s:%s:%v", table, keyField, valueField, value)

		// 尝试从缓存获取
		if manager.enabled {
			manager.cacheMutex.RLock()
			if cached, ok := manager.cache[cacheKey]; ok {
				manager.cacheMutex.RUnlock()
				return cached, nil
			}
			manager.cacheMutex.RUnlock()
		}

		// 从数据库查询
		if manager.translator == nil {
			return "", fmt.Errorf("database translator not registered")
		}

		result, err := manager.translator.Query(table, keyField, valueField, value)
		if err != nil {
			return "", err
		}

		// 存入缓存
		if manager.enabled && result != "" {
			manager.cacheMutex.Lock()
			manager.cache[cacheKey] = result
			manager.cacheMutex.Unlock()
		}

		return result, nil
	})
}

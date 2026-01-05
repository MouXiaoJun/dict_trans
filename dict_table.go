package dict

import (
	"database/sql"
	"fmt"
	"sync"
)

// DictTableTranslator 字典表翻译器接口
// 用于从数据库字典表读取数据
type DictTableTranslator interface {
	// QueryDict 查询字典表
	// dictType: 字典类型（如 "sex", "status"）
	// dictKey: 字典键（如 "1", "2"）
	// 返回: 字典值（如 "男", "女"）
	QueryDict(dictType, dictKey string) (string, error)
}

// DictTableTranslatorFunc 字典表翻译器函数类型
type DictTableTranslatorFunc func(dictType, dictKey string) (string, error)

// QueryDict 实现 DictTableTranslator 接口
func (f DictTableTranslatorFunc) QueryDict(dictType, dictKey string) (string, error) {
	return f(dictType, dictKey)
}

// dictTableManager 字典表管理器
type dictTableManager struct {
	translator DictTableTranslator
	cache      map[string]string // 结果缓存: "dictType:dictKey" -> value
	cacheMutex sync.RWMutex
	enabled    bool // 是否启用缓存
}

var defaultDictTableManager = &dictTableManager{
	cache:   make(map[string]string),
	enabled: true,
}

// RegisterDictTableTranslator 注册字典表翻译器
func RegisterDictTableTranslator(translator DictTableTranslator) {
	defaultDictTableManager.translator = translator
}

// EnableDictTableCache 启用/禁用字典表缓存
func EnableDictTableCache(enabled bool) {
	defaultDictTableManager.enabled = enabled
}

// ClearDictTableCache 清空字典表缓存
func ClearDictTableCache() {
	defaultDictTableManager.cacheMutex.Lock()
	defer defaultDictTableManager.cacheMutex.Unlock()
	defaultDictTableManager.cache = make(map[string]string)
}

// createDictTableTranslator 创建字典表翻译器实例
func createDictTableTranslator(dictType string) Translator {
	return TranslatorFunc(func(value interface{}, fieldName string, tagValue string) (string, error) {
		manager := defaultDictTableManager

		// 将 value 转换为字符串
		dictKey := fmt.Sprintf("%v", value)

		// 构建缓存键
		cacheKey := fmt.Sprintf("%s:%s", dictType, dictKey)

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
			return "", fmt.Errorf("dict table translator not registered")
		}

		result, err := manager.translator.QueryDict(dictType, dictKey)
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

// CreateDictTableTranslatorFromDB 从数据库连接创建字典表翻译器
// 适用于标准的字典表结构：
//   - dict_type: 字典类型字段
//   - dict_key: 字典键字段
//   - dict_value: 字典值字段
//   - table_name: 字典表名（默认 "sys_dict"）
func CreateDictTableTranslatorFromDB(db *sql.DB, tableName string) DictTableTranslator {
	if tableName == "" {
		tableName = "sys_dict"
	}

	return DictTableTranslatorFunc(func(dictType, dictKey string) (string, error) {
		query := fmt.Sprintf("SELECT dict_value FROM %s WHERE dict_type = ? AND dict_key = ? AND status = '1'", tableName)

		var result string
		err := db.QueryRow(query, dictType, dictKey).Scan(&result)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", nil // 未找到记录，返回空字符串
			}
			return "", fmt.Errorf("查询字典表失败: %v", err)
		}

		return result, nil
	})
}

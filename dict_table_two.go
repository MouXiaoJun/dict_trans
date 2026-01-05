package dict

import (
	"database/sql"
	"fmt"
	"sync"
)

// DictTableTwoTranslator 双表字典翻译器接口
// 支持字典类型表和字典数据表分离的设计
type DictTableTwoTranslator interface {
	// QueryDict 查询字典数据
	// dictTypeCode: 字典类型编码（如 "sex", "status"）
	// dictKey: 字典键（如 "1", "2"）
	// 返回: 字典值（如 "男", "女"）
	QueryDict(dictTypeCode, dictKey string) (string, error)
}

// DictTableTwoTranslatorFunc 双表字典翻译器函数类型
type DictTableTwoTranslatorFunc func(dictTypeCode, dictKey string) (string, error)

// QueryDict 实现 DictTableTwoTranslator 接口
func (f DictTableTwoTranslatorFunc) QueryDict(dictTypeCode, dictKey string) (string, error) {
	return f(dictTypeCode, dictKey)
}

// dictTableTwoManager 双表字典管理器
type dictTableTwoManager struct {
	translator DictTableTwoTranslator
	cache      map[string]string // 结果缓存: "dictTypeCode:dictKey" -> value
	cacheMutex sync.RWMutex
	enabled    bool // 是否启用缓存
}

var defaultDictTableTwoManager = &dictTableTwoManager{
	cache:   make(map[string]string),
	enabled: true,
}

// RegisterDictTableTwoTranslator 注册双表字典翻译器
func RegisterDictTableTwoTranslator(translator DictTableTwoTranslator) {
	defaultDictTableTwoManager.translator = translator
}

// EnableDictTableTwoCache 启用/禁用双表字典缓存
func EnableDictTableTwoCache(enabled bool) {
	defaultDictTableTwoManager.enabled = enabled
}

// ClearDictTableTwoCache 清空双表字典缓存
func ClearDictTableTwoCache() {
	defaultDictTableTwoManager.cacheMutex.Lock()
	defer defaultDictTableTwoManager.cacheMutex.Unlock()
	defaultDictTableTwoManager.cache = make(map[string]string)
}

// createDictTableTwoTranslator 创建双表字典翻译器实例
func createDictTableTwoTranslator(dictTypeCode string) Translator {
	return TranslatorFunc(func(value interface{}, fieldName string, tagValue string) (string, error) {
		manager := defaultDictTableTwoManager

		// 将 value 转换为字符串
		dictKey := fmt.Sprintf("%v", value)

		// 构建缓存键
		cacheKey := fmt.Sprintf("%s:%s", dictTypeCode, dictKey)

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
			return "", fmt.Errorf("dict table two translator not registered")
		}

		result, err := manager.translator.QueryDict(dictTypeCode, dictKey)
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

// CreateDictTableTwoTranslatorFromDB 从数据库连接创建双表字典翻译器
// 适用于标准的双表字典结构：
//   - dict_type 表：存储字典类型（dict_type_code, dict_type_name）
//   - dict_data 表：存储字典数据（dict_type_code, dict_key, dict_value）
func CreateDictTableTwoTranslatorFromDB(db *sql.DB, dictTypeTable, dictDataTable string) DictTableTwoTranslator {
	return CreateDictTableTwoTranslatorFromDBWithConfig(
		db,
		DefaultDictTypeTableConfig(dictTypeTable),
		DefaultDictDataTableConfig(dictDataTable),
	)
}

// CreateDictTableTwoTranslatorFromDBWithConfig 从数据库连接创建双表字典翻译器（支持自定义表结构）
func CreateDictTableTwoTranslatorFromDBWithConfig(db *sql.DB, typeConfig, dataConfig *TableConfig) DictTableTwoTranslator {
	if typeConfig == nil {
		typeConfig = DefaultDictTypeTableConfig("sys_dict_type")
	}
	if dataConfig == nil {
		dataConfig = DefaultDictDataTableConfig("sys_dict_data")
	}

	return DictTableTwoTranslatorFunc(func(dictTypeCode, dictKey string) (string, error) {
		// 先验证字典类型是否存在且启用
		typeQuery, typeArgs := typeConfig.BuildTypeCheckQuery(dictTypeCode)
		var typeCount int
		err := db.QueryRow(typeQuery, typeArgs...).Scan(&typeCount)
		if err != nil {
			return "", fmt.Errorf("查询字典类型失败: %v", err)
		}
		if typeCount == 0 {
			return "", nil // 字典类型不存在或已禁用
		}

		// 查询字典数据
		dataQuery, dataArgs := dataConfig.BuildQueryWithKey(dictTypeCode, dictKey)
		var result string
		err = db.QueryRow(dataQuery, dataArgs...).Scan(&result)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", nil // 未找到记录，返回空字符串
			}
			return "", fmt.Errorf("查询字典数据失败: %v", err)
		}

		return result, nil
	})
}

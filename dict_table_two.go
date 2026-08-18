package dict

import (
	"context"
	"database/sql"
	"fmt"
)

// DictTableTwoTranslator 双表字典翻译器接口（字典类型表 + 字典数据表）。
// 可选实现 DictTableContextTranslator（ctx）、DictTableBatchTranslator（批量）、DictTableLoader（预加载）。
type DictTableTwoTranslator interface {
	// QueryDict 查询字典数据
	// dictTypeCode: 字典类型编码, dictKey: 字典键
	QueryDict(dictTypeCode, dictKey string) (string, error)
}

// DictTableTwoTranslatorFunc 双表字典翻译器函数类型
type DictTableTwoTranslatorFunc func(dictTypeCode, dictKey string) (string, error)

// QueryDict 实现 DictTableTwoTranslator 接口
func (f DictTableTwoTranslatorFunc) QueryDict(dictTypeCode, dictKey string) (string, error) {
	return f(dictTypeCode, dictKey)
}

// CreateDictTableTwoTranslatorFromDB 从数据库创建双表字典翻译器（默认表结构）
func CreateDictTableTwoTranslatorFromDB(db *sql.DB, dictTypeTable, dictDataTable string) DictTableTwoTranslator {
	return CreateDictTableTwoTranslatorFromDBWithConfig(
		db,
		DefaultDictTypeTableConfig(dictTypeTable),
		DefaultDictDataTableConfig(dictDataTable),
	)
}

// CreateDictTableTwoTranslatorFromDBWithConfig 从数据库创建双表字典翻译器（自定义表结构）。
// 返回值同时实现 DictTableContextTranslator / DictTableBatchTranslator / DictTableLoader。
func CreateDictTableTwoTranslatorFromDBWithConfig(db *sql.DB, typeConfig, dataConfig *TableConfig) DictTableTwoTranslator {
	if typeConfig == nil {
		typeConfig = DefaultDictTypeTableConfig("sys_dict_type")
	}
	if dataConfig == nil {
		dataConfig = DefaultDictDataTableConfig("sys_dict_data")
	}
	return &sqlDictTableTwo{db: db, typeCfg: typeConfig, dataCfg: dataConfig}
}

// sqlDictTableTwo 基于 database/sql 的双表字典后端：先确认类型存在且启用，再查数据表
type sqlDictTableTwo struct {
	db      *sql.DB
	typeCfg *TableConfig
	dataCfg *TableConfig
}

func (t *sqlDictTableTwo) typeEnabled(ctx context.Context, dictTypeCode string) (bool, error) {
	query, args := t.typeCfg.BuildTypeCheckQuery(dictTypeCode)
	var n int
	if err := t.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return false, fmt.Errorf("查询字典类型失败: %w", err)
	}
	return n > 0, nil
}

func (t *sqlDictTableTwo) QueryDict(dictTypeCode, dictKey string) (string, error) {
	return t.QueryDictContext(context.Background(), dictTypeCode, dictKey)
}

func (t *sqlDictTableTwo) QueryDictContext(ctx context.Context, dictTypeCode, dictKey string) (string, error) {
	ok, err := t.typeEnabled(ctx, dictTypeCode)
	if err != nil || !ok {
		return "", err
	}
	query, args := t.dataCfg.BuildQueryWithKey(dictTypeCode, dictKey)
	var result string
	if err := t.db.QueryRowContext(ctx, query, args...).Scan(&result); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("查询字典数据失败: %w", err)
	}
	return result, nil
}

func (t *sqlDictTableTwo) QueryDictBatch(ctx context.Context, dictTypeCode string, dictKeys []string) (map[string]string, error) {
	if len(dictKeys) == 0 {
		return map[string]string{}, nil
	}
	ok, err := t.typeEnabled(ctx, dictTypeCode)
	if err != nil || !ok {
		return nil, err
	}
	query, args := t.dataCfg.BuildQueryIn(dictTypeCode, dictKeys)
	return scanKeyValues(ctx, t.db, query, args, "查询字典数据失败")
}

func (t *sqlDictTableTwo) LoadDict(ctx context.Context, dictTypeCode string) (map[string]string, error) {
	ok, err := t.typeEnabled(ctx, dictTypeCode)
	if err != nil || !ok {
		return nil, err
	}
	query, args := t.dataCfg.BuildQueryAll(dictTypeCode)
	return scanKeyValues(ctx, t.db, query, args, "预加载字典数据失败")
}

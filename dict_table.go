package dict

import (
	"context"
	"database/sql"
	"fmt"
)

// DictTableTranslator 字典表翻译器接口（单表：dict_type / dict_key / dict_value）。
// 可选实现 DictTableContextTranslator（ctx）、DictTableBatchTranslator（批量）、DictTableLoader（预加载）。
type DictTableTranslator interface {
	// QueryDict 查询字典表
	// dictType: 字典类型, dictKey: 字典键
	QueryDict(dictType, dictKey string) (string, error)
}

// DictTableTranslatorFunc 字典表翻译器函数类型
type DictTableTranslatorFunc func(dictType, dictKey string) (string, error)

// QueryDict 实现 DictTableTranslator 接口
func (f DictTableTranslatorFunc) QueryDict(dictType, dictKey string) (string, error) {
	return f(dictType, dictKey)
}

// CreateDictTableTranslatorFromDB 从数据库创建字典表翻译器（默认表结构）
func CreateDictTableTranslatorFromDB(db *sql.DB, tableName string) DictTableTranslator {
	return CreateDictTableTranslatorFromDBWithConfig(db, DefaultTableConfig(tableName))
}

// CreateDictTableTranslatorFromDBWithConfig 从数据库创建字典表翻译器（自定义表结构）。
// 返回值同时实现 DictTableContextTranslator / DictTableBatchTranslator / DictTableLoader。
func CreateDictTableTranslatorFromDBWithConfig(db *sql.DB, config *TableConfig) DictTableTranslator {
	if config == nil {
		config = DefaultTableConfig("sys_dict")
	}
	return &sqlDictTable{db: db, cfg: config}
}

// sqlDictTable 基于 database/sql 的单表字典后端
type sqlDictTable struct {
	db  *sql.DB
	cfg *TableConfig
}

func (t *sqlDictTable) QueryDict(dictType, dictKey string) (string, error) {
	return t.QueryDictContext(context.Background(), dictType, dictKey)
}

func (t *sqlDictTable) QueryDictContext(ctx context.Context, dictType, dictKey string) (string, error) {
	query, args := t.cfg.BuildQueryWithKey(dictType, dictKey)
	var result string
	if err := t.db.QueryRowContext(ctx, query, args...).Scan(&result); err != nil {
		if err == sql.ErrNoRows {
			return "", nil // 未找到记录
		}
		return "", fmt.Errorf("查询字典表失败: %w", err)
	}
	return result, nil
}

func (t *sqlDictTable) QueryDictBatch(ctx context.Context, dictType string, dictKeys []string) (map[string]string, error) {
	if len(dictKeys) == 0 {
		return map[string]string{}, nil
	}
	query, args := t.cfg.BuildQueryIn(dictType, dictKeys)
	return scanKeyValues(ctx, t.db, query, args, "查询字典表失败")
}

func (t *sqlDictTable) LoadDict(ctx context.Context, dictType string) (map[string]string, error) {
	query, args := t.cfg.BuildQueryAll(dictType)
	return scanKeyValues(ctx, t.db, query, args, "预加载字典表失败")
}

// scanKeyValues 执行 SELECT key, value ... 并装成 map
func scanKeyValues(ctx context.Context, db *sql.DB, query string, args []any, errPrefix string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("%s: %w", errPrefix, err)
		}
		out[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	return out, nil
}

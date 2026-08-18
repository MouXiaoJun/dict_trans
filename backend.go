package dict

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// ---------------------------------------------------------------------------
// 可选接口（Go 惯用的"实现了就用"升级方式，不破坏已有接口）
// ---------------------------------------------------------------------------

// ContextTranslator 翻译器可选接口：需要超时 / 取消 / trace 时实现它，
// TranslateWith(v, WithContext(ctx)) 会优先调用 TranslateContext。
type ContextTranslator interface {
	TranslateContext(ctx context.Context, value any, fieldName string, tagValue string) (string, error)
}

// DBContextTranslator DBTranslator 的 ctx 版可选接口
type DBContextTranslator interface {
	QueryContext(ctx context.Context, table, keyField, valueField string, key any) (string, error)
}

// DBBatchTranslator DBTranslator 的批量可选接口：一次查多个 key，返回 key -> value（缺失的 key 不出现在 map 里）
type DBBatchTranslator interface {
	QueryBatch(ctx context.Context, table, keyField, valueField string, keys []string) (map[string]string, error)
}

// DictTableContextTranslator DictTableTranslator / DictTableTwoTranslator 的 ctx 版可选接口
type DictTableContextTranslator interface {
	QueryDictContext(ctx context.Context, dictType, dictKey string) (string, error)
}

// DictTableBatchTranslator DictTableTranslator / DictTableTwoTranslator 的批量可选接口
type DictTableBatchTranslator interface {
	QueryDictBatch(ctx context.Context, dictType string, dictKeys []string) (map[string]string, error)
}

// DictTableLoader DictTableTranslator 的预加载可选接口：一次取出某个字典类型的全部 key -> value，
// Framework.Init 按 Config.Performance.PreloadDicts 调用它预热缓存。
type DictTableLoader interface {
	LoadDict(ctx context.Context, dictType string) (map[string]string, error)
}

// ---------------------------------------------------------------------------
// 结果缓存：Decorator——包在 DB 后端外面。默认进程内 map；Config.Cache.Enabled 且设置了 CustomCache（如 Redis）时走它，
// TTL 取 Config.Cache.TTL。EnableXCache(false) 相当于摘掉这层装饰器。
// ---------------------------------------------------------------------------

type resultCache struct {
	name    string // 前缀，三类缓存共用一个 CustomCache 时不撞 key
	enabled atomic.Bool
	mu      sync.RWMutex
	m       map[string]string
}

func newResultCache(name string) *resultCache {
	c := &resultCache{name: name, m: make(map[string]string)}
	c.enabled.Store(true)
	return c
}

func (c *resultCache) get(key string) (string, bool) {
	if !c.enabled.Load() {
		return "", false
	}
	if cfg := GetConfig(); cfg.Cache.Enabled && cfg.Cache.CustomCache != nil {
		return cfg.Cache.CustomCache.Get(c.name + ":" + key)
	}
	c.mu.RLock()
	v, ok := c.m[key]
	c.mu.RUnlock()
	return v, ok
}

func (c *resultCache) set(key, value string) {
	if !c.enabled.Load() || value == "" {
		return
	}
	if cfg := GetConfig(); cfg.Cache.Enabled && cfg.Cache.CustomCache != nil {
		_ = cfg.Cache.CustomCache.Set(c.name+":"+key, value, cfg.Cache.TTL)
		return
	}
	c.mu.Lock()
	c.m[key] = value
	c.mu.Unlock()
}

// clear 清空本地缓存；配置了 CustomCache 时调用它的 Clear（由实现决定范围）
func (c *resultCache) clear() {
	if cfg := GetConfig(); cfg.Cache.Enabled && cfg.Cache.CustomCache != nil {
		_ = cfg.Cache.CustomCache.Clear()
	}
	c.mu.Lock()
	c.m = make(map[string]string)
	c.mu.Unlock()
}

// ---------------------------------------------------------------------------
// lookupManager：一类 DB 后端（db / dictTable / dictTableTwo）= 已注册后端 + 结果缓存
// group 是查找分组（字典类型，或 table|keyField|valueField），key 是字典键
// ---------------------------------------------------------------------------

type lookupManager struct {
	name    string
	backend atomic.Pointer[lookupBackend] // 用户注册的后端，写时整体替换
	cache   *resultCache
}

// lookupBackend 把三类后端接口统一成 one / many / load 三个能力；many / load 为 nil 表示不支持
type lookupBackend struct {
	one  func(ctx context.Context, group, key string) (string, error)
	many func(ctx context.Context, group string, keys []string) (map[string]string, error)
	load func(ctx context.Context, group string) (map[string]string, error)
}

func newLookupManager(name string) *lookupManager {
	return &lookupManager{name: name, cache: newResultCache(name)}
}

func (m *lookupManager) lookup(ctx context.Context, group, key string) (string, error) {
	cacheKey := group + ":" + key
	if v, ok := m.cache.get(cacheKey); ok {
		return v, nil
	}
	b := m.backend.Load()
	if b == nil {
		return "", fmt.Errorf("%s translator not registered", m.name)
	}
	v, err := b.one(ctx, group, key)
	if err != nil {
		return "", err
	}
	m.cache.set(cacheKey, v)
	return v, nil
}

// prefetch 批量预热：只查未命中缓存的 key；后端不支持批量则什么都不做（后续按单 key 走）
func (m *lookupManager) prefetch(ctx context.Context, group string, keys []string) error {
	b := m.backend.Load()
	if b == nil || b.many == nil || !m.cache.enabled.Load() {
		return nil
	}
	opt := NewBatchQueryOptimizer()
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		if _, ok := m.cache.get(group + ":" + k); ok {
			continue
		}
		k := k // go 1.21：循环变量仍是共享的
		opt.AddQuery(group, k, func(v string, err error) {
			if err == nil {
				m.cache.set(group+":"+k, v)
			}
		})
	}
	var batchErr error
	opt.ExecuteBatch(group, func(ks []string) (map[string]string, error) {
		res, err := b.many(ctx, group, ks)
		batchErr = err
		return res, err
	})
	return batchErr
}

// preload 预加载整个分组
func (m *lookupManager) preload(ctx context.Context, group string) (map[string]string, error) {
	b := m.backend.Load()
	if b == nil || b.load == nil {
		return nil, fmt.Errorf("%s translator does not support preload (implement DictTableLoader)", m.name)
	}
	data, err := b.load(ctx, group)
	if err != nil {
		return nil, err
	}
	for k, v := range data {
		m.cache.set(group+":"+k, v)
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// lookupTranslator：挂在字段上的翻译器（一个 struct tag 对应一个），实现 Translator + ContextTranslator
// ---------------------------------------------------------------------------

type lookupTranslator struct {
	mgr   *lookupManager
	group string
}

func (t *lookupTranslator) Translate(value any, fieldName string, tagValue string) (string, error) {
	return t.TranslateContext(context.Background(), value, fieldName, tagValue)
}

func (t *lookupTranslator) TranslateContext(ctx context.Context, value any, _ string, _ string) (string, error) {
	return t.mgr.lookup(ctx, t.group, fmt.Sprintf("%v", value))
}

// ---------------------------------------------------------------------------
// 三个包级管理器 + 对应的注册 / 开关 / 清空函数
// ---------------------------------------------------------------------------

var (
	defaultDBTranslatorManager = newLookupManager("db")
	defaultDictTableManager    = newLookupManager("dictTable")
	defaultDictTableTwoManager = newLookupManager("dictTableTwo")
	dbGroupSep                 = "\x00"
)

func dbGroup(table, keyField, valueField string) string {
	return table + dbGroupSep + keyField + dbGroupSep + valueField
}

func splitDBGroup(group string) (table, keyField, valueField string) {
	parts := strings.SplitN(group, dbGroupSep, 3)
	return parts[0], parts[1], parts[2]
}

// RegisterDBTranslator 注册数据库翻译器（可选实现 DBContextTranslator / DBBatchTranslator）
func RegisterDBTranslator(translator DBTranslator) {
	b := &lookupBackend{
		one: func(ctx context.Context, group, key string) (string, error) {
			table, kf, vf := splitDBGroup(group)
			if ct, ok := translator.(DBContextTranslator); ok {
				return ct.QueryContext(ctx, table, kf, vf, key)
			}
			return translator.Query(table, kf, vf, key)
		},
	}
	if bt, ok := translator.(DBBatchTranslator); ok {
		b.many = func(ctx context.Context, group string, keys []string) (map[string]string, error) {
			table, kf, vf := splitDBGroup(group)
			return bt.QueryBatch(ctx, table, kf, vf, keys)
		}
	}
	defaultDBTranslatorManager.backend.Store(b)
}

// EnableDBCache 启用 / 禁用数据库翻译结果缓存
func EnableDBCache(enabled bool) { defaultDBTranslatorManager.cache.enabled.Store(enabled) }

// ClearDBCache 清空数据库翻译结果缓存。若配置了 Config.Cache.CustomCache，会调用它的 Clear（Cache 接口无按前缀清理，三类共用时会一起清空）
func ClearDBCache() { defaultDBTranslatorManager.cache.clear() }

func createDBTranslator(table, keyField, valueField string) Translator {
	return &lookupTranslator{mgr: defaultDBTranslatorManager, group: dbGroup(table, keyField, valueField)}
}

func dictTableBackend(translator interface {
	QueryDict(dictType, dictKey string) (string, error)
}) *lookupBackend {
	b := &lookupBackend{
		one: func(ctx context.Context, group, key string) (string, error) {
			if ct, ok := translator.(DictTableContextTranslator); ok {
				return ct.QueryDictContext(ctx, group, key)
			}
			return translator.QueryDict(group, key)
		},
	}
	if bt, ok := translator.(DictTableBatchTranslator); ok {
		b.many = bt.QueryDictBatch
	}
	if ld, ok := translator.(DictTableLoader); ok {
		b.load = ld.LoadDict
	}
	return b
}

// RegisterDictTableTranslator 注册字典表翻译器（可选实现 DictTableContextTranslator / DictTableBatchTranslator / DictTableLoader）
func RegisterDictTableTranslator(translator DictTableTranslator) {
	defaultDictTableManager.backend.Store(dictTableBackend(translator))
}

// EnableDictTableCache 启用 / 禁用字典表翻译结果缓存
func EnableDictTableCache(enabled bool) { defaultDictTableManager.cache.enabled.Store(enabled) }

// ClearDictTableCache 清空字典表翻译结果缓存。若配置了 Config.Cache.CustomCache，会调用它的 Clear（Cache 接口无按前缀清理，三类共用时会一起清空）
func ClearDictTableCache() { defaultDictTableManager.cache.clear() }

func createDictTableTranslator(dictType string) Translator {
	return &lookupTranslator{mgr: defaultDictTableManager, group: dictType}
}

// RegisterDictTableTwoTranslator 注册双表字典翻译器（可选实现 DictTableContextTranslator / DictTableBatchTranslator / DictTableLoader）
func RegisterDictTableTwoTranslator(translator DictTableTwoTranslator) {
	defaultDictTableTwoManager.backend.Store(dictTableBackend(translator))
}

// EnableDictTableTwoCache 启用 / 禁用双表字典翻译结果缓存
func EnableDictTableTwoCache(enabled bool) { defaultDictTableTwoManager.cache.enabled.Store(enabled) }

// ClearDictTableTwoCache 清空双表字典翻译结果缓存。若配置了 Config.Cache.CustomCache，会调用它的 Clear（Cache 接口无按前缀清理，三类共用时会一起清空）
func ClearDictTableTwoCache() { defaultDictTableTwoManager.cache.clear() }

func createDictTableTwoTranslator(dictTypeCode string) Translator {
	return &lookupTranslator{mgr: defaultDictTableTwoManager, group: dictTypeCode}
}

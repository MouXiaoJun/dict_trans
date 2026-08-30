package dict

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
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
// 结果缓存：默认复用 MemoryCache，也可使用 CustomCache。
// 全局 Cache.Enabled 和 EnableXCache 必须同时启用；两种存储均使用 Cache.TTL。
// ---------------------------------------------------------------------------

type resultCache struct {
	name       string // 前缀，三类缓存共用一个 CustomCache 时不撞 key
	enabled    *atomic.Bool
	mu         sync.Mutex
	memory     Cache
	maxEntries int
	ttl        int
}

func newResultCache(name string, enabled *atomic.Bool) *resultCache {
	// 独立命名空间也隔离共享 CustomCache 中其他进程 / 注册代次的结果。
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		panic(fmt.Errorf("dict: initialize result cache namespace: %w", err))
	}
	return &resultCache{name: name + ":" + hex.EncodeToString(nonce[:]), enabled: enabled}
}

func (c *resultCache) storage() (Cache, int) {
	cfg := GetConfig().Cache
	if !c.enabled.Load() || !cfg.Enabled {
		return nil, 0
	}
	if cfg.CustomCache != nil {
		return cfg.CustomCache, cfg.TTL
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.memory == nil || c.maxEntries != cfg.MaxEntries || c.ttl != cfg.TTL {
		c.memory = NewMemoryCache(cfg.MaxEntries)
		c.maxEntries, c.ttl = cfg.MaxEntries, cfg.TTL
	}
	return c.memory, cfg.TTL
}

func (c *resultCache) get(key string) (string, bool) {
	if cache, _ := c.storage(); cache != nil {
		return cache.Get(c.name + ":" + key)
	}
	return "", false
}

func (c *resultCache) set(key, value string) {
	if value == "" {
		return
	}
	if cache, ttl := c.storage(); cache != nil {
		_ = cache.Set(c.name+":"+key, value, ttl)
	}
}

// ---------------------------------------------------------------------------
// lookupManager：一类 DB 后端（db / dictTable / dictTableTwo）= 已注册后端 + 结果缓存
// parts 是查找分组（dictTable：[dictType]；db：[table, keyField, valueField]），key 是字典键
// ---------------------------------------------------------------------------

type lookupManager struct {
	name         string
	backend      atomic.Pointer[lookupBackend] // 用户注册的后端，写时整体替换
	cacheEnabled atomic.Bool
}

// lookupBackend 把三类后端接口统一成 one / many / load 三个能力；many / load 为 nil 表示不支持
type lookupBackend struct {
	cache *resultCache // 与后端一起发布；旧请求只能写入旧代缓存
	one   func(ctx context.Context, parts []string, key string) (string, error)
	many  func(ctx context.Context, parts []string, keys []string) (map[string]string, error)
	load  func(ctx context.Context, parts []string) (map[string]string, error)
}

// cacheGroup 用长度前缀编码任意分量，避免分隔符与用户数据歧义。
func cacheGroup(parts []string) string {
	var key strings.Builder
	for _, part := range parts {
		key.WriteString(strconv.Itoa(len(part)))
		key.WriteByte(':')
		key.WriteString(part)
	}
	return key.String()
}

func newLookupManager(name string) *lookupManager {
	m := &lookupManager{name: name}
	m.cacheEnabled.Store(true)
	return m
}

func (m *lookupManager) register(b *lookupBackend) {
	b.cache = newResultCache(m.name, &m.cacheEnabled)
	m.backend.Store(b)
}

// clear 只切换本类命名空间，不调用共享 CustomCache.Clear。
func (m *lookupManager) clear() {
	for {
		old := m.backend.Load()
		if old == nil {
			return
		}
		next := *old
		next.cache = newResultCache(m.name, &m.cacheEnabled)
		if m.backend.CompareAndSwap(old, &next) {
			return
		}
	}
}

// lookupResultKey 按后端快照隔离本次调用的批查结果；空字符串也代表已查过。
type lookupResultKey struct {
	backend *lookupBackend
	group   string
	key     string
}

func (m *lookupManager) lookup(ctx context.Context, group string, parts []string, key string, results map[lookupResultKey]string) (string, error) {
	b := m.backend.Load()
	if b == nil {
		return "", fmt.Errorf("%s translator not registered", m.name)
	}
	if value, ok := results[lookupResultKey{b, group, key}]; ok {
		return value, nil
	}
	cacheKey := cacheGroup([]string{group, key})
	if v, ok := b.cache.get(cacheKey); ok {
		return v, nil
	}
	v, err := b.one(ctx, parts, key)
	if err != nil {
		return "", err
	}
	b.cache.set(cacheKey, v)
	return v, nil
}

// prefetch 批量预热：只查未命中缓存的 key；后端不支持批量则什么都不做（后续按单 key 走）
func (m *lookupManager) prefetch(ctx context.Context, group string, parts []string, keys []string, results map[lookupResultKey]string) error {
	b := m.backend.Load()
	if b == nil || b.many == nil {
		return nil
	}
	opt := NewBatchQueryOptimizer()
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		resultKey := lookupResultKey{b, group, k}
		if _, ok := results[resultKey]; ok {
			continue
		}
		cacheKey := cacheGroup([]string{group, k})
		if v, ok := b.cache.get(cacheKey); ok {
			results[resultKey] = v
			continue
		}
		opt.AddQuery(group, k, func(v string, err error) {
			if err == nil {
				results[resultKey] = v
				b.cache.set(cacheKey, v)
			}
		})
	}
	var batchErr error
	opt.ExecuteBatch(group, func(ks []string) (map[string]string, error) {
		res, err := b.many(ctx, parts, ks)
		batchErr = err
		return res, err
	})
	return batchErr
}

// preload 预加载整个分组
func (m *lookupManager) preload(ctx context.Context, parts []string) (map[string]string, error) {
	b := m.backend.Load()
	if b == nil || b.load == nil {
		return nil, fmt.Errorf("%s translator does not support preload (implement DictTableLoader)", m.name)
	}
	group := cacheGroup(parts)
	data, err := b.load(ctx, parts)
	if err != nil {
		return nil, err
	}
	for k, v := range data {
		b.cache.set(cacheGroup([]string{group, k}), v)
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// lookupTranslator：挂在字段上的翻译器（一个 struct tag 对应一个），实现 Translator + ContextTranslator
// ---------------------------------------------------------------------------

type lookupTranslator struct {
	mgr   *lookupManager
	parts []string // dictTable：[dictType]；db：[table, keyField, valueField]
	group string   // cacheGroup(parts)，预先算好
}

func newLookupTranslator(mgr *lookupManager, parts ...string) *lookupTranslator {
	return &lookupTranslator{mgr: mgr, parts: parts, group: cacheGroup(parts)}
}

func (t *lookupTranslator) Translate(value any, fieldName string, tagValue string) (string, error) {
	return t.TranslateContext(context.Background(), value, fieldName, tagValue)
}

func (t *lookupTranslator) TranslateContext(ctx context.Context, value any, _ string, _ string) (string, error) {
	return t.mgr.lookup(ctx, t.group, t.parts, fmt.Sprintf("%v", value), nil)
}

// ---------------------------------------------------------------------------
// 三个包级管理器 + 对应的注册 / 开关 / 清空函数
// ---------------------------------------------------------------------------

var (
	defaultDBTranslatorManager = newLookupManager("db")
	defaultDictTableManager    = newLookupManager("dictTable")
	defaultDictTableTwoManager = newLookupManager("dictTableTwo")
)

// RegisterDBTranslator 注册数据库翻译器（可选实现 DBContextTranslator / DBBatchTranslator）
func RegisterDBTranslator(translator DBTranslator) {
	b := &lookupBackend{
		one: func(ctx context.Context, p []string, key string) (string, error) {
			if ct, ok := translator.(DBContextTranslator); ok {
				return ct.QueryContext(ctx, p[0], p[1], p[2], key)
			}
			return translator.Query(p[0], p[1], p[2], key)
		},
	}
	if bt, ok := translator.(DBBatchTranslator); ok {
		b.many = func(ctx context.Context, p []string, keys []string) (map[string]string, error) {
			return bt.QueryBatch(ctx, p[0], p[1], p[2], keys)
		}
	}
	defaultDBTranslatorManager.register(b)
}

// EnableDBCache 启用 / 禁用数据库翻译结果缓存
func EnableDBCache(enabled bool) { defaultDBTranslatorManager.cacheEnabled.Store(enabled) }

// ClearDBCache 使本类结果缓存失效，不清空共享 CustomCache；外部旧条目按其 TTL 回收。
func ClearDBCache() { defaultDBTranslatorManager.clear() }

func createDBTranslator(table, keyField, valueField string) Translator {
	return newLookupTranslator(defaultDBTranslatorManager, table, keyField, valueField)
}

func dictTableBackend(translator interface {
	QueryDict(dictType, dictKey string) (string, error)
}) *lookupBackend {
	b := &lookupBackend{
		one: func(ctx context.Context, p []string, key string) (string, error) {
			if ct, ok := translator.(DictTableContextTranslator); ok {
				return ct.QueryDictContext(ctx, p[0], key)
			}
			return translator.QueryDict(p[0], key)
		},
	}
	if bt, ok := translator.(DictTableBatchTranslator); ok {
		b.many = func(ctx context.Context, p []string, keys []string) (map[string]string, error) {
			return bt.QueryDictBatch(ctx, p[0], keys)
		}
	}
	if ld, ok := translator.(DictTableLoader); ok {
		b.load = func(ctx context.Context, p []string) (map[string]string, error) { return ld.LoadDict(ctx, p[0]) }
	}
	return b
}

// RegisterDictTableTranslator 注册字典表翻译器（可选实现 DictTableContextTranslator / DictTableBatchTranslator / DictTableLoader）
func RegisterDictTableTranslator(translator DictTableTranslator) {
	defaultDictTableManager.register(dictTableBackend(translator))
}

// EnableDictTableCache 启用 / 禁用字典表翻译结果缓存
func EnableDictTableCache(enabled bool) { defaultDictTableManager.cacheEnabled.Store(enabled) }

// ClearDictTableCache 使本类结果缓存失效，不清空共享 CustomCache；外部旧条目按其 TTL 回收。
func ClearDictTableCache() { defaultDictTableManager.clear() }

func createDictTableTranslator(dictType string) Translator {
	return newLookupTranslator(defaultDictTableManager, dictType)
}

// RegisterDictTableTwoTranslator 注册双表字典翻译器（可选实现 DictTableContextTranslator / DictTableBatchTranslator / DictTableLoader）
func RegisterDictTableTwoTranslator(translator DictTableTwoTranslator) {
	defaultDictTableTwoManager.register(dictTableBackend(translator))
}

// EnableDictTableTwoCache 启用 / 禁用双表字典翻译结果缓存
func EnableDictTableTwoCache(enabled bool) { defaultDictTableTwoManager.cacheEnabled.Store(enabled) }

// ClearDictTableTwoCache 使本类结果缓存失效，不清空共享 CustomCache；外部旧条目按其 TTL 回收。
func ClearDictTableTwoCache() { defaultDictTableTwoManager.clear() }

func createDictTableTwoTranslator(dictTypeCode string) Translator {
	return newLookupTranslator(defaultDictTableTwoManager, dictTypeCode)
}

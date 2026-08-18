package dict

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

// countingDictTable 假字典表后端：记录单查 / 批查 / 预加载次数
type countingDictTable struct {
	single, batch, load int64
	data                map[string]map[string]string
}

func (c *countingDictTable) QueryDict(dictType, key string) (string, error) {
	atomic.AddInt64(&c.single, 1)
	return c.data[dictType][key], nil
}
func (c *countingDictTable) QueryDictBatch(_ context.Context, dictType string, keys []string) (map[string]string, error) {
	atomic.AddInt64(&c.batch, 1)
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := c.data[dictType][k]; ok {
			out[k] = v
		}
	}
	return out, nil
}
func (c *countingDictTable) LoadDict(_ context.Context, dictType string) (map[string]string, error) {
	atomic.AddInt64(&c.load, 1)
	return c.data[dictType], nil
}

func resetDictTableFor(t *testing.T, backend DictTableTranslator) {
	t.Helper()
	EnableDictTableCache(true)
	ClearDictTableCache()
	RegisterDictTableTranslator(backend)
	t.Cleanup(func() { EnableDictTableCache(true); ClearDictTableCache() })
}

// N+1：100 行 × 2 个 dictTable 字段 → 每个分组 1 次批量查询，0 次单查
func TestPrefetchBatchesDictTableQueries(t *testing.T) {
	be := &countingDictTable{data: map[string]map[string]string{
		"sex":    {"1": "男", "2": "女"},
		"status": {"0": "禁用", "1": "启用"},
	}}
	resetDictTableFor(t, be)

	type Row struct {
		Sex        string `dictTable:"sex" dictField:"SexName"`
		SexName    string
		Status     string `dictTable:"status" dictField:"StatusName"`
		StatusName string
	}
	rows := make([]Row, 100)
	for i := range rows {
		rows[i].Sex, rows[i].Status = "12"[i%2:i%2+1], "01"[i%2:i%2+1]
	}
	if err := Translate(&rows); err != nil {
		t.Fatal(err)
	}
	if rows[0].SexName != "男" || rows[1].SexName != "女" || rows[0].StatusName != "禁用" || rows[1].StatusName != "启用" {
		t.Fatalf("翻译结果不对: %+v %+v", rows[0], rows[1])
	}
	if b, s := atomic.LoadInt64(&be.batch), atomic.LoadInt64(&be.single); b != 2 || s != 0 {
		t.Fatalf("期望 2 次批量 0 次单查，实际 batch=%d single=%d", b, s)
	}

	// 并行路径同样走预取
	ClearDictTableCache()
	atomic.StoreInt64(&be.batch, 0)
	if err := BatchTranslate(&rows, true); err != nil {
		t.Fatal(err)
	}
	if b, s := atomic.LoadInt64(&be.batch), atomic.LoadInt64(&be.single); b != 2 || s != 0 {
		t.Fatalf("并行：期望 2 次批量 0 次单查，实际 batch=%d single=%d", b, s)
	}

	// 关掉预取 → 退回单查（缓存命中后只查不同 key）
	ClearDictTableCache()
	atomic.StoreInt64(&be.batch, 0)
	atomic.StoreInt64(&be.single, 0)
	if err := TranslateWith(&rows, WithoutPrefetch()); err != nil {
		t.Fatal(err)
	}
	if b, s := atomic.LoadInt64(&be.batch), atomic.LoadInt64(&be.single); b != 0 || s != 4 {
		t.Fatalf("WithoutPrefetch：期望 0 次批量 4 次单查（4 个不同 key），实际 batch=%d single=%d", b, s)
	}
}

// 后端没实现批量接口 → 不报错，按单查走
func TestPrefetchFallsBackWithoutBatchSupport(t *testing.T) {
	var calls int64
	resetDictTableFor(t, DictTableTranslatorFunc(func(dictType, key string) (string, error) {
		atomic.AddInt64(&calls, 1)
		return "v" + key, nil
	}))
	type Row struct {
		K  string `dictTable:"t" dictField:"KN"`
		KN string
	}
	rows := make([]Row, 20)
	for i := range rows {
		rows[i].K = "k"
	}
	if err := Translate(&rows); err != nil {
		t.Fatal(err)
	}
	if rows[19].KN != "vk" || atomic.LoadInt64(&calls) != 1 {
		t.Fatalf("单查 + 缓存：期望 1 次调用，实际 %d，值 %q", calls, rows[19].KN)
	}
}

// 无 DB 类字段的平铺切片：预取遍历不应发生（mayLookup=false）
func TestPrefetchSkippedForPlainStructs(t *testing.T) {
	type Row struct {
		S  string `dict:"plain_s" dictField:"SN"`
		SN string
	}
	cfg := NewDictManager().getOrCreateConfig(reflectTypeOf(Row{}))
	if cfg.mayLookup {
		t.Fatal("纯内存字典的平铺结构体不应标记 mayLookup")
	}
}

// ctx：取消后返回 ctx 错误；实现了 ContextTranslator 的翻译器能拿到 ctx
type ctxProbe struct{ got atomic.Bool }

func (p *ctxProbe) Translate(any, string, string) (string, error) { return "plain", nil }
func (p *ctxProbe) TranslateContext(ctx context.Context, _ any, _ string, _ string) (string, error) {
	if ctx.Value(ctxKey{}) == "yes" {
		p.got.Store(true)
	}
	return "ctx", nil
}

type ctxKey struct{}

func TestTranslateWithContext(t *testing.T) {
	probe := &ctxProbe{}
	dm := NewDictManager()
	dm.RegisterTranslator("ctxprobe", probe)
	type Row struct {
		A  string `translate:"ctxprobe" dictField:"AN"`
		AN string
	}
	r := &Row{A: "x"}
	ctx := context.WithValue(context.Background(), ctxKey{}, "yes")
	if err := dm.TranslateWith(r, WithContext(ctx)); err != nil || r.AN != "ctx" || !probe.got.Load() {
		t.Fatalf("ContextTranslator 未被调用: err=%v AN=%q got=%v", err, r.AN, probe.got.Load())
	}
	r2 := &Row{A: "x"}
	if err := dm.Translate(r2); err != nil || r2.AN != "plain" {
		t.Fatalf("无 ctx 时应走 Translate: err=%v AN=%q", err, r2.AN)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rows := make([]Row, 50)
	err := dm.TranslateWith(&rows, WithContext(cancelled))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("已取消的 ctx 应返回 context.Canceled，实际 %v", err)
	}
	err = dm.TranslateWith(&rows, WithContext(cancelled), WithParallel())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("并行：已取消的 ctx 应返回 context.Canceled，实际 %v", err)
	}
}

// CustomCache：配置了就用它存 DB 结果
type recordingCache struct {
	m         map[string]string
	gets, set int64
}

func (c *recordingCache) Get(k string) (string, bool)  { c.gets++; v, ok := c.m[k]; return v, ok }
func (c *recordingCache) Set(k, v string, _ int) error { c.set++; c.m[k] = v; return nil }
func (c *recordingCache) Delete(k string) error        { delete(c.m, k); return nil }
func (c *recordingCache) Clear() error                 { c.m = map[string]string{}; return nil }

func TestCustomCacheUsedForDBResults(t *testing.T) {
	old := GetConfig()
	rc := &recordingCache{m: map[string]string{}}
	cfg := *old
	cfg.Cache.Enabled = true
	cfg.Cache.CustomCache = rc
	SetConfig(&cfg)
	t.Cleanup(func() { SetConfig(old) })

	be := &countingDictTable{data: map[string]map[string]string{"sex": {"1": "男"}}}
	resetDictTableFor(t, be)
	type Row struct {
		Sex     string `dictTable:"sex" dictField:"SexName"`
		SexName string
	}
	r := &Row{Sex: "1"}
	_ = Translate(r)
	_ = Translate(&Row{Sex: "1"})
	if r.SexName != "男" || rc.set != 1 || rc.gets < 2 || atomic.LoadInt64(&be.single) != 1 {
		t.Fatalf("CustomCache 未生效: name=%q set=%d gets=%d single=%d", r.SexName, rc.set, rc.gets, be.single)
	}
	if _, ok := rc.m["dictTable:sex:1"]; !ok {
		t.Fatalf("CustomCache 里应有带前缀的 key，实际 %v", rc.m)
	}
}

// Framework：PreloadDicts 预热 + GetMetrics 记录
func TestFrameworkPreloadAndMetrics(t *testing.T) {
	be := &countingDictTable{data: map[string]map[string]string{"sex": {"1": "男", "2": "女"}}}
	resetDictTableFor(t, be)

	cfg := *GetConfig()
	cfg.Performance.PreloadDicts = []string{"sex"}
	fw := NewFramework(&cfg)
	if err := fw.Init(); err != nil {
		t.Fatal(err)
	}
	if v, ok := fw.Preloaded("sex", "2"); !ok || v != "女" || atomic.LoadInt64(&be.load) != 1 {
		t.Fatalf("预加载失败: v=%q ok=%v load=%d", v, ok, be.load)
	}
	type Row struct {
		Sex     string `dictTable:"sex" dictField:"SexName"`
		SexName string
	}
	r := &Row{Sex: "1"}
	if err := fw.Translate(r); err != nil || r.SexName != "男" {
		t.Fatalf("翻译失败: %v %q", err, r.SexName)
	}
	if atomic.LoadInt64(&be.single) != 0 {
		t.Fatalf("预加载后不应再单查，实际 %d 次", be.single)
	}
	m := fw.GetMetrics()["translate"]
	if m == nil || m.Count != 1 || m.ErrorCount != 0 {
		t.Fatalf("GetMetrics 未记录: %+v", m)
	}
}

func TestBuildQueryInAndAll(t *testing.T) {
	tc := DefaultTableConfig("sys_dict")
	q, args := tc.BuildQueryIn("sex", []string{"1", "2"})
	if !strings.Contains(q, "dict_key IN (?, ?)") || len(args) != 4 || args[0] != "sex" || args[3] != "1" {
		t.Fatalf("BuildQueryIn: %s %v", q, args)
	}
	q, args = tc.BuildQueryAll("sex")
	if !strings.HasPrefix(q, "SELECT dict_key, dict_value FROM sys_dict WHERE dict_type = ?") || len(args) != 2 {
		t.Fatalf("BuildQueryAll: %s %v", q, args)
	}
}

func reflectTypeOf(v any) reflect.Type { return reflect.TypeOf(v) }

// ClearDBCache 不应清掉 dictTable 的缓存（默认配置下三类缓存各自独立）
func TestClearOneKindDoesNotTouchOthers(t *testing.T) {
	be := &countingDictTable{data: map[string]map[string]string{"sex": {"1": "男"}}}
	resetDictTableFor(t, be)
	type Row struct {
		Sex     string `dictTable:"sex" dictField:"SexName"`
		SexName string
	}
	_ = Translate(&Row{Sex: "1"})
	ClearDBCache()
	_ = Translate(&Row{Sex: "1"})
	if n := atomic.LoadInt64(&be.single); n != 1 {
		t.Fatalf("ClearDBCache 后 dictTable 缓存应仍命中，实际查询 %d 次", n)
	}
}

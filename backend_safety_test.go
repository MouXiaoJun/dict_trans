package dict

import (
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

type safetyBackendKind struct {
	name     string
	tag      reflect.StructTag
	register func(func() (string, error))
	clear    func()
	enable   func(bool)
}

func safetyBackendKinds() []safetyBackendKind {
	return []safetyBackendKind{
		{"db", `db:"cache-safety:id:name"`, func(f func() (string, error)) {
			RegisterDBTranslator(DBTranslatorFunc(func(string, string, string, any) (string, error) { return f() }))
		}, ClearDBCache, EnableDBCache},
		{"dictTable", `dictTable:"cache-safety"`, func(f func() (string, error)) {
			RegisterDictTableTranslator(DictTableTranslatorFunc(func(string, string) (string, error) { return f() }))
		}, ClearDictTableCache, EnableDictTableCache},
		{"dictTableTwo", `dictTableTwo:"cache-safety"`, func(f func() (string, error)) {
			RegisterDictTableTwoTranslator(DictTableTwoTranslatorFunc(func(string, string) (string, error) { return f() }))
		}, ClearDictTableTwoCache, EnableDictTableTwoCache},
	}
}

func translateTagged(tag reflect.StructTag, key string) (string, error) {
	typ := reflect.StructOf([]reflect.StructField{
		{Name: "Key", Type: reflect.TypeOf(""), Tag: tag + ` dictField:"Name"`},
		{Name: "Name", Type: reflect.TypeOf("")},
	})
	v := reflect.New(typ)
	v.Elem().Field(0).SetString(key)
	err := Translate(v.Interface())
	return v.Elem().Field(1).String(), err
}

func assertTaggedValue(t *testing.T, tag reflect.StructTag, key, want string) {
	t.Helper()
	if got, err := translateTagged(tag, key); err != nil || got != want {
		t.Fatalf("got %q, err=%v; want %q", got, err, want)
	}
}

func TestBackendReplacementIsolatesCache(t *testing.T) {
	for _, custom := range []bool{false, true} {
		for _, kind := range safetyBackendKinds() {
			t.Run(kind.name+"/custom="+strconv.FormatBool(custom), func(t *testing.T) {
				cfg := cacheSafetyConfig(t)
				if custom {
					cfg.Cache.CustomCache = NewMemoryCache(100)
				}
				kind.enable(true)
				kind.clear()
				t.Cleanup(kind.clear)
				kind.register(func() (string, error) { return "A", nil })
				assertTaggedValue(t, kind.tag, "warm", "A")
				kind.register(func() (string, error) { return "B", nil })
				assertTaggedValue(t, kind.tag, "warm", "B")
			})
		}
	}
}

func TestBackendReplacementRejectsLateCacheWrite(t *testing.T) {
	for _, custom := range []bool{false, true} {
		for _, kind := range safetyBackendKinds() {
			t.Run(kind.name+"/custom="+strconv.FormatBool(custom), func(t *testing.T) {
				cfg := cacheSafetyConfig(t)
				if custom {
					cfg.Cache.CustomCache = NewMemoryCache(100)
				}
				kind.enable(true)
				kind.clear()
				t.Cleanup(kind.clear)
				started, release := make(chan struct{}), make(chan struct{})
				kind.register(func() (string, error) {
					close(started)
					<-release
					return "A", nil
				})
				done := make(chan error, 1)
				go func() { _, err := translateTagged(kind.tag, "late"); done <- err }()
				<-started
				kind.register(func() (string, error) { return "B", nil })
				// Release and join even when an assertion fails.
				got, err := translateTagged(kind.tag, "late")
				close(release)
				if oldErr := <-done; oldErr != nil {
					t.Fatal(oldErr)
				}
				if err != nil || got != "B" {
					t.Fatalf("new backend: got %q, err=%v", got, err)
				}
				assertTaggedValue(t, kind.tag, "late", "B")
			})
		}
	}
}

type sharedSafetyCache struct {
	Cache
	clears atomic.Int64
}

func (c *sharedSafetyCache) Clear() error {
	c.clears.Add(1)
	return c.Cache.Clear()
}

func TestClearCacheDoesNotClearSharedStorage(t *testing.T) {
	for _, target := range safetyBackendKinds() {
		t.Run(target.name, func(t *testing.T) {
			cfg := cacheSafetyConfig(t)
			shared := &sharedSafetyCache{Cache: NewMemoryCache(100)}
			cfg.Cache.CustomCache = shared
			_ = shared.Set("application-owned", "keep", 0)
			counts := map[string]*atomic.Int64{}
			for _, kind := range safetyBackendKinds() {
				calls := &atomic.Int64{}
				counts[kind.name] = calls
				kind.enable(true)
				kind.register(func() (string, error) { calls.Add(1); return "value", nil })
				assertTaggedValue(t, kind.tag, "clear", "value")
			}
			target.clear()
			for _, kind := range safetyBackendKinds() {
				assertTaggedValue(t, kind.tag, "clear", "value")
				want := int64(1)
				if kind.name == target.name {
					want = 2
				}
				if got := counts[kind.name].Load(); got != want {
					t.Errorf("%s calls=%d, want %d", kind.name, got, want)
				}
			}
			if v, ok := shared.Get("application-owned"); !ok || v != "keep" || shared.clears.Load() != 0 {
				t.Fatalf("shared cache cleared: value=%q, exists=%v, clear calls=%d", v, ok, shared.clears.Load())
			}
		})
	}
}

func TestResultCacheKeysDoNotCollide(t *testing.T) {
	cacheSafetyConfig(t)
	t.Run("group-key-boundary", func(t *testing.T) {
		RegisterDictTableTranslator(DictTableTranslatorFunc(func(group, key string) (string, error) {
			return group + " / " + key, nil
		}))
		assertTaggedValue(t, `dictTable:"a:b"`, "c", "a:b / c")
		assertTaggedValue(t, `dictTable:"a"`, "b:c", "a / b:c")
	})
	t.Run("group-components", func(t *testing.T) {
		RegisterDBTranslator(DBTranslatorFunc(func(table, keyField, valueField string, key any) (string, error) {
			return table + " / " + keyField, nil
		}))
		for _, parts := range [][3]string{{"a\x00b", "c", "d"}, {"a", "b\x00c", "d"}} {
			tag := reflect.StructTag("db:" + strconv.Quote(parts[0]+":"+parts[1]+":"+parts[2]))
			assertTaggedValue(t, tag, "key", parts[0]+" / "+parts[1])
		}
	})
}

type cacheSafetyRow struct {
	Key  string `dictTable:"cache-safety" dictField:"Name"`
	Name string
}

func cacheSafetyConfig(t *testing.T) *Config {
	t.Helper()
	old := GetConfig()
	cfg := *old
	cfg.Cache = CacheConfig{Enabled: true, Type: "memory", MaxEntries: 10000}
	SetConfig(&cfg)
	EnableDictTableCache(true)
	ClearDictTableCache()
	t.Cleanup(func() {
		EnableDictTableCache(true)
		ClearDictTableCache()
		SetConfig(old)
	})
	return &cfg
}

func translateCacheSafety(t *testing.T, key string) string {
	t.Helper()
	r := cacheSafetyRow{Key: key}
	if err := Translate(&r); err != nil {
		t.Fatal(err)
	}
	return r.Name
}

func TestResultCacheHonorsEnableSwitches(t *testing.T) {
	for _, custom := range []bool{false, true} {
		for _, global := range []bool{false, true} {
			for _, kind := range []bool{false, true} {
				t.Run(fmtCacheCase(custom, global, kind), func(t *testing.T) {
					cfg := cacheSafetyConfig(t)
					cfg.Cache.Enabled = global
					if custom {
						cfg.Cache.CustomCache = NewMemoryCache(10)
					}
					EnableDictTableCache(kind)
					be := &countingDictTable{data: map[string]map[string]string{"cache-safety": {"k": "value"}}}
					RegisterDictTableTranslator(be)
					translateCacheSafety(t, "k")
					translateCacheSafety(t, "k")
					want := int64(2)
					if global && kind {
						want = 1
					}
					if got := atomic.LoadInt64(&be.single); got != want {
						t.Fatalf("single=%d, want %d", got, want)
					}
				})
			}
		}
	}
}

func fmtCacheCase(custom, global, kind bool) string {
	name := "memory"
	if custom {
		name = "custom"
	}
	if global {
		name += "/global-on"
	} else {
		name += "/global-off"
	}
	if kind {
		name += "/kind-on"
	} else {
		name += "/kind-off"
	}
	return name
}

func TestDefaultResultCacheCapacity(t *testing.T) {
	cfg := cacheSafetyConfig(t)
	cfg.Cache.MaxEntries = 1
	be := &countingDictTable{data: map[string]map[string]string{"cache-safety": {"a": "A", "b": "B"}}}
	RegisterDictTableTranslator(be)
	for _, key := range []string{"a", "b", "a"} {
		translateCacheSafety(t, key)
	}
	if got := atomic.LoadInt64(&be.single); got != 3 {
		t.Fatalf("capacity 1 should evict a before requery: single=%d, want 3", got)
	}
}

func TestDefaultResultCacheTTL(t *testing.T) {
	cfg := cacheSafetyConfig(t)
	cfg.Cache.TTL = 1
	be := &countingDictTable{data: map[string]map[string]string{"cache-safety": {"k": "old"}}}
	RegisterDictTableTranslator(be)
	if got := translateCacheSafety(t, "k"); got != "old" {
		t.Fatal(got)
	}
	be.data["cache-safety"]["k"] = "new"
	if got := translateCacheSafety(t, "k"); got != "old" {
		t.Fatal("cache did not hit before expiry")
	}
	time.Sleep(1100 * time.Millisecond)
	if got := translateCacheSafety(t, "k"); got != "new" {
		t.Fatalf("after TTL: got %q, want new", got)
	}
	if got := atomic.LoadInt64(&be.single); got != 2 {
		t.Fatalf("single=%d, want 2", got)
	}
}

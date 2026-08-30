package dict

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestBatchResultsRetainedForOneCall(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		for _, mode := range []string{"memory", "capacity-one", "global-disabled", "kind-disabled"} {
			for _, values := range []string{"all-found", "partial", "all-missing", "empty-value"} {
				name := mode + "/" + values
				if parallel {
					name += "/parallel"
				}
				t.Run(name, func(t *testing.T) {
					cfg := cacheSafetyConfig(t)
					cfg.Performance.BatchQueryThreshold = 10
					switch mode {
					case "capacity-one":
						cfg.Cache.MaxEntries = 1
					case "global-disabled":
						cfg.Cache.Enabled = false
					case "kind-disabled":
						EnableDictTableCache(false)
					}
					data := map[string]string{}
					switch values {
					case "all-found":
						data["a"], data["b"] = "A", "B"
					case "partial":
						data["a"] = "A"
					case "empty-value":
						data["a"], data["b"] = "", ""
					}
					be := &countingDictTable{data: map[string]map[string]string{"cache-safety": data}}
					RegisterDictTableTranslator(be)
					rows := make([]cacheSafetyRow, 20)
					for i := range rows {
						rows[i] = cacheSafetyRow{Key: []string{"a", "b"}[i%2], Name: "unchanged"}
					}
					if err := BatchTranslate(&rows, parallel); err != nil {
						t.Fatal(err)
					}
					for _, row := range rows {
						want := data[row.Key]
						if want == "" {
							want = "unchanged"
						}
						if row.Name != want {
							t.Fatalf("row=%+v, want %q", row, want)
						}
					}
					if b, s := atomic.LoadInt64(&be.batch), atomic.LoadInt64(&be.single); b != 1 || s != 0 {
						t.Fatalf("batch=%d, single=%d; want 1, 0", b, s)
					}
					// Missing and empty values must not become persistent negative cache entries.
					if values != "all-found" {
						data["b"] = "created-later"
						if got := translateCacheSafety(t, "b"); got != "created-later" {
							t.Fatalf("next call got %q", got)
						}
					}
				})
			}
		}
	}
}

type failingBatchBackend struct {
	countingDictTable
	err error
}

func (b *failingBatchBackend) QueryDictBatch(context.Context, string, []string) (map[string]string, error) {
	atomic.AddInt64(&b.batch, 1)
	return map[string]string{"a": "partial-result"}, b.err
}

func TestBatchResultErrorIsNotCached(t *testing.T) {
	cacheSafetyConfig(t)
	wantErr := errors.New("batch unavailable")
	be := &failingBatchBackend{err: wantErr}
	RegisterDictTableTranslator(be)
	rows := make([]cacheSafetyRow, 20)
	for i := range rows {
		rows[i].Key = "a"
	}
	for i := 0; i < 2; i++ {
		if err := Translate(&rows); !errors.Is(err, wantErr) {
			t.Fatalf("got %v, want %v", err, wantErr)
		}
	}
	if b, s := atomic.LoadInt64(&be.batch), atomic.LoadInt64(&be.single); b != 2 || s != 0 {
		t.Fatalf("batch=%d, single=%d; want 2, 0", b, s)
	}
	for _, row := range rows {
		if row.Name != "" {
			t.Fatal("batch error partially filled rows")
		}
	}
}

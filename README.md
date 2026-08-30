# dict-trans

[![Go Reference](https://pkg.go.dev/badge/github.com/MouXiaoJun/dict_trans.svg)](https://pkg.go.dev/github.com/MouXiaoJun/dict_trans)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg?style=flat-square)](LICENSE)
[![GitHub release](https://img.shields.io/github/release/MouXiaoJun/dict_trans.svg?style=flat-square)](https://github.com/MouXiaoJun/dict_trans/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/MouXiaoJun/dict_trans?style=flat-square)](https://goreportcard.com/report/github.com/MouXiaoJun/dict_trans)

[中文文档](README_zh.md)

Translate coded fields into display text with struct tags — `Status: "1"` becomes `StatusName: "Enabled"` — driven by in-memory dictionaries, enums, database dictionary tables, or your own translators. Zero dependencies (standard library only).

## Install

```bash
go get github.com/MouXiaoJun/dict_trans
```

## Quick start

```go
package main

import (
    "fmt"

    dict "github.com/MouXiaoJun/dict_trans"
)

type User struct {
    Sex     string `dict:"sex" dictField:"SexName"`
    SexName string
}

func main() {
    dict.RegisterDict("sex", map[string]string{"1": "Male", "2": "Female"})

    u := User{Sex: "1"}
    _ = dict.Translate(&u)
    fmt.Println(u.SexName) // Male
}
```

`Translate` accepts a pointer to a struct or to a slice of structs / struct pointers. Nested structs, struct pointers and slices are translated recursively; wrapper types (e.g. `Page`, `Result`) are unwrapped via registered `UnWrapper`s.

## Struct tags

| Tag | Source | Example |
| --- | --- | --- |
| `dict:"name"` | in-memory dictionary registered with `RegisterDict` | `Sex string \`dict:"sex" dictField:"SexName"\`` |
| `enum:"name"` | enum registered with `RegisterEnum` | `Status string \`enum:"deviceStatus" dictField:"StatusName"\`` |
| `translate:"name"` | custom `Translator` registered with `RegisterTranslator` | `ID string \`translate:"user" dictField:"Name"\`` |
| `db:"table=t,key=k,value=v"` | look up `v` from table `t` where `k = value`, via `RegisterDBTranslator` | `DeptID string \`db:"table=dept,key=id,value=name" dictField:"DeptName"\`` |
| `dictTable:"type"` | single dictionary table (`sys_dict`) via `RegisterDictTableTranslator` | `Sex string \`dictTable:"sex" dictField:"SexName"\`` |
| `dictTableTwo:"type"` | dictionary type table + data table via `RegisterDictTableTwoTranslator` | `Sex string \`dictTableTwo:"sex" dictField:"SexName"\`` |
| `dictField:"Field"` | target field (string) that receives the translated text; required with every tag above | |
| `mask:"format"` | mask a string field in place — `phone`/`idcard`/`bankcard`/`email`/`name`/`address`/`password`/`*`, or `"3,4"` to keep first 3 + last 4 runes | `Phone string \`mask:"phone"\`` |

Priority when several tags are present on one field: `translate` > `db` > `dictTableTwo` > `dictTable` > `enum` > `dict`.

## Data masking

`dict.Mask(&user)` masks string fields in place; structs, struct pointers, struct slices and top-level slices are recursed automatically. It composes with translation — translate first, then mask:

```go
type User struct {
	Name     string `mask:"name"`    // 张三 → 张*
	Phone    string `mask:"phone"`   // 13800138000 → 138****8000
	IDCard   string `mask:"idcard"`  // 110101199003071234 → 110101********1234
	Email    string `mask:"email"`   // zhangsan@example.com → z*******@example.com
	Password string `mask:"password"`
}

user := User{Name: "张三", Phone: "13800138000", Password: "secret"}
dict.Translate(&user)
dict.Mask(&user) // in place

users := []User{{Name: "李四"}}
dict.Mask(&users) // batch

dict.MaskOf(&user)       // generic entry
dict.RegisterMaskFormat("carplate", func(s string) string { /* custom */ })
```

Built-in formats: `phone` (3+4), `idcard` (6+4), `bankcard` (4+4), `email` (first char of local part), `name` (first char), `address` (first 6), `password` / `*` (all), and generic `"n,m"` keep form. All are rune-safe for Chinese text. `mask:"-"` skips a field; non-string fields are skipped.

Masking visits each typed struct object once per call, including cycles and shared children. Nil pointers are skipped; interface-typed fields (including `any` and `[]any` fields) are not dynamically traversed. Custom formats need not be idempotent. Do not mask the same object concurrently.

## Database-backed dictionaries

```go
db, _ := sql.Open("mysql", os.Getenv("DICT_TRANS_DSN"))

// single table: sys_dict(dict_type, dict_key, dict_value, status)
dict.RegisterDictTableTranslator(dict.CreateDictTableTranslatorFromDB(db, "sys_dict"))

// two tables: sys_dict_type + sys_dict_data (column names configurable via TableConfig)
dict.RegisterDictTableTwoTranslator(dict.CreateDictTableTwoTranslatorFromDB(db, "sys_dict_type", "sys_dict_data"))
```

Query results are cached in memory (`EnableDictTableCache`, `ClearDictTableCache`, and the `DB*` equivalents). See [examples/](examples/) for runnable programs.

## Options, context and batching

`TranslateWith` takes functional options; `Translate` / `BatchTranslate` are thin wrappers over it.

```go
err := dict.TranslateWith(&rows,
    dict.WithContext(ctx),   // cancellation checked per struct; passed to translators implementing ContextTranslator
    dict.WithParallel(),     // worker pool for slices with >= 10 elements
    dict.WithoutPrefetch(),  // opt out of the DB prefetch below
)
```

Generic entrypoints move the pointer/slice checks to compile time: `dict.TranslateOf(&u)` (`*T`), `dict.BatchTranslateOf(items, true)` (`[]*T`).

**Batch lookup.** For slices with at least `Config.Performance.BatchQueryThreshold` (default 10) elements, database-backed fields (`db`, `dictTable`, `dictTableTwo`) are collected in one pass and fetched through one batch call per backend/group with uncached keys. The number of SQL statements inside that call depends on the backend. Backends opt in by implementing the optional interfaces:

| Backend | Optional interface | Enables |
| --- | --- | --- |
| `DBTranslator` | `DBContextTranslator`, `DBBatchTranslator` | ctx, batch |
| `DictTableTranslator` / `DictTableTwoTranslator` | `DictTableContextTranslator`, `DictTableBatchTranslator`, `DictTableLoader` | ctx, batch, preload |
| any `Translator` | `ContextTranslator` | receives `ctx` from `WithContext` |

`CreateDictTableTranslatorFromDB` / `CreateDictTableTwoTranslatorFromDB` already implement all of them (`QueryRowContext`, `IN` queries, full-dictionary load). A backend without batch support silently falls back to per-key lookups.

Batch results, including missing keys and empty display values, are retained only for the current translation call, including parallel filling. Missing/empty values leave destination fields unchanged and are not stored as persistent negative cache entries, so later calls can see newly created data. This per-call record also works with result caching disabled or a small cache capacity. `WithoutPrefetch()` disables batching and preserves per-key fallback. Batch errors are returned without caching partial results.

**Result cache.** The global configuration installed by `SetConfig` controls DB results, independently of a `Framework` instance's configuration. Results are cached per kind (`EnableDBCache`, `ClearDBCache`, `EnableDictTableCache`, ...):

- Both `Config.Cache.Enabled` and the relevant `Enable*Cache` switch must be true for persistent cache reads/writes. Disabling bypasses caching; it does not invalidate existing entries. Use `Clear*Cache` when re-enabling must start fresh.
- The default `MemoryCache` applies `Cache.TTL` (seconds; 0 means no expiration) and `Cache.MaxEntries` per kind (default 10000; <= 0 means unbounded). Changing TTL/capacity rebuilds that kind's local store on its next use. A supplied `CustomCache` receives TTL and controls its own capacity; setting `Cache.Type` alone does not connect Redis.
- Registration replaces the backend and its result-cache namespace atomically. Already-running queries may return their old backend's result, but cannot populate the new backend's cache. Cache keys use unambiguous length-prefixed components and a private registration namespace; their representation is not a public API and entries are not shared across process lifetimes/registrations.
- `ClearDBCache`, `ClearDictTableCache` and `ClearDictTableTwoCache` invalidate only their own namespace. They never call `CustomCache.Clear()` or delete application-owned/other-kind data. In external storage, obsolete namespace entries remain until that storage expires/evicts them: configure a positive TTL or an explicit reclamation policy. TTL 0 does **not** physically reclaim old external entries.

Compatibility: existing custom-cache keys become cold after this key-format change. The separate `Framework.ClearCache()` API still delegates to its configured cache's `Clear`; do not use that operation on a shared cache whose `Clear` flushes all data.

**Framework extras.** `NewFramework(cfg).Init()` preloads `cfg.Performance.PreloadDicts` through `DictTableLoader` (`fw.Preloaded(type, key)`), and `fw.GetMetrics()["translate"]` reports count / min / max / avg latency and error count for `fw.Translate`. `NewDictManager()` gives an isolated manager (own dictionaries, translators and config cache) for multi-tenant or test setups.

## Performance

Apple M-series, `go test -bench . -benchmem` (numbers move ±20% run to run):

| Benchmark | v1.0.0 | v1.2.1 |
| --- | --- | --- |
| single struct, in-memory dict | 939 ns/op · 4 allocs | **175 ns/op · 1 alloc** |
| 1000-element slice, sequential | 966 µs/op · 3000 allocs | **205 µs/op · 0 allocs** |
| 1000-element slice, parallel | 291 µs/op | ~230 µs/op (parallel only pays for I/O-bound translators) |
| nested struct | 1081 ns/op · 6 allocs | **324 ns/op · 0 allocs** |

How it got there: per-type config cache with precomputed traversal steps (only nested or tagged fields are visited), pre-split tag names and target-field indexes, copy-on-write registries instead of RWMutex, lazy visited set for cycle safety, and a two-phase prefetch that replaces N database round-trips with one `IN` query per group. To profile yourself:

```bash
go test -run xxx -bench 'BenchmarkTranslateBatch$' -cpuprofile cpu.out -o dict.test ./
go tool pprof -top -nodecount=20 dict.test cpu.out      # or: -http=:8080 for the flame graph
```

CI posts a `benchstat` comparison (base vs head) in every pull request's job summary; `go test -fuzz FuzzParseDBTag` fuzzes the tag parser.

## Concurrency and limits

- `Translate` / `BatchTranslate` are safe for concurrent use. Registration (`RegisterDict`, `RegisterTranslator`, ...) is also safe to call concurrently with translation; the registry is copy-on-write, so register at startup — each call copies the small registry maps.
- Registering a translator invalidates the per-type configuration cache, so it takes effect for types that were already translated.
- Cyclic structures (self-referencing pointers, parent/child links) are handled: each pointer target is translated once per `Translate` call.
- Translation is best-effort: a missing dictionary, missing target field or non-string target is silently skipped, not an error. Errors come only from translators (e.g. database failures).
- Source fields must be `string` or integer kinds; target fields must be `string`.
- `WithParallel` / `BatchTranslate(..., true)`: nested pointer targets shared by several elements are translated exactly once (a shared visited set), but the *same pointer appearing several times as a top-level element* is translated by whichever worker gets it — de-duplicate such slices before translating in parallel. Parallel mode pays off for I/O-bound translators (database lookups); for in-memory dictionaries the sequential path is usually faster.

## Framework mode

`Framework` bundles a `DictManager` with config, middleware and plugin hooks (`NewFramework`, `GetFramework`). See [FRAMEWORK.md](FRAMEWORK.md).

## License

[MIT](LICENSE)

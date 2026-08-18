# dict-trans

[![Go Reference](https://pkg.go.dev/badge/github.com/MouXiaoJun/dict_trans.svg)](https://pkg.go.dev/github.com/MouXiaoJun/dict_trans)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg?style=flat-square)](LICENSE)
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

Priority when several tags are present on one field: `translate` > `db` > `dictTableTwo` > `dictTable` > `enum` > `dict`.

## Database-backed dictionaries

```go
db, _ := sql.Open("mysql", os.Getenv("DICT_TRANS_DSN"))

// single table: sys_dict(dict_type, dict_key, dict_value, status)
dict.RegisterDictTableTranslator(dict.CreateDictTableTranslatorFromDB(db, "sys_dict"))

// two tables: sys_dict_type + sys_dict_data (column names configurable via TableConfig)
dict.RegisterDictTableTwoTranslator(dict.CreateDictTableTwoTranslatorFromDB(db, "sys_dict_type", "sys_dict_data"))
```

Query results are cached in memory (`EnableDictTableCache`, `ClearDictTableCache`, and the `DB*` equivalents). See [examples/](examples/) for runnable programs.

## Batch translation

```go
items := make([]*Item, 0, 10000)
// ...
err := dict.BatchTranslate(&items, true) // parallel worker pool for len >= 10
```

On the first translator error the remaining workers stop and the error is returned; already-translated elements keep their values.

Generic entrypoints move the pointer/slice checks to compile time:

```go
_ = dict.TranslateOf(&u)             // *T
_ = dict.BatchTranslateOf(items, true) // []*T
```

## Concurrency and limits

- `Translate` / `BatchTranslate` are safe for concurrent use. Registration (`RegisterDict`, `RegisterTranslator`, ...) is also safe to call concurrently with translation; the registry is copy-on-write, so register at startup — each call copies the small registry maps.
- Registering a translator invalidates the per-type configuration cache, so it takes effect for types that were already translated.
- Cyclic structures (self-referencing pointers, parent/child links) are handled: each pointer target is translated once per `Translate` call.
- Translation is best-effort: a missing dictionary, missing target field or non-string target is silently skipped, not an error. Errors come only from translators (e.g. database failures).
- Source fields must be `string` or integer kinds; target fields must be `string`.

## Framework mode

`Framework` bundles a `DictManager` with config, middleware and plugin hooks (`NewFramework`, `GetFramework`). See [FRAMEWORK.md](FRAMEWORK.md).

## License

[Mulan PSL v2](LICENSE)

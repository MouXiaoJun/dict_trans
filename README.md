# dict-trans

[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg?style=flat-square)](LICENSE)
[![GitHub release](https://img.shields.io/github/release/MouXiaoJun/dict_trans.svg?style=flat-square)](https://github.com/MouXiaoJun/dict_trans/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/MouXiaoJun/dict_trans?style=flat-square)](https://goreportcard.com/report/github.com/MouXiaoJun/dict_trans)

一个**高效率、高扩展性、高自定义**的 Go 语言翻译框架，支持字典翻译、数据脱敏、嵌套翻译等功能。

> 🚀 **高性能翻译框架**：批量查询优化、预加载机制、智能缓存、并行处理、性能监控
> 
> 🔌 **高扩展性**：中间件系统、插件机制、策略模式、工厂模式
> 
> 🎨 **高自定义**：灵活配置、自定义缓存、自定义翻译器、自定义策略

## 特性

- ✅ **字典翻译**：通过 struct tags 自动翻译字典值（内存字典）
- ✅ **字典表翻译**：从数据库字典表读取数据翻译（单表结构）
- ✅ **双表字典翻译**：从字典类型表+字典数据表读取数据翻译（双表结构）
- ✅ **枚举转换**：支持枚举类型的自动转换
- ✅ **数据库翻译**：支持从数据库自动查表翻译（类似 Easy Trans）
- ✅ **嵌套翻译**：支持嵌套结构体的自动翻译
- ✅ **自定义翻译器**：支持自定义翻译逻辑（可扩展支持 Redis、本地缓存等）
- ✅ **包装类型支持**：支持 Page、Result 等包装类型的自动解包
- ✅ **配置缓存**：转换配置首次处理后缓存，减少反射开销
- ✅ **结果缓存**：数据库翻译结果自动缓存，避免重复查询
- ✅ **批量并行翻译**：支持大批量数据的并行处理，提升性能
- ✅ **框架模式**：完整的高性能翻译框架（中间件、插件、策略、监控）
- ✅ **零依赖**：仅使用 Go 标准库（框架核心）

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "github.com/mouxiaojun/dict-trans"
)

type User struct {
    Sex     string `dict:"sex" dictField:"SexName"`
    SexName string
}

func main() {
    // 注册字典
    dict.RegisterDict("sex", map[string]string{
        "1": "男",
        "2": "女",
    })
    
    // 翻译
    user := User{Sex: "1"}
    dict.Translate(&user)
    fmt.Println(user.SexName) // 输出: 男
}
```

### 切片翻译

```go
type Item struct {
    Status     string `dict:"status" dictField:"StatusName"`
    StatusName string
}

items := []Item{
    {Status: "1"},
    {Status: "0"},
}
dict.Translate(&items)
```

### 嵌套翻译

```go
type Device struct {
    Status     string `dict:"status" dictField:"StatusName"`
    StatusName string
}

type User struct {
    Sex     string `dict:"sex" dictField:"SexName"`
    SexName string
    Device  Device
}

user := User{
    Sex:    "1",
    Device: Device{Status: "1"},
}
dict.Translate(&user)
```

### 枚举转换

```go
// 注册枚举
dict.RegisterEnum("deviceStatus", map[string]string{
    "1": "未使用",
    "2": "试运行",
    "3": "运行中",
})

type Device struct {
    Status     string `enum:"deviceStatus" dictField:"StatusName"`
    StatusName string
}

device := Device{Status: "1"}
dict.Translate(&device)
// device.StatusName = "未使用"
```

### 自定义翻译器

```go
// 注册自定义翻译器
dict.RegisterTranslator("custom", dict.TranslatorFunc(func(value interface{}, fieldName string, tagValue string) (string, error) {
    // 自定义翻译逻辑，可以连接 Redis、数据库等
    return "翻译结果", nil
}))

type User struct {
    ID     string `translate:"custom" dictField:"Name"`
    Name   string
}
```

### 字典表翻译（单表结构）

从单张字典表读取数据，适用于简单的字典场景。

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/mouxiaojun/dict-trans"
)

// 连接数据库
db, _ := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/dbname")

// 注册字典表翻译器（从 sys_dict 表读取）
dictTableTranslator := dict.CreateDictTableTranslatorFromDB(db, "sys_dict")
dict.RegisterDictTableTranslator(dictTableTranslator)

// 使用
type User struct {
    Sex     string `dictTable:"sex" dictField:"SexName"`
    SexName string
}

user := User{Sex: "1"}
dict.Translate(&user)
// user.SexName = "男"（从数据库字典表查询）
```

**单表字典结构：**
```sql
CREATE TABLE sys_dict (
  dict_type VARCHAR(50) NOT NULL COMMENT '字典类型',
  dict_key VARCHAR(50) NOT NULL COMMENT '字典键',
  dict_value VARCHAR(200) NOT NULL COMMENT '字典值',
  status CHAR(1) DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
  PRIMARY KEY (dict_type, dict_key)
);
```

### 双表字典翻译（字典类型表 + 字典数据表）⭐

从两张表读取数据，字典类型表和字典数据表分离，更符合数据库规范化设计。

#### 使用默认表结构

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/mouxiaojun/dict-trans"
)

// 连接数据库
db, _ := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/dbname")

// 注册双表字典翻译器（从 sys_dict_type 和 sys_dict_data 表读取）
dictTableTwoTranslator := dict.CreateDictTableTwoTranslatorFromDB(db, "sys_dict_type", "sys_dict_data")
dict.RegisterDictTableTwoTranslator(dictTableTwoTranslator)

// 使用
type User struct {
    Sex     string `dictTableTwo:"sex" dictField:"SexName"`
    SexName string
}

user := User{Sex: "1"}
dict.Translate(&user)
// user.SexName = "男"（从双表字典查询）

// 缓存管理
dict.EnableDictTableTwoCache(true)  // 默认启用
dict.ClearDictTableTwoCache()       // 清空缓存
```

#### 自定义双表结构 ⭐

```go
// 自定义字典类型表配置
typeConfig := &dict.TableConfig{
    TableName: "dict_category",
    Fields: dict.TableFields{
        TypeField:  "category_code",
        ValueField: "category_name",
    },
    StatusField: &dict.StatusFieldConfig{
        FieldName:     "enabled",
        EnabledValue:  "1",
        DisabledValue: "0",
    },
}

// 自定义字典数据表配置
dataConfig := &dict.TableConfig{
    TableName: "dict_item",
    Fields: dict.TableFields{
        TypeField:  "category_code",
        KeyField:   "item_code",
        ValueField: "item_name",
    },
    StatusField: &dict.StatusFieldConfig{
        FieldName:     "enabled",
        EnabledValue:  "1",
        DisabledValue: "0",
    },
}

// 使用自定义配置创建翻译器
translator := dict.CreateDictTableTwoTranslatorFromDBWithConfig(db, typeConfig, dataConfig)
dict.RegisterDictTableTwoTranslator(translator)
```

**默认双表字典结构：**
```sql
-- 字典类型表
CREATE TABLE sys_dict_type (
  dict_type_code VARCHAR(50) NOT NULL COMMENT '字典类型编码',
  dict_type_name VARCHAR(100) NOT NULL COMMENT '字典类型名称',
  status CHAR(1) DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
  PRIMARY KEY (dict_type_code)
);

-- 字典数据表
CREATE TABLE sys_dict_data (
  dict_type_code VARCHAR(50) NOT NULL COMMENT '字典类型编码',
  dict_key VARCHAR(50) NOT NULL COMMENT '字典键',
  dict_value VARCHAR(200) NOT NULL COMMENT '字典值',
  status CHAR(1) DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
  PRIMARY KEY (dict_type_code, dict_key)
);
```

**自定义双表结构示例：**
```go
// 自定义字典类型表配置
typeConfig := &dict.TableConfig{
    TableName: "dict_category",
    Fields: dict.TableFields{
        TypeField:  "category_code",
        ValueField: "category_name",
    },
    StatusField: &dict.StatusFieldConfig{
        FieldName:     "enabled",
        EnabledValue:  "1",
        DisabledValue: "0",
    },
}

// 自定义字典数据表配置
dataConfig := &dict.TableConfig{
    TableName: "dict_item",
    Fields: dict.TableFields{
        TypeField:  "category_code",
        KeyField:   "item_code",
        ValueField: "item_name",
    },
    StatusField: &dict.StatusFieldConfig{
        FieldName:     "enabled",
        EnabledValue:  "1",
        DisabledValue: "0",
    },
}

translator := dict.CreateDictTableTwoTranslatorFromDBWithConfig(db, typeConfig, dataConfig)
dict.RegisterDictTableTwoTranslator(translator)
```

**双表字典的优势：**
- ✅ 字典类型信息更完整（名称、排序、备注等）
- ✅ 字典类型可以独立管理（启用/禁用）
- ✅ 更符合数据库规范化设计
- ✅ 支持字典类型的元数据管理

### 数据库翻译（类似 Easy Trans）

```go
// 注册数据库翻译器（实际项目中连接真实数据库）
dict.RegisterDBTranslator(dict.DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
    // 这里实现数据库查询逻辑
    // 例如使用 GORM、database/sql 等
    return "查询结果", nil
}))

// 使用简化格式
type User struct {
    UserID   string `db:"user:id:name" dictField:"UserName"`
    UserName string
}

// 或使用完整格式
type Device struct {
    DeviceID   string `db:"table=device,key=id,value=name" dictField:"DeviceName"`
    DeviceName string
}

user := User{UserID: "1"}
dict.Translate(&user)
// user.UserName = "查询结果"（从数据库查询）

// 启用/禁用缓存
dict.EnableDBCache(true)  // 默认启用
dict.ClearDBCache()       // 清空缓存
```

### 包装类型支持

```go
// 定义包装类型
type Page struct {
    Data []Item `json:"data"`
}

type Item struct {
    Status     string `dict:"status" dictField:"StatusName"`
    StatusName string
}

// 注册解包器
dict.RegisterUnWrapper(dict.UnWrapperFunc(func(value interface{}) (interface{}, error) {
    if page, ok := value.(*Page); ok {
        return &page.Data, nil
    }
    return nil, nil
}))

page := &Page{Data: []Item{{Status: "1"}}}
dict.Translate(page)
```

### 批量并行翻译

```go
type Item struct {
    Status     string `dict:"status" dictField:"StatusName"`
    StatusName string
}

// 创建大量数据
items := make([]Item, 1000)
for i := range items {
    items[i] = Item{Status: "1"}
}

// 顺序处理（小批量或禁用并行）
dict.BatchTranslate(&items, false)

// 并行处理（大批量数据，自动使用 worker pool）
dict.BatchTranslate(&items, true)
```

## Struct Tags 说明

- `dict:"字典名"` - 指定使用的字典名称（内存字典翻译）
- `dictTable:"字典类型"` - 指定字典类型，从数据库单表字典读取（单表字典翻译）
- `dictTableTwo:"字典类型编码"` - 指定字典类型编码，从双表字典读取（双表字典翻译）
- `enum:"枚举名"` - 指定使用的枚举名称（枚举转换）
- `db:"table:key:value"` 或 `db:"table=table,key=key,value=value"` - 数据库翻译（类似 Easy Trans）
- `translate:"翻译器名"` - 指定使用的自定义翻译器
- `dictField:"字段名"` - 指定翻译结果存储的字段名（大小写不敏感）

## 标签优先级

翻译标签的优先级顺序：`translate` > `db` > `dictTableTwo` > `dictTable` > `enum` > `dict`

## 字典翻译方式对比

| 特性 | 内存字典 (`dict`) | 单表字典 (`dictTable`) | 双表字典 (`dictTableTwo`) |
|------|------------------|----------------------|-------------------------|
| 数据来源 | 内存（代码中注册） | 数据库单表 | 数据库双表 |
| 表结构 | 无 | 1张表 | 2张表（类型表+数据表） |
| 适用场景 | 固定不变的字典 | 简单字典场景 | 复杂字典场景 |
| 性能 | 最快（内存访问） | 较快（有缓存） | 较快（有缓存） |
| 灵活性 | 需要修改代码 | 可动态修改数据库 | 可动态修改数据库 |
| 类型管理 | 无 | 无 | 支持类型元数据管理 |
| 使用方式 | `dict:"sex"` | `dictTable:"sex"` | `dictTableTwo:"sex"` |

## 安装

```bash
go get github.com/mouxiaojun/dict-trans
```

## 文档

详细文档请参考 [文档目录](./doc)

## 框架模式（高级功能）

dict-trans 不仅是一个简单的翻译组件，更是一个完整的**高性能翻译框架**。

### 核心优势

- 🚀 **高性能**：批量查询优化、预加载、智能缓存、并行处理
- 🔌 **高扩展性**：中间件、插件、策略、工厂模式
- 🎨 **高自定义**：灵活配置、自定义缓存、自定义翻译器

### 快速开始

```go
// 1. 配置框架
config := &dict.Config{
    Performance: dict.PerformanceConfig{
        BatchQueryThreshold: 10,
        ParallelThreshold:   100,
        PreloadDicts:        []string{"sex", "status"},
    },
    Cache: dict.CacheConfig{
        Enabled:   true,
        TTL:      3600,
        MaxEntries: 50000,
    },
}
dict.SetConfig(config)

// 2. 使用框架
framework := dict.GetFramework()
framework.Translate(&user)
```

### 详细文档

查看 [FRAMEWORK.md](./FRAMEWORK.md) 了解完整的框架功能和使用方法。

### 框架示例

```bash
cd examples/framework
go run main.go
```

## License

MulanPSL-2.0


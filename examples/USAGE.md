# dict-trans 使用指南

## 快速开始

### 1. 安装

```bash
go get github.com/mouxiaojun/dict-trans
```

### 2. 最简单的示例

```go
package main

import (
    "fmt"
    "github.com/mouxiaojun/dict-trans"
)

func main() {
    // 注册字典
    dict.RegisterDict("sex", map[string]string{
        "1": "男",
        "2": "女",
    })

    // 定义结构体
    type User struct {
        Sex     string `dict:"sex" dictField:"SexName"`
        SexName string
    }

    // 翻译
    user := User{Sex: "1"}
    dict.Translate(&user)
    fmt.Println(user.SexName) // 输出: 男
}
```

## 功能示例

### 字典翻译

```go
dict.RegisterDict("status", map[string]string{
    "1": "启用",
    "0": "禁用",
})

type Item struct {
    Status     string `dict:"status" dictField:"StatusName"`
    StatusName string
}
```

### 枚举转换

```go
dict.RegisterEnum("priority", map[string]string{
    "1": "低",
    "2": "中",
    "3": "高",
})

type Task struct {
    Priority     string `enum:"priority" dictField:"PriorityName"`
    PriorityName string
}
```

### 数据库翻译（MySQL）

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/mouxiaojun/dict-trans"
)

// 连接数据库
db, _ := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/dbname")

// 注册数据库翻译器
dict.RegisterDBTranslator(dict.DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
    query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", valueField, table, keyField)
    var result string
    err := db.QueryRow(query, key).Scan(&result)
    return result, err
}))

// 使用
type User struct {
    UserID   string `db:"user:id:name" dictField:"UserName"`
    UserName string
}
```

### 自定义翻译器

```go
// 注册自定义翻译器（可以连接 Redis、外部 API 等）
dict.RegisterTranslator("custom", dict.TranslatorFunc(func(value interface{}, fieldName string, tagValue string) (string, error) {
    // 实现自定义逻辑
    return "翻译结果", nil
}))

type Item struct {
    Key   string `translate:"custom" dictField:"Value"`
    Value string
}
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
    Device  Device  // 嵌套结构体
}
```

### 批量并行翻译

```go
items := make([]Item, 1000)
// ... 填充数据

// 顺序处理
dict.BatchTranslate(&items, false)

// 并行处理（大批量数据）
dict.BatchTranslate(&items, true)
```

## 标签说明

| 标签 | 说明 | 示例 |
|------|------|------|
| `dict` | 字典翻译 | `dict:"status"` |
| `enum` | 枚举转换 | `enum:"priority"` |
| `db` | 数据库翻译 | `db:"user:id:name"` 或 `db:"table=user,key=id,value=name"` |
| `translate` | 自定义翻译器 | `translate:"custom"` |
| `dictField` | 翻译结果字段 | `dictField:"StatusName"` |

## 标签优先级

翻译标签的优先级顺序：`translate` > `db` > `enum` > `dict`

## 缓存管理

### 数据库翻译缓存

```go
// 启用缓存（默认启用）
dict.EnableDBCache(true)

// 禁用缓存
dict.EnableDBCache(false)

// 清空缓存
dict.ClearDBCache()
```

## 完整示例

查看 `examples/` 目录下的完整示例代码：

- `basic/` - 基础示例
- `advanced/` - 高级示例
- `db/` - 数据库翻译示例（模拟）
- `mysql/` - MySQL 数据库翻译示例（真实数据库）
- `complete/` - 完整功能示例

## 注意事项

1. **必须使用指针**：`dict.Translate()` 需要传入指针类型
2. **字段必须可设置**：目标字段必须是可导出的（首字母大写）
3. **数据库翻译**：需要先注册 `DBTranslator`，否则会返回错误
4. **缓存机制**：数据库翻译结果会自动缓存，相同查询不会重复访问数据库

## 常见问题

### Q: 为什么翻译后字段还是空的？

A: 检查以下几点：
- 字典/枚举是否已注册
- 源字段的值是否在字典/枚举中存在
- `dictField` 标签是否正确
- 字段名是否大小写匹配

### Q: 如何禁用数据库翻译缓存？

A: 使用 `dict.EnableDBCache(false)` 禁用缓存。

### Q: 批量翻译什么时候使用并行模式？

A: 当数据量 >= 10 条且 `parallel=true` 时，会自动使用并行模式。


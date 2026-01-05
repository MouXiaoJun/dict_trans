# dict-trans 示例代码

本目录包含 dict-trans 的各种使用示例，涵盖所有功能特性。

## 📚 示例列表

### 1. basic - 基础示例 ⭐
最简单的使用示例，适合初学者。

**包含功能：**
- 字典翻译
- 切片翻译
- 嵌套翻译

**运行方式：**
```bash
cd examples/basic
go run main.go
```

### 2. advanced - 高级示例
包含枚举转换、自定义翻译器等高级功能。

**包含功能：**
- 枚举转换
- 自定义翻译器
- 混合使用

**运行方式：**
```bash
cd examples/advanced
go run main.go
```

### 3. db - 数据库翻译示例（模拟数据）
使用模拟数据的数据库翻译示例，不需要真实数据库。

**包含功能：**
- 数据库翻译（模拟）
- 缓存测试
- 混合使用

**运行方式：**
```bash
cd examples/db
go run main.go
```

### 4. mysql - MySQL 数据库翻译示例（真实数据库）🔥
连接真实 MySQL 数据库的完整示例。

**数据库配置：**
- 主机: 127.0.0.1
- 端口: 3306
- 用户名: root
- 密码: MSms0427
- 数据库: dict_trans

**前置要求：**
- 安装 MySQL 数据库
- 安装 Go MySQL 驱动: `go get github.com/go-sql-driver/mysql`

**运行方式：**

方式1：使用脚本（推荐）
```bash
cd examples/mysql
./run.sh
```

方式2：手动运行
```bash
cd examples/mysql
# 1. 初始化数据库
mysql -u root -pMSms0427 < setup.sql
# 2. 安装依赖
go mod tidy
# 3. 运行示例
go run main.go
```

**包含功能：**
- 真实 MySQL 数据库翻译
- 缓存测试
- 批量翻译
- 混合使用（字典+数据库）

### 5. complete - 完整示例 ⭐⭐⭐
包含所有功能的完整示例代码，推荐查看。

**包含功能：**
- ✅ 字典翻译
- ✅ 枚举转换
- ✅ 嵌套翻译
- ✅ 切片翻译
- ✅ 包装类型支持
- ✅ 自定义翻译器
- ✅ 数据库翻译（模拟）
- ✅ 批量并行翻译
- ✅ 混合使用

**运行方式：**
```bash
cd examples/complete
go run main.go
```

## 功能对照表

| 功能 | basic | advanced | db | mysql | complete |
|------|-------|----------|----|----|---------|
| 字典翻译 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 枚举转换 | ❌ | ✅ | ❌ | ❌ | ✅ |
| 嵌套翻译 | ✅ | ✅ | ❌ | ❌ | ✅ |
| 自定义翻译器 | ❌ | ✅ | ❌ | ❌ | ✅ |
| 数据库翻译 | ❌ | ❌ | ✅ | ✅ | ✅ |
| 包装类型 | ❌ | ❌ | ❌ | ❌ | ✅ |
| 批量翻译 | ❌ | ❌ | ❌ | ✅ | ✅ |
| 混合使用 | ❌ | ✅ | ✅ | ✅ | ✅ |

## 快速开始

### 最简单的示例

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

## 更多示例

查看各个子目录中的示例代码，每个示例都有详细的注释说明。


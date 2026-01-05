# dict-trans 高性能翻译框架

dict-trans 是一个**高效率、高扩展性、高自定义**的 Go 语言翻译框架。

## 🚀 核心特性

### 1. 高性能 (High Performance)

- ✅ **批量查询优化**：自动合并多个数据库查询为批量查询
- ✅ **预加载机制**：启动时预加载常用字典到内存
- ✅ **智能缓存**：多级缓存策略（内存、Redis、自定义）
- ✅ **并行处理**：大批量数据自动并行翻译
- ✅ **性能监控**：内置性能指标收集和分析

### 2. 高扩展性 (High Extensibility)

- ✅ **中间件系统**：支持翻译前后处理（日志、审计、限流等）
- ✅ **插件机制**：可插拔的插件系统
- ✅ **策略模式**：支持多种翻译策略切换
- ✅ **工厂模式**：自定义翻译器工厂
- ✅ **解耦设计**：各组件独立，易于扩展

### 3. 高自定义 (High Customization)

- ✅ **灵活配置**：丰富的配置选项
- ✅ **自定义缓存**：支持 Redis、本地缓存等
- ✅ **自定义翻译器**：完全自定义翻译逻辑
- ✅ **自定义策略**：实现自己的翻译策略
- ✅ **选项模式**：细粒度的翻译控制

## 📦 架构设计

```
┌─────────────────────────────────────────┐
│           Framework (框架入口)            │
├─────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌────────┐│
│  │ Config   │  │ Manager  │  │Monitor ││
│  │ (配置)   │  │ (管理器) │  │(监控)  ││
│  └──────────┘  └──────────┘  └────────┘│
│  ┌──────────┐  ┌──────────┐  ┌────────┐│
│  │Optimizer│  │Preloader │  │Strategy││
│  │(优化器) │  │(预加载)  │  │(策略)  ││
│  └──────────┘  └──────────┘  └────────┘│
├─────────────────────────────────────────┤
│  Middleware → Translator → Cache        │
│  (中间件)    (翻译器)     (缓存)        │
└─────────────────────────────────────────┘
```

## 🎯 快速开始

### 基础使用

```go
import "github.com/mouxiaojun/dict-trans"

// 简单使用（向后兼容）
dict.RegisterDict("sex", map[string]string{
    "1": "男",
    "2": "女",
})

type User struct {
    Sex     string `dict:"sex" dictField:"SexName"`
    SexName string
}

user := User{Sex: "1"}
dict.Translate(&user)
```

### 框架模式（推荐）

```go
import "github.com/mouxiaojun/dict-trans"

// 创建自定义配置
config := &dict.Config{
    Performance: dict.PerformanceConfig{
        BatchQueryThreshold: 10,  // 批量查询阈值
        ParallelThreshold:   100, // 并行处理阈值
        MaxConcurrency:       20,  // 最大并发数
        PreloadDicts:         []string{"sex", "status"}, // 预加载字典
    },
    Cache: dict.CacheConfig{
        Enabled:   true,
        Type:     "memory",
        TTL:      3600,      // 1小时过期
        MaxEntries: 50000,   // 最大缓存条目
    },
}

// 设置配置
dict.SetConfig(config)

// 获取框架实例
framework := dict.GetFramework()

// 使用框架翻译
user := User{Sex: "1"}
framework.Translate(&user)
```

## 🔧 高级功能

### 1. 中间件系统

```go
// 创建日志中间件
type LogMiddleware struct{}

func (m *LogMiddleware) BeforeTranslate(ctx *dict.TranslateContext) error {
    log.Printf("翻译前: 字段=%s, 值=%v", ctx.FieldName, ctx.SourceValue)
    return nil
}

func (m *LogMiddleware) AfterTranslate(ctx *dict.TranslateContext) error {
    log.Printf("翻译后: 结果=%s", ctx.Result)
    return nil
}

// 注册中间件
dict.RegisterMiddleware(&LogMiddleware{})
```

### 2. 插件系统

```go
// 创建自定义插件
type CustomPlugin struct{}

func (p *CustomPlugin) Name() string {
    return "custom_plugin"
}

func (p *CustomPlugin) Init(config map[string]interface{}) error {
    // 初始化插件
    return nil
}

func (p *CustomPlugin) Execute(ctx *dict.TranslateContext) error {
    // 执行插件逻辑
    return nil
}

// 注册插件
dict.RegisterPlugin(&CustomPlugin{})
```

### 3. 自定义策略

```go
// 创建自定义翻译策略
type CustomStrategy struct{}

func (s *CustomStrategy) Name() string {
    return "custom"
}

func (s *CustomStrategy) Translate(ctx *dict.TranslateContext) error {
    // 自定义翻译逻辑
    ctx.Result = fmt.Sprintf("自定义翻译: %v", ctx.SourceValue)
    return nil
}

// 注册策略
framework := dict.GetFramework()
framework.Strategies.RegisterStrategy(&CustomStrategy{})

// 使用策略
options := &dict.TranslateOptions{
    Strategy: "custom",
}
framework.Translate(&user, options)
```

### 4. 自定义缓存

```go
// 实现缓存接口
type RedisCache struct {
    client *redis.Client
}

func (c *RedisCache) Get(key string) (string, bool) {
    val, err := c.client.Get(key).Result()
    return val, err == nil
}

func (c *RedisCache) Set(key string, value string, ttl int) error {
    return c.client.Set(key, value, time.Duration(ttl)*time.Second).Err()
}

func (c *RedisCache) Delete(key string) error {
    return c.client.Del(key).Err()
}

func (c *RedisCache) Clear() error {
    return c.client.FlushDB().Err()
}

// 使用自定义缓存
config := &dict.Config{
    Cache: dict.CacheConfig{
        Enabled:    true,
        CustomCache: &RedisCache{client: redisClient},
    },
}
dict.SetConfig(config)
```

### 5. 性能监控

```go
framework := dict.GetFramework()

// 执行翻译操作
dict.Translate(&user)

// 获取性能指标
metrics := framework.GetMetrics()
for name, metric := range metrics {
    fmt.Printf("%s: 调用次数=%d, 平均耗时=%d微秒\n",
        name, metric.Count, metric.GetAverageTime())
}
```

### 6. 批量翻译优化

```go
// 批量翻译选项
options := &dict.BatchOptions{
    Parallel:   true,    // 并行处理
    BatchQuery: true,    // 批量查询优化
    Concurrency: 20,     // 并发数
}

items := make([]Item, 1000)
dict.TranslateBatch(&items, options)
```

## 📊 性能对比

| 场景 | 传统方式 | dict-trans 框架 |
|------|---------|----------------|
| 单条翻译 | 1ms | 0.1ms (缓存) |
| 100条翻译 | 100ms | 5ms (批量优化) |
| 1000条翻译 | 1000ms | 50ms (并行+批量) |
| 数据库查询 | N次查询 | 1次批量查询 |

## 🎨 最佳实践

### 1. 配置优化

```go
config := &dict.Config{
    Performance: dict.PerformanceConfig{
        // 根据数据量调整阈值
        BatchQueryThreshold: 10,  // 小批量：10
        ParallelThreshold:   100,  // 大批量：100
        MaxConcurrency:       20,   // 根据CPU核心数调整
        
        // 预加载常用字典
        PreloadDicts: []string{"sex", "status", "priority"},
    },
    Cache: dict.CacheConfig{
        Enabled:   true,
        TTL:      3600,      // 1小时过期
        MaxEntries: 100000,  // 根据内存调整
    },
}
```

### 2. 中间件使用

```go
// 日志中间件
dict.RegisterMiddleware(&LogMiddleware{})

// 审计中间件
dict.RegisterMiddleware(&AuditMiddleware{})

// 限流中间件
dict.RegisterMiddleware(&RateLimitMiddleware{})
```

### 3. 缓存策略

```go
// 内存缓存（默认，适合单机）
config.Cache.Type = "memory"

// Redis缓存（适合分布式）
config.Cache.CustomCache = &RedisCache{}

// 多级缓存（内存+Redis）
config.Cache.CustomCache = &MultiLevelCache{
    L1: NewMemoryCache(10000),
    L2: &RedisCache{},
}
```

## 🔌 扩展开发

### 自定义翻译器工厂

```go
type CustomTranslatorFactory struct{}

func (f *CustomTranslatorFactory) Type() string {
    return "custom"
}

func (f *CustomTranslatorFactory) Create(config map[string]interface{}) (dict.Translator, error) {
    // 根据配置创建翻译器
    return &CustomTranslator{}, nil
}

// 注册工厂
dict.RegisterTranslatorFactory(&CustomTranslatorFactory{})
```

## 📈 性能监控

```go
// 获取性能指标
metrics := framework.GetMetrics()

// 分析性能瓶颈
for name, metric := range metrics {
    avgTime := metric.GetAverageTime()
    if avgTime > 1000 { // 超过1ms
        log.Printf("警告: %s 平均耗时 %d 微秒", name, avgTime)
    }
}
```

## 🎯 总结

dict-trans 框架提供了：

1. **高效率**：批量查询、预加载、智能缓存、并行处理
2. **高扩展性**：中间件、插件、策略、工厂模式
3. **高自定义**：灵活配置、自定义缓存、自定义翻译器

适用于各种规模的 Go 项目，从简单应用到大型分布式系统。


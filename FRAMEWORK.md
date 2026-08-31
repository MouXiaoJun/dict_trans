# dict-trans 高性能翻译框架

本页描述现有 Framework 包装层。维护范围不包含补齐历史扩展设想；以 [README 的当前执行边界](README_zh.md#框架模式高级功能) 为准。

注意：`MaxConcurrency`、`DBPoolSize`、`BatchOptions.Concurrency` 目前不控制 worker/数据库连接池；`TranslateOptions` 仅 `Strategy` 被 `Framework.Translate` 消费。`Plugin.Execute` 和已注册工厂不自动调用，中间件是调用级钩子（字段信息不会自动填充）。下方接口示意不能视为这些能力已接通。

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
- **工厂接口**：可注册保存，但未接入自动翻译流程
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
import "github.com/MouXiaoJun/dict_trans"

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
import "github.com/MouXiaoJun/dict_trans"

// 创建自定义配置
config := &dict.Config{
    Performance: dict.PerformanceConfig{
        BatchQueryThreshold: 10,  // 批量查询阈值
        ParallelThreshold:   100, // 并行处理阈值
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

// 创建实例：GetFramework 在包初始化时创建，不随 SetConfig 重建
framework := dict.NewFramework(config)
framework.RegisterDict("sex", map[string]string{"1": "男", "2": "女"})
if err := framework.Init(); err != nil { panic(err) }

// 使用框架翻译
user := User{Sex: "1"}
if err := framework.Translate(&user); err != nil { panic(err) }
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

func (p *CustomPlugin) Init(config map[string]any) error {
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
    // 共享 Redis 不提供全库清理，避免影响其他业务数据。
    return fmt.Errorf("shared cache: Clear is disabled")
}

// 使用自定义缓存
config := &dict.Config{
    Cache: dict.CacheConfig{
        Enabled:    true,
        TTL:        300, // 回收 Clear*Cache / 后端重注册后不可达的旧命名空间
        CustomCache: &RedisCache{client: redisClient},
    },
}
dict.SetConfig(config)
```

`ClearDBCache` / `ClearDictTableCache` / `ClearDictTableTwoCache` 仅切换对应结果缓存的命名空间，不调用此 `Clear`。外部旧条目由 TTL / 缓存自身淘汰回收；TTL 为 0 时不会自动回收。`Framework.ClearCache()` 仍直接调用自定义缓存的 `Clear`，上面的共享缓存示例会明确拒绝它。默认 DB 结果缓存读取 `SetConfig` 的全局配置，不是 `NewFramework(cfg)` 的实例配置。

### 5. 性能监控

```go
framework := dict.GetFramework()

// 只有经本实例执行的操作会计入其指标；字典也须注册到本实例
framework.Translate(&user)

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
    BatchQuery: true,    // 批量查询优化：切片 >= BatchQueryThreshold 时先收集 DB 类字段的 key，每组一次 IN 查询预热缓存（后端需实现 *BatchTranslator 可选接口）
}

items := make([]Item, 1000)
dict.TranslateBatch(&items, options)
```

## 📊 性能验证

使用仓库中的可运行基准，见 [README](README.md#performance)。数据库语句数由具体后端实现决定，一次批查接口调用不等于一次 SQL；不承诺固定加速比。

## 🎨 最佳实践

### 1. 配置优化

```go
config := &dict.Config{
    Performance: dict.PerformanceConfig{
        // 根据数据量调整阈值
        BatchQueryThreshold: 10,  // 小批量：10
        ParallelThreshold:   100,  // 大批量：100
        
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

func (f *CustomTranslatorFactory) Create(config map[string]any) (dict.Translator, error) {
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
2. **当前扩展点**：调用级中间件、插件初始化、显式策略；工厂及其他预留选项不自动参与翻译
3. **高自定义**：灵活配置、自定义缓存、自定义翻译器

选用前核对本页开头的执行边界；本库不提供通用分布式框架保证。

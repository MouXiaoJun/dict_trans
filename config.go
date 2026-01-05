package dict

import (
	"sync"
)

// Config 翻译框架配置
type Config struct {
	// 性能优化配置
	Performance PerformanceConfig

	// 缓存配置
	Cache CacheConfig

	// 扩展配置
	Extensions ExtensionsConfig

	// 自定义配置
	Custom map[string]interface{}
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	// 批量查询阈值：当批量翻译数量超过此值时，启用批量查询优化
	BatchQueryThreshold int

	// 并行处理阈值：当批量翻译数量超过此值时，启用并行处理
	ParallelThreshold int

	// 最大并发数
	MaxConcurrency int

	// 预加载字典：启动时预加载常用字典到内存
	PreloadDicts []string

	// 连接池配置（数据库相关）
	DBPoolSize int
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// 启用缓存
	Enabled bool

	// 缓存类型：memory, redis, custom
	Type string

	// 缓存过期时间（秒），0表示不过期
	TTL int

	// 最大缓存条目数
	MaxEntries int

	// 自定义缓存实现
	CustomCache Cache
}

// ExtensionsConfig 扩展配置
type ExtensionsConfig struct {
	// 中间件列表
	Middlewares []Middleware

	// 插件列表
	Plugins []Plugin

	// 自定义翻译器工厂
	TranslatorFactories map[string]TranslatorFactory
}

// Cache 缓存接口
type Cache interface {
	Get(key string) (string, bool)
	Set(key string, value string, ttl int) error
	Delete(key string) error
	Clear() error
}

// Middleware 中间件接口
type Middleware interface {
	// BeforeTranslate 翻译前处理
	BeforeTranslate(ctx *TranslateContext) error

	// AfterTranslate 翻译后处理
	AfterTranslate(ctx *TranslateContext) error
}

// Plugin 插件接口
type Plugin interface {
	// Name 插件名称
	Name() string

	// Init 初始化插件
	Init(config map[string]interface{}) error

	// Execute 执行插件逻辑
	Execute(ctx *TranslateContext) error
}

// TranslatorFactory 翻译器工厂接口
type TranslatorFactory interface {
	// Create 创建翻译器
	Create(config map[string]interface{}) (Translator, error)

	// Type 返回工厂类型
	Type() string
}

// TranslateContext 翻译上下文
type TranslateContext struct {
	// 源值
	SourceValue interface{}

	// 字段信息
	FieldName string
	FieldType string

	// 标签值
	TagValue string

	// 翻译结果
	Result string

	// 元数据
	Metadata map[string]interface{}

	// 是否跳过翻译
	Skip bool

	// 错误信息
	Error error
}

var (
	defaultConfig = &Config{
		Performance: PerformanceConfig{
			BatchQueryThreshold: 10,
			ParallelThreshold:   100,
			MaxConcurrency:      10,
			DBPoolSize:          10,
		},
		Cache: CacheConfig{
			Enabled:    true,
			Type:       "memory",
			TTL:        0, // 不过期
			MaxEntries: 10000,
		},
		Extensions: ExtensionsConfig{
			Middlewares:         make([]Middleware, 0),
			Plugins:             make([]Plugin, 0),
			TranslatorFactories: make(map[string]TranslatorFactory),
		},
		Custom: make(map[string]interface{}),
	}
	configMutex sync.RWMutex
)

// SetConfig 设置全局配置
func SetConfig(config *Config) {
	configMutex.Lock()
	defer configMutex.Unlock()
	defaultConfig = config
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return defaultConfig
}

// ResetConfig 重置为默认配置
func ResetConfig() {
	configMutex.Lock()
	defer configMutex.Unlock()
	defaultConfig = &Config{
		Performance: PerformanceConfig{
			BatchQueryThreshold: 10,
			ParallelThreshold:   100,
			MaxConcurrency:      10,
			DBPoolSize:          10,
		},
		Cache: CacheConfig{
			Enabled:    true,
			Type:       "memory",
			TTL:        0,
			MaxEntries: 10000,
		},
		Extensions: ExtensionsConfig{
			Middlewares:         make([]Middleware, 0),
			Plugins:             make([]Plugin, 0),
			TranslatorFactories: make(map[string]TranslatorFactory),
		},
		Custom: make(map[string]interface{}),
	}
}

// RegisterMiddleware 注册中间件
func RegisterMiddleware(middleware Middleware) {
	configMutex.Lock()
	defer configMutex.Unlock()
	defaultConfig.Extensions.Middlewares = append(defaultConfig.Extensions.Middlewares, middleware)
}

// RegisterPlugin 注册插件
func RegisterPlugin(plugin Plugin) error {
	configMutex.Lock()
	defer configMutex.Unlock()
	defaultConfig.Extensions.Plugins = append(defaultConfig.Extensions.Plugins, plugin)
	return nil
}

// RegisterTranslatorFactory 注册翻译器工厂
func RegisterTranslatorFactory(factory TranslatorFactory) {
	configMutex.Lock()
	defer configMutex.Unlock()
	defaultConfig.Extensions.TranslatorFactories[factory.Type()] = factory
}

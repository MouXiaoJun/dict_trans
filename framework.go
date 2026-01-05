package dict

import (
	"fmt"
	"reflect"
)

// Framework 翻译框架主入口
type Framework struct {
	config     *Config
	manager    *DictManager
	optimizer  *BatchQueryOptimizer
	preloader  *PreloadManager
	monitor    *PerformanceMonitor
	Strategies *StrategyManager
}

// NewFramework 创建翻译框架实例
func NewFramework(config *Config) *Framework {
	if config == nil {
		config = GetConfig()
	}

	return &Framework{
		config: config,
		manager: &DictManager{
			dicts:       make(map[string]map[string]string),
			translators: make(map[string]Translator),
			unwrappers:  make([]UnWrapper, 0),
			configCache: make(map[reflect.Type]*structConfig),
		},
		optimizer:  NewBatchQueryOptimizer(),
		preloader:  NewPreloadManager(),
		monitor:    NewPerformanceMonitor(),
		Strategies: NewStrategyManager(),
	}
}

// Init 初始化框架
func (f *Framework) Init() error {
	// 初始化缓存
	if f.config.Cache.Enabled {
		if f.config.Cache.CustomCache != nil {
			// 使用自定义缓存
		} else {
			// 使用默认内存缓存
			cache := NewMemoryCache(f.config.Cache.MaxEntries)
			f.config.Cache.CustomCache = cache
		}
	}

	// 初始化插件
	for _, plugin := range f.config.Extensions.Plugins {
		if err := plugin.Init(f.config.Custom); err != nil {
			return fmt.Errorf("初始化插件 %s 失败: %v", plugin.Name(), err)
		}
	}

	// 预加载字典
	for _, dictType := range f.config.Performance.PreloadDicts {
		// TODO: 实现预加载逻辑
		_ = dictType
	}

	return nil
}

// Translate 翻译（使用框架配置）
func (f *Framework) Translate(v interface{}, options ...*TranslateOptions) error {
	var opts *TranslateOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = &TranslateOptions{}
	}

	// 使用策略
	if opts.Strategy != "" {
		strategy := f.Strategies.GetStrategy(opts.Strategy)
		if strategy != nil {
			ctx := &TranslateContext{
				SourceValue: v,
				Metadata:    make(map[string]interface{}),
			}
			return strategy.Translate(ctx)
		}
	}

	// 使用默认翻译
	return f.manager.TranslateWithOptions(v, opts)
}

// RegisterDict 注册字典
func (f *Framework) RegisterDict(name string, dict map[string]string) {
	f.manager.dicts[name] = dict
}

// RegisterTranslator 注册翻译器
func (f *Framework) RegisterTranslator(tagName string, translator Translator) {
	f.manager.translators[tagName] = translator
}

// GetMetrics 获取性能指标
func (f *Framework) GetMetrics() map[string]*Metric {
	return f.monitor.GetMetrics()
}

// ClearCache 清空缓存
func (f *Framework) ClearCache() error {
	if f.config.Cache.CustomCache != nil {
		return f.config.Cache.CustomCache.Clear()
	}
	return nil
}

// GetDefaultFramework 获取默认框架实例
var defaultFramework *Framework

func init() {
	defaultFramework = NewFramework(nil)
	if err := defaultFramework.Init(); err != nil {
		panic(fmt.Sprintf("初始化默认框架失败: %v", err))
	}
}

// GetFramework 获取默认框架
func GetFramework() *Framework {
	return defaultFramework
}

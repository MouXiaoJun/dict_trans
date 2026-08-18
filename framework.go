package dict

import (
	"context"
	"fmt"
	"time"
)

// Framework 翻译框架主入口
type Framework struct {
	config     *Config
	manager    *DictManager
	cache      Cache // Init 时自动创建的内存缓存（用户未提供 CustomCache 时）
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
		config:     config,
		manager:    NewDictManager(),
		preloader:  NewPreloadManager(),
		monitor:    NewPerformanceMonitor(),
		Strategies: NewStrategyManager(),
	}
}

// Init 初始化框架
func (f *Framework) Init() error {
	// 初始化缓存：用户没给 CustomCache 时创建一个内存缓存挂在框架上（f.cache），
	// 不写回 config——只有用户显式设置的 Config.Cache.CustomCache 才会被 DB 结果缓存使用，
	// 否则三类 DB 缓存各自独立、ClearDBCache 不会波及 dictTable。
	if f.config.Cache.Enabled && f.config.Cache.CustomCache == nil && f.cache == nil {
		f.cache = NewMemoryCache(f.config.Cache.MaxEntries)
	}

	// 初始化插件
	for _, plugin := range f.config.Extensions.Plugins {
		if err := plugin.Init(f.config.Custom); err != nil {
			return fmt.Errorf("初始化插件 %s 失败: %v", plugin.Name(), err)
		}
	}

	// 预加载字典：要求已注册的字典表翻译器实现 DictTableLoader（CreateDictTableTranslatorFromDB 的返回值已实现），
	// 整表读入 preloader 并预热结果缓存
	for _, dictType := range f.config.Performance.PreloadDicts {
		dt := dictType
		err := f.preloader.Preload(dt, func() (map[string]string, error) {
			return defaultDictTableManager.preload(context.Background(), []string{dt})
		})
		if err != nil {
			return fmt.Errorf("预加载字典 %s 失败: %w", dt, err)
		}
	}

	return nil
}

// Preloaded 读取预加载的数据（Init 时按 Config.Performance.PreloadDicts 加载）
func (f *Framework) Preloaded(dictType, key string) (string, bool) {
	return f.preloader.Get(dictType, key)
}

// Translate 翻译（使用框架配置）
func (f *Framework) Translate(v any, options ...*TranslateOptions) error {
	var opts *TranslateOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = &TranslateOptions{}
	}

	// 使用策略，否则默认翻译；耗时与错误记入 PerformanceMonitor（GetMetrics()["translate"]）
	start := time.Now()
	var err error
	if strategy := f.strategyFor(opts.Strategy); strategy != nil {
		err = strategy.Translate(&TranslateContext{
			SourceValue: v,
			Metadata:    make(map[string]any),
		})
	} else {
		err = f.manager.TranslateWithOptions(v, opts)
	}
	f.monitor.Record("translate", time.Since(start).Microseconds(), err)
	return err
}

func (f *Framework) strategyFor(name string) TranslateStrategy {
	if name == "" {
		return nil
	}
	return f.Strategies.GetStrategy(name)
}

// RegisterDict 注册字典
func (f *Framework) RegisterDict(name string, dict map[string]string) {
	f.manager.RegisterDict(name, dict)
}

// RegisterTranslator 注册翻译器
func (f *Framework) RegisterTranslator(tagName string, translator Translator) {
	f.manager.RegisterTranslator(tagName, translator)
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
	if f.cache != nil {
		return f.cache.Clear()
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

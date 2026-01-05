package dict

import (
	"reflect"
	"sync"
)

// BatchQueryOptimizer 批量查询优化器
// 用于优化数据库查询，将多个查询合并为批量查询
type BatchQueryOptimizer struct {
	pendingQueries map[string][]pendingQuery
	mutex          sync.Mutex
}

type pendingQuery struct {
	key      string
	callback func(string, error)
}

// NewBatchQueryOptimizer 创建批量查询优化器
func NewBatchQueryOptimizer() *BatchQueryOptimizer {
	return &BatchQueryOptimizer{
		pendingQueries: make(map[string][]pendingQuery),
	}
}

// AddQuery 添加查询请求
func (o *BatchQueryOptimizer) AddQuery(queryKey string, key string, callback func(string, error)) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.pendingQueries[queryKey] == nil {
		o.pendingQueries[queryKey] = make([]pendingQuery, 0)
	}

	o.pendingQueries[queryKey] = append(o.pendingQueries[queryKey], pendingQuery{
		key:      key,
		callback: callback,
	})
}

// ExecuteBatch 执行批量查询
func (o *BatchQueryOptimizer) ExecuteBatch(queryKey string, batchQuery func([]string) (map[string]string, error)) {
	o.mutex.Lock()
	queries := o.pendingQueries[queryKey]
	delete(o.pendingQueries, queryKey)
	o.mutex.Unlock()

	if len(queries) == 0 {
		return
	}

	// 收集所有键
	keys := make([]string, len(queries))
	for i, q := range queries {
		keys[i] = q.key
	}

	// 执行批量查询
	results, err := batchQuery(keys)
	if err != nil {
		// 所有查询都返回错误
		for _, q := range queries {
			q.callback("", err)
		}
		return
	}

	// 分发结果
	for _, q := range queries {
		if value, ok := results[q.key]; ok {
			q.callback(value, nil)
		} else {
			q.callback("", nil)
		}
	}
}

// PreloadManager 预加载管理器
type PreloadManager struct {
	preloaded map[string]map[string]string
	mutex     sync.RWMutex
}

// NewPreloadManager 创建预加载管理器
func NewPreloadManager() *PreloadManager {
	return &PreloadManager{
		preloaded: make(map[string]map[string]string),
	}
}

// Preload 预加载字典
func (p *PreloadManager) Preload(dictType string, loader func() (map[string]string, error)) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	data, err := loader()
	if err != nil {
		return err
	}

	p.preloaded[dictType] = data
	return nil
}

// Get 获取预加载的数据
func (p *PreloadManager) Get(dictType, key string) (string, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if dict, ok := p.preloaded[dictType]; ok {
		if value, ok := dict[key]; ok {
			return value, true
		}
	}
	return "", false
}

// Clear 清空预加载数据
func (p *PreloadManager) Clear() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.preloaded = make(map[string]map[string]string)
}

// TranslateStrategy 翻译策略接口
type TranslateStrategy interface {
	// Translate 执行翻译
	Translate(ctx *TranslateContext) error

	// Name 策略名称
	Name() string
}

// StrategyManager 策略管理器
type StrategyManager struct {
	strategies      map[string]TranslateStrategy
	defaultStrategy TranslateStrategy
}

// NewStrategyManager 创建策略管理器
func NewStrategyManager() *StrategyManager {
	return &StrategyManager{
		strategies: make(map[string]TranslateStrategy),
	}
}

// RegisterStrategy 注册策略
func (s *StrategyManager) RegisterStrategy(strategy TranslateStrategy) {
	s.strategies[strategy.Name()] = strategy
}

// SetDefaultStrategy 设置默认策略
func (s *StrategyManager) SetDefaultStrategy(strategy TranslateStrategy) {
	s.defaultStrategy = strategy
}

// GetStrategy 获取策略
func (s *StrategyManager) GetStrategy(name string) TranslateStrategy {
	if strategy, ok := s.strategies[name]; ok {
		return strategy
	}
	return s.defaultStrategy
}

// DefaultTranslateStrategy 默认翻译策略
type DefaultTranslateStrategy struct{}

func (s *DefaultTranslateStrategy) Name() string {
	return "default"
}

func (s *DefaultTranslateStrategy) Translate(ctx *TranslateContext) error {
	// 默认策略：使用现有的翻译逻辑
	// 这里可以调用现有的翻译器
	return nil
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	metrics map[string]*Metric
	mutex   sync.RWMutex
}

type Metric struct {
	Count      int64
	TotalTime  int64 // 微秒
	MinTime    int64
	MaxTime    int64
	ErrorCount int64
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*Metric),
	}
}

// Record 记录性能指标
func (m *PerformanceMonitor) Record(operation string, duration int64, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	metric, ok := m.metrics[operation]
	if !ok {
		metric = &Metric{
			MinTime: duration,
			MaxTime: duration,
		}
		m.metrics[operation] = metric
	}

	metric.Count++
	metric.TotalTime += duration
	if duration < metric.MinTime {
		metric.MinTime = duration
	}
	if duration > metric.MaxTime {
		metric.MaxTime = duration
	}
	if err != nil {
		metric.ErrorCount++
	}
}

// GetMetrics 获取性能指标
func (m *PerformanceMonitor) GetMetrics() map[string]*Metric {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*Metric)
	for k, v := range m.metrics {
		result[k] = &Metric{
			Count:      v.Count,
			TotalTime:  v.TotalTime,
			MinTime:    v.MinTime,
			MaxTime:    v.MaxTime,
			ErrorCount: v.ErrorCount,
		}
	}
	return result
}

// GetAverageTime 获取平均时间（微秒）
func (m *Metric) GetAverageTime() int64 {
	if m.Count == 0 {
		return 0
	}
	return m.TotalTime / m.Count
}

// TranslateOptions 翻译选项
type TranslateOptions struct {
	// 使用策略
	Strategy string

	// 跳过缓存
	SkipCache bool

	// 批量模式
	BatchMode bool

	// 并行处理
	Parallel bool

	// 自定义上下文
	Context map[string]interface{}
}

// BatchOptions 批量翻译选项
type BatchOptions struct {
	// 并行处理
	Parallel bool

	// 批量查询优化
	BatchQuery bool

	// 并发数
	Concurrency int
}

// TranslateBatch 批量翻译（优化版）
func TranslateBatch(items interface{}, options *BatchOptions) error {
	return defaultManager.TranslateBatch(items, options)
}

// TranslateBatch 批量翻译（实例方法）
func (dm *DictManager) TranslateBatch(items interface{}, options *BatchOptions) error {
	rv := reflect.ValueOf(items)
	if rv.Kind() != reflect.Ptr {
		return ErrNotPointer
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Slice {
		return ErrNotSlice
	}

	if options == nil {
		options = &BatchOptions{
			Parallel:   true,
			BatchQuery: true,
		}
	}

	// 根据配置决定是否并行处理
	if options.Parallel {
		config := GetConfig()
		length := rv.Len()
		if length >= config.Performance.ParallelThreshold {
			return dm.batchTranslateParallel(rv)
		}
	}

	return dm.translateSlice(rv)
}

// TranslateWithOptions 使用选项翻译
func TranslateWithOptions(v interface{}, options *TranslateOptions) error {
	return defaultManager.TranslateWithOptions(v, options)
}

// TranslateWithOptions 使用选项翻译（实例方法）
func (dm *DictManager) TranslateWithOptions(v interface{}, options *TranslateOptions) error {
	if options == nil {
		options = &TranslateOptions{}
	}

	// 应用中间件
	ctx := &TranslateContext{
		Metadata: make(map[string]interface{}),
	}

	// BeforeTranslate 中间件
	config := GetConfig()
	for _, middleware := range config.Extensions.Middlewares {
		if err := middleware.BeforeTranslate(ctx); err != nil {
			return err
		}
		if ctx.Skip {
			return nil
		}
	}

	// 执行翻译
	err := dm.Translate(v)
	if err != nil {
		ctx.Error = err
	}

	// AfterTranslate 中间件
	for _, middleware := range config.Extensions.Middlewares {
		if err := middleware.AfterTranslate(ctx); err != nil {
			return err
		}
	}

	return err
}

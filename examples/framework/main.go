package main

import (
	"fmt"
	"time"

	"github.com/mouxiaojun/dict-trans"
)

// 示例：高性能、高扩展性、高自定义的翻译框架使用

func main() {
	fmt.Println("=== 高性能翻译框架示例 ===\n")

	// ========== 示例1: 基础配置 ==========
	fmt.Println("【示例1】框架配置")
	example1_Config()

	// ========== 示例2: 性能优化 ==========
	fmt.Println("\n【示例2】性能优化（批量查询、预加载）")
	example2_Performance()

	// ========== 示例3: 中间件 ==========
	fmt.Println("\n【示例3】中间件扩展")
	example3_Middleware()

	// ========== 示例4: 插件系统 ==========
	fmt.Println("\n【示例4】插件系统")
	example4_Plugin()

	// ========== 示例5: 自定义策略 ==========
	fmt.Println("\n【示例5】自定义翻译策略")
	example5_Strategy()

	// ========== 示例6: 性能监控 ==========
	fmt.Println("\n【示例6】性能监控")
	example6_Monitor()
}

// 示例1: 框架配置
func example1_Config() {
	// 创建自定义配置
	config := &dict.Config{
		Performance: dict.PerformanceConfig{
			BatchQueryThreshold: 5,                         // 5条以上启用批量查询
			ParallelThreshold:   50,                        // 50条以上启用并行处理
			MaxConcurrency:      20,                        // 最大并发数
			PreloadDicts:        []string{"sex", "status"}, // 预加载字典
		},
		Cache: dict.CacheConfig{
			Enabled:    true,
			Type:       "memory",
			TTL:        3600, // 1小时过期
			MaxEntries: 50000,
		},
		Custom: map[string]interface{}{
			"custom_key": "custom_value",
		},
	}

	// 设置全局配置
	dict.SetConfig(config)
	fmt.Println("  ✓ 框架配置已设置")
}

// 示例2: 性能优化
func example2_Performance() {
	// 注册字典
	dict.RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	// 批量翻译（自动使用批量查询优化）
	items := make([]Item, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = Item{Status: "1"}
	}

	start := time.Now()
	dict.Translate(&items)
	duration := time.Since(start)

	fmt.Printf("  ✓ 批量翻译 1000 条数据，耗时: %v\n", duration)
	fmt.Printf("  ✓ 平均每条: %v\n", duration/1000)
}

// 示例3: 中间件
func example3_Middleware() {
	// 创建日志中间件
	logMiddleware := &LogMiddleware{}

	// 注册中间件
	dict.RegisterMiddleware(logMiddleware)

	// 使用中间件进行翻译
	type User struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
	}

	dict.RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
	})

	user := User{Sex: "1"}
	dict.Translate(&user)
	fmt.Printf("  ✓ 中间件已执行，翻译结果: %s\n", user.SexName)
}

// LogMiddleware 日志中间件示例
type LogMiddleware struct{}

func (m *LogMiddleware) BeforeTranslate(ctx *dict.TranslateContext) error {
	fmt.Printf("  [中间件] 翻译前: 字段=%s, 值=%v\n", ctx.FieldName, ctx.SourceValue)
	return nil
}

func (m *LogMiddleware) AfterTranslate(ctx *dict.TranslateContext) error {
	fmt.Printf("  [中间件] 翻译后: 结果=%s\n", ctx.Result)
	return nil
}

// 示例4: 插件系统
func example4_Plugin() {
	// 创建自定义插件
	plugin := &CustomPlugin{}

	// 注册插件
	dict.RegisterPlugin(plugin)

	fmt.Println("  ✓ 插件已注册")
}

// CustomPlugin 自定义插件示例
type CustomPlugin struct{}

func (p *CustomPlugin) Name() string {
	return "custom_plugin"
}

func (p *CustomPlugin) Init(config map[string]interface{}) error {
	fmt.Println("  [插件] 初始化自定义插件")
	return nil
}

func (p *CustomPlugin) Execute(ctx *dict.TranslateContext) error {
	fmt.Println("  [插件] 执行自定义逻辑")
	return nil
}

// 示例5: 自定义策略
func example5_Strategy() {
	// 创建自定义翻译策略
	strategy := &CustomStrategy{}

	// 获取框架实例
	framework := dict.GetFramework()
	framework.Strategies.RegisterStrategy(strategy)

	fmt.Println("  ✓ 自定义策略已注册")
}

// CustomStrategy 自定义策略示例
type CustomStrategy struct{}

func (s *CustomStrategy) Name() string {
	return "custom"
}

func (s *CustomStrategy) Translate(ctx *dict.TranslateContext) error {
	// 自定义翻译逻辑
	ctx.Result = fmt.Sprintf("自定义翻译: %v", ctx.SourceValue)
	return nil
}

// 示例6: 性能监控
func example6_Monitor() {
	framework := dict.GetFramework()

	// 执行一些翻译操作
	type Item struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	items := make([]Item, 100)
	for i := 0; i < 100; i++ {
		items[i] = Item{Status: "1"}
	}
	dict.Translate(&items)

	// 获取性能指标
	metrics := framework.GetMetrics()
	fmt.Printf("  ✓ 性能指标:\n")
	for name, metric := range metrics {
		fmt.Printf("    %s: 调用次数=%d, 平均耗时=%d微秒\n",
			name, metric.Count, metric.GetAverageTime())
	}
}

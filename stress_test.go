package dict

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// 初始化测试数据
func initTestData() {
	// 注册字典
	RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
		"2": "待审核",
		"3": "已拒绝",
	})

	RegisterDict("sex", map[string]string{
		"1": "男",
		"2": "女",
		"0": "未知",
	})

	RegisterDict("type", map[string]string{
		"1": "类型A",
		"2": "类型B",
		"3": "类型C",
	})

	// 注册枚举
	RegisterEnum("priority", map[string]string{
		"high":   "高优先级",
		"medium": "中优先级",
		"low":    "低优先级",
	})
}

// 测试结构体
type TestUser struct {
	ID           int
	Status       string `dict:"status" dictField:"StatusName"`
	StatusName   string
	Sex          string `dict:"sex" dictField:"SexName"`
	SexName      string
	Type         string `dict:"type" dictField:"TypeName"`
	TypeName     string
	Priority     string `enum:"priority" dictField:"PriorityName"`
	PriorityName string
}

type TestUserNested struct {
	ID         int
	Status     string `dict:"status" dictField:"StatusName"`
	StatusName string
	User       TestUser
	Device     TestDevice
}

type TestDevice struct {
	Status     string `dict:"status" dictField:"StatusName"`
	StatusName string
	Type       string `dict:"type" dictField:"TypeName"`
	TypeName   string
}

// ==================== Benchmark 测试 ====================

// BenchmarkTranslateSingle 单次翻译性能测试
func BenchmarkTranslateSingle(b *testing.B) {
	initTestData()
	user := &TestUser{
		ID:       1,
		Status:   "1",
		Sex:      "1",
		Type:     "1",
		Priority: "high",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Translate(user)
		// 重置状态，避免缓存影响
		user.StatusName = ""
		user.SexName = ""
		user.TypeName = ""
		user.PriorityName = ""
	}
}

// BenchmarkTranslateBatch 批量翻译性能测试（顺序）
func BenchmarkTranslateBatch(b *testing.B) {
	initTestData()
	items := make([]TestUser, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = TestUser{
			ID:     i,
			Status: fmt.Sprintf("%d", i%4),
			Sex:    fmt.Sprintf("%d", i%3),
			Type:   fmt.Sprintf("%d", i%3+1),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = BatchTranslate(&items, false)
		// 重置状态
		for j := range items {
			items[j].StatusName = ""
			items[j].SexName = ""
			items[j].TypeName = ""
		}
	}
}

// BenchmarkTranslateBatchParallel 批量翻译性能测试（并行）
func BenchmarkTranslateBatchParallel(b *testing.B) {
	initTestData()
	items := make([]TestUser, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = TestUser{
			ID:     i,
			Status: fmt.Sprintf("%d", i%4),
			Sex:    fmt.Sprintf("%d", i%3),
			Type:   fmt.Sprintf("%d", i%3+1),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = BatchTranslate(&items, true)
		// 重置状态
		for j := range items {
			items[j].StatusName = ""
			items[j].SexName = ""
			items[j].TypeName = ""
		}
	}
}

// BenchmarkTranslateNested 嵌套结构翻译性能测试
func BenchmarkTranslateNested(b *testing.B) {
	initTestData()
	user := &TestUserNested{
		ID:     1,
		Status: "1",
		User: TestUser{
			Status: "1",
			Sex:    "1",
		},
		Device: TestDevice{
			Status: "1",
			Type:   "1",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Translate(user)
		// 重置状态
		user.StatusName = ""
		user.User.StatusName = ""
		user.User.SexName = ""
		user.Device.StatusName = ""
		user.Device.TypeName = ""
	}
}

// BenchmarkTranslateEnum 枚举翻译性能测试
func BenchmarkTranslateEnum(b *testing.B) {
	initTestData()
	user := &TestUser{
		Priority: "high",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Translate(user)
		user.PriorityName = ""
	}
}

// BenchmarkConfigCache 配置缓存性能测试
func BenchmarkConfigCache(b *testing.B) {
	initTestData()
	users := make([]*TestUser, 100)
	for i := 0; i < 100; i++ {
		users[i] = &TestUser{
			ID:     i,
			Status: "1",
			Sex:    "1",
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		user := users[i%100]
		_ = Translate(user)
		user.StatusName = ""
		user.SexName = ""
	}
}

// ==================== 并发压力测试 ====================

// TestConcurrentTranslate 并发翻译压力测试
func TestConcurrentTranslate(t *testing.T) {
	initTestData()
	const goroutineCount = 100
	const iterationsPerGoroutine = 1000

	var wg sync.WaitGroup
	errors := make(chan error, goroutineCount)

	start := time.Now()

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				user := &TestUser{
					ID:     id*iterationsPerGoroutine + j,
					Status: fmt.Sprintf("%d", j%4),
					Sex:    fmt.Sprintf("%d", j%3),
					Type:   fmt.Sprintf("%d", j%3+1),
				}
				if err := Translate(user); err != nil {
					errors <- fmt.Errorf("goroutine %d, iteration %d: %v", id, j, err)
					return
				}
				// 验证翻译结果
				if user.StatusName == "" {
					errors <- fmt.Errorf("goroutine %d, iteration %d: StatusName is empty", id, j)
					return
				}
			}
		}(i)
	}

	// 等待所有 goroutine 完成
	go func() {
		wg.Wait()
		close(errors)
	}()

	// 收集错误
	var errorCount int
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 10 {
			t.Log("Too many errors, stopping error collection")
			break
		}
	}

	duration := time.Since(start)
	totalOps := goroutineCount * iterationsPerGoroutine
	t.Logf("并发测试完成: %d 个 goroutine, 每个 %d 次迭代", goroutineCount, iterationsPerGoroutine)
	t.Logf("总操作数: %d", totalOps)
	t.Logf("耗时: %v", duration)
	t.Logf("QPS: %.0f", float64(totalOps)/duration.Seconds())
	t.Logf("平均延迟: %v", duration/time.Duration(totalOps))

	if errorCount > 0 {
		t.Fatalf("发现 %d 个错误", errorCount)
	}
}

// TestConcurrentBatchTranslate 并发批量翻译压力测试
func TestConcurrentBatchTranslate(t *testing.T) {
	initTestData()
	const goroutineCount = 50
	const itemsPerBatch = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutineCount)

	start := time.Now()

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			items := make([]TestUser, itemsPerBatch)
			for j := 0; j < itemsPerBatch; j++ {
				items[j] = TestUser{
					ID:     id*itemsPerBatch + j,
					Status: fmt.Sprintf("%d", j%4),
					Sex:    fmt.Sprintf("%d", j%3),
					Type:   fmt.Sprintf("%d", j%3+1),
				}
			}
			if err := BatchTranslate(&items, true); err != nil {
				errors <- fmt.Errorf("goroutine %d: %v", id, err)
				return
			}
			// 验证结果
			if items[0].StatusName == "" {
				errors <- fmt.Errorf("goroutine %d: translation failed", id)
				return
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	var errorCount int
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 10 {
			break
		}
	}

	duration := time.Since(start)
	totalItems := goroutineCount * itemsPerBatch
	t.Logf("并发批量测试完成: %d 个 goroutine, 每个 %d 条数据", goroutineCount, itemsPerBatch)
	t.Logf("总数据量: %d", totalItems)
	t.Logf("耗时: %v", duration)
	t.Logf("处理速度: %.0f items/s", float64(totalItems)/duration.Seconds())

	if errorCount > 0 {
		t.Fatalf("发现 %d 个错误", errorCount)
	}
}

// TestConcurrentConfigCache 并发配置缓存压力测试
func TestConcurrentConfigCache(t *testing.T) {
	initTestData()
	const goroutineCount = 200
	const iterationsPerGoroutine = 500

	var wg sync.WaitGroup
	errors := make(chan error, goroutineCount)

	// 创建多种不同的结构体类型，测试配置缓存
	type Type1 struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}
	type Type2 struct {
		Sex     string `dict:"sex" dictField:"SexName"`
		SexName string
	}
	type Type3 struct {
		Type     string `dict:"type" dictField:"TypeName"`
		TypeName string
	}

	start := time.Now()

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				var err error
				switch j % 3 {
				case 0:
					obj := &Type1{Status: "1"}
					err = Translate(obj)
				case 1:
					obj := &Type2{Sex: "1"}
					err = Translate(obj)
				case 2:
					obj := &Type3{Type: "1"}
					err = Translate(obj)
				}
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, iteration %d: %v", id, j, err)
					return
				}
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	var errorCount int
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 10 {
			break
		}
	}

	duration := time.Since(start)
	totalOps := goroutineCount * iterationsPerGoroutine
	t.Logf("配置缓存并发测试完成: %d 个 goroutine, 每个 %d 次迭代", goroutineCount, iterationsPerGoroutine)
	t.Logf("总操作数: %d", totalOps)
	t.Logf("耗时: %v", duration)
	t.Logf("QPS: %.0f", float64(totalOps)/duration.Seconds())

	if errorCount > 0 {
		t.Fatalf("发现 %d 个错误", errorCount)
	}
}

// ==================== 性能对比测试 ====================

// TestPerformanceComparison 性能对比测试
func TestPerformanceComparison(t *testing.T) {
	initTestData()
	const dataSize = 10000

	// 准备数据
	items := make([]TestUser, dataSize)
	for i := 0; i < dataSize; i++ {
		items[i] = TestUser{
			ID:     i,
			Status: fmt.Sprintf("%d", i%4),
			Sex:    fmt.Sprintf("%d", i%3),
			Type:   fmt.Sprintf("%d", i%3+1),
		}
	}

	// 测试顺序处理
	items1 := make([]TestUser, dataSize)
	copy(items1, items)
	start1 := time.Now()
	err1 := BatchTranslate(&items1, false)
	duration1 := time.Since(start1)
	if err1 != nil {
		t.Fatalf("顺序处理失败: %v", err1)
	}

	// 测试并行处理
	items2 := make([]TestUser, dataSize)
	copy(items2, items)
	start2 := time.Now()
	err2 := BatchTranslate(&items2, true)
	duration2 := time.Since(start2)
	if err2 != nil {
		t.Fatalf("并行处理失败: %v", err2)
	}

	// 验证结果一致性
	for i := 0; i < 100; i++ { // 抽样验证
		if items1[i].StatusName != items2[i].StatusName {
			t.Errorf("结果不一致 at index %d: 顺序=%s, 并行=%s", i, items1[i].StatusName, items2[i].StatusName)
		}
	}

	t.Logf("数据量: %d", dataSize)
	t.Logf("顺序处理耗时: %v (%.0f items/s)", duration1, float64(dataSize)/duration1.Seconds())
	t.Logf("并行处理耗时: %v (%.0f items/s)", duration2, float64(dataSize)/duration2.Seconds())
	t.Logf("性能提升: %.2f%%", (1-float64(duration2)/float64(duration1))*100)
}

// TestMemoryUsage 内存使用测试
func TestMemoryUsage(t *testing.T) {
	initTestData()
	const iterations = 10000

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 执行大量翻译操作
	for i := 0; i < iterations; i++ {
		user := &TestUser{
			ID:     i,
			Status: "1",
			Sex:    "1",
			Type:   "1",
		}
		_ = Translate(user)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocated := m2.TotalAlloc - m1.TotalAlloc
	heapInUse := m2.HeapInuse - m1.HeapInuse

	t.Logf("迭代次数: %d", iterations)
	t.Logf("总分配内存: %.2f KB", float64(allocated)/1024)
	t.Logf("堆内存增长: %.2f KB", float64(heapInUse)/1024)
	t.Logf("平均每次分配: %.2f bytes", float64(allocated)/float64(iterations))
}

// TestLargeBatch 大批量数据测试
func TestLargeBatch(t *testing.T) {
	initTestData()
	sizes := []int{100, 1000, 10000, 50000}

	for _, size := range sizes {
		items := make([]TestUser, size)
		for i := 0; i < size; i++ {
			items[i] = TestUser{
				ID:     i,
				Status: fmt.Sprintf("%d", i%4),
				Sex:    fmt.Sprintf("%d", i%3),
				Type:   fmt.Sprintf("%d", i%3+1),
			}
		}

		start := time.Now()
		err := BatchTranslate(&items, true)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("批量翻译失败 (size=%d): %v", size, err)
		}

		// 验证结果
		if items[0].StatusName == "" {
			t.Fatalf("翻译失败 (size=%d)", size)
		}

		t.Logf("数据量: %d, 耗时: %v, 速度: %.0f items/s", size, duration, float64(size)/duration.Seconds())
	}
}

// TestNestedDepth 嵌套深度测试
func TestNestedDepth(t *testing.T) {
	initTestData()

	type Level1 struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	type Level2 struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
		L1         Level1
	}

	type Level3 struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
		L2         Level2
	}

	type Level4 struct {
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
		L3         Level3
	}

	obj := &Level4{
		Status: "1",
		L3: Level3{
			Status: "1",
			L2: Level2{
				Status: "1",
				L1: Level1{
					Status: "1",
				},
			},
		},
	}

	start := time.Now()
	err := Translate(obj)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("嵌套翻译失败: %v", err)
	}

	// 验证所有层级都翻译成功
	if obj.StatusName == "" || obj.L3.StatusName == "" || obj.L3.L2.StatusName == "" || obj.L3.L2.L1.StatusName == "" {
		t.Fatal("嵌套翻译不完整")
	}

	t.Logf("嵌套深度: 4, 耗时: %v", duration)
}

// TestMixedTranslation 混合翻译类型测试
func TestMixedTranslation(t *testing.T) {
	initTestData()

	type Mixed struct {
		// 字典翻译
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
		Sex        string `dict:"sex" dictField:"SexName"`
		SexName    string
		// 枚举翻译
		Priority     string `enum:"priority" dictField:"PriorityName"`
		PriorityName string
		// 嵌套结构
		Device TestDevice
	}

	items := make([]Mixed, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = Mixed{
			Status:   fmt.Sprintf("%d", i%4),
			Sex:      fmt.Sprintf("%d", i%3),
			Priority: []string{"high", "medium", "low"}[i%3],
			Device: TestDevice{
				Status: fmt.Sprintf("%d", i%4),
				Type:   fmt.Sprintf("%d", i%3+1),
			},
		}
	}

	start := time.Now()
	err := BatchTranslate(&items, true)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("混合翻译失败: %v", err)
	}

	// 验证结果
	if items[0].StatusName == "" {
		t.Error("StatusName 为空")
	}
	if items[0].SexName == "" {
		t.Error("SexName 为空")
	}
	if items[0].PriorityName == "" {
		t.Error("PriorityName 为空")
	}
	if items[0].Device.StatusName == "" {
		t.Error("Device.StatusName 为空")
	}
	if items[0].StatusName == "" || items[0].SexName == "" || items[0].PriorityName == "" || items[0].Device.StatusName == "" {
		t.Fatalf("混合翻译不完整: StatusName=%s, SexName=%s, PriorityName=%s, Device.StatusName=%s",
			items[0].StatusName, items[0].SexName, items[0].PriorityName, items[0].Device.StatusName)
	}

	t.Logf("混合翻译测试: 1000 条数据, 耗时: %v, 速度: %.0f items/s", duration, float64(1000)/duration.Seconds())
}

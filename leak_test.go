package dict

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"
)

// 反复"并行批量 + 中途取消 / 首错即停"，goroutine 数与堆占用都不能持续增长
func TestNoGoroutineOrHeapLeakUnderCancellation(t *testing.T) {
	RegisterTranslator("leak_slow", TranslatorFunc(func(v any, _ string, _ string) (string, error) {
		time.Sleep(50 * time.Microsecond)
		if v == "boom" {
			return "", errors.New("boom")
		}
		return "ok", nil
	}))
	type Row struct {
		Code     string `translate:"leak_slow" dictField:"CodeName"`
		CodeName string
	}
	rows := make([]Row, 200)
	for i := range rows {
		rows[i].Code = "x"
	}
	rows[150].Code = "boom" // 落在中间某个 worker 的 chunk 里

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	goroutinesBefore := runtime.NumGoroutine()

	for i := 0; i < 300; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Microsecond)
		_ = TranslateWith(&rows, WithContext(ctx), WithParallel()) // 取消或首错都可能先到
		cancel()
		_ = BatchTranslate(&rows, true) // 纯首错即停路径
	}

	// 给已收到 done/ctx 的 worker 一点时间退出，再检查
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > goroutinesBefore+2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > goroutinesBefore+2 {
		t.Fatalf("goroutine 泄漏: before=%d after=%d", goroutinesBefore, n)
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if grew := int64(after.HeapInuse) - int64(before.HeapInuse); grew > 4<<20 {
		t.Fatalf("堆持续增长 %d 字节（600 轮取消/出错后不应留下垃圾）", grew)
	}
}

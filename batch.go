package dict

import (
	"context"
	"reflect"
	"sync"
)

// BatchTranslate 批量翻译（支持并行处理）
// items: 要翻译的切片指针
// parallel: 是否并行处理（长度 >= 10 时启用 worker pool）
// 等价于 TranslateWith(items, WithParallel())，但只接受切片。
func BatchTranslate(items any, parallel bool) error {
	return defaultManager.BatchTranslate(items, parallel)
}

// BatchTranslate 批量翻译（实例方法）
func (dm *DictManager) BatchTranslate(items any, parallel bool) error {
	rv := reflect.ValueOf(items)
	if rv.Kind() != reflect.Ptr {
		return ErrNotPointer
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Slice {
		return ErrNotSlice
	}
	if rv.Len() == 0 {
		return nil
	}
	return dm.translateSliceOpts(rv, &translateOpts{parallel: parallel})
}

// batchTranslateParallel 并行批量翻译
// 任一 worker 出错即通知其他 worker 停止，返回第一个错误。
// ponytail: 需要超时/取消时再加 BatchTranslateContext
func (dm *DictManager) batchTranslateParallel(sliceValue reflect.Value, ctx context.Context) error {
	length := sliceValue.Len()
	var ctxDone <-chan struct{} // ctx 为 nil 时保持 nil channel，select 里永远不就绪
	if ctx != nil {
		ctxDone = ctx.Done()
	}

	// 使用 worker pool 模式，避免创建过多 goroutine
	workerCount := 10
	if length < workerCount {
		workerCount = length
	}

	var wg sync.WaitGroup
	// 容量 >= workerCount，每个 worker 最多发一次，发送永不阻塞
	errChan := make(chan error, workerCount)
	// 首个错误出现时关闭，其他 worker 检查到即退出
	done := make(chan struct{})
	var closeOnce sync.Once

	// 所有 worker 共享一份 visited（带锁）：被多个元素共享的嵌套指针目标只由一个 worker 翻译，
	// 避免两个 goroutine 同时写同一个字段。顶层元素本身不记（平铺 []*Row 不碰锁）——
	// 同一指针作为顶层元素重复出现时会被并发翻译，调用方需自行去重（见 README 限制）。
	// ponytail: 单把互斥锁，只在 mark 嵌套指针目标时持有；成为瓶颈再分片
	shared := &walk{ctx: ctx, mu: &sync.Mutex{}}

	// 每个 worker 处理一部分数据
	chunkSize := (length + workerCount - 1) / workerCount

	for i := 0; i < workerCount; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > length {
			end = length
		}

		if start >= length {
			break
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()

			// 处理这个区间的数据
			for j := start; j < end; j++ {
				select {
				case <-done:
					return
				case <-ctxDone:
					return
				default:
				}
				elem, ok := sliceElemStruct(sliceValue.Index(j))
				if !ok {
					continue
				}
				if err := dm.translateStructV(elem, shared, false); err != nil {
					errChan <- err
					closeOnce.Do(func() { close(done) })
					return
				}
			}
		}(start, end)
	}

	wg.Wait()
	close(errChan)

	// 返回第一个错误；没有错误但 ctx 已取消时返回取消原因
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	if ctx != nil {
		return ctx.Err()
	}
	return nil
}

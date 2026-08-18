package dict

import (
	"reflect"
	"sync"
)

// BatchTranslate 批量翻译（支持并行处理）
// items: 要翻译的切片
// parallel: 是否并行处理（默认 false，保持向后兼容）
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

	length := rv.Len()
	if length == 0 {
		return nil
	}

	if !parallel || length < 10 {
		// 小批量或禁用并行时，顺序处理
		return dm.translateSlice(rv)
	}

	// 并行处理
	return dm.batchTranslateParallel(rv)
}

// batchTranslateParallel 并行批量翻译
// 任一 worker 出错即通知其他 worker 停止，返回第一个错误。
// ponytail: 需要超时/取消时再加 BatchTranslateContext
func (dm *DictManager) batchTranslateParallel(sliceValue reflect.Value) error {
	length := sliceValue.Len()

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
			seen := &visited{} // 每个 worker 一份：本 chunk 内共享的指针目标只走一次

			// 处理这个区间的数据
			for j := start; j < end; j++ {
				select {
				case <-done:
					return
				default:
				}
				elem := sliceValue.Index(j)
				if elem.Kind() == reflect.Ptr {
					if elem.IsNil() {
						continue
					}
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					if err := dm.translateStructV(elem, seen, false); err != nil {
						errChan <- err
						closeOnce.Do(func() { close(done) })
						return
					}
				}
			}
		}(start, end)
	}

	wg.Wait()
	close(errChan)

	// 返回第一个错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

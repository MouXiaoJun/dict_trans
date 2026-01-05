package dict

import (
	"reflect"
	"sync"
)

// BatchTranslate 批量翻译（支持并行处理）
// items: 要翻译的切片
// parallel: 是否并行处理（默认 false，保持向后兼容）
func BatchTranslate(items interface{}, parallel bool) error {
	return defaultManager.BatchTranslate(items, parallel)
}

// BatchTranslate 批量翻译（实例方法）
func (dm *DictManager) BatchTranslate(items interface{}, parallel bool) error {
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
func (dm *DictManager) batchTranslateParallel(sliceValue reflect.Value) error {
	length := sliceValue.Len()

	// 使用 worker pool 模式，避免创建过多 goroutine
	workerCount := 10
	if length < workerCount {
		workerCount = length
	}

	var wg sync.WaitGroup
	errChan := make(chan error, workerCount)

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
				elem := sliceValue.Index(j)
				if elem.Kind() == reflect.Ptr {
					if elem.IsNil() {
						continue
					}
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					if err := dm.translateStruct(elem); err != nil {
						errChan <- err
						return
					}
				}
			}
		}(start, end)
	}

	// 等待所有 worker 完成并关闭错误通道
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(errChan)
		close(done)
	}()

	// 收集错误
	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				// 通道已关闭，等待 done
				<-done
				return nil
			}
			if err != nil {
				return err
			}
		case <-done:
			return nil
		}
	}

	return nil
}

package dict

import "errors"

var (
	// ErrNotPointer 不是指针类型
	ErrNotPointer = errors.New("dict-trans: value must be a pointer")
	// ErrNotStruct 不是结构体类型
	ErrNotStruct = errors.New("dict-trans: value must be a struct")
	// ErrNotSlice 不是切片类型
	ErrNotSlice = errors.New("dict-trans: value must be a slice")
)

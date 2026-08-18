package dict

// TranslateOf 是 Translate 的泛型入口：编译期保证传入的是 *T，
// 传错类型不再是运行时的 ErrNotPointer。反射核心不变（struct tag 只能运行时读）。
func TranslateOf[T any](v *T) error {
	return Translate(v)
}

// BatchTranslateOf 是 BatchTranslate 的泛型入口：编译期保证传入的是 []*T。
func BatchTranslateOf[T any](items []*T, parallel bool) error {
	return BatchTranslate(&items, parallel)
}

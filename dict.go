// Package dict translates coded struct fields into display text using struct tags.
//
// A field such as
//
//	Status string `dict:"status" dictField:"StatusName"`
//
// is looked up in the dictionary named "status" and the result is written into
// the sibling field StatusName. Sources are in-memory dictionaries (RegisterDict),
// enums (RegisterEnum), database dictionary tables (RegisterDictTableTranslator,
// RegisterDictTableTwoTranslator, RegisterDBTranslator) or custom translators
// (RegisterTranslator). Translate walks nested structs, pointers and slices;
// BatchTranslate adds a parallel worker pool for large slices.
//
// Translation is best-effort: unknown dictionaries or missing target fields are
// skipped silently; only translator errors (for example database failures) are
// returned. All functions are safe for concurrent use; the registry is
// copy-on-write, so register dictionaries at startup.
//
// The package has no dependencies outside the standard library.
package dict

import (
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

// DictManager 字典管理器
type DictManager struct {
	reg         atomic.Pointer[registry]       // 字典 / 翻译器注册表（写时复制，读路径无锁）
	regWrite    sync.Mutex                     // 串行化注册表写者
	unwrappers  []UnWrapper                    // 包装类型解包器
	configCache map[reflect.Type]*structConfig // 配置缓存
	configMutex sync.RWMutex                   // 配置缓存互斥锁
}

// registry 字典与自定义翻译器注册表。读多写少（注册在启动期、翻译在热路径），
// 用 atomic.Pointer 做写时复制：读是一次原子 Load，不与其他 goroutine 争锁；
// 写在 regWrite 下拷贝一份再 Store。
// ponytail: 每次注册全量拷贝两张小 map，注册次数少可接受。
type registry struct {
	dicts       map[string]map[string]string // dictName -> {key: value}
	translators map[string]Translator        // tagName -> Translator
}

var emptyRegistry = &registry{}

// loadReg 读取当前注册表（未注册过时返回空表）
func (dm *DictManager) loadReg() *registry {
	if r := dm.reg.Load(); r != nil {
		return r
	}
	return emptyRegistry
}

// updateReg 写时复制地更新注册表
func (dm *DictManager) updateReg(fn func(*registry)) {
	dm.regWrite.Lock()
	defer dm.regWrite.Unlock()
	old := dm.loadReg()
	next := &registry{
		dicts:       make(map[string]map[string]string, len(old.dicts)+1),
		translators: make(map[string]Translator, len(old.translators)+1),
	}
	for k, v := range old.dicts {
		next.dicts[k] = v
	}
	for k, v := range old.translators {
		next.translators[k] = v
	}
	fn(next)
	dm.reg.Store(next)
}

// newDictManager 创建管理器
func newDictManager() *DictManager {
	return &DictManager{
		unwrappers:  make([]UnWrapper, 0),
		configCache: make(map[reflect.Type]*structConfig),
	}
}

// visited 记录一次翻译过程中已进入过的指针目标，防止自引用/环形结构无限递归。
// 只有经指针到达的结构体才需要记录（值类型嵌套不可能成环），map 惰性分配，无指针的常见场景零开销。
// 以 (地址, 类型) 为键：外层结构体与其第一个字段地址相同，只用地址会误判。
// 每次顶层翻译私有，不放在 DictManager 上。
type visited struct {
	small [4]visitKey // 前几个指针目标放栈上，常见 DTO 不碰堆
	n     int
	m     map[visitKey]struct{} // 溢出后才分配
}

type visitKey struct {
	addr uintptr
	typ  reflect.Type
}

// mark 记录一个指针目标；已记录过返回 false（调用方跳过）。
func (seen *visited) mark(rv reflect.Value) bool {
	if !rv.CanAddr() {
		return true
	}
	key := visitKey{addr: rv.UnsafeAddr(), typ: rv.Type()}
	for i := 0; i < seen.n; i++ {
		if seen.small[i] == key {
			return false
		}
	}
	if _, ok := seen.m[key]; ok {
		return false
	}
	if seen.n < len(seen.small) {
		seen.small[seen.n] = key
		seen.n++
		return true
	}
	if seen.m == nil {
		seen.m = make(map[visitKey]struct{})
	}
	seen.m[key] = struct{}{}
	return true
}

var defaultManager = newDictManager()

// structConfig 结构体配置缓存
type structConfig struct {
	fields  []fieldConfig
	byIndex []*fieldConfig // 按字段下标预建的索引，长度 = NumField，无配置处为 nil
	noop    bool           // 没有任何可设置字段（如 time.Time），翻译时直接跳过、不记 visited
}

// fieldConfig 字段配置
type fieldConfig struct {
	fieldIndex       int
	targetField      string
	targetFieldIndex int // 缓存目标字段索引，避免每次查找
	translatorTag    string
	translator       Translator
}

// RegisterDict 注册字典
func RegisterDict(name string, dict map[string]string) {
	defaultManager.RegisterDict(name, dict)
}

// GetDict 获取字典
func GetDict(name string) map[string]string {
	return defaultManager.GetDict(name)
}

// Translate 翻译结构体
func Translate(v any) error {
	return defaultManager.Translate(v)
}

// RegisterTranslator 注册自定义翻译器
func RegisterTranslator(tagName string, translator Translator) {
	defaultManager.RegisterTranslator(tagName, translator)
}

// RegisterDict 注册字典（实例方法，并发安全）
func (dm *DictManager) RegisterDict(name string, dict map[string]string) {
	dm.updateReg(func(r *registry) { r.dicts[name] = dict })
}

// GetDict 获取字典（实例方法，并发安全）
func (dm *DictManager) GetDict(name string) map[string]string {
	return dm.loadReg().dicts[name]
}

// RegisterTranslator 注册自定义翻译器（实例方法，并发安全）。
// 结构体配置缓存里固化了翻译器实例，注册后清空缓存，保证已翻译过的类型也能拿到新翻译器。
func (dm *DictManager) RegisterTranslator(tagName string, translator Translator) {
	dm.updateReg(func(r *registry) { r.translators[tagName] = translator })

	dm.configMutex.Lock()
	dm.configCache = make(map[reflect.Type]*structConfig)
	dm.configMutex.Unlock()
}

// Translate 翻译结构体（实例方法）
func (dm *DictManager) Translate(v any) error {
	// 尝试解包包装类型
	if unwrapped, ok := dm.tryUnwrap(v); ok {
		v = unwrapped
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return ErrNotPointer
	}

	rv = rv.Elem()

	// 支持切片类型
	if rv.Kind() == reflect.Slice {
		return dm.translateSlice(rv)
	}

	if rv.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	return dm.translateStruct(rv)
}

// translateStruct 翻译顶层结构体（私有一份 visited）
func (dm *DictManager) translateStruct(rv reflect.Value) error {
	return dm.translateStructV(rv, &visited{}, false)
}

// translateSlice 翻译顶层切片：整个切片共享一份 visited。
// 顶层元素本身不记 visited（平铺 []*Row 零开销），只记从元素内部经指针到达的目标，
// 所以父子链 / 树形数据里被多个元素共享的子图只走一次，总体 O(n) 而不是 O(n²)。
func (dm *DictManager) translateSlice(sliceValue reflect.Value) error {
	seen := &visited{}
	for i := 0; i < sliceValue.Len(); i++ {
		elem, ok := sliceElemStruct(sliceValue.Index(i))
		if !ok {
			continue
		}
		if err := dm.translateStructV(elem, seen, false); err != nil {
			return err
		}
	}
	return nil
}

// sliceElemStruct 取切片元素指向的结构体（解一层指针，nil 跳过）
func sliceElemStruct(elem reflect.Value) (reflect.Value, bool) {
	if elem.Kind() == reflect.Ptr {
		if elem.IsNil() {
			return elem, false
		}
		elem = elem.Elem()
	}
	return elem, elem.Kind() == reflect.Struct
}

// translateStructV 翻译结构体。viaPtr=true 表示 rv 是经指针到达的目标，需要记 visited 防环；
// 值类型嵌套不可能成环，不记。
func (dm *DictManager) translateStructV(rv reflect.Value, seen *visited, viaPtr bool) error {
	rt := rv.Type()

	// 获取或创建配置缓存（含按字段下标预建的索引）
	config := dm.getOrCreateConfig(rt)
	if config.noop {
		return nil
	}
	if viaPtr && !seen.mark(rv) {
		return nil
	}

	// 遍历所有字段（不仅仅是配置中的字段，因为嵌套结构体可能没有翻译标签）
	for i := 0; i < rt.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// 跳过不可设置的字段
		if !field.CanSet() {
			continue
		}

		// 处理嵌套结构体（先处理嵌套，再处理当前字段的翻译）
		if field.Kind() == reflect.Struct {
			if err := dm.translateStructV(field, seen, false); err != nil {
				return err
			}
		}

		// 处理指针类型的嵌套结构体（指针目标可能成环，viaPtr=true 记 visited）
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if target := field.Elem(); target.Kind() == reflect.Struct {
				if err := dm.translateStructV(target, seen, true); err != nil {
					return err
				}
			}
		}

		// 处理切片中的结构体
		if field.Kind() == reflect.Slice {
			if err := dm.translateSliceV(field, seen); err != nil {
				return err
			}
		}

		// 处理翻译标签（dict, enum, translator等）
		if fieldCfg := config.byIndex[i]; fieldCfg != nil {
			if fieldCfg.translator != nil {
				if err := dm.translateFieldWithTranslator(field, fieldType, *fieldCfg, rv); err != nil {
					return err
				}
			} else if fieldCfg.translatorTag != "" {
				// 兼容旧的 dict 标签
				if err := dm.translateField(field, fieldType, *fieldCfg, rv); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// getOrCreateConfig 获取或创建配置缓存（线程安全）
func (dm *DictManager) getOrCreateConfig(rt reflect.Type) *structConfig {
	// 先尝试读锁获取
	dm.configMutex.RLock()
	if config, ok := dm.configCache[rt]; ok {
		dm.configMutex.RUnlock()
		return config
	}
	dm.configMutex.RUnlock()

	// 需要创建配置，使用写锁
	dm.configMutex.Lock()
	defer dm.configMutex.Unlock()

	// 双重检查，防止并发创建
	if config, ok := dm.configCache[rt]; ok {
		return config
	}

	config := &structConfig{
		fields: make([]fieldConfig, 0),
	}

	for i := 0; i < rt.NumField(); i++ {
		fieldType := rt.Field(i)

		// 检查各种翻译标签
		dictTag := fieldType.Tag.Get("dict")
		dictTableTag := fieldType.Tag.Get("dictTable")
		dictTableTwoTag := fieldType.Tag.Get("dictTableTwo")
		enumTag := fieldType.Tag.Get("enum")
		translateTag := fieldType.Tag.Get("translate")
		dbTag := fieldType.Tag.Get("db")
		dictFieldTag := fieldType.Tag.Get("dictField")

		fieldCfg := fieldConfig{
			fieldIndex:       i,
			targetField:      dictFieldTag,
			targetFieldIndex: -1, // 初始化为 -1，表示未找到
		}

		// 优先级: translate > db > dictTableTwo > dictTable > enum > dict
		if translateTag != "" {
			// 自定义翻译器
			parts := strings.Split(translateTag, ",")
			tagName := parts[0]
			if translator, ok := dm.loadReg().translators[tagName]; ok {
				fieldCfg.translator = translator
				fieldCfg.translatorTag = translateTag
			}
		} else if dbTag != "" {
			// 数据库翻译（类似 Easy Trans 的自动查表）
			// 格式: db:"table=user,key=id,value=name"
			translator := parseDBTag(dbTag)
			if translator != nil {
				fieldCfg.translator = translator
				fieldCfg.translatorTag = dbTag
			}
		} else if dictTableTwoTag != "" {
			// 双表字典翻译（字典类型表+字典数据表）
			translator := createDictTableTwoTranslator(dictTableTwoTag)
			fieldCfg.translator = translator
			fieldCfg.translatorTag = dictTableTwoTag
		} else if dictTableTag != "" {
			// 字典表翻译（从数据库字典表读取，单表）
			translator := createDictTableTranslator(dictTableTag)
			fieldCfg.translator = translator
			fieldCfg.translatorTag = dictTableTag
		} else if enumTag != "" {
			// 枚举翻译
			fieldCfg.translator = DefaultEnumTranslator()
			fieldCfg.translatorTag = enumTag
		} else if dictTag != "" {
			// 字典翻译（兼容旧版本，内存字典）
			fieldCfg.translatorTag = dictTag
		}

		if fieldCfg.translator != nil || fieldCfg.translatorTag != "" {
			// 查找并缓存目标字段索引，避免运行时查找
			if fieldCfg.targetField != "" {
				for j := 0; j < rt.NumField(); j++ {
					structField := rt.Field(j)
					if structField.Name == fieldCfg.targetField || strings.EqualFold(structField.Name, fieldCfg.targetField) {
						fieldCfg.targetFieldIndex = j
						break
					}
				}
			}
			config.fields = append(config.fields, fieldCfg)
		}
	}

	// 按字段下标预建索引，避免每次翻译都重建 map
	config.byIndex = make([]*fieldConfig, rt.NumField())
	for i := range config.fields {
		config.byIndex[config.fields[i].fieldIndex] = &config.fields[i]
	}

	// 没有导出字段的结构体（time.Time 等）翻译时什么都做不了，标记为 noop
	config.noop = true
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).IsExported() {
			config.noop = false
			break
		}
	}

	dm.configCache[rt] = config
	return config
}

// translateSliceV 翻译嵌套切片（共享父结构体的 visited，因为元素可能指回父节点）
func (dm *DictManager) translateSliceV(sliceValue reflect.Value, seen *visited) error {
	for i := 0; i < sliceValue.Len(); i++ {
		raw := sliceValue.Index(i)
		elem, ok := sliceElemStruct(raw)
		if !ok {
			continue
		}
		// 指针元素才可能成环，viaPtr 交给 translateStructV 记 visited
		if err := dm.translateStructV(elem, seen, raw.Kind() == reflect.Ptr); err != nil {
			return err
		}
	}
	return nil
}

// translateField 翻译字段
func (dm *DictManager) translateField(field reflect.Value, fieldType reflect.StructField, fieldCfg fieldConfig, structValue reflect.Value) error {
	// 解析标签
	tagParts := strings.Split(fieldCfg.translatorTag, ",")
	dictName := tagParts[0]

	// 获取字典（原子读注册表快照，无锁）
	dict := dm.loadReg().dicts[dictName]
	if dict == nil {
		return nil // 字典不存在，跳过
	}

	// 获取源字段值
	if field.Kind() != reflect.String {
		return nil // 只支持字符串类型
	}

	sourceValue := field.String()
	if sourceValue == "" {
		return nil
	}

	// 获取翻译后的值
	translatedValue := dict[sourceValue]
	if translatedValue == "" {
		return nil
	}

	// 获取目标字段名
	dictFieldTag := fieldCfg.targetField
	if dictFieldTag == "" {
		return nil
	}

	// 优先使用缓存的字段索引（O(1)）
	if fieldCfg.targetFieldIndex >= 0 {
		targetField := structValue.Field(fieldCfg.targetFieldIndex)
		if targetField.CanSet() && targetField.Kind() == reflect.String {
			targetField.SetString(translatedValue)
			return nil
		}
	}

	// 回退到遍历查找
	structType := structValue.Type()
	for i := 0; i < structType.NumField(); i++ {
		structField := structType.Field(i)
		// 支持字段名匹配（大小写不敏感）
		if structField.Name == dictFieldTag || strings.EqualFold(structField.Name, dictFieldTag) {
			targetField := structValue.Field(i)
			if targetField.CanSet() && targetField.Kind() == reflect.String {
				targetField.SetString(translatedValue)
				break
			}
		}
	}

	return nil
}

// translateFieldWithTranslator 使用翻译器翻译字段
func (dm *DictManager) translateFieldWithTranslator(field reflect.Value, fieldType reflect.StructField, fieldCfg fieldConfig, structValue reflect.Value) error {
	// 获取源字段值
	var sourceValue any
	switch field.Kind() {
	case reflect.String:
		sourceValue = field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sourceValue = field.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		sourceValue = field.Uint()
	default:
		sourceValue = field.Interface()
	}

	// 调用翻译器
	translatedValue, err := fieldCfg.translator.Translate(sourceValue, fieldType.Name, fieldCfg.translatorTag)
	if err != nil {
		return err
	}

	if translatedValue == "" {
		return nil
	}

	// 设置目标字段
	if fieldCfg.targetField == "" {
		return nil
	}

	// 优先使用缓存的字段索引（O(1)），否则回退到遍历查找（O(n)）
	if fieldCfg.targetFieldIndex >= 0 {
		targetField := structValue.Field(fieldCfg.targetFieldIndex)
		if targetField.CanSet() && targetField.Kind() == reflect.String {
			targetField.SetString(translatedValue)
			return nil
		}
	}

	// 回退到遍历查找（兼容旧代码或字段索引未找到的情况）
	structType := structValue.Type()
	for i := 0; i < structType.NumField(); i++ {
		structField := structType.Field(i)
		if structField.Name == fieldCfg.targetField || strings.EqualFold(structField.Name, fieldCfg.targetField) {
			targetField := structValue.Field(i)
			if targetField.CanSet() && targetField.Kind() == reflect.String {
				targetField.SetString(translatedValue)
				break
			}
		}
	}

	return nil
}

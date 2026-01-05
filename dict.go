package dict

import (
	"reflect"
	"strings"
	"sync"
)

// DictManager 字典管理器
type DictManager struct {
	dicts       map[string]map[string]string   // 字典存储: dictName -> {key: value}
	translators map[string]Translator          // 自定义翻译器: tagName -> Translator
	unwrappers  []UnWrapper                    // 包装类型解包器
	configCache map[reflect.Type]*structConfig // 配置缓存
	configMutex sync.RWMutex                   // 配置缓存互斥锁
}

var defaultManager = &DictManager{
	dicts:       make(map[string]map[string]string),
	translators: make(map[string]Translator),
	unwrappers:  make([]UnWrapper, 0),
	configCache: make(map[reflect.Type]*structConfig),
}

// structConfig 结构体配置缓存
type structConfig struct {
	fields []fieldConfig
}

// fieldConfig 字段配置
type fieldConfig struct {
	fieldIndex    int
	sourceField   string
	targetField   string
	translatorTag string
	translator    Translator
}

// RegisterDict 注册字典
func RegisterDict(name string, dict map[string]string) {
	defaultManager.dicts[name] = dict
}

// GetDict 获取字典
func GetDict(name string) map[string]string {
	return defaultManager.dicts[name]
}

// Translate 翻译结构体
func Translate(v interface{}) error {
	return defaultManager.Translate(v)
}

// RegisterTranslator 注册自定义翻译器
func RegisterTranslator(tagName string, translator Translator) {
	defaultManager.translators[tagName] = translator
}

// Translate 翻译结构体（实例方法）
func (dm *DictManager) Translate(v interface{}) error {
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

// translateStruct 翻译结构体
func (dm *DictManager) translateStruct(rv reflect.Value) error {
	rt := rv.Type()

	// 获取或创建配置缓存
	config := dm.getOrCreateConfig(rt)

	// 创建字段配置映射，方便快速查找
	fieldConfigMap := make(map[int]*fieldConfig)
	for i := range config.fields {
		fieldConfigMap[config.fields[i].fieldIndex] = &config.fields[i]
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
			if err := dm.translateStruct(field); err != nil {
				return err
			}
		}

		// 处理指针类型的嵌套结构体
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if field.Elem().Kind() == reflect.Struct {
				if err := dm.translateStruct(field.Elem()); err != nil {
					return err
				}
			}
		}

		// 处理切片中的结构体
		if field.Kind() == reflect.Slice {
			if err := dm.translateSlice(field); err != nil {
				return err
			}
		}

		// 处理翻译标签（dict, enum, translator等）
		if fieldCfg, ok := fieldConfigMap[i]; ok {
			if fieldCfg.translator != nil {
				if err := dm.translateFieldWithTranslator(field, fieldType, *fieldCfg, rv); err != nil {
					return err
				}
			} else if fieldCfg.translatorTag != "" {
				// 兼容旧的 dict 标签
				if err := dm.translateField(field, fieldType, fieldCfg.translatorTag, rv); err != nil {
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
			fieldIndex:  i,
			targetField: dictFieldTag,
		}

		// 优先级: translate > db > dictTableTwo > dictTable > enum > dict
		if translateTag != "" {
			// 自定义翻译器
			parts := strings.Split(translateTag, ",")
			tagName := parts[0]
			if translator, ok := dm.translators[tagName]; ok {
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
			config.fields = append(config.fields, fieldCfg)
		}
	}

	dm.configCache[rt] = config
	return config
}

// translateSlice 翻译切片
func (dm *DictManager) translateSlice(sliceValue reflect.Value) error {
	for i := 0; i < sliceValue.Len(); i++ {
		elem := sliceValue.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		if elem.Kind() == reflect.Struct {
			if err := dm.translateStruct(elem); err != nil {
				return err
			}
		}
	}
	return nil
}

// translateField 翻译字段
func (dm *DictManager) translateField(field reflect.Value, fieldType reflect.StructField, dictTag string, structValue reflect.Value) error {
	// 解析标签
	tagParts := strings.Split(dictTag, ",")
	dictName := tagParts[0]

	// 获取字典
	dict := dm.dicts[dictName]
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
	dictFieldTag := fieldType.Tag.Get("dictField")
	if dictFieldTag == "" {
		return nil
	}

	// 在同一结构体中查找目标字段
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
	var sourceValue interface{}
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

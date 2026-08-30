package dict

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// MaskFormatter 脱敏格式：输入原文，输出脱敏后的字符串。
type MaskFormatter func(s string) string

// builtinMaskFormats 内置脱敏格式（所有管理器共享，只读）：
// 覆盖国内业务最常见的敏感字段。所有格式按 rune 处理，中文不截断。
var builtinMaskFormats = map[string]MaskFormatter{
	"phone":    maskKeepBoth(3, 4), // 手机号：138****8000
	"idcard":   maskKeepBoth(6, 4), // 身份证：110101********1234
	"bankcard": maskKeepBoth(4, 4), // 银行卡：6222********1234
	"address":  maskKeepBoth(6, 0), // 地址：保留前 6 个字符
	"email":    maskEmail,          // 邮箱：a***@b.com
	"name":     maskName,           // 姓名：张* / 张**
	"password": maskAll,            // 密码/密钥：全部掩掉
	"*":        maskAll,            // 通配：全部掩掉
}

// maskKeepBoth 保留前 head 与后 tail 个字符（按 rune 计），中间掩掉；
// 字符串太短（不足首尾之和）时全部掩掉。
func maskKeepBoth(head, tail int) MaskFormatter {
	return func(s string) string {
		r := []rune(s)
		if head >= len(r) || tail >= len(r)-head {
			return strings.Repeat("*", len(r))
		}
		return string(r[:head]) + strings.Repeat("*", len(r)-head-tail) + string(r[len(r)-tail:])
	}
}

// maskEmail 本地部分保留首字符，域名保留：zhangsan@example.com → z*******@example.com。
func maskEmail(s string) string {
	at := strings.IndexByte(s, '@')
	if at < 0 {
		return maskAll(s)
	}
	local, domain := s[:at], s[at:]
	r := []rune(local)
	if len(r) <= 1 {
		return "*" + domain
	}
	return string(r[:1]) + strings.Repeat("*", len(r)-1) + domain
}

// maskName 保留首字符，其余掩掉：张三 → 张*，张三四 → 张**。
func maskName(s string) string {
	r := []rune(s)
	if len(r) <= 1 {
		return s
	}
	return string(r[:1]) + strings.Repeat("*", len(r)-1)
}

// maskAll 全部掩掉（保留原长度）。
func maskAll(s string) string {
	return strings.Repeat("*", len([]rune(s)))
}

// maskErr 构建期发现的脱敏配置错误。
type maskErr struct{ field, name string }

// maskFieldConfig 单字段脱敏配置（getMaskConfig 时一次算好，脱敏热路径零解析）。
type maskFieldConfig struct {
	fieldIndex int
	format     MaskFormatter // 非 nil 表示该字段需脱敏；nil 表示仅嵌套递归
	nested     nestedKind
}

// maskConfig 按类型缓存的脱敏配置。
type maskConfig struct {
	fields []maskFieldConfig
	errs   []maskErr
}

// RegisterMaskFormat 注册自定义脱敏格式，name 即 struct tag 里的格式名。
// 注册后已脱敏过的类型也会用上新格式（脱敏配置缓存清空重建）。
// 示例：
//
//	RegisterMaskFormat("carplate", func(s string) string {
//	    r := []rune(s)
//	    if len(r) < 4 { return s }
//	    return string(r[:2]) + "***" + string(r[len(r)-2:])
//	})
//
//	type Car struct{ Plate string `mask:"carplate"` }
func RegisterMaskFormat(name string, fn MaskFormatter) {
	defaultManager.RegisterMaskFormat(name, fn)
}

// RegisterMaskFormat 注册自定义脱敏格式（实例方法，按管理器隔离，并发安全）。
func (dm *DictManager) RegisterMaskFormat(name string, fn MaskFormatter) {
	if name == "" || fn == nil {
		return
	}
	dm.updateReg(func(r *registry) { r.formats[name] = fn })

	dm.maskConfigMutex.Lock()
	dm.maskConfigCache = make(map[reflect.Type]*maskConfig)
	dm.maskConfigMutex.Unlock()
}

// maskFormat 查脱敏格式：自定义优先，回退内置。
func (dm *DictManager) maskFormat(name string) MaskFormatter {
	if fn := dm.loadReg().formats[name]; fn != nil {
		return fn
	}
	return builtinMaskFormats[name]
}

// Mask 就地脱敏结构体或结构体切片（修改 v 指向的字段值）。
// v 必须是结构体指针或结构体切片指针。配置错误（未注册格式）返回错误。
//
// struct tag 语法：
//   - `mask:"phone"`：使用内置/注册格式
//   - `mask:"3,4"`：通用格式，保留前 3 与后 4 个字符，中间掩掉
//   - `mask:"-"`：忽略该字段
//   - `mask:"*"`：全部掩掉
//
// 嵌套结构体、结构体指针、结构体切片自动递归脱敏；非字符串字段跳过。
func Mask(v any) error {
	return defaultManager.Mask(v)
}

// MaskOf 是 Mask 的泛型入口：编译期保证 *T。
func MaskOf[T any](v *T) error {
	return Mask(v)
}

// Mask 就地脱敏结构体或结构体切片（实例方法）。
func (dm *DictManager) Mask(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return ErrNotPointer
	}
	elem := rv.Elem()
	var errs []string
	seen := &walk{}
	switch elem.Kind() {
	case reflect.Struct:
		dm.maskStruct(elem, "", &errs, seen)
	case reflect.Slice, reflect.Array:
		for i := 0; i < elem.Len(); i++ {
			ev := elem.Index(i)
			for ev.Kind() == reflect.Ptr || ev.Kind() == reflect.Interface {
				if ev.IsNil() {
					ev = reflect.Value{}
					break
				}
				ev = ev.Elem()
			}
			if ev.IsValid() && ev.Kind() == reflect.Struct {
				dm.maskStruct(ev, fmt.Sprintf("[%d].", i), &errs, seen)
			}
		}
	default:
		return ErrNotStruct
	}
	if len(errs) > 0 {
		return fmt.Errorf("dict-trans: mask: %s", strings.Join(errs, "; "))
	}
	return nil
}

// maskStruct 递归脱敏结构体，path 用于错误消息。
func (dm *DictManager) maskStruct(rv reflect.Value, path string, errs *[]string, seen *walk) {
	if !seen.mark(rv) {
		return
	}
	cfg := dm.getMaskConfig(rv.Type())
	for _, me := range cfg.errs {
		*errs = append(*errs, fmt.Sprintf("字段 %s: mask 格式 %q 未注册", me.field, me.name))
	}
	for _, fc := range cfg.fields {
		fv := rv.Field(fc.fieldIndex)
		fieldName := path + rv.Type().Field(fc.fieldIndex).Name

		// 字符串字段：应用格式；非字符串字段带 tag 时忽略格式，仅当嵌套时继续递归。
		if fc.format != nil {
			if fv.Kind() == reflect.String {
				fv.SetString(fc.format(fv.String()))
				continue
			}
		}
		if fc.nested == nestedNone || !fv.IsValid() {
			continue
		}
		switch fc.nested {
		case nestedStruct:
			dm.maskStruct(fv, fieldName+".", errs, seen)
		case nestedPtr:
			if fv.IsNil() {
				continue
			}
			elem := fv.Elem()
			for elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
				if elem.IsNil() {
					elem = reflect.Value{}
					break
				}
				elem = elem.Elem()
			}
			if elem.IsValid() && elem.Kind() == reflect.Struct {
				dm.maskStruct(elem, fieldName+".", errs, seen)
			}
		case nestedSlice:
			for i := 0; i < fv.Len(); i++ {
				ev := fv.Index(i)
				for ev.Kind() == reflect.Ptr || ev.Kind() == reflect.Interface {
					if ev.IsNil() {
						ev = reflect.Value{}
						break
					}
					ev = ev.Elem()
				}
				if ev.IsValid() && ev.Kind() == reflect.Struct {
					dm.maskStruct(ev, fmt.Sprintf("%s[%d].", fieldName, i), errs, seen)
				}
			}
		}
	}
}

// getMaskConfig 获取类型的脱敏配置（构建并缓存）。
func (dm *DictManager) getMaskConfig(t reflect.Type) *maskConfig {
	dm.maskConfigMutex.RLock()
	cfg := dm.maskConfigCache[t]
	dm.maskConfigMutex.RUnlock()
	if cfg != nil {
		return cfg
	}
	dm.maskConfigMutex.Lock()
	defer dm.maskConfigMutex.Unlock()
	if cfg = dm.maskConfigCache[t]; cfg != nil {
		return cfg
	}
	cfg = dm.buildMaskConfig(t)
	dm.maskConfigCache[t] = cfg
	return cfg
}

// buildMaskConfig 构建某类型的脱敏配置：只保留有 mask tag 或需递归的字段。
func (dm *DictManager) buildMaskConfig(t reflect.Type) *maskConfig {
	cfg := &maskConfig{}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue // 未导出字段跳过
		}
		tag := sf.Tag.Get("mask")
		fc := maskFieldConfig{fieldIndex: i, nested: nestedKindOf(sf.Type)}
		if tag == "" {
			if fc.nested != nestedNone {
				cfg.fields = append(cfg.fields, fc)
			}
			continue
		}
		if tag == "-" {
			continue
		}
		// 通用格式：mask:"3,4"（保留前 3 后 4）
		if strings.Contains(tag, ",") {
			head, tail, ok := parseMaskKeep(tag)
			if !ok {
				cfg.errs = append(cfg.errs, maskErr{sf.Name, tag})
				continue
			}
			fc.format = maskKeepBoth(head, tail)
		} else {
			fn := dm.maskFormat(tag)
			if fn == nil {
				cfg.errs = append(cfg.errs, maskErr{sf.Name, tag})
				continue
			}
			fc.format = fn
		}
		cfg.fields = append(cfg.fields, fc)
	}
	return cfg
}

// parseMaskKeep 解析 "3,4" 形式的通用保留参数。
func parseMaskKeep(tag string) (head, tail int, ok bool) {
	parts := strings.Split(tag, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	ta, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || h < 0 || ta < 0 {
		return 0, 0, false
	}
	return h, ta, true
}

// nestedKindOf 判断类型的嵌套方式（结构体 / 结构体指针 / 结构体切片）。
func nestedKindOf(t reflect.Type) nestedKind {
	if t.Kind() == reflect.Struct {
		return nestedStruct
	}
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		if t.Kind() == reflect.Interface {
			return nestedNone // 接口无法静态确定具体类型，不递归
		}
		et := t.Elem()
		for et.Kind() == reflect.Ptr || et.Kind() == reflect.Interface {
			if et.Kind() == reflect.Interface {
				return nestedNone
			}
			et = et.Elem()
		}
		if et.Kind() == reflect.Struct {
			return nestedPtr
		}
	}
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		et := t.Elem()
		for et.Kind() == reflect.Ptr || et.Kind() == reflect.Interface {
			if et.Kind() == reflect.Interface {
				return nestedNone
			}
			et = et.Elem()
		}
		if et.Kind() == reflect.Struct {
			return nestedSlice
		}
	}
	return nestedNone
}

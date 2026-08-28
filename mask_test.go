package dict

import (
	"strings"
	"testing"
)

func TestMaskBasic(t *testing.T) {
	type User struct {
		Name     string `mask:"name"`
		Phone    string `mask:"phone"`
		IDCard   string `mask:"idcard"`
		Email    string `mask:"email"`
		Password string `mask:"password"`
		Note     string
	}
	u := User{
		Name: "张三", Phone: "13800138000",
		IDCard:   "110101199003071234",
		Email:    "zhangsan@example.com",
		Password: "secret", Note: "保留",
	}
	if err := Mask(&u); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if u.Name != "张*" || u.Phone != "138****8000" || u.IDCard != "110101********1234" ||
		u.Email != "z*******@example.com" || u.Password != "******" || u.Note != "保留" {
		t.Fatalf("mask result wrong: %+v", u)
	}
}

func TestMaskGenericKeep(t *testing.T) {
	type Account struct {
		No string `mask:"4,4"`
	}
	a := Account{No: "6222020200112233"}
	if err := Mask(&a); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if a.No != "6222********2233" {
		t.Fatalf("keep wrong: %q", a.No)
	}
}

func TestMaskWildcardAndIgnore(t *testing.T) {
	type Req struct {
		Token string `mask:"*"`
		Keep  string `mask:"-"`
	}
	r := Req{Token: "abc123", Keep: "保留"}
	if err := Mask(&r); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if r.Token != "******" || r.Keep != "保留" {
		t.Fatalf("wildcard/ignore wrong: %+v", r)
	}
}

func TestMaskNestedSlicePtr(t *testing.T) {
	type Card struct {
		No string `mask:"bankcard"`
	}
	type User struct {
		Name string `mask:"name"`
		Card Card   // 嵌套结构体
	}
	u := User{Name: "张三", Card: Card{No: "6222020200112233445"}}
	if err := Mask(&u); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if u.Card.No != "6222***********3445" {
		t.Fatalf("nested wrong: %q", u.Card.No)
	}

	// 切片 + 指针
	users := []User{{Name: "李四"}, {Name: "王五"}}
	if err := Mask(&users); err != nil {
		t.Fatalf("Mask slice failed: %v", err)
	}
	if users[0].Name != "李*" || users[1].Name != "王*" {
		t.Fatalf("slice wrong: %+v", users)
	}

	ptr := &User{Name: "赵六"}
	if err := Mask(ptr); err != nil {
		t.Fatalf("Mask ptr failed: %v", err)
	}
	if ptr.Name != "赵*" {
		t.Fatalf("ptr wrong: %+v", ptr)
	}
}

func TestMaskWithTranslate(t *testing.T) {
	// dict + mask 共存：先翻译后脱敏
	type User struct {
		Sex     string `dict:"sex" dictField:"SexName" mask:"name"` // mask 作用于源字段 Sex？Sex 是 string，会被脱敏
		SexName string
		Phone   string `mask:"phone"`
	}
	RegisterDict("sex", map[string]string{"1": "男", "2": "女"})
	u := User{Sex: "1", Phone: "13800138000"}
	if err := Translate(&u); err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if u.SexName != "男" {
		t.Fatalf("translate wrong: %+v", u)
	}
	if err := Mask(&u); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if u.Phone != "138****8000" {
		t.Fatalf("mask phone wrong: %+v", u)
	}
	// mask:"name" 作用于 Sex（"1" 是单字符，maskName 不掩，保持原值）
	if u.Sex != "1" {
		t.Fatalf("mask on sex field wrong: %q", u.Sex)
	}
}

func TestMaskErrors(t *testing.T) {
	type Req struct{ Name string }
	var req Req
	if err := Mask(req); err == nil || !strings.Contains(err.Error(), "pointer") {
		t.Fatalf("non-ptr err = %v", err)
	}
	if err := Mask(nil); err == nil {
		t.Fatal("nil should error")
	}
	var n int
	if err := Mask(&n); err == nil {
		t.Fatal("non-struct should error")
	}
}

func TestMaskUnregisteredFormat(t *testing.T) {
	type Req struct {
		Foo string `mask:"not_a_format"`
	}
	err := Mask(&Req{})
	if err == nil || !strings.Contains(err.Error(), "未注册") {
		t.Fatalf("unregistered format should error: %v", err)
	}
}

func TestMaskCustomFormat(t *testing.T) {
	type Req struct {
		Plate string `mask:"carplate"`
	}
	RegisterMaskFormat("carplate", func(s string) string {
		r := []rune(s)
		if len(r) < 4 {
			return s
		}
		return string(r[:2]) + "***" + string(r[len(r)-2:])
	})
	r := Req{Plate: "京A12345"}
	if err := Mask(&r); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if r.Plate != "京A***45" {
		t.Fatalf("custom format wrong: %q", r.Plate)
	}
}

func TestMaskNonStringSkipped(t *testing.T) {
	type Req struct {
		Age int `mask:"phone"` // 非 string：不脱敏不报错
	}
	r := Req{Age: 30}
	if err := Mask(&r); err != nil {
		t.Fatalf("non-string should be skipped: %v", err)
	}
	if r.Age != 30 {
		t.Fatalf("non-string mutated: %+v", r)
	}
}

func TestMaskOfGeneric(t *testing.T) {
	type User struct {
		Phone string `mask:"phone"`
	}
	u := User{Phone: "13800138000"}
	if err := MaskOf(&u); err != nil {
		t.Fatalf("MaskOf failed: %v", err)
	}
	if u.Phone != "138****8000" {
		t.Fatalf("MaskOf wrong: %q", u.Phone)
	}
}

func TestMaskIndependentManager(t *testing.T) {
	// 独立管理器：内置格式可用，自定义格式按管理器隔离
	mgr := NewDictManager()
	type User struct {
		Phone string `mask:"phone"`
	}
	u := User{Phone: "13800138000"}
	if err := mgr.Mask(&u); err != nil {
		t.Fatalf("manager Mask failed: %v", err)
	}
	if u.Phone != "138****8000" {
		t.Fatalf("manager mask wrong: %q", u.Phone)
	}
}

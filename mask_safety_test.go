package dict

import (
	"reflect"
	"strconv"
	"testing"
)

func TestMaskRetainCountOverflow(t *testing.T) {
	max := strconv.Itoa(int(^uint(0) >> 1))
	for _, format := range []string{max + ",1", "1," + max, max + "," + max, "1,1"} {
		t.Run(format, func(t *testing.T) {
			typ := reflect.StructOf([]reflect.StructField{{
				Name: "Secret", Type: reflect.TypeOf(""), Tag: reflect.StructTag("mask:" + strconv.Quote(format)),
			}})
			v := reflect.New(typ)
			v.Elem().Field(0).SetString("中文")
			if err := Mask(v.Interface()); err != nil {
				t.Fatal(err)
			}
			if got := v.Elem().Field(0).String(); got != "**" {
				t.Fatalf("Secret = %q, want **", got)
			}
		})
	}
}

func TestMaskCyclicAndSharedNodes(t *testing.T) {
	calls := 0
	RegisterMaskFormat("cycle-once", func(s string) string {
		calls++
		if calls > 4 {
			panic("the same object was visited repeatedly")
		}
		return s + "!"
	})
	type part struct {
		Secret string `mask:"cycle-once"`
	}
	type node struct {
		Part   part
		Secret string `mask:"cycle-once"`
		Next   *node
		Items  []*node
		Skip   *node `mask:"-"`
	}
	root := &node{Part: part{Secret: "a"}, Secret: "b"}
	child := &node{Part: part{Secret: "c"}, Secret: "d", Next: root}
	root.Next = root
	root.Items = []*node{child, child, nil}
	root.Skip = &node{Secret: "untouched"}
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("Mask did not terminate safely: %v", p)
		}
	}()
	if err := Mask(root); err != nil {
		t.Fatal(err)
	}
	if calls != 4 || root.Part.Secret != "a!" || root.Secret != "b!" ||
		child.Part.Secret != "c!" || child.Secret != "d!" || root.Skip.Secret != "untouched" {
		t.Fatalf("calls=%d, root=%+v, child=%+v", calls, root, child)
	}
}

func TestMaskDeepPointers(t *testing.T) {
	type child struct {
		Secret string `mask:"*"`
	}
	p := &child{Secret: "secret"}
	pp := &p
	v := struct{ Child ***child }{Child: &pp}
	if err := Mask(&v); err != nil {
		t.Fatal(err)
	}
	if p.Secret != "******" {
		t.Fatalf("Secret = %q", p.Secret)
	}
}

func TestMaskInterfaceFields(t *testing.T) {
	type child struct {
		Secret string `mask:"*"`
	}
	for _, value := range []any{nil, "plain", &child{Secret: "secret"}} {
		v := struct {
			Extra   any
			Skipped any `mask:"-"`
			Items   []any
			Secret  string `mask:"*"`
		}{Extra: value, Skipped: value, Items: []any{value}, Secret: "secret"}
		if err := Mask(&v); err != nil {
			t.Fatal(err)
		}
		if v.Secret != "******" {
			t.Fatalf("Secret = %q", v.Secret)
		}
		if c, ok := value.(*child); ok && c.Secret != "secret" {
			t.Fatal("interface fields should remain outside static traversal")
		}
	}
}

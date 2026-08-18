package dict

import (
	"sync"
	"sync/atomic"
	"testing"
)

type cycleNode struct {
	Status     string `dict:"cycle_status" dictField:"StatusName"`
	StatusName string
	Next       *cycleNode
	Children   []*cycleNode
}

// #1 自引用 / 环形指针不应栈溢出，且每个节点仍被翻译
func TestTranslateCyclicPointers(t *testing.T) {
	RegisterDict("cycle_status", map[string]string{"1": "启用"})
	a := &cycleNode{Status: "1"}
	b := &cycleNode{Status: "1", Next: a}
	a.Next = b
	a.Children = []*cycleNode{a, b}
	if err := Translate(a); err != nil {
		t.Fatal(err)
	}
	if a.StatusName != "启用" || b.StatusName != "启用" {
		t.Fatalf("环中节点应都被翻译: a=%q b=%q", a.StatusName, b.StatusName)
	}
}

type firstFieldInner struct {
	Status     string `dict:"cycle_status" dictField:"StatusName"`
	StatusName string
}
type firstFieldOuter struct {
	Inner firstFieldInner // 与外层地址相同，visited 不能把它当成已访问
	Tail  string
}

// #1 回归：外层结构体与第一个字段地址相同，不能误判为环
func TestTranslateFirstFieldSameAddress(t *testing.T) {
	RegisterDict("cycle_status", map[string]string{"1": "启用"})
	o := &firstFieldOuter{Inner: firstFieldInner{Status: "1"}}
	if err := Translate(o); err != nil {
		t.Fatal(err)
	}
	if o.Inner.StatusName != "启用" {
		t.Fatalf("第一个字段应被翻译，得到 %q", o.Inner.StatusName)
	}
}

// #2 注册与翻译并发无数据竞争（go test -race）
func TestRegisterAndTranslateConcurrently(t *testing.T) {
	type row struct {
		Status     string `dict:"race_status" dictField:"StatusName"`
		StatusName string
		Kind       string `translate:"race_kind" dictField:"KindName"`
		KindName   string
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); RegisterDict("race_status", map[string]string{"1": "ok"}) }()
		go func() {
			defer wg.Done()
			RegisterTranslator("race_kind", TranslatorFunc(func(v any, _ string, _ string) (string, error) { return "k", nil }))
		}()
		go func() { defer wg.Done(); _ = Translate(&row{Status: "1", Kind: "x"}) }()
	}
	wg.Wait()
	_ = GetDict("race_status")
}

// #3 先 Translate 再 RegisterTranslator，再 Translate 应生效（配置缓存需失效）。用独立 manager，-count>1 安全
func TestRegisterTranslatorAfterFirstTranslate(t *testing.T) {
	type row struct {
		Code     string `translate:"late_tr" dictField:"CodeName"`
		CodeName string
	}
	dm := NewDictManager()
	r := &row{Code: "x"}
	if err := dm.Translate(r); err != nil {
		t.Fatal(err)
	}
	if r.CodeName != "" {
		t.Fatalf("未注册翻译器时不应翻译，得到 %q", r.CodeName)
	}
	dm.RegisterTranslator("late_tr", TranslatorFunc(func(v any, _ string, _ string) (string, error) { return "late-" + v.(string), nil }))
	r2 := &row{Code: "x"}
	if err := dm.Translate(r2); err != nil {
		t.Fatal(err)
	}
	if r2.CodeName != "late-x" {
		t.Fatalf("注册后应生效，得到 %q", r2.CodeName)
	}
}

type chainNode struct {
	Code     string `translate:"chain_count" dictField:"CodeName"`
	CodeName string
	Parent   *chainNode
}

// #1 回归：父子链切片（每个元素指向前一个）应是 O(n)：翻译器调用次数 ≤ 2n，而不是 n²/2
func TestTranslateSliceSharedPointersLinear(t *testing.T) {
	var calls int64
	RegisterTranslator("chain_count", TranslatorFunc(func(v any, _ string, _ string) (string, error) {
		atomic.AddInt64(&calls, 1)
		return "x", nil
	}))
	const n = 2000
	items := make([]*chainNode, n)
	for i := range items {
		items[i] = &chainNode{Code: "c"}
		if i > 0 {
			items[i].Parent = items[i-1]
		}
	}
	if err := Translate(&items); err != nil {
		t.Fatal(err)
	}
	for i, it := range items {
		if it.CodeName != "x" {
			t.Fatalf("items[%d] 未翻译", i)
		}
	}
	if c := atomic.LoadInt64(&calls); c > 2*n {
		t.Fatalf("父子链切片翻译器调用 %d 次，超过 2n=%d，说明退化为 O(n²)", c, 2*n)
	}
}

// enum 注册与翻译并发无数据竞争（go test -race）
func TestRegisterEnumAndTranslateConcurrently(t *testing.T) {
	type row struct {
		Status     string `enum:"race_enum" dictField:"StatusName"`
		StatusName string
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); RegisterEnum("race_enum", map[string]string{"1": "ok"}) }()
		go func() { defer wg.Done(); _ = Translate(&row{Status: "1"}) }()
	}
	wg.Wait()
	r := &row{Status: "1"}
	if err := Translate(r); err != nil || r.StatusName != "ok" {
		t.Fatalf("enum 翻译失败: err=%v name=%q", err, r.StatusName)
	}
}

// 并行批量：多个元素共享同一个子对象（落在不同 worker 的 chunk）不能被两个 goroutine 同时翻译（-race）
func TestParallelSharedChildTranslatedOnce(t *testing.T) {
	type child struct {
		Status     string `dict:"shared_child" dictField:"StatusName"`
		StatusName string
	}
	type parent struct {
		Kid *child
	}
	RegisterDict("shared_child", map[string]string{"1": "ok"})
	kid := &child{Status: "1"}
	items := make([]parent, 100)
	for i := range items {
		items[i].Kid = kid
	}
	if err := BatchTranslate(&items, true); err != nil {
		t.Fatal(err)
	}
	if kid.StatusName != "ok" {
		t.Fatalf("共享子对象未翻译: %q", kid.StatusName)
	}
}

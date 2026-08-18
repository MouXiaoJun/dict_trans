package dict

import "testing"

// db 标签两种写法（"t:k:v" / "table=t,key=k,value=v"）的解析器不能 panic，
// 三段齐全时必须返回翻译器，缺任一段必须返回 nil
func FuzzParseDBTag(f *testing.F) {
	for _, seed := range []string{
		"", "user:id:name", "table=user,key=id,value=name", "table=user,key=id",
		"a:b", "a:b:c:d", "table=,key=,value=", ":::", "table=t,key=k,value=v,extra=x", "t:k:v=1",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, tag string) {
		tr := parseDBTag(tag)
		if tr == nil {
			return
		}
		lt, ok := tr.(*lookupTranslator)
		if !ok {
			t.Fatalf("parseDBTag 应返回 *lookupTranslator，得到 %T", tr)
		}
		if len(lt.parts) != 3 || lt.parts[0] == "" || lt.parts[1] == "" || lt.parts[2] == "" {
			t.Fatalf("三段应齐全: tag=%q -> %q", tag, lt.parts)
		}
	})
}

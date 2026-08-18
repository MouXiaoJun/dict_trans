package dict

import "testing"

func TestGenericEntrypoints(t *testing.T) {
	type row struct {
		Status     string `dict:"generic_status" dictField:"StatusName"`
		StatusName string
	}
	RegisterDict("generic_status", map[string]string{"1": "on"})

	r := &row{Status: "1"}
	if err := TranslateOf(r); err != nil || r.StatusName != "on" {
		t.Fatalf("TranslateOf: err=%v name=%q", err, r.StatusName)
	}

	rows := []*row{{Status: "1"}, {Status: "1"}}
	if err := BatchTranslateOf(rows, false); err != nil {
		t.Fatal(err)
	}
	for i, x := range rows {
		if x.StatusName != "on" {
			t.Fatalf("rows[%d] = %q", i, x.StatusName)
		}
	}
}

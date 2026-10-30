package kosac

import (
	"slices"
	"testing"
)

func TestRecordSetKeepsLoc(t *testing.T) {
	loc := Loc{SheetVars, "B4"}
	r := Record{"VAR_NAME": {Value: "IDX", Loc: loc}}
	r.Set("VAR_NAME", "ID")
	if got := r["VAR_NAME"]; got != (Field{Value: "ID", Loc: loc}) {
		t.Errorf("after Set: %+v", got)
	}
	r.Set("NOTE", "x")
	if got := r["NOTE"]; got != (Field{Value: "x"}) {
		t.Errorf("Set of a new key: %+v", got)
	}
	if r.Get("MISSING") != "" || r.Get("VAR_NAME") != "ID" {
		t.Errorf("Get: %q %q", r.Get("MISSING"), r.Get("VAR_NAME"))
	}
}

func rec(kv ...string) Record {
	r := Record{}
	for i := 0; i < len(kv); i += 2 {
		r.Set(kv[i], kv[i+1])
	}
	return r
}

func TestLookups(t *testing.T) {
	f1, f2 := &File{Rec: rec("FILE_NO", "1")}, &File{Rec: rec("FILE_NO", "2")}
	a1 := &Var{Rec: rec("FILE_NO", "1", "VAR_NAME", "A")}
	a2 := &Var{Rec: rec("FILE_NO", "2", "VAR_NAME", "A")}
	b1 := &Var{Rec: rec("FILE_NO", "1", "VAR_NAME", "B")}
	c1 := &Code{Rec: rec("FILE_NO", "1", "VAR_NAME", "A", "CODE", "1")}
	c2 := &Code{Rec: rec("FILE_NO", "2", "VAR_NAME", "A", "CODE", "1")}
	c3 := &Code{Rec: rec("FILE_NO", "1", "VAR_NAME", "A", "CODE", "2")}
	cb := &Codebook{Files: []*File{f1, f2}, Vars: []*Var{a1, a2, b1}, Codes: []*Code{c1, c2, c3}}

	if cb.File("2") != f2 || cb.File("3") != nil {
		t.Error("File")
	}
	if cb.Var("2", "A") != a2 || cb.Var("1", "B") != b1 || cb.Var("2", "B") != nil {
		t.Error("Var")
	}
	if got := cb.CodesOf(a1); !slices.Equal(got, []*Code{c1, c3}) {
		t.Errorf("CodesOf(a1) = %v", got)
	}
	if got := cb.CodesOf(b1); len(got) != 0 {
		t.Errorf("CodesOf(b1) = %v", got)
	}
}

func TestUnverified(t *testing.T) {
	for _, c := range []struct {
		in, reason string
		ok         bool
	}{
		{"확인 불가: 보고서 미기재", "보고서 미기재", true},
		{"확인불가：x", "x", true},
		{"  확인  불가 :  사유  ", "사유", true},
		{"확인 불가:", "", true},
		{"확인 불가: 보고서\n미기재", "보고서\n미기재", true},
		{"확인 불가", "", false},
		{"보고서 확인 불가: x", "", false},
		{"", "", false},
	} {
		reason, ok := Unverified(c.in)
		if reason != c.reason || ok != c.ok {
			t.Errorf("Unverified(%q) = %q, %v; want %q, %v", c.in, reason, ok, c.reason, c.ok)
		}
	}
}

func TestKeyLists(t *testing.T) {
	for name, c := range map[string]struct {
		keys []string
		n    int
	}{"Study": {StudyKeys, 44}, "File": {FileKeys, 10}, "Var": {VarKeys, 15}, "Code": {CodeKeys, 5}} {
		if len(c.keys) != c.n {
			t.Errorf("%sKeys has %d keys, want %d", name, len(c.keys), c.n)
		}
	}
	if len(StudyRequired) != len(StudyKeys) {
		t.Errorf("StudyRequired has %d keys", len(StudyRequired))
	}
	for _, k := range StudyKeys {
		if v := StudyRequired[k]; !slices.Contains([]string{"필수", "선택", "조건부", "재단"}, v) {
			t.Errorf("StudyRequired[%s] = %q", k, v)
		}
	}
}

func TestPrefix(t *testing.T) {
	for name, want := range map[string]string{"QA5_1_1": "QA5_1", "Q5_other": "Q5", "C8_1": "C8", "B2": "", "_x": ""} {
		got, ok := Prefix(name)
		if got != want || ok != (want != "") {
			t.Errorf("Prefix(%q) = %q, %v; want %q", name, got, ok, want)
		}
	}
}

func TestParsePages(t *testing.T) {
	for in, want := range map[string]string{
		"REPORT:p336":     "336",
		"REPORT:p336-":    "336-",
		"REPORT:p336-340": "336-340",
		"최종보고서 부록 수록 (PDF 336쪽~)":    "336-",
		"최종보고서 부록 수록 (PDF 336~340쪽)": "336-340",
		"보고서 부록 수록 336쪽":             "336",
		"보고서 부록 수록 (336 – 340)":      "336-340",
		"부록 수록 (336～340쪽)":           "336-340",
		"부록 수록 (336∼340쪽)":           "336-340",
		"부록 수록 (336〜340쪽)":           "336-340",
		"부록 수록 (PDF 336쪽～)":          "336-",
		"보고서 참조":                     "",
		"3-2024_최종보고서.pdf":           "",
		"REPORT:p340-336":            "",
		"REPORT:p0":                  "",
		"최종보고서 부록 (PDF 336쪽~)":       "",
	} {
		got, ok := ParsePages(in)
		if got != want || ok != (want != "") {
			t.Errorf("ParsePages(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}

func TestMissingCodes(t *testing.T) {
	var got []string
	for _, m := range MissingCodes {
		got = append(got, m.Code+" "+m.Type+" "+yes(m.Declared))
		if m.Declared && len(DefaultMissingLabels[m.Code]) == 0 {
			t.Errorf("declared %s has no default label", m.Code)
		}
	}
	if want := []string{"-8 MISSING Y", "-7 DK Y", "-9 MISSING Y", "-6 MISSING "}; !slices.Equal(got, want) {
		t.Errorf("MissingCodes %q, want %q", got, want)
	}
}

func TestSystemVars(t *testing.T) {
	var names []string
	for _, s := range SystemVars {
		names = append(names, s.Name)
		if s.Label == "" || s.Role != "SYSTEM" {
			t.Errorf("%+v", s)
		}
		if (s.MeasLevel == "date") != (s.Category == "date") {
			t.Errorf("%s: category %q for measLevel %q", s.Name, s.Category, s.MeasLevel)
		}
	}
	want := []string{"walla_response_id", "walla_row_no", "walla_customer_key", "walla_submitted_at", "walla_started_at", "walla_duration_sec"}
	if !slices.Equal(names, want) {
		t.Errorf("SystemVars names = %q", names)
	}
	if !IsSystemVar("walla_response_id") || IsSystemVar("IDX") {
		t.Error("IsSystemVar")
	}
}

package check

import (
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"slices"
	"testing"
)

func v4Model() *kosac.Codebook {
	rec := func(m map[string]string) kosac.Record {
		r := kosac.Record{}
		for k, v := range m {
			r.Set(k, v)
		}
		return r
	}
	return &kosac.Codebook{Template: kosac.TemplateMeta{Version: kosac.Field{Value: "4.0"}},
		Study:   rec(map[string]string{"TITLE": "example", "SERIES_NAME": "series", "STUDY_ROUND": "2026", "ACCESS_RESTRICTIONS": "내부 이용", "CLEANING_LOG": "확인 완료", "SOURCE_FILES": "raw.csv"}),
		Files:   []*kosac.File{{Rec: rec(map[string]string{"FILE_NO": "1", "FILE_NAME": "data.csv", "VARIABLE_N": "1", "UNIQUE_KEY": "q", "ID_VAR": "q"})}},
		Vars:    []*kosac.Var{{Rec: rec(map[string]string{"FILE_NO": "1", "VAR_NAME": "q", "DATA_TYPE": "문자", "DDI_INTERVAL": "discrete", "MEAS_LEVEL": "명목", "ACCESS_LEVEL": "내부 이용", "SOURCE_VAR": "raw_q", "RECODE_STATUS": "처리 완료"})}},
		Codes:   []*kosac.Code{{Rec: rec(map[string]string{"FILE_NO": "1", "VAR_NAME": "q", "CODE": "01", "CODE_LABEL": "남자", "IS_MISSING": "아니오", "MISSING_REASON": "해당 없음", "RECODE_STATUS": "처리 완료"})}, {Rec: rec(map[string]string{"FILE_NO": "1", "VAR_NAME": "q", "CODE": "-9", "CODE_LABEL": "무응답", "IS_MISSING": "예", "MISSING_REASON": "무응답", "RECODE_STATUS": "처리 완료"})}},
		Recodes: []kosac.Record{rec(map[string]string{"FILE_NO": "1", "SOURCE_FILE": "raw.csv", "SOURCE_VAR": "raw_q", "SOURCE_CODE_OR_CONDITION": "1", "TARGET_VAR": "q", "TARGET_CODE_OR_RULE": "01", "TRANSFORM_TYPE": "코드 통합", "BASIS": "codebook", "RECORD_COUNT": "12", "STATUS": "처리 완료"})}}
}

func TestV4Rules(t *testing.T) {
	for _, tc := range []struct {
		name   string
		modify func(*kosac.Codebook)
		field  string
	}{
		{"valid", func(cb *kosac.Codebook) {}, ""},
		{"missing study", func(cb *kosac.Codebook) { cb.Study.Set("SOURCE_FILES", "") }, "SOURCE_FILES"},
		{"variable count", func(cb *kosac.Codebook) { cb.Files[0].Rec.Set("VARIABLE_N", "2") }, "VARIABLE_N"},
		{"data type", func(cb *kosac.Codebook) { cb.Vars[0].Rec.Set("DATA_TYPE", "bad") }, "DATA_TYPE"},
		{"interval", func(cb *kosac.Codebook) { cb.Vars[0].Rec.Set("DDI_INTERVAL", "continuous") }, "DDI_INTERVAL"},
		{"status reference", func(cb *kosac.Codebook) { cb.Vars[0].Rec.Set("MISSING_STATUS_VAR", "missing") }, "MISSING_STATUS_VAR"},
		{"missing declaration", func(cb *kosac.Codebook) { cb.Codes[1].Rec.Set("IS_MISSING", "아니오") }, "IS_MISSING"},
		{"missing reason", func(cb *kosac.Codebook) { cb.Codes[1].Rec.Set("MISSING_REASON", "") }, "MISSING_REASON"},
		{"recode file", func(cb *kosac.Codebook) { cb.Recodes[0].Set("FILE_NO", "9") }, "FILE_NO"},
		{"recode evidence", func(cb *kosac.Codebook) { cb.Recodes[0].Set("BASIS", "") }, "BASIS"},
		{"target variable", func(cb *kosac.Codebook) { cb.Recodes[0].Set("TARGET_VAR", "unknown") }, "TARGET_VAR"},
		{"negative count", func(cb *kosac.Codebook) { cb.Recodes[0].Set("RECORD_COUNT", "-1") }, "RECORD_COUNT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cb := v4Model()
			tc.modify(cb)
			checker := newChecker(cb, kosac.RefData{})
			checker.v4()
			is := checker.issues
			var relevant []kosac.Issue
			for _, i := range is {
				if i.Rule >= "C19" && i.Rule <= "C27" || i.Rule == "C01" {
					relevant = append(relevant, i)
				}
			}
			if tc.field == "" && len(relevant) > 0 {
				t.Fatal(relevant)
			}
			if tc.field != "" && !slices.ContainsFunc(relevant, func(i kosac.Issue) bool { return i.Level == kosac.R }) {
				t.Fatalf("missing validation for %s: %v", tc.field, relevant)
			}
		})
	}
}
func TestV4MissingCodesAndStringCodes(t *testing.T) {
	cb := v4Model()
	before := len(cb.Codes)
	kosac.DeriveDefaults(cb, kosac.RefData{})
	if len(cb.Codes) != before || cb.Codes[1].Type != "MISSING" || cb.Codes[1].Rec.Get("CODE_LABEL") != "무응답" {
		t.Fatal("invented or relabeled missing values")
	}
	cb.Codes[0].Rec.Set("CODE", "A01")
	for _, i := range Check(cb, kosac.RefData{}) {
		if i.Level == kosac.R && (i.Rule == "C14" || i.Rule == "C15") {
			t.Fatal(i)
		}
	}
}

func TestV4XMLValidation(t *testing.T) {
	cb := engineModel()
	cb.Template.Version.Value = "4.0"
	for _, code := range cb.Codes {
		code.Rec.Set("IS_MISSING", "아니오")
	}
	cb.Study.Set("SERIES_NAME", "series")
	cb.Study.Set("STUDY_ROUND", "2026")
	cb.Study.Set("ACCESS_RESTRICTIONS", "내부 이용")
	cb.Recodes = []kosac.Record{v4Model().Recodes[0]}
	source, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	if issues := ValidateXML(source, xmllint(t)); len(issues) > 0 {
		t.Fatal(issues)
	}
}

func TestV4DeclaredMissingIsNotAValidCategory(t *testing.T) {
	cb := v4Model()
	cb.Vars[0].Rec.Set("MEAS_LEVEL", "명목")
	cb.Codes[1].Rec.Set("CODE", "99")
	issues := Check(cb, kosac.RefData{})
	if !slices.ContainsFunc(issues, func(i kosac.Issue) bool { return i.Rule == "C07" && i.Level == kosac.R }) {
		t.Fatal("declared missing category counted as valid", issues)
	}
}

func TestV4RangeTargetIsNotGuessed(t *testing.T) {
	cb := v4Model()
	cb.Recodes[0].Set("TARGET_VAR", "Q1~Q9993")
	relevant := []kosac.Issue{}
	for _, i := range Check(cb, kosac.RefData{}) {
		if i.Rule == "C25" {
			relevant = append(relevant, i)
		}
	}
	if len(relevant) != 1 || relevant[0].Level != kosac.R {
		t.Fatal("unknown range endpoints must be rejected", relevant)
	}
}

func TestV4TargetsUseDeclaredOrder(t *testing.T) {
	cb := v4Model()
	for _, name := range []string{"a", "b", "DQ1"} {
		r := kosac.Record{}
		r.Set("FILE_NO", "1")
		r.Set("VAR_NAME", name)
		cb.Vars = append(cb.Vars, &kosac.Var{Rec: r})
	}
	c := newChecker(cb, kosac.RefData{})
	for _, expression := range []string{"a~DQ1", "a~DQ1 (3개: a, b, DQ1)"} {
		got, err := c.v4targets("1", expression, "")
		if err != nil || !slices.Equal(got, []string{"a", "b", "DQ1"}) {
			t.Fatal(expression, got, err)
		}
	}
}

func TestV4PendingThreshold(t *testing.T) {
	for _, limit := range []float64{0, .5} {
		cb := v4Model()
		cb.Vars[0].Rec.Set("RECODE_STATUS", "확인 필요")
		c := newChecker(cb, kosac.RefData{PendingRecodeLimit: limit})
		c.v4recodeStatus()
		reject := slices.ContainsFunc(c.issues, func(i kosac.Issue) bool { return i.Level == kosac.R })
		if reject != (limit == 0) {
			t.Fatal(limit, c.issues)
		}
	}
}
func TestV4CodeNormalizationDoesNotRewrite(t *testing.T) {
	for _, typ := range []string{"문자", "숫자"} {
		cb := v4Model()
		cb.Vars[0].Rec.Set("DATA_TYPE", typ)
		r := kosac.Record{}
		r.Set("FILE_NO", "1")
		r.Set("VAR_NAME", "q")
		r.Set("CODE", "1")
		cb.Codes = append(cb.Codes, &kosac.Code{Rec: r})
		c := newChecker(cb, kosac.RefData{})
		c.v4codes()
		if (len(c.issues) > 0) != (typ == "숫자") {
			t.Fatal(typ, c.issues)
		}
		if cb.Codes[0].Rec.Get("CODE") != "01" {
			t.Fatal("source code rewritten")
		}
	}
}
func TestV4StatusDomainAndAccess(t *testing.T) {
	cb := v4Model()
	cb.Vars[0].Rec.Set("MISSING_STATUS_VAR", "status")
	r := kosac.Record{}
	r.Set("FILE_NO", "1")
	r.Set("VAR_NAME", "status")
	r.Set("MEAS_LEVEL", "명목")
	cb.Vars = append(cb.Vars, &kosac.Var{Rec: r})
	cd := kosac.Record{}
	cd.Set("FILE_NO", "1")
	cd.Set("VAR_NAME", "status")
	cd.Set("CODE", "9")
	cb.Codes = append(cb.Codes, &kosac.Code{Rec: cd})
	c := newChecker(cb, kosac.RefData{})
	c.v4statusVars()
	if !slices.ContainsFunc(c.issues, func(i kosac.Issue) bool { return i.Rule == "C20" && i.Level == kosac.R }) {
		t.Fatal(c.issues)
	}
	cb = v4Model()
	cb.Vars[0].Rec.Set("ACCESS_LEVEL", "공개 가능")
	cb.Vars[0].Rec.Set("PII", "Y")
	c = newChecker(cb, kosac.RefData{})
	c.v4access()
	if len(c.issues) != 1 || c.issues[0].Rule != "C21" || c.issues[0].Level != kosac.W {
		t.Fatal(c.issues)
	}
}

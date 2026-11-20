package check

import (
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/kosac"
)

// rec423 builds a record with every v4.2.3 KEY of the sheet, as Read does.
func rec423(sheet string, kv ...string) kosac.Record {
	l := kosac.LayoutFor(kosac.Profile423)
	keys := map[string][]string{kosac.SheetStudy: l.Study, kosac.SheetFiles: l.Files, kosac.SheetVars: l.Vars, kosac.SheetCodes: l.Codes}[sheet]
	r := kosac.Record{}
	for _, k := range keys {
		r[k] = kosac.Field{Loc: kosac.Loc{Sheet: sheet, Cell: k}}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i], kv[i+1])
	}
	return r
}

func var423(no, name string, kv ...string) *kosac.Var {
	base := []string{"FILE_NO", no, "VAR_NAME", name, "VAR_LABEL", name + " 라벨", "Q_TEXT", name + " 문항", "MEAS_LEVEL", "해당 없음", "DATA_TYPE", "문자", "PII", "N", "SENSITIVE", "N"}
	return &kosac.Var{Rec: rec423(kosac.SheetVars, append(base, kv...)...)}
}

func code423(no, name, code, label string) *kosac.Code {
	return &kosac.Code{Rec: rec423(kosac.SheetCodes, "FILE_NO", no, "VAR_NAME", name, "CODE", code, "CODE_LABEL", label)}
}

// model423 is a synthetic two-wave codebook that passes every S rule.
func model423() *kosac.Codebook {
	study := []string{"RULE_VERSION", "20261105-r3", "TITLE", "예시 조사", "PURPOSE", "목적", "SUMMARY", "내용", "FIELD_ORG", "기관",
		"FIELD_START", "2026-08-01", "FIELD_END", "2026-08-31", "KEYWORDS", "가;나;다", "QUESTIONNAIRE", "q.pdf", "REPORT_FILE", "r.pdf",
		"UNIVERSE", "대상", "POP_SOURCE", "자료", "ANALYSIS_UNIT", "개인", "SAMPLING_TYPE", "편의(비확률)", "SAMPLING_DESC", "설명",
		"TARGET_N", "100", "VALID_N", "사전 100, 사후 90", "SPLIT_BALLOT", "N", "MODE", "온라인(웹·모바일)", "EXCLUDE_N", "0",
		"WEIGHT_YN", "N", "SURVEY_TYPE", "연구조사", "ACCESS_RESTRICTIONS", "내부 이용", "SOURCE_FILES", "raw.csv"}
	files := []*kosac.File{
		{Rec: rec423(kosac.SheetFiles, "FILE_NO", "1", "FILE_NAME", "pre.csv", "SURVEY_TIME", "사전", "UNIQUE_KEY", "PID")},
		{Rec: rec423(kosac.SheetFiles, "FILE_NO", "2", "FILE_NAME", "post.csv", "SURVEY_TIME", "사후", "UNIQUE_KEY", "PID",
			"LINK_FILE_NO", "1", "LINK_VAR", "PID", "LINK_TARGET_VAR", "PID")},
	}
	var vars []*kosac.Var
	var codes []*kosac.Code
	for _, no := range []string{"1", "2"} {
		vars = append(vars,
			var423(no, "PID", "KEY_ROLE", "참여번호"),
			var423(no, "SEX", "VAR_LABEL", "성별", "MEAS_LEVEL", "명목", "DATA_TYPE", "문자", "STD_ID", "PERSON.GENDER"),
			var423(no, "LVL", "VAR_LABEL", "학교급", "MEAS_LEVEL", "명목", "DATA_TYPE", "문자", "STD_ID", "SCHOOL.LEVEL"),
			var423(no, "Q1", "MEAS_LEVEL", "서열", "DATA_TYPE", "숫자", "ITEM_KEY", "SAT", "SCALE", "만족>전반", "UNIVERSE_EXPR", "LVL in ('02', '03') AND SEX = 1"),
		)
		codes = append(codes,
			code423(no, "SEX", "1", "남"), code423(no, "SEX", "2", "여"), code423(no, "SEX", "-9", "무응답"),
			code423(no, "LVL", "02", "초등학교"), code423(no, "LVL", "03", "중학교"),
			code423(no, "Q1", "1", "불만"), code423(no, "Q1", "2", "만족"), code423(no, "Q1", "-8", "비해당"),
		)
	}
	return &kosac.Codebook{Template: kosac.TemplateMeta{Version: kosac.Field{Value: "4.2.3"}},
		Study: rec423(kosac.SheetStudy, study...), Files: files, Vars: vars, Codes: codes}
}

func TestV423Valid(t *testing.T) {
	if is := Check(model423(), kosac.RefData{}); len(is) > 0 {
		t.Fatal(is)
	}
}

func TestV423SchoolIdentifierLevel(t *testing.T) {
	for _, id := range []string{"SCHOOL.ID", "SCHOOL.KEDI_ID"} {
		t.Run(id, func(t *testing.T) {
			cb := model423()
			cb.Vars[0].Rec.Set("STD_ID", id)
			cb.Vars[0].Rec.Set("CODE_SYSTEM", "2025 교육통계")
			for level, want := range map[string]kosac.Level{"해당 없음": kosac.I, "서열": kosac.R} {
				cb.Vars[0].Rec.Set("MEAS_LEVEL", level)
				var got []kosac.Level
				for _, issue := range Check(cb, kosac.RefData{}) {
					if issue.Rule == "S11" && issue.Cell == "MEAS_LEVEL" {
						got = append(got, issue.Level)
					}
				}
				if !slices.Equal(got, []kosac.Level{want}) {
					t.Fatalf("%s: got %v, want %s", level, got, want)
				}
			}
		})
	}
}

// The trimmed version selects the profile; no C rule runs on it.
func TestV423Dispatch(t *testing.T) {
	cb := model423()
	cb.Template.Version.Value = " 4.2.3 "
	cb.Study.Set("TITLE", "")
	is := Check(cb, kosac.RefData{})
	if !slices.ContainsFunc(is, func(i kosac.Issue) bool { return i.Rule == "S01" && i.Level == kosac.R }) {
		t.Fatal(is)
	}
	if slices.ContainsFunc(is, func(i kosac.Issue) bool { return strings.HasPrefix(i.Rule, "C") }) {
		t.Fatal(is)
	}
	// Older versions keep the C rules and never get S rules.
	old := v4Model()
	for _, i := range Check(old, kosac.RefData{}) {
		if strings.HasPrefix(i.Rule, "S") {
			t.Fatal(i)
		}
	}
}

func TestV423Rules(t *testing.T) {
	R, W := kosac.R, kosac.W
	for _, tc := range []struct {
		name   string
		modify func(*kosac.Codebook)
		rule   string
		level  kosac.Level
		cell   string
	}{
		{"required blank", func(cb *kosac.Codebook) { cb.Study.Set("SOURCE_FILES", "") }, "S01", R, "SOURCE_FILES"},
		{"required with reason", func(cb *kosac.Codebook) {
			cb.Study.Set("SOURCE_FILES", "")
			cb.Supplemental = map[string][][]string{Sheet423Unconfirmed: {{"KEY", "사유"}, {"SOURCE_FILES", "원본 목록 미수령"}}}
		}, "S01", R, "SOURCE_FILES"},
		{"conditional weight", func(cb *kosac.Codebook) { cb.Study.Set("WEIGHT_YN", "Y") }, "S01", R, "WEIGHT_VAR"},
		{"date format", func(cb *kosac.Codebook) { cb.Study.Set("FIELD_END", "2026/08/31") }, "S02", R, "FIELD_END"},
		{"date order", func(cb *kosac.Codebook) { cb.Study.Set("FIELD_END", "2026-07-01") }, "S02", R, "FIELD_END"},
		{"file id form", func(cb *kosac.Codebook) { cb.Files[1].Rec.Set("FILE_NO", "A") }, "S03", R, "FILE_NO"},
		{"file id collision", func(cb *kosac.Codebook) { cb.Files[1].Rec.Set("FILE_NO", "01") }, "S03", R, "FILE_NO"},
		{"var key", func(cb *kosac.Codebook) { cb.Vars[1].Rec.Set("VAR_NAME", "PID") }, "S04", R, "VAR_NAME"},
		{"var name form", func(cb *kosac.Codebook) { cb.Vars[3].Rec.Set("VAR_NAME", "1Q") }, "S05", R, "VAR_NAME"},
		{"var name case", func(cb *kosac.Codebook) { cb.Vars[3].Rec.Set("VAR_NAME", "pid") }, "S05", R, "VAR_NAME"},
		{"label duplicate", func(cb *kosac.Codebook) { cb.Vars[3].Rec.Set("VAR_LABEL", "성별") }, "S06", W, "VAR_LABEL"},
		{"code var", func(cb *kosac.Codebook) { cb.Codes[0].Rec.Set("VAR_NAME", "NOPE") }, "S07", R, "VAR_NAME"},
		{"nominal codes", func(cb *kosac.Codebook) { cb.Vars[0].Rec.Set("MEAS_LEVEL", "명목") }, "S07", R, "MEAS_LEVEL"},
		{"missing meaning", func(cb *kosac.Codebook) { cb.Codes[2].Rec.Set("CODE_LABEL", "모름") }, "S08", R, "CODE_LABEL"},
		{"unconfirmed label", func(cb *kosac.Codebook) { cb.Codes[6].Rec.Set("CODE_LABEL", "미확인") }, "S08", R, "CODE_LABEL"},
		{"report unconfirmed", func(cb *kosac.Codebook) { cb.Study.Set("TOOL", "미확인: 플랫폼명 확인 중") }, "S09", kosac.I, "TOOL"},
		{"report list", func(cb *kosac.Codebook) { cb.Study.Set("MODE", "온라인") }, "S09", kosac.I, "MODE"},
		{"report sheet", func(cb *kosac.Codebook) {
			cb.Supplemental = map[string][][]string{Sheet423Unconfirmed: {{"KEY", "사유"}, {"TOOL", "확인 중"}}}
		}, "S09", kosac.I, "A2"},
		{"std id", func(cb *kosac.Codebook) { cb.Vars[1].Rec.Set("STD_ID", "BIZ.ONLY") }, "S10", R, "STD_ID"},
		{"std level", func(cb *kosac.Codebook) { cb.Vars[1].Rec.Set("MEAS_LEVEL", "서열") }, "S11", R, "MEAS_LEVEL"},
		{"std type", func(cb *kosac.Codebook) { cb.Vars[2].Rec.Set("DATA_TYPE", "숫자") }, "S11", R, "DATA_TYPE"},
		{"std code", func(cb *kosac.Codebook) { cb.Codes[1].Rec.Set("CODE", "5") }, "S11", R, "CODE"},
		{"leading zero", func(cb *kosac.Codebook) { cb.Codes[3].Rec.Set("CODE", "2") }, "S11", R, "CODE"},
		{"reversed label", func(cb *kosac.Codebook) { cb.Codes[0].Rec.Set("CODE_LABEL", "여") }, "S11", R, "CODE_LABEL"},
		{"sido period", func(cb *kosac.Codebook) {
			cb.Vars[2].Rec.Set("STD_ID", "REGION.SIDO")
			cb.Codes[3].Rec.Set("CODE", "29")
			cb.Codes[3].Rec.Set("CODE_LABEL", "광주광역시")
			cb.Codes[4].Rec.Set("CODE", "11")
			cb.Codes[4].Rec.Set("CODE_LABEL", "서울특별시")
		}, "S11", R, "CODE"},
		{"std recommend", func(cb *kosac.Codebook) { cb.Vars[1].Rec.Set("STD_ID", "") }, "S12", W, "STD_ID"},
		{"part number", func(cb *kosac.Codebook) { cb.Vars[0].Rec.Set("KEY_ROLE", "") }, "S13", R, "FILE_NO"},
		{"wave", func(cb *kosac.Codebook) { cb.Files[1].Rec.Set("SURVEY_TIME", "") }, "S13", R, "TIME_VAR"},
		{"row key", func(cb *kosac.Codebook) { cb.Files[0].Rec.Set("UNIQUE_KEY", "") }, "S14", R, "UNIQUE_KEY"},
		{"key var", func(cb *kosac.Codebook) { cb.Files[0].Rec.Set("UNIQUE_KEY", "PID WAVE") }, "S14", R, "UNIQUE_KEY"},
		{"link incomplete", func(cb *kosac.Codebook) { cb.Files[1].Rec.Set("LINK_TARGET_VAR", "") }, "S14", R, "LINK_TARGET_VAR"},
		{"link count", func(cb *kosac.Codebook) { cb.Files[1].Rec.Set("LINK_TARGET_VAR", "PID SEX") }, "S14", R, "LINK_TARGET_VAR"},
		{"link target", func(cb *kosac.Codebook) { cb.Files[1].Rec.Set("LINK_TARGET_VAR", "NOPE") }, "S14", R, "LINK_TARGET_VAR"},
		{"universe grammar", func(cb *kosac.Codebook) { cb.Vars[3].Rec.Set("UNIVERSE_EXPR", "LVL in ('02'") }, "S15", R, "UNIVERSE_EXPR"},
		{"universe var", func(cb *kosac.Codebook) { cb.Vars[3].Rec.Set("UNIVERSE_EXPR", "NOPE = 1 OR SEX != 2") }, "S15", R, "UNIVERSE_EXPR"},
		{"item key codes", func(cb *kosac.Codebook) { cb.Codes[13].Rec.Set("CODE_LABEL", "매우 만족") }, "S16", W, "ITEM_KEY"},
		{"scale codes", func(cb *kosac.Codebook) { cb.Codes[13].Rec.Set("CODE_LABEL", "매우 만족") }, "S17", W, "SCALE"},
		{"scale level", func(cb *kosac.Codebook) { cb.Vars[7].Rec.Set("MEAS_LEVEL", "명목") }, "S17", W, "MEAS_LEVEL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cb := model423()
			tc.modify(cb)
			is := Check(cb, kosac.RefData{})
			if !slices.ContainsFunc(is, func(i kosac.Issue) bool {
				return i.Rule == tc.rule && i.Level == tc.level && i.Cell == tc.cell
			}) {
				t.Fatalf("want %s %s at %s, got %v", tc.rule, tc.level, tc.cell, is)
			}
		})
	}
}

// A conditional key is not required while its condition does not hold, and
// auto or foundation keys are never required.
func TestV423NotBulkRequired(t *testing.T) {
	cb := model423()
	for _, k := range []string{"RULE_VERSION", "FUNDER", "DATA_NO", "STUDY_ID", "PI_ORG", "SERIES_NAME", "STRATA_VARS", "CLUSTER_VAR"} {
		cb.Study.Set(k, "")
	}
	for _, i := range Check(cb, kosac.RefData{}) {
		if i.Rule == "S01" {
			t.Fatal(i)
		}
	}
}

// Negative valid codes are allowed when their label is not a reserved code's.
func TestV423NegativeValid(t *testing.T) {
	cb := model423()
	cb.Codes[5].Rec.Set("CODE", "-1")
	for _, i := range Check(cb, kosac.RefData{}) {
		if i.Rule == "S08" {
			t.Fatal(i)
		}
	}
}

func TestV423UniverseGrammar(t *testing.T) {
	for _, tc := range []struct {
		s  string
		ok bool
	}{
		{"A = 1", true},
		{"A != 1 AND B in (1, 2, 3)", true},
		{"(A = 1 OR A = 2) AND B = 'x'", true},
		{`A in ("01")`, true},
		{"A", false},
		{"A = ", false},
		{"A = 1 AND", false},
		{"A in 1, 2", false},
		{"A = 1 B = 2", false},
		{"(A = 1", false},
	} {
		if _, err := parseUniverse423(tc.s); (err == nil) != tc.ok {
			t.Errorf("%q: %v", tc.s, err)
		}
	}
}

func TestProfile423SchoolCodeEdition(t *testing.T) {
	cb := model423()
	v := var423("1", "SCHOOL", "STD_ID", "SCHOOL.ID", "MEAS_LEVEL", "명목", "DATA_TYPE", "문자")
	cb.Vars = append(cb.Vars, v)
	c := code423("1", "SCHOOL", "C150029", "학교")
	cb.Codes = append(cb.Codes, c)
	cb.Study.Set("SCHOOL_CODE_YEAR", "2026")
	check := func(want bool) {
		t.Helper()
		bad := false
		for _, i := range Check(cb, kosac.RefData{}) {
			if i.Rule == "S11" && i.VarName == "SCHOOL" && i.Code == c.Rec.Get("CODE") {
				bad = true
			}
		}
		if bad != want {
			t.Fatalf("school code %s, rejection %t", c.Rec.Get("CODE"), bad)
		}
	}
	check(false)
	c.Rec.Set("CODE", "XXXXXXX")
	check(true)
	cb.Study.Set("SCHOOL_CODE_YEAR", "2022")
	check(false)
}

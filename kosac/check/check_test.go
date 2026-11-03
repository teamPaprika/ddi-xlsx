package check

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/template"
	"github.com/teamPaprika/ddi-xlsx/kosac/universe"
)

var update = flag.Bool("update", false, "rewrite testdata/kosac/v32_sample_2024.issues.json")

// rec builds a record with every KEY of the sheet, as Read does. Loc.Cell is
// the KEY itself, so an issue says which cell it points at without a workbook.
func rec(sheet string, kv ...string) kosac.Record {
	keys := map[string][]string{kosac.SheetStudy: kosac.StudyKeys, kosac.SheetFiles: kosac.FileKeys, kosac.SheetVars: kosac.VarKeys, kosac.SheetCodes: kosac.CodeKeys}
	r := kosac.Record{}
	for _, k := range keys[sheet] {
		r[k] = kosac.Field{Loc: kosac.Loc{Sheet: sheet, Cell: k}}
	}
	for i := 0; i < len(kv); i += 2 {
		r.Set(kv[i], kv[i+1])
	}
	return r
}

// v builds a 03 row of file 1 that passes every rule on its own; kv overrides.
func v(name string, kv ...string) *kosac.Var {
	base := []string{"FILE_NO", "1", "VAR_NAME", name, "VAR_LABEL", name + " 라벨", "Q_TEXT", name + " 문항",
		"VAR_GROUP", "영역", "MEAS_LEVEL", "연속", "PII", "N", "SENSITIVE", "N"}
	return &kosac.Var{Rec: rec(kosac.SheetVars, append(base, kv...)...)}
}

// codes builds 04 rows of file 1 for name.
func codes(name string, cs ...string) []*kosac.Code {
	var out []*kosac.Code
	for _, c := range cs {
		out = append(out, &kosac.Code{Rec: rec(kosac.SheetCodes, "FILE_NO", "1", "VAR_NAME", name, "CODE", c, "CODE_LABEL", "보기 "+c)})
	}
	return out
}

func file(no string, kv ...string) *kosac.File {
	base := []string{"FILE_NO", no, "FILE_NAME", "data" + no + ".csv", "FILE_FORMAT", "CSV", "ID_VAR", "ID"}
	return &kosac.File{Rec: rec(kosac.SheetFiles, append(base, kv...)...)}
}

// base is a codebook with no issues.
func base() *kosac.Codebook {
	kv := []string{
		"RULE_VERSION", "v1.0", "TITLE", "조사", "PURPOSE", "목적", "SUMMARY", "요약", "PI_ORG", "기관",
		"FIELD_ORG", "조사기관", "FIELD_START", "2024-10-04", "FIELD_END", "2024-10-28", "KEYWORDS", "과학",
		"QUESTIONNAIRE", "q.pdf", "REPORT_FILE", "r.pdf", "UNIVERSE", "국민", "POP_SOURCE", "주민등록",
		"ANALYSIS_UNIT", "개인", "SAMPLING_TYPE", "편의(비확률)", "SAMPLING_DESC", "편의", "TARGET_N", "100",
		"VALID_N", "100", "SPLIT_BALLOT", "N", "MODE", "온라인", "EXCLUDE_N", "0", "EDIT_METHOD", "검토",
		"WEIGHT_YN", "N", "SURVEY_TYPE", "연구조사",
	}
	return &kosac.Codebook{Study: rec(kosac.SheetStudy, kv...), Files: []*kosac.File{file("1")}, Vars: []*kosac.Var{v("ID")}}
}

// sum is an issue as "Level Cell VarName Code Actual", blank parts left out.
func sum(i kosac.Issue) string {
	var parts []string
	for _, p := range []string{string(i.Level), i.Cell, i.VarName, i.Code, i.Actual} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " ")
}

func of(rule string, is []kosac.Issue) []string {
	out := []string{}
	for _, i := range is {
		if i.Rule == rule {
			out = append(out, sum(i))
		}
	}
	return out
}

func set(cb *kosac.Codebook, kv ...string) {
	for i := 0; i < len(kv); i += 2 {
		cb.Study.Set(kv[i], kv[i+1])
	}
}

func parseErr(s string) string {
	_, err := universe.Parse(s)
	return err.Error()
}

func TestBaseHasNoIssues(t *testing.T) {
	if is := Check(base(), kosac.RefData{}); len(is) != 0 {
		t.Errorf("issues on the base codebook: %+v", is)
	}
}

func TestRules(t *testing.T) {
	type tc struct {
		rule, name string
		mod        func(cb *kosac.Codebook)
		ref        kosac.RefData
		want       []string
	}
	addVars := func(vs ...*kosac.Var) func(cb *kosac.Codebook) {
		return func(cb *kosac.Codebook) { cb.Vars = append(cb.Vars, vs...) }
	}
	long := strings.Repeat("가", 31)
	cases := []tc{
		{rule: "C01", name: "required empty", mod: func(cb *kosac.Codebook) { set(cb, "TITLE", " ") }, want: []string{"R TITLE"}},
		{rule: "C01", name: "STRATA_VARS on 할당(비확률)", mod: func(cb *kosac.Codebook) { set(cb, "SAMPLING_TYPE", "할당(비확률)") }, want: []string{"R STRATA_VARS"}},
		{rule: "C01", name: "STRATA_VARS given", mod: func(cb *kosac.Codebook) {
			set(cb, "SAMPLING_TYPE", "할당(비확률)", "STRATA_VARS", "ID")
		}, want: []string{}},
		{rule: "C01", name: "확률-집락", mod: func(cb *kosac.Codebook) { set(cb, "SAMPLING_TYPE", "확률-집락") },
			want: []string{"R CLUSTER_VAR", "R CONTACT_N", "R RESPONSE_RATE"}},
		{rule: "C01", name: "RR_FORMULA after RESPONSE_RATE", mod: func(cb *kosac.Codebook) { set(cb, "RESPONSE_RATE", "12.5") }, want: []string{"R RR_FORMULA"}},
		{rule: "C01", name: "unverified optional TOOL", mod: func(cb *kosac.Codebook) { set(cb, "TOOL", "확인 불가: 보고서 미기재") },
			want: []string{"W TOOL 확인 불가: 보고서 미기재"}},
		{rule: "C01", name: "unverified EXCLUDE_N requires no EXCLUDE_RULE", mod: func(cb *kosac.Codebook) { set(cb, "EXCLUDE_N", "확인 불가: x") },
			want: []string{"W EXCLUDE_N 확인 불가: x"}},
		{rule: "C01", name: "EXCLUDE_N ≥ 1", mod: func(cb *kosac.Codebook) { set(cb, "EXCLUDE_N", "1,200") }, want: []string{"R EXCLUDE_RULE"}},
		{rule: "C01", name: "SCHOOL.ID var", mod: addVars(v("SCH", "STD_ID", "SCHOOL.ID", "MEAS_LEVEL", "문자")), want: []string{"R SCHOOL_CODE_YEAR"}},
		{rule: "C01", name: "사업성과", mod: func(cb *kosac.Codebook) { set(cb, "SURVEY_TYPE", "사업성과") }, want: []string{"R BIZ_NAME", "R BIZ_YEAR"}},

		{rule: "C02", name: "thousands separator", mod: func(cb *kosac.Codebook) { set(cb, "TARGET_N", "10,000", "CONTACT_N", "1234567") }, want: []string{}},
		{rule: "C02", name: "bad integers", mod: func(cb *kosac.Codebook) { set(cb, "VALID_N", "1.5", "TARGET_N", "10,00") },
			want: []string{"R TARGET_N 10,00", "R VALID_N 1.5"}},
		{rule: "C02", name: "not a real date", mod: func(cb *kosac.Codebook) { set(cb, "FIELD_START", "2024-02-30") }, want: []string{"R FIELD_START 2024-02-30"}},
		{rule: "C02", name: "end before start", mod: func(cb *kosac.Codebook) { set(cb, "FIELD_END", "2024-10-01") }, want: []string{"R FIELD_END 2024-10-01"}},
		{rule: "C02", name: "DATA_NO passes", mod: func(cb *kosac.Codebook) { set(cb, "DATA_NO", "문화-01-01") }, want: []string{}},
		{rule: "C02", name: "DATA_NO fails", mod: func(cb *kosac.Codebook) { set(cb, "DATA_NO", "문화 01/01") }, want: []string{"R DATA_NO 문화 01/01"}},
		{rule: "C02", name: "lists", mod: func(cb *kosac.Codebook) {
			set(cb, "MODE", "웹", "SAMPLING_TYPE", "할당(비확률) ", "RULE_VERSION", "v1.1")
		},
			want: []string{"R RULE_VERSION v1.1", "R SAMPLING_TYPE 할당(비확률)", "R MODE 웹"}},
		{rule: "C02", name: "rate and years", mod: func(cb *kosac.Codebook) {
			set(cb, "RESPONSE_RATE", "45%", "BIZ_YEAR", "24", "SCHOOL_CODE_YEAR", "2024")
		}, want: []string{"R RESPONSE_RATE 45%", "R BIZ_YEAR 24"}},

		{rule: "C03", name: "weighted", mod: func(cb *kosac.Codebook) {
			set(cb, "WEIGHT_YN", "Y", "WEIGHT_VAR", "WT", "WEIGHT_METHOD", "사후층화")
			cb.Vars = append(cb.Vars, v("WT"))
		}, want: []string{}},
		{rule: "C03", name: "empty WEIGHT_METHOD", mod: func(cb *kosac.Codebook) {
			set(cb, "WEIGHT_YN", "Y", "WEIGHT_VAR", "WT")
			cb.Vars = append(cb.Vars, v("WT"))
		}, want: []string{"R WEIGHT_METHOD"}},
		{rule: "C03", name: "weight var not 연속", mod: func(cb *kosac.Codebook) {
			set(cb, "WEIGHT_YN", "Y", "WEIGHT_VAR", "WT", "WEIGHT_METHOD", "m")
			cb.Vars = append(cb.Vars, v("WT", "MEAS_LEVEL", "명목"))
		}, want: []string{"R WEIGHT_VAR WT"}},
		{rule: "C03", name: "weight var missing", mod: func(cb *kosac.Codebook) { set(cb, "WEIGHT_YN", "Y", "WEIGHT_VAR", "WT", "WEIGHT_METHOD", "m") },
			want: []string{"R WEIGHT_VAR WT"}},
		{rule: "C03", name: "N with a weight var", mod: func(cb *kosac.Codebook) { set(cb, "WEIGHT_VAR", "WT") }, want: []string{"R WEIGHT_VAR WT"}},
		// Assembly resolves WEIGHT_VAR whatever WEIGHT_YN says.
		{rule: "C03", name: "unverified WEIGHT_YN", mod: func(cb *kosac.Codebook) { set(cb, "WEIGHT_YN", "확인 불가: x", "WEIGHT_VAR", "WT") },
			want: []string{"R WEIGHT_VAR WT"}},

		{rule: "C04", name: "split var", mod: func(cb *kosac.Codebook) {
			set(cb, "SPLIT_BALLOT", "Y", "SPLIT_VAR", "Random_Ballot")
			cb.Vars = append(cb.Vars, v("Random_Ballot", "MEAS_LEVEL", "명목"))
		}, want: []string{}},
		{rule: "C04", name: "split var not 명목", mod: func(cb *kosac.Codebook) {
			set(cb, "SPLIT_BALLOT", "Y", "SPLIT_VAR", "Random_Ballot")
			cb.Vars = append(cb.Vars, v("Random_Ballot", "MEAS_LEVEL", "서열"))
		}, want: []string{"R SPLIT_VAR Random_Ballot"}},
		{rule: "C04", name: "split var empty", mod: func(cb *kosac.Codebook) { set(cb, "SPLIT_BALLOT", "Y") }, want: []string{"R SPLIT_VAR"}},
		{rule: "C04", name: "strata and cluster vars", mod: func(cb *kosac.Codebook) {
			set(cb, "STRATA_VARS", "SQ1; SQX,ID", "CLUSTER_VAR", "PSU")
			cb.Vars = append(cb.Vars, v("SQ1"))
		}, want: []string{"R STRATA_VARS SQX", "R CLUSTER_VAR PSU"}},
		// Assembly resolves SPLIT_VAR whatever SPLIT_BALLOT says.
		{rule: "C04", name: "stale split var", mod: func(cb *kosac.Codebook) { set(cb, "SPLIT_VAR", "NOPE") }, want: []string{"R SPLIT_VAR NOPE"}},
		{rule: "C04", name: "split var under N", mod: func(cb *kosac.Codebook) { set(cb, "SPLIT_VAR", "ID") }, want: []string{}},
		// kosac:sampling@clusterVar holds one var.
		{rule: "C04", name: "one cluster var", mod: func(cb *kosac.Codebook) {
			set(cb, "CLUSTER_VAR", "ID SQ1")
			cb.Vars = append(cb.Vars, v("SQ1"))
		}, want: []string{"R CLUSTER_VAR ID SQ1"}},

		{rule: "C05", name: "undeclared system ID_VAR", mod: func(cb *kosac.Codebook) { cb.Files[0].Rec.Set("ID_VAR", "walla_response_id") }, want: []string{"R ID_VAR walla_response_id"}},
		{rule: "C05", name: "declared system ID_VAR", mod: func(cb *kosac.Codebook) {
			cb.Files[0].Rec.Set("ID_VAR", "walla_response_id")
			cb.Vars = append(cb.Vars, v("walla_response_id"))
		}, want: []string{}},
		{rule: "C05", name: "ID_VAR missing", mod: func(cb *kosac.Codebook) { cb.Files[0].Rec.Set("ID_VAR", "NOPE") }, want: []string{"R ID_VAR NOPE"}},
		{rule: "C05", name: "FILE_NO gap and format", mod: func(cb *kosac.Codebook) {
			cb.Files = append(cb.Files, file("3", "FILE_FORMAT", "SAV", "ID_VAR", "ID"))
			cb.Vars = append(cb.Vars, v("ID", "FILE_NO", "3"))
		}, want: []string{"R FILE_NO 3", "R FILE_FORMAT SAV"}},
		{rule: "C05", name: "link", mod: func(cb *kosac.Codebook) {
			cb.Files = append(cb.Files, file("2", "ID_VAR", "walla_response_id", "LINK_VAR", "PID", "LINK_FILE_NO", "1"))
			cb.Vars = append(cb.Vars, v("PID", "FILE_NO", "2"), v("walla_response_id", "FILE_NO", "2"))
		}, want: []string{"R LINK_VAR PID"}},
		{rule: "C05", name: "link to an unknown file", mod: func(cb *kosac.Codebook) {
			cb.Files[0].Rec.Set("LINK_VAR", "ID")
			cb.Files[0].Rec.Set("LINK_FILE_NO", "7")
		}, want: []string{"R LINK_FILE_NO 7"}},
		{rule: "C05", name: "LINK_FILE_NO without LINK_VAR", mod: func(cb *kosac.Codebook) { cb.Files[0].Rec.Set("LINK_FILE_NO", "7") },
			want: []string{"R LINK_FILE_NO 7"}},
		// Assembly takes one spelling of a FILE_NO; a "확인 불가" one is none.
		{rule: "C05", name: "FILE_NO spelling", mod: func(cb *kosac.Codebook) {
			cb.Files = append(cb.Files, file("02", "ID_VAR", "walla_response_id"), file("+3", "ID_VAR", "walla_response_id"),
				file("확인 불가: x", "ID_VAR", "walla_response_id"))
			for _, no := range []string{"02", "+3", "확인 불가: x"} {
				cb.Vars = append(cb.Vars, v("walla_response_id", "FILE_NO", no))
			}
		}, want: []string{"R FILE_NO 02", "R FILE_NO +3", "R FILE_NO 확인 불가: x", "W FILE_NO 확인 불가: x"}},
		{rule: "C05", name: "uploaded files", mod: func(cb *kosac.Codebook) {
			cb.Files = append(cb.Files, file("2", "FILE_NAME", "data1.csv", "ID_VAR", "ID"))
			cb.Vars = append(cb.Vars, v("ID", "FILE_NO", "2"))
		}, ref: kosac.RefData{UploadedFiles: []string{"data1.csv"}}, want: []string{"R FILE_NAME data1.csv"}},
		{rule: "C05", name: "file not uploaded", ref: kosac.RefData{UploadedFiles: []string{"other.csv"}}, want: []string{"R FILE_NAME data1.csv"}},

		{rule: "C06", name: "duplicate, unknown file, required", mod: addVars(v("ID"), v("Q1", "FILE_NO", "9"), v("Q2", "Q_TEXT", "", "PII", "")),
			want: []string{"R VAR_NAME ID ID", "R FILE_NO Q1 9", "R Q_TEXT Q2", "R PII Q2"}},
		{rule: "C06", name: "unverified FILE_NO", mod: addVars(v("Q1", "FILE_NO", "확인 불가: x")),
			want: []string{"R FILE_NO Q1 확인 불가: x", "W FILE_NO Q1 확인 불가: x"}},
		// The XSD enumerates Y and N exactly, so a space is not trimmed away.
		{rule: "C06", name: "PII and SENSITIVE are Y or N", mod: addVars(v("Q1", "PII", "예"), v("Q2", "PII", "Y "), v("Q3", "SENSITIVE", "y")),
			want: []string{"R PII Q1 예", "R PII Q2 Y", "R SENSITIVE Q3 y"}},

		{rule: "C05", name: "VALID_N is an integer", mod: func(cb *kosac.Codebook) {
			cb.Files[0].Rec.Set("VALID_N", "1,000")
			cb.Files = append(cb.Files, file("2", "ID_VAR", "ID", "VALID_N", "약 500"), file("3", "ID_VAR", "ID", "VALID_N", "1.5"))
			cb.Vars = append(cb.Vars, v("ID", "FILE_NO", "2"), v("ID", "FILE_NO", "3"))
		}, want: []string{"R VALID_N 약 500", "R VALID_N 1.5"}},
		{rule: "C05", name: "UNIT list", mod: func(cb *kosac.Codebook) {
			cb.Files[0].Rec.Set("UNIT", "학교")
			cb.Files = append(cb.Files, file("2", "ID_VAR", "ID", "UNIT", "개인 "), file("3", "ID_VAR", "ID", "UNIT", "가구"))
			cb.Vars = append(cb.Vars, v("ID", "FILE_NO", "2"), v("ID", "FILE_NO", "3"))
		}, want: []string{"R UNIT 개인", "R UNIT 가구"}},

		{rule: "C07", name: "valid codes only", mod: func(cb *kosac.Codebook) {
			cb.Vars = append(cb.Vars, v("Q1", "MEAS_LEVEL", "명목"), v("Q2", "MEAS_LEVEL", "서열"))
			cb.Codes = append(cb.Codes, codes("Q1", "1", "-8", "-9")...)
			cb.Codes = append(cb.Codes, codes("Q2", "1", "2", "-7")...)
		}, want: []string{"R VAR_NAME Q1 1"}},
		{rule: "C07", name: "unknown level", mod: addVars(v("Q1", "MEAS_LEVEL", "범주")), want: []string{"R MEAS_LEVEL Q1 범주"}},

		{rule: "C08", name: "one issue per condition", mod: addVars(v("Q1", "VAR_LABEL", long, "Q_TEXT", long)),
			want: []string{"W VAR_LABEL Q1 31", "W VAR_LABEL Q1 " + long}},
		{rule: "C08", name: "english word or name", mod: addVars(v("Q1", "VAR_LABEL", "gender"), v("Q2", "VAR_LABEL", "Q2")),
			want: []string{"W VAR_LABEL Q1 gender", "W VAR_LABEL Q2 Q2"}},
		{rule: "C08", name: "30 runes pass", mod: addVars(v("Q1", "VAR_LABEL", strings.Repeat("가", 30))), want: []string{}},

		{rule: "C09", name: "question numbers", mod: addVars(v("Q1", "VAR_GROUP", "Q 3"), v("Q2", "VAR_GROUP", "문항3"), v("Q3", "VAR_GROUP", "Q 3")),
			want: []string{"R VAR_GROUP Q1 Q 3", "R VAR_GROUP Q2 문항3"}},
		{rule: "C09", name: "length", mod: addVars(v("Q1", "VAR_GROUP", strings.Repeat("가", 21)), v("Q2", "VAR_GROUP", strings.Repeat("가", 20))),
			want: []string{"R VAR_GROUP Q1 21"}},

		{rule: "C10", name: "values", mod: func(cb *kosac.Codebook) {
			cb.Vars = append(cb.Vars, v("Q5", "MEAS_LEVEL", "명목"), v("AGE"),
				v("Q6", "RESP_UNIVERSE", "Q5=1,2"), v("Q7", "RESP_UNIVERSE", "Q5=9"),
				v("Q8", "RESP_UNIVERSE", "AGE=20~29 AND Q5!=-8"), v("Q9", "RESP_UNIVERSE", "Q5=1~9"))
			cb.Codes = append(cb.Codes, codes("Q5", "1", "2")...)
		}, want: []string{"R RESP_UNIVERSE Q7 Q5=9"}},
		{rule: "C10", name: "var of another file", mod: func(cb *kosac.Codebook) {
			cb.Files = append(cb.Files, file("2", "ID_VAR", "walla_response_id"))
			cb.Vars = append(cb.Vars, v("Z1", "FILE_NO", "2"), v("Q6", "RESP_UNIVERSE", "Z1=1 OR walla_row_no=1"))
		}, want: []string{"R RESP_UNIVERSE Q6 Z1", "R RESP_UNIVERSE Q6 walla_row_no"}},
		{rule: "C10", name: "grammar", mod: addVars(v("Q6", "RESP_UNIVERSE", "ID=")), want: []string{"R RESP_UNIVERSE Q6 " + parseErr("ID=")}},

		{rule: "C11", name: "text var outside the group", mod: func(cb *kosac.Codebook) {
			cb.Vars = append(cb.Vars, v("Q5_1", "MEAS_LEVEL", "명목", "MR_TYPE", "보기별 열"), v("Q5_2", "MEAS_LEVEL", "명목", "MR_TYPE", "보기별 열"),
				v("Q5_other", "MEAS_LEVEL", "문자"))
			cb.Codes = append(cb.Codes, codes("Q5_1", "0", "1", "-8")...)
			cb.Codes = append(cb.Codes, codes("Q5_2", "0", "1")...)
		}, want: []string{}},
		{rule: "C11", name: "nominal var in the group", mod: func(cb *kosac.Codebook) {
			cb.Vars = append(cb.Vars, v("Q5_1", "MEAS_LEVEL", "명목", "MR_TYPE", "보기별 열"), v("Q5_other", "MEAS_LEVEL", "명목"))
			cb.Codes = append(cb.Codes, codes("Q5_1", "0", "1")...)
		}, want: []string{"R MR_TYPE Q5_other"}},
		{rule: "C11", name: "list and codes", mod: func(cb *kosac.Codebook) {
			cb.Vars = append(cb.Vars, v("Q5", "MEAS_LEVEL", "명목", "MR_TYPE", "복수"), v("Q6", "MEAS_LEVEL", "명목", "MR_TYPE", "보기별 열"))
			cb.Codes = append(cb.Codes, codes("Q6", "1", "2")...)
		}, want: []string{"R MR_TYPE Q5 복수", "R CODE Q6 2 2"}},

		{rule: "C12", name: "prefix group", mod: addVars(
			v("C7_1", "MEAS_LEVEL", "서열", "VAR_GROUP", "G"),
			v("C8_1", "MEAS_LEVEL", "서열", "VAR_GROUP", "G", "REVERSE", "Y"),
			v("C8_2", "MEAS_LEVEL", "서열", "VAR_GROUP", "G", "REVERSE", "Y"),
			v("QA3_1", "MEAS_LEVEL", "서열", "VAR_GROUP", "G"),
			v("QA3_4", "MEAS_LEVEL", "서열", "VAR_GROUP", "G", "REVERSE", "Y"),
			v("B2", "MEAS_LEVEL", "서열", "VAR_GROUP", "G", "REVERSE", "Y"),
		), want: []string{"W REVERSE C8_1 Y", "W REVERSE C8_2 Y"}},
		{rule: "C12", name: "not 서열", mod: addVars(v("B3", "MEAS_LEVEL", "명목", "REVERSE", "Y")), want: []string{"W MEAS_LEVEL B3 명목"}},
		{rule: "C12", name: "REVERSE is empty, Y or N", mod: addVars(
			v("B4", "MEAS_LEVEL", "서열", "REVERSE", "N "), v("B5", "MEAS_LEVEL", "서열", "REVERSE", "X"),
			v("B6", "MEAS_LEVEL", "서열", "REVERSE", "N"), v("B7", "MEAS_LEVEL", "서열", "REVERSE", " "),
		), want: []string{"R REVERSE B4 N", "R REVERSE B5 X"}},

		{rule: "C13", name: "derivation", mod: addVars(v("QA1_1"), v("QA1_2"),
			v("INT_SCORE", "Q_TEXT", "파생", "DERIVATION", "QA1_1, QA1_2를 0·50·100점으로 환산한 평균"),
			v("PCA", "Q_TEXT", "파생", "DERIVATION", "PCA 점수"),
			v("D1", "Q_TEXT", "파생", "DERIVATION", "QX9 평균"),
			v("D2", "DERIVATION", "QA1_1~QA1_2 합"),
		), want: []string{"R DERIVATION D1 QX9", "R Q_TEXT D2 D2 문항"}},

		{rule: "C14", name: "codes", mod: func(cb *kosac.Codebook) {
			cb.Vars = append(cb.Vars, v("Q1", "MEAS_LEVEL", "명목"))
			cs := codes("Q1", "1", "1", "1.5", "3", "4", "5")
			cs[3].Rec.Set("CODE_LABEL", "")
			cs[4].Rec.Set("CODE_LABEL", "④")
			cs[5].Rec.Set("CODE_LABEL", "5")
			cb.Codes = append(cb.Codes, cs...)
			cb.Codes = append(cb.Codes, codes("QX", "1")...)
		}, want: []string{"R CODE Q1 1 1", "R CODE Q1 1.5 1.5", "R CODE_LABEL Q1 3", "W CODE_LABEL Q1 4 ④", "W CODE_LABEL Q1 5 5", "R VAR_NAME QX 1 QX"}},

		{rule: "C15", name: "missing labels", mod: func(cb *kosac.Codebook) {
			cs := codes("ID", "-8", "-7", "-9")
			cs[0].Rec.Set("CODE_LABEL", "묻지 않음")
			cs[1].Rec.Set("CODE_LABEL", "무응답")
			cs[2].Rec.Set("CODE_LABEL", "모름")
			cb.Codes = append(cb.Codes, cs...)
		}, want: []string{"R CODE_LABEL ID -9 모름"}},
		{rule: "C15", name: "labels from RefData", mod: func(cb *kosac.Codebook) {
			cs := codes("ID", "-9")
			cs[0].Rec.Set("CODE_LABEL", "해당 없음")
			cb.Codes = append(cb.Codes, cs...)
		}, ref: kosac.RefData{MissingLabels: map[string][]string{"-9": {"해당 없음"}}}, want: []string{}},

		{rule: "C16", name: "dictionary", mod: addVars(
			v("SQ1", "MEAS_LEVEL", "명목", "STD_ID", "PERSON.GENDER"),
			v("SQ2", "MEAS_LEVEL", "서열", "STD_ID", "PERSON.GENDER"),
			v("SQ3", "STD_ID", "X.Y"),
		), ref: kosac.RefData{StdItems: map[string]kosac.StdItem{"PERSON.GENDER": {StdID: "PERSON.GENDER", MeasLevel: "nominal"}}},
			want: []string{"R MEAS_LEVEL SQ2 서열", "R STD_ID SQ3 X.Y"}},

		{rule: "C17", name: "identifiers", mod: addVars(
			v("SCH", "MEAS_LEVEL", "문자", "STD_ID", "SCHOOL.ID"),
			v("SCH2", "MEAS_LEVEL", "문자", "STD_ID", "SCHOOL.ID", "NOTE", "NEIS 2024년 기준"),
			v("ORG", "MEAS_LEVEL", "문자", "VAR_LABEL", "기관 코드"),
			v("NAME", "MEAS_LEVEL", "명목", "PII", "Y"),
		), want: []string{"W NOTE SCH", "W NOTE ORG", "W PII NAME Y"}},

		{rule: "C18", name: "report pages beyond the PDF", mod: func(cb *kosac.Codebook) { set(cb, "QUESTIONNAIRE", "REPORT:p336-") },
			ref: kosac.RefData{PDFPages: map[string]int{"r.pdf": 300}}, want: []string{"R QUESTIONNAIRE REPORT:p336-"}},
		{rule: "C18", name: "report pages", mod: func(cb *kosac.Codebook) { set(cb, "QUESTIONNAIRE", "최종보고서 부록 수록 (PDF 336쪽~)") },
			ref: kosac.RefData{PDFPages: map[string]int{"r.pdf": 400}}, want: []string{}},
		{rule: "C18", name: "neither", mod: func(cb *kosac.Codebook) { set(cb, "QUESTIONNAIRE", "보고서 참조", "REPORT_FILE", "REPORT:p1") },
			want: []string{"R QUESTIONNAIRE 보고서 참조", "R REPORT_FILE REPORT:p1"}},
		{rule: "C18", name: "uploaded files", mod: func(cb *kosac.Codebook) { set(cb, "EXTRA_FILES", "a.pdf; b.hwp") },
			ref: kosac.RefData{UploadedFiles: []string{"r.pdf", "a.pdf"}}, want: []string{"R QUESTIONNAIRE q.pdf", "R EXTRA_FILES b.hwp"}},
		// "p.336" also ends like a file extension; pages are read first.
		{rule: "C18", name: "pages before file name: not a missing upload", mod: func(cb *kosac.Codebook) { set(cb, "QUESTIONNAIRE", "부록 수록 p.336") },
			ref: kosac.RefData{UploadedFiles: []string{"r.pdf"}}, want: []string{}},
		{rule: "C18", name: "pages before file name: beyond the PDF", mod: func(cb *kosac.Codebook) { set(cb, "QUESTIONNAIRE", "부록 수록 p.336") },
			ref: kosac.RefData{PDFPages: map[string]int{"r.pdf": 300}}, want: []string{"R QUESTIONNAIRE 부록 수록 p.336"}},
		{rule: "C18", name: "pages before file name: report", mod: func(cb *kosac.Codebook) { set(cb, "REPORT_FILE", "부록 수록 p.336") },
			want: []string{"R REPORT_FILE 부록 수록 p.336"}},
	}
	seen := map[string]bool{}
	for _, c := range cases {
		seen[c.rule] = true
		t.Run(c.rule+" "+c.name, func(t *testing.T) {
			cb := base()
			if c.mod != nil {
				c.mod(cb)
			}
			if got := of(c.rule, Check(cb, c.ref)); !slices.Equal(got, c.want) {
				t.Errorf("got %q\nwant %q", got, c.want)
			}
		})
	}
	for i := 1; i <= 18; i++ {
		if r := fmt.Sprintf("C%02d", i); !seen[r] {
			t.Errorf("no case for %s", r)
		}
	}
}

func TestUnverifiedEverywhere(t *testing.T) {
	cb := base()
	set(cb, "POP_SOURCE", "확인 불가: x", "FIELD_START", "확인 불가: s", "TARGET_N", "확인 불가: n")
	cb.Files[0].Rec.Set("VALID_N", "확인 불가: z")
	cb.Vars = append(cb.Vars, v("Q1", "RESP_UNIVERSE", "확인 불가: y"), v("Q2", "MEAS_LEVEL", "명목"))
	cb.Codes = append(cb.Codes, codes("Q2", "1", "2")...)
	cb.Codes[1].Rec.Set("CODE", "확인 불가: c")
	var got []string
	for _, i := range Check(cb, kosac.RefData{}) {
		got = append(got, i.Rule+" "+sum(i))
	}
	want := []string{
		"C01 W FIELD_START 확인 불가: s", "C01 W POP_SOURCE 확인 불가: x", "C01 W TARGET_N 확인 불가: n",
		"C05 W VALID_N 확인 불가: z",
		"C10 W RESP_UNIVERSE Q1 확인 불가: y",
		"C14 W CODE Q2 확인 불가: c 확인 불가: c",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestRefDataNil(t *testing.T) {
	cb := base()
	cb.Vars = append(cb.Vars, v("SQ1", "STD_ID", "X.Y"))
	set(cb, "QUESTIONNAIRE", "REPORT:p999", "EXTRA_FILES", "a.pdf")
	for _, i := range Check(cb, kosac.RefData{}) {
		t.Errorf("issue with a zero RefData: %+v", i)
	}
	Check(&kosac.Codebook{}, kosac.RefData{})
}

func TestCheckNil(t *testing.T) {
	if is := Check(nil, kosac.RefData{}); is != nil {
		t.Errorf("Check(nil) = %+v, want nil", is)
	}
}

func TestSort(t *testing.T) {
	is := []kosac.Issue{
		{Rule: "C08", Sheet: kosac.SheetVars, Cell: "C4"},
		{Rule: "C07", Sheet: kosac.SheetCodes, Cell: "A4"},
		{Rule: "C07", Sheet: kosac.SheetVars, Cell: "AA10"},
		{Rule: "C07", Sheet: kosac.SheetVars, Cell: "B10"},
		{Rule: "C07", Sheet: kosac.SheetVars, Cell: "Z9"},
		{Rule: "C07", Sheet: kosac.SheetStudy, Cell: "E40"},
	}
	sortIssues(is)
	var got []string
	for _, i := range is {
		got = append(got, i.Rule+" "+i.Sheet+"!"+i.Cell)
	}
	want := []string{
		"C07 01_조사정보!E40", "C07 03_변수정의!Z9", "C07 03_변수정의!B10", "C07 03_변수정의!AA10",
		"C07 04_코드값!A4", "C08 03_변수정의!C4",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestSampleGolden(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "kosac")
	x, err := os.ReadFile(filepath.Join(root, "v32_sample_2024.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	cb, read, err := template.Read(x, template.ReadOptions{})
	if err != nil || len(read) != 0 {
		t.Fatalf("Read: %v %v", read, err)
	}
	is := Check(cb, kosac.RefData{})

	counts := map[string]int{}
	var c12 []string
	for _, i := range is {
		counts[i.Rule+" "+string(i.Level)]++
		switch i.Rule {
		case "C07", "C10":
			if i.VarName != map[string]string{"C07": "SQ1", "C10": "QC2"}[i.Rule] {
				t.Errorf("%s on %s", i.Rule, i.VarName)
			}
		case "C12":
			c12 = append(c12, i.VarName)
		}
	}
	want := map[string]int{"C01 W": 4, "C07 R": 1, "C08 W": 145, "C10 W": 1, "C12 W": 4}
	if fmt.Sprint(counts) != fmt.Sprint(want) {
		t.Errorf("counts = %v, want %v", counts, want)
	}
	if !slices.Equal(c12, []string{"C8_1", "C8_2", "C9_1", "C9_2"}) {
		t.Errorf("C12 on %q", c12)
	}

	got, err := json.MarshalIndent(is, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	golden := filepath.Join(root, "v32_sample_2024.issues.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want2, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want2) {
		t.Error("issues differ from v32_sample_2024.issues.json; rerun with -update and review the diff")
	}
}

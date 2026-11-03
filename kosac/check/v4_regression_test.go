package check

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/template"
)

func TestV4GroupDomainAndSeverity(t *testing.T) {
	for _, tc := range []struct {
		value string
		warn  bool
	}{
		{"과학기술 정책 태도(R&D): QA22", false},
		{strings.Repeat("가", 20) + ": Q123456789", false},
		{"상위: " + strings.Repeat("가", 17) + ": Q1", true},
		{strings.Repeat("가", 21), true},
		{"Q 3", true},
		{"문항3", true},
		{"Q3 관련 경험", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			cb := v4Model()
			cb.Vars[0].Rec.Set("VAR_GROUP", tc.value)
			c := newChecker(cb, kosac.RefData{})
			c.c09()
			if len(c.issues) != boolCount(tc.warn) || tc.warn && c.issues[0].Level != kosac.W {
				t.Fatal(c.issues)
			}
			if cb.Vars[0].Rec.Get("VAR_GROUP") != tc.value {
				t.Fatal("source group changed")
			}
		})
	}
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestV4StudyCountNotes(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"10000 (단일 시점, 중복 없음)", true},
		{"30530 (학생 30,004 + 교사 526, 중복 없음)", true},
		{"30,530(중복 없음)", true},
		{"0 (유효 응답 없음)", true},
		{"1.5 (설명)", false},
		{"10,00 (설명)", false},
		{"약 10000 (설명)", false},
		{"10000 (설명) + 526", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			cb := v4Model()
			cb.Study.Set("VALID_N", tc.value)
			c := newChecker(cb, kosac.RefData{})
			c.c02()
			if (len(c.issues) == 0) != tc.valid {
				t.Fatal(c.issues)
			}
			if cb.Study.Get("VALID_N") != tc.value {
				t.Fatal("source count note changed")
			}
		})
	}
}

func TestV4ReverseDirectionEvidence(t *testing.T) {
	for _, tc := range []struct {
		level, note, derivation string
		want                    kosac.Level
	}{
		{"서열", "1=가장 높음, 역순 척도", "", ""},
		{"서열", "", "원점수 1~5를 역순으로 환산", ""},
		{"서열", "", "", kosac.W},
		{"서열", "해당 없음", "확인 불가: 근거 없음", kosac.W},
		{"명목", "역문항", "", kosac.R},
	} {
		t.Run(tc.level+tc.note+tc.derivation, func(t *testing.T) {
			cb := v4Model()
			cb.Vars = []*kosac.Var{v("Q1_1", "REVERSE", "Y", "MEAS_LEVEL", tc.level, "NOTE", tc.note, "DERIVATION", tc.derivation)}
			c := newChecker(cb, kosac.RefData{})
			c.c12()
			if len(c.issues) != boolCount(tc.want != "") || tc.want != "" && c.issues[0].Level != tc.want {
				t.Fatal(c.issues)
			}
		})
	}
}

func TestV4SourceReferencesInKoreanAndRanges(t *testing.T) {
	for _, tc := range []struct {
		source, derivation string
		valid              bool
	}{
		{"Q1;Q2", "Q1·Q2를 환산한 평균", true},
		{"Q1~Q3", "Q1, Q2, Q3의 평균", true},
		{"Q2", "Q1 ~ Q3에서 결측을 제외", true},
		{"Q1 ~ Q3", "Q1~Q3의 평균", true},
		{"raw_q", "raw_q가 있으면 0", true},
		{"Q1", "Q10의 평균", false},
		{"Q1", "Q1_extra의 평균", false},
		{"Q2", "Q3~Q1의 평균", false},
		{"Q2", "Q1~Q999의 평균", false},
		{"Q1~Q3", "Q1과 Q3의 평균", false},
	} {
		t.Run(tc.source+" / "+tc.derivation, func(t *testing.T) {
			cb := v4Model()
			cb.Vars = nil
			for _, name := range []string{"Q1", "Q2", "Q3", "Q10", "Q1_extra"} {
				cb.Vars = append(cb.Vars, v(name, "SOURCE_VAR", name))
			}
			cb.Vars = append(cb.Vars, v("out", "SOURCE_VAR", tc.source, "DERIVATION", tc.derivation))
			cb.Recodes[0].Set("TARGET_VAR", "out")
			c := newChecker(cb, kosac.RefData{})
			c.v4sources()
			if (len(c.issues) == 0) != tc.valid {
				t.Fatal(c.issues)
			}
		})
	}
	if literalName("다른원본.csv", "원본.csv") || literalName("raw.csv2", "raw.csv") || literalName("-90", "-9") {
		t.Fatal("file and missing-code boundaries were weakened")
	}
}

func TestV4InstitutionCodeSystemAndStatus(t *testing.T) {
	cb := v4Model()
	cb.Vars = []*kosac.Var{
		v("SCHOOL_CD", "VAR_LABEL", "학교 코드", "CODE_SYSTEM", "NEIS (2026)", "MISSING_STATUS_VAR", "SCHOOL_CD_ST"),
		v("SCHOOL_CD_ST", "VAR_LABEL", "학교 코드 결측 상태", "MEAS_LEVEL", "명목"),
		v("ORG", "VAR_LABEL", "기관 코드", "NOTE", "NEIS 2026년 기준"),
		v("UNKNOWN", "STD_ID", "SCHOOL.ID", "CODE_SYSTEM", "확인 불가: 체계 미확인"),
		v("OTHER_FILE", "FILE_NO", "2", "VAR_NAME", "SCHOOL_CD_ST", "VAR_LABEL", "학교 코드"),
	}
	c := newChecker(cb, kosac.RefData{})
	c.c17()
	var names []string
	for _, issue := range c.issues {
		if issue.Level != kosac.W || issue.Expected != "코드 체계·기준연도" {
			t.Fatal(issue)
		}
		names = append(names, issue.FileNo+"/"+issue.VarName)
	}
	if !slices.Equal(names, []string{"1/ORG", "1/UNKNOWN", "2/SCHOOL_CD_ST"}) {
		t.Fatal(names)
	}
}

func TestV4QuestionnairePageNormalization(t *testing.T) {
	for _, tc := range []struct {
		value string
		pages int
		want  kosac.Level
	}{
		{"REPORT:p331-", 400, ""},
		{"최종보고서 부록 수록(부록 설문지, 보고서 331쪽 이후)", 400, kosac.W},
		{"최종보고서 부록 수록(331쪽~)", 300, kosac.R},
		{"최종보고서 부록 수록(초등 97쪽, 중학 104쪽, 고등 111쪽, 교사 119쪽~). 변수가이드(학생 251128·교사 251117)", 400, kosac.R},
	} {
		t.Run(tc.value, func(t *testing.T) {
			cb := v4Model()
			cb.Study.Set("QUESTIONNAIRE", tc.value)
			cb.Study.Set("REPORT_FILE", "report.pdf")
			c := newChecker(cb, kosac.RefData{PDFPages: map[string]int{"report.pdf": tc.pages}})
			c.c18()
			if len(c.issues) != boolCount(tc.want != "") || tc.want != "" && c.issues[0].Level != tc.want {
				t.Fatal(c.issues)
			}
		})
	}
}

func TestV4CountedRangeAnnotations(t *testing.T) {
	cb := v4Model()
	cb.Vars = []*kosac.Var{v("Q1"), v("Q2"), v("Q3")}
	c := newChecker(cb, kosac.RefData{})
	for _, expression := range []string{"Q1~Q3 (3개)", "Q1~Q3 (2개)", "Q1~Q99 (3개)", "(전체) (2개)"} {
		got, err := c.v4targets("1", expression, "")
		if expression == "Q1~Q3 (3개)" {
			if err != nil || !slices.Equal(got, []string{"Q1", "Q2", "Q3"}) {
				t.Fatal(got, err)
			}
		} else if err == nil {
			t.Fatal("invalid range accepted", expression, got)
		}
	}
}

func TestV4SuppliedCodebookRegressions(t *testing.T) {
	want := map[string]map[string]map[kosac.Level]int{
		"2024": {"C02": {}, "C09": {}, "C12": {}, "C17": {}, "C18": {kosac.W: 1}, "C22": {}, "C25": {kosac.W: 2}, "C10": {kosac.R: 1}},
		"2025": {"C02": {kosac.R: 1}, "C09": {kosac.W: 1}, "C12": {}, "C17": {}, "C18": {kosac.R: 3}, "C22": {kosac.R: 1}, "C25": {kosac.W: 3}, "C10": {kosac.R: 6}, "C15": {kosac.R: 2}},
	}
	for year, rules := range want {
		t.Run(year, func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join("..", "..", "testdata", "kosac", "v40_sample_"+year+".xlsx"))
			if err != nil {
				t.Fatal(err)
			}
			cb, structure, err := template.Read(input, template.ReadOptions{})
			if err != nil || cb == nil || len(structure) > 0 {
				t.Fatal(structure, err)
			}
			before, _ := json.Marshal(cb)
			issues := Check(cb, kosac.RefData{})
			for rule, counts := range rules {
				got := map[kosac.Level]int{}
				for _, issue := range issues {
					if issue.Rule == rule {
						got[issue.Level]++
					}
				}
				if got[kosac.R] != counts[kosac.R] || got[kosac.W] != counts[kosac.W] {
					t.Errorf("%s: got %v, want %v", rule, got, counts)
				}
			}
			after, _ := json.Marshal(cb)
			if !bytes.Equal(before, after) {
				t.Fatal("checking rewrote source metadata")
			}
		})
	}
}

func TestV4HistoryStatusesStaySeparateFromVariableStatuses(t *testing.T) {
	for _, status := range []string{"미착수", "확인 필요", "규칙 확정", "처리 완료", "검증 완료", "적용 제외", "임의 상태"} {
		t.Run(status, func(t *testing.T) {
			cb := v4Model()
			cb.Recodes[0].Set("STATUS", status)
			c := newChecker(cb, kosac.RefData{PendingRecodeLimit: .9})
			c.v4recodeStatus()
			rejected := slices.ContainsFunc(c.issues, func(i kosac.Issue) bool { return i.Level == kosac.R })
			if rejected != (status == "임의 상태") || status != "처리 완료" && len(c.issues) == 0 {
				t.Fatal(c.issues)
			}
		})
	}
	for _, codeRow := range []bool{false, true} {
		cb := v4Model()
		r := cb.Vars[0].Rec
		if codeRow {
			r = cb.Codes[0].Rec
		}
		r.Set("RECODE_STATUS", "검증 완료")
		c := newChecker(cb, kosac.RefData{})
		c.v4recodeStatus()
		if !slices.ContainsFunc(c.issues, func(i kosac.Issue) bool { return i.Level == kosac.R }) {
			t.Fatal("history status leaked into variable/code status", c.issues)
		}
	}
}

func TestV4TransformationTypeApproval(t *testing.T) {
	for _, tc := range []struct {
		value string
		list  []string
		valid bool
	}{
		{"ID 생성", nil, true},
		{"", nil, false},
		{"ID 생성", []string{}, false},
		{"ID 생성", []string{"ID 생성"}, true},
		{"ID 생성", []string{"결측 코드화"}, false},
	} {
		cb := v4Model()
		cb.Recodes[0].Set("TRANSFORM_TYPE", tc.value)
		c := newChecker(cb, kosac.RefData{TransformTypes: tc.list})
		c.v4recodes()
		if (len(c.issues) == 0) != tc.valid {
			t.Fatalf("%q, %v: %v", tc.value, tc.list, c.issues)
		}
	}
}

func TestV4NonDataHistoryDoesNotBypassReferencesOrProvenance(t *testing.T) {
	for _, tc := range []struct {
		sourceFile, source, target string
		valid                      bool
	}{
		{"raw.csv", "q", "(코드북 04)", true},
		{"raw.csv", "unknown", "(코드북 04)", false},
		{"(코드북 추가 매핑)", "raw_q", "q", true},
		{"(코드북 추가 매핑)", "raw_q", "unknown", false},
		{"(임의 매핑)", "raw_q", "q", false},
		{"unknown.csv", "raw_q", "q", false},
		{"raw.csv", "-", "-", true},
		{"raw.csv", "전체 숫자 열", "동일 변수", true},
		{"raw.csv", "q", "(임의 대상)", false},
	} {
		cb := v4Model()
		r := cb.Recodes[0]
		r.Set("SOURCE_FILE", tc.sourceFile)
		r.Set("SOURCE_VAR", tc.source)
		r.Set("TARGET_VAR", tc.target)
		c := newChecker(cb, kosac.RefData{})
		c.v4recodes()
		rejected := slices.ContainsFunc(c.issues, func(i kosac.Issue) bool { return i.Level == kosac.R })
		if rejected == tc.valid {
			t.Fatalf("%+v: %v", tc, c.issues)
		}
		if tc.valid && !slices.ContainsFunc(c.issues, func(i kosac.Issue) bool { return i.Level == kosac.W }) {
			t.Fatal("record acceptance was presented as completed verification", c.issues)
		}
		if tc.target == "동일 변수" || tc.target == "-" || tc.target == "(코드북 04)" {
			c = newChecker(cb, kosac.RefData{})
			c.v4sources()
			if !slices.ContainsFunc(c.issues, func(i kosac.Issue) bool { return i.Rule == "C22" && i.Level == kosac.R }) {
				t.Fatal("review/metadata record satisfied a new variable's provenance", c.issues)
			}
		}
	}
}

func TestV4StatusVariableNeedsOneValidCodeWithoutRelaxingOtherCategories(t *testing.T) {
	cb := v4Model()
	cb.Vars = []*kosac.Var{v("school", "MISSING_STATUS_VAR", "status", "MEAS_LEVEL", "해당 없음"), v("status", "MEAS_LEVEL", "명목")}
	cb.Codes = codes("status", "0", "-9")
	c := newChecker(cb, kosac.RefData{})
	c.c07()
	if len(c.issues) != 0 {
		t.Fatal(c.issues)
	}
	cb.Vars[0].Rec.Set("MISSING_STATUS_VAR", "")
	c = newChecker(cb, kosac.RefData{})
	c.c07()
	if len(c.issues) != 1 || c.issues[0].Level != kosac.R {
		t.Fatal("ordinary categorical minimum was relaxed", c.issues)
	}
}

func TestV4HistoryMarkersDoNotHideMissingCodeChecks(t *testing.T) {
	for _, tc := range []struct{ sourceFile, source, target string }{
		{"raw.csv", "q", "(코드북 04)"},
		{"raw.csv", "-", "-"},
		{"raw.csv", "전체 숫자 열", "동일 변수"},
		{"(코드북 추가 매핑)", "q", "q"},
	} {
		cb := v4Model()
		r := cb.Recodes[0]
		r.Set("SOURCE_FILE", tc.sourceFile)
		r.Set("SOURCE_VAR", tc.source)
		r.Set("TARGET_VAR", tc.target)
		r.Set("TRANSFORM_TYPE", "결측 코드화")
		r.Set("TARGET_CODE_OR_RULE", "-8")
		c := newChecker(cb, kosac.RefData{})
		c.v4recodes()
		if !slices.ContainsFunc(c.issues, func(i kosac.Issue) bool { return i.Level == kosac.R }) {
			t.Fatal("missing-code validation was bypassed", tc, c.issues)
		}
	}
}

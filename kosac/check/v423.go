package check

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/teamPaprika/ddi-xlsx/kosac"
)

// The v4.2.3 profile (TEMPLATE_VERSION "4.2.3") has its own rule set S01–S17
// from the template's 07 sheet. Only the workbook rules run here. The XML
// rules S18–S20 belong to ValidateXML, and the data rules D01–D11 need the
// data files, so neither runs in Check.

const report423 = "[보고] "

// Sheet423Unconfirmed is the optional sheet of unconfirmed items and reasons.
const Sheet423Unconfirmed = "10_미확인사유"

func is423(cb *kosac.Codebook) bool {
	return strings.TrimSpace(cb.Template.Version.Value) == kosac.Profile423
}

func (c *checker) rules423() []func() {
	return []func(){c.s01, c.s02, c.s03, c.s04, c.s05, c.s06, c.s07, c.s08, c.s09, c.s10, c.s11, c.s12, c.s13, c.s14, c.s15, c.s16, c.s17}
}

// unconfirmed423 is a value the producer marked as not yet confirmed.
func unconfirmed423(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "미확인") || strings.HasPrefix(s, "확인 불가") || strings.HasPrefix(s, "확인불가")
}

func filled423(s string) bool { return !blank(s) && !unconfirmed423(s) }

var required423 = strings.Fields("TITLE PURPOSE SUMMARY FIELD_ORG FIELD_START FIELD_END KEYWORDS QUESTIONNAIRE REPORT_FILE UNIVERSE POP_SOURCE ANALYSIS_UNIT SAMPLING_TYPE SAMPLING_DESC TARGET_N VALID_N SPLIT_BALLOT MODE EXCLUDE_N WEIGHT_YN SURVEY_TYPE ACCESS_RESTRICTIONS SOURCE_FILES")

// reasoned423 reports whether the 10 sheet names key in a row below its header.
func (c *checker) reasoned423(key string) bool {
	rows := c.cb.Supplemental[Sheet423Unconfirmed]
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if slices.ContainsFunc(row, func(cell string) bool { return literalVarName(cell, key) }) {
			return true
		}
	}
	return false
}

// s01: blank required values. Auto, foundation and conditional values are not
// required as a whole; a conditional one only when its condition holds.
func (c *checker) s01() {
	st := c.cb.Study
	g := func(k string) string { return strings.TrimSpace(st.Get(k)) }
	if v := g("KEYWORDS"); filled423(v) {
		count := 0
		for _, keyword := range strings.Split(v, ";") {
			if !blank(keyword) {
				count++
			}
		}
		if count < 3 || count > 7 {
			c.add("S01", kosac.R, st, "KEYWORDS", v, "세미콜론으로 구분한 3~7개", "키워드 개수를 확인하세요.")
		}
	}
	for _, key := range []string{"STRATA_VARS", "CLUSTER_VAR", "WEIGHT_VAR", "SPLIT_VAR"} {
		if !filled423(g(key)) {
			continue
		}
		for _, name := range refNames423(g(key)) {
			if c.byName[name] == nil {
				c.add("S01", kosac.R, st, key, name, "03 변수정의의 변수명", "조사정보가 참조하는 변수가 없습니다.")
			}
		}
	}
	if g("RULE_VERSION") != "" && g("RULE_VERSION") != kosac.RuleVersion423 {
		c.add("S01", kosac.R, st, "RULE_VERSION", g("RULE_VERSION"), "20261105-r3", "배포 양식의 고정 규칙 버전을 확인하세요.")
	}
	when := map[string]string{}
	if g("WEIGHT_YN") == "Y" {
		when["WEIGHT_VAR"], when["WEIGHT_METHOD"] = "WEIGHT_YN=Y", "WEIGHT_YN=Y"
	}
	if g("SPLIT_BALLOT") == "Y" {
		when["SPLIT_VAR"] = "SPLIT_BALLOT=Y"
	}
	if n, ok := integer(g("EXCLUDE_N")); ok && n >= 1 {
		when["EXCLUDE_RULE"] = "EXCLUDE_N ≥ 1"
	}
	if filled423(g("RESPONSE_RATE")) {
		when["RR_FORMULA"] = "RESPONSE_RATE 있음"
	}
	if s := g("SAMPLING_TYPE"); strings.HasPrefix(s, "확률-") {
		when["CONTACT_N"], when["RESPONSE_RATE"] = "SAMPLING_TYPE="+s, "SAMPLING_TYPE="+s
	}
	if g("SURVEY_TYPE") == "사업성과" {
		for _, k := range []string{"BIZ_NAME", "BIZ_YEAR", "RELATED_PROGRAM_CD"} {
			when[k] = "SURVEY_TYPE=사업성과"
		}
	}
	if slices.ContainsFunc(c.cb.Vars, func(v *kosac.Var) bool { return strings.TrimSpace(v.Rec.Get("STD_ID")) == "SCHOOL.ID" }) {
		when["SCHOOL_CODE_YEAR"] = "STD_ID=SCHOOL.ID인 변수 있음"
	}
	for _, k := range kosac.LayoutFor(kosac.Profile423).Study {
		if !blank(st.Get(k)) {
			continue
		}
		expected := ""
		switch {
		case slices.Contains(required423, k):
			expected = "필수"
		case when[k] != "":
			expected = when[k] + "일 때 필수"
		default:
			continue
		}
		if c.reasoned423(k) {
			c.add("S01", kosac.R, st, k, "", expected, k+"의 빈칸 사유가 "+Sheet423Unconfirmed+"에 있습니다. 재단 확인 전에는 최종 확정하지 않습니다.")
		} else {
			c.add("S01", kosac.R, st, k, "", expected, k+"의 값이 비어 있습니다. 원자료에서 확인할 수 없으면 "+Sheet423Unconfirmed+"에 사유를 기록하세요.")
		}
	}
}

// s02: 01 value formats.
func (c *checker) s02() {
	st := c.cb.Study
	for _, k := range []string{"FIELD_START", "FIELD_END", "CB_DATE"} {
		if v := st.Get(k); filled423(v) {
			if _, ok := date(strings.TrimSpace(v)); !ok {
				c.add("S02", kosac.R, st, k, v, "YYYY-MM-DD", k+"의 값이 실제 날짜(YYYY-MM-DD)가 아닙니다.")
			}
		}
	}
	start, ok1 := date(strings.TrimSpace(st.Get("FIELD_START")))
	end, ok2 := date(strings.TrimSpace(st.Get("FIELD_END")))
	if ok1 && ok2 && end.Before(start) {
		c.add("S02", kosac.R, st, "FIELD_END", st.Get("FIELD_END"), "FIELD_START 이후", "조사 종료일(FIELD_END)이 시작일(FIELD_START)보다 앞섭니다.")
	}
	for _, k := range []string{"TARGET_N", "CONTACT_N", "EXCLUDE_N"} {
		if v := st.Get(k); filled423(v) {
			if _, ok := integer(strings.TrimSpace(v)); !ok {
				c.add("S02", kosac.R, st, k, v, "숫자만", k+"의 값은 0 이상의 정수여야 합니다.")
			}
		}
	}
	if v := st.Get("RESPONSE_RATE"); filled423(v) && !decimalRe.MatchString(strings.TrimSpace(v)) {
		c.add("S02", kosac.R, st, "RESPONSE_RATE", v, "숫자만", "RESPONSE_RATE의 값은 숫자여야 합니다.")
	}
	for _, k := range []string{"BIZ_YEAR", "SCHOOL_CODE_YEAR"} {
		if v := st.Get(k); filled423(v) && !yearRe.MatchString(strings.TrimSpace(v)) {
			c.add("S02", kosac.R, st, k, v, "YYYY", k+"의 값은 네 자리 연도(YYYY)여야 합니다.")
		}
	}
}

var fileNo423Re = regexp.MustCompile(`^[0-9]{1,2}$`)

// fileID423 mirrors the XML file ID the v4.2.3 export writes.
func fileID423(no string) string {
	n, _ := strconv.Atoi(no)
	return fmt.Sprintf("F%02d", n)
}

// s03: FILE_NO is unique, gives an F01-style XML ID, and every reference
// names a 02 FILE_NO exactly as written.
func (c *checker) s03() {
	seen, ids := map[string]bool{}, map[string]string{}
	for _, f := range c.cb.Files {
		r := f.Rec
		no := r.Get("FILE_NO")
		switch {
		case !fileNo423Re.MatchString(no) || strings.Trim(no, "0") == "":
			c.add("S03", kosac.R, r, "FILE_NO", no, "1~99의 숫자", "FILE_NO는 XML 파일 ID(F01 형식)를 만들 수 있는 1~99의 숫자여야 합니다.")
		case seen[no]:
			c.add("S03", kosac.R, r, "FILE_NO", no, "파일마다 다른 번호", "같은 FILE_NO가 두 번 이상 있습니다.")
		case ids[fileID423(no)] != "":
			c.add("S03", kosac.R, r, "FILE_NO", no, "파일마다 다른 XML 파일 ID", fmt.Sprintf("FILE_NO %s와 %s가 같은 XML 파일 ID %s가 됩니다.", ids[fileID423(no)], no, fileID423(no)))
		default:
			ids[fileID423(no)] = no
		}
		seen[no] = true
	}
	for _, v := range c.cb.Vars {
		if no := v.Rec.Get("FILE_NO"); c.cb.File(no) == nil {
			c.add("S03", kosac.R, v.Rec, "FILE_NO", no, "02_데이터파일의 FILE_NO", "변수의 FILE_NO가 02_데이터파일 시트에 없습니다.")
		}
	}
	for _, cd := range c.cb.Codes {
		if no := cd.Rec.Get("FILE_NO"); c.cb.File(no) == nil {
			c.add("S03", kosac.R, cd.Rec, "FILE_NO", no, "02_데이터파일의 FILE_NO", "코드의 FILE_NO가 02_데이터파일 시트에 없습니다.")
		}
	}
}

// s04: (FILE_NO, VAR_NAME) is unique.
func (c *checker) s04() {
	seen := map[key]bool{}
	for _, v := range c.cb.Vars {
		k := varKey(v.Rec)
		if seen[k] && !blank(k.name) {
			c.add("S04", kosac.R, v.Rec, "VAR_NAME", k.name, "파일 안에서 유일한 이름", "같은 파일에 VAR_NAME이 두 번 이상 있습니다.")
		}
		seen[k] = true
	}
}

var varName423Re = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,31}$`)

// s05: var name form, and no names in one file that differ only by case.
func (c *checker) s05() {
	folded := map[key]string{}
	for _, v := range c.cb.Vars {
		r := v.Rec
		name := r.Get("VAR_NAME")
		if !varName423Re.MatchString(name) {
			c.add("S05", kosac.R, r, "VAR_NAME", name, "영문자 시작, 영문·숫자·_, 32자 이하", "변수명 형식이 맞지 않습니다.")
			continue
		}
		k := key{r.Get("FILE_NO"), strings.ToLower(name)}
		if prev, ok := folded[k]; ok && prev != name {
			c.add("S05", kosac.R, r, "VAR_NAME", name, prev, "같은 파일에 대소문자만 다른 변수명이 있습니다.")
		} else if !ok {
			folded[k] = name
		}
	}
}

// s06: duplicate var labels within a file.
func (c *checker) s06() {
	seen := map[key]string{}
	for _, v := range c.cb.Vars {
		r := v.Rec
		label := strings.TrimSpace(r.Get("VAR_LABEL"))
		if label == "" {
			continue
		}
		k := key{r.Get("FILE_NO"), label}
		if prev, ok := seen[k]; ok {
			c.add("S06", kosac.W, r, "VAR_LABEL", label, "파일 안에서 다른 라벨", "같은 파일의 변수 "+prev+"와 변수 라벨이 같습니다.")
		} else {
			seen[k] = r.Get("VAR_NAME")
		}
	}
}

// s07: 04 rows name a 03 var; nominal and ordinal vars have a code.
func (c *checker) s07() {
	for _, cd := range c.cb.Codes {
		if c.vars[varKey(cd.Rec)] == nil {
			c.add("S07", kosac.R, cd.Rec, "VAR_NAME", cd.Rec.Get("VAR_NAME"), "같은 FILE_NO의 03_변수정의 변수", "코드의 변수가 03_변수정의 시트에 없습니다.")
		}
	}
	for _, v := range c.cb.Vars {
		ml := strings.TrimSpace(v.Rec.Get("MEAS_LEVEL"))
		if (ml == "명목" || ml == "서열") && len(c.codes[varKey(v.Rec)]) == 0 {
			c.add("S07", kosac.R, v.Rec, "MEAS_LEVEL", ml, "04 코드 1개 이상", "명목·서열 변수의 코드가 04_코드값 시트에 없습니다.")
		}
	}
}

// s08: every code's missing status follows from the var definition and the
// label. -9/-8/-7 count as missing only under their standard meaning; the
// export stops on any code whose status is not confirmed.
func (c *checker) s08() {
	seen := map[[3]string]bool{}
	for _, cd := range c.cb.Codes {
		r := cd.Rec
		code := r.Get("CODE")
		if blank(code) || strings.TrimSpace(code) != code {
			c.add("S08", kosac.R, r, "CODE", code, "공백 없는 코드", "CODE는 비어 있거나 앞뒤 공백을 포함할 수 없습니다.")
			continue
		}
		k := [3]string{r.Get("FILE_NO"), r.Get("VAR_NAME"), code}
		if seen[k] {
			c.add("S08", kosac.R, r, "CODE", code, "변수 안에서 유일한 코드", "같은 변수에 같은 CODE가 두 번 이상 있습니다.")
		}
		seen[k] = true
		if _, ok := kosac.Missing423(code, r.Get("CODE_LABEL")); !ok {
			expected := "라벨 확정"
			if m := map[string]string{"-9": "무응답", "-8": "비해당", "-7": "모름·거절"}[code]; m != "" {
				expected = m
			}
			c.add("S08", kosac.R, r, "CODE_LABEL", r.Get("CODE_LABEL"), expected, "코드의 유효·결측 여부를 확정할 수 없습니다. 표준 결측 코드의 뜻과 다르면 별도 처리 기준을 확인하세요.")
		}
	}
	for _, v := range c.cb.Vars {
		r := v.Rec
		for _, name := range refNames423(r.Get("MISSING_STATUS_VAR")) {
			if c.vars[key{r.Get("FILE_NO"), name}] == nil {
				c.add("S08", kosac.R, r, "MISSING_STATUS_VAR", name, "같은 파일의 변수명", "결측 상태 변수가 같은 파일에 없습니다.")
			}
		}
	}
}

// Value lists the v4.2.3 export maps; another value would be dropped from the
// XML rather than replaced by a default.
var (
	studyLists423 = map[string][]string{
		"ANALYSIS_UNIT": {"개인", "학교", "기관", "기타"},
		"SAMPLING_TYPE": {"확률-단순무작위", "확률-층화", "확률-집락", "확률-다단계", "할당(비확률)", "편의(비확률)", "전수", "기타"},
		"MODE":          {"온라인(웹·모바일)", "면접", "전화", "지면 자기기입", "혼합", "기타"},
		"SURVEY_TYPE":   {"사업성과", "연구조사"},
		"WEIGHT_YN":     {"Y", "N"},
		"SPLIT_BALLOT":  {"Y", "N"},
		"RULE_VERSION":  {"20261105-r3"},
	}
	varLists423 = map[string][]string{
		"MEAS_LEVEL":   {"명목", "서열", "등간", "비율", "해당 없음"},
		"DATA_TYPE":    {"숫자", "문자", "날짜", "일시"},
		"DDI_INTERVAL": {"discrete", "contin", "해당 없음"},
		"KEY_ROLE":     {"응답 ID", "참여번호", "실행단위 ID", "연결 대상 참여번호", "조사 시점", "가중치", "일반"},
		"PII":          {"Y", "N"},
		"SENSITIVE":    {"Y", "N"},
		"REVERSE":      {"Y", "N"},
	}
)

// s09: reports unconfirmed values and their reasons, and values the export
// would drop.
func (c *checker) s09() {
	layout := kosac.LayoutFor(kosac.Profile423)
	unconfirmed := func(r kosac.Record, keys []string) {
		for _, k := range keys {
			if v := r.Get(k); unconfirmed423(v) {
				c.add("S09", kosac.I, r, k, v, "재단 확인", report423+k+"의 값이 미확인입니다. 재단 확인 후 최종 확정합니다.")
			}
		}
	}
	listed := func(r kosac.Record, lists map[string][]string, keys []string) {
		for _, k := range keys {
			l := lists[k]
			if v := r.Get(k); l != nil && filled423(v) && !slices.Contains(l, v) {
				c.add("S09", kosac.I, r, k, v, strings.Join(l, ", "), k+"의 값이 양식 목록에 없습니다. XML에는 이 값을 기본값으로 바꾸지 않고 빼고 씁니다.")
			}
		}
	}
	unconfirmed(c.cb.Study, layout.Study)
	listed(c.cb.Study, studyLists423, layout.Study)
	for _, f := range c.cb.Files {
		unconfirmed(f.Rec, layout.Files)
	}
	for _, v := range c.cb.Vars {
		unconfirmed(v.Rec, layout.Vars)
		listed(v.Rec, varLists423, layout.Vars)
	}
	for _, cd := range c.cb.Codes {
		unconfirmed(cd.Rec, layout.Codes)
	}
	for i, row := range c.cb.Supplemental[Sheet423Unconfirmed] {
		if i == 0 || !slices.ContainsFunc(row, func(s string) bool { return !blank(s) }) {
			continue
		}
		var cells []string
		for _, s := range row {
			if s = strings.TrimSpace(s); s != "" {
				cells = append(cells, s)
			}
		}
		c.issues = append(c.issues, kosac.Issue{Rule: "S09", Level: kosac.I, Sheet: Sheet423Unconfirmed, Cell: "A" + strconv.Itoa(i+1),
			Actual: strings.Join(cells, " | "), Expected: "재단 확인", Message: report423 + "미확인 항목과 사유입니다. 재단 확인 후 최종 확정합니다."})
	}
}

// s10: STD_ID is a survey item of the standard code table.
func (c *checker) s10() {
	for _, v := range c.cb.Vars {
		id := strings.TrimSpace(v.Rec.Get("STD_ID"))
		if filled423(id) && std423Items[id].ID == "" {
			c.add("S10", kosac.R, v.Rec, "STD_ID", id, "표준코드표(설문용) 02의 STD_ID", "STD_ID가 설문에 쓸 수 있는 표준 항목이 아닙니다.")
		}
	}
}

func normLabel423(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return -1
		}
		return r
	}, s)
}

// labelMatch423 is 1 when label names code in set, -1 when it names another
// code of the set, and 0 when it names none.
func labelMatch423(set []std423Code, code, label string) int {
	l := normLabel423(label)
	if l == "" {
		return 0
	}
	for _, exact := range []bool{true, false} {
		own, other := false, false
		for _, s := range set {
			sl := normLabel423(s.Label)
			hit := l == sl
			if !exact {
				hit = strings.Contains(l, sl) || strings.Contains(sl, l)
			}
			if hit && s.Code == code {
				own = true
			} else if hit {
				other = true
			}
		}
		if own {
			return 1
		}
		if other {
			return -1
		}
	}
	return 0
}

func isReserved423(code string) bool { return code == "-9" || code == "-8" || code == "-7" }

// s11: a var with a STD_ID has the item's level, data type and codes.
func (c *checker) s11() {
	start, hasStart := date(strings.TrimSpace(c.cb.Study.Get("FIELD_START")))
	end, hasEnd := date(strings.TrimSpace(c.cb.Study.Get("FIELD_END")))
	for _, v := range c.cb.Vars {
		r := v.Rec
		item := std423Items[strings.TrimSpace(r.Get("STD_ID"))]
		if item.ID == "" {
			continue
		}
		if ml := r.Get("MEAS_LEVEL"); ml != item.Level {
			if ml == "해당 없음" && (item.ID == "SCHOOL.ID" || item.ID == "SCHOOL.KEDI_ID") {
				c.add("S11", kosac.I, r, "MEAS_LEVEL", ml, item.Level, item.ID+"의 측정수준 표기는 재단 확인이 필요합니다.")
			} else {
				c.add("S11", kosac.R, r, "MEAS_LEVEL", ml, item.Level, item.ID+"의 측정수준이 표준코드표와 다릅니다.")
			}
		}
		if typ := r.Get("DATA_TYPE"); typ != item.Type {
			c.add("S11", kosac.R, r, "DATA_TYPE", typ, item.Type, item.ID+"의 자료형이 표준코드표와 다릅니다.")
		}
		if item.ID == "SCHOOL.KEDI_ID" && !noteYearRe.MatchString(r.Get("CODE_SYSTEM")) {
			c.add("S11", kosac.W, r, "CODE_SYSTEM", r.Get("CODE_SYSTEM"), "기준연도", "KEDI 학교코드 변수의 CODE_SYSTEM에 기준연도가 없습니다.")
		}
		var set []std423Code
		for _, id := range item.Sets {
			set = append(set, std423Sets[id]...)
		}
		for _, cd := range c.codes[varKey(r)] {
			cr := cd.Rec
			code := cr.Get("CODE")
			if isReserved423(code) || blank(code) {
				continue
			}
			switch {
			case item.ID == "SCHOOL.ID" && strings.TrimSpace(c.cb.Study.Get("SCHOOL_CODE_YEAR")) == "2026":
				if _, ok := slices.BinarySearch(schoolCodes423, code); !ok {
					c.add("S11", kosac.R, cr, "CODE", code, "2026 NEIS 학교코드", "재단 표준코드표의 학교 목록에서 코드를 찾을 수 없습니다.")
				}
			case item.Pattern != nil:
				if !item.Pattern.MatchString(code) {
					c.add("S11", kosac.R, cr, "CODE", code, item.Pattern.String(), item.ID+"의 코드 자릿수가 표준과 다릅니다. 문자형으로 원래 형식을 유지하세요.")
				}
			case item.Max > 0:
				if n, err := strconv.Atoi(code); err != nil || n < item.Min || n > item.Max {
					c.add("S11", kosac.R, cr, "CODE", code, fmt.Sprintf("%d~%d", item.Min, item.Max), item.ID+"의 코드가 표준 범위 밖입니다.")
				}
			case len(set) > 0:
				i := slices.IndexFunc(set, func(s std423Code) bool { return s.Code == code })
				if i < 0 {
					expected := codeList423(set)
					msg := item.ID + "의 코드가 표준코드표(설문용) 02 목록에 없습니다."
					if j := slices.IndexFunc(set, func(s std423Code) bool { return sameValue(s.Code, code) }); j >= 0 {
						expected, msg = set[j].Code, item.ID+"의 코드 표기가 표준과 다릅니다. 앞자리 0을 보존하세요."
					}
					c.add("S11", kosac.R, cr, "CODE", code, expected, msg)
					continue
				}
				s := set[i]
				if s.End != "" && hasStart {
					if e, _ := date(s.End); start.After(e) {
						c.add("S11", kosac.R, cr, "CODE", code, s.End+"까지 사용", item.ID+"의 코드가 조사 시작일 전에 사용 종료되었습니다. 조사 시점 기준으로 코드를 고르세요.")
					}
				}
				if s.Start != "" && hasEnd {
					if b, _ := date(s.Start); end.Before(b) {
						c.add("S11", kosac.R, cr, "CODE", code, s.Start+"부터 사용", item.ID+"의 코드가 조사 종료일 뒤에 사용을 시작합니다. 조사 시점 기준으로 코드를 고르세요.")
					}
				}
				switch labelMatch423(set, code, cr.Get("CODE_LABEL")) {
				case -1:
					c.add("S11", kosac.R, cr, "CODE_LABEL", cr.Get("CODE_LABEL"), s.Label, item.ID+"의 코드 라벨이 다른 표준 코드의 뜻입니다. 원자료 코드가 다르면 표준이라고 선언하지 마세요.")
				case 0:
					c.add("S11", kosac.W, cr, "CODE_LABEL", cr.Get("CODE_LABEL"), s.Label, item.ID+"의 코드 라벨이 표준 라벨과 다릅니다. 뜻이 같은지 확인하세요.")
				}
			}
		}
	}
}

func codeList423(set []std423Code) string {
	var out []string
	for _, s := range set {
		out = append(out, s.Code)
	}
	return strings.Join(out, ", ")
}

// s12: a var without STD_ID whose codes and labels are a standard set's and
// whose label names the item.
func (c *checker) s12() {
	ids := make([]string, 0, len(std423Items))
	for id := range std423Items {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, v := range c.cb.Vars {
		r := v.Rec
		if !blank(r.Get("STD_ID")) {
			continue
		}
		var codes []*kosac.Code
		for _, cd := range c.codes[varKey(r)] {
			if !isReserved423(cd.Rec.Get("CODE")) {
				codes = append(codes, cd)
			}
		}
		if len(codes) < 2 {
			continue
		}
		best := ""
		for _, id := range ids {
			item := std423Items[id]
			if len(item.Sets) == 0 || !strings.Contains(normLabel423(r.Get("VAR_LABEL")), normLabel423(item.Name)) {
				continue
			}
			var set []std423Code
			for _, s := range item.Sets {
				set = append(set, std423Sets[s]...)
			}
			if !slices.ContainsFunc(codes, func(cd *kosac.Code) bool {
				return labelMatch423(set, cd.Rec.Get("CODE"), cd.Rec.Get("CODE_LABEL")) != 1
			}) && (best == "" || len(item.Name) > len(std423Items[best].Name)) {
				best = id
			}
		}
		if best != "" {
			c.add("S12", kosac.W, r, "STD_ID", "", best, "코드와 라벨이 표준 코드셋과 같습니다. STD_ID "+best+"를 쓰는 것을 권합니다.")
		}
	}
}

func surveyTimes423(s string) (pre, post bool) {
	return strings.Contains(s, "사전"), strings.Contains(s, "사후")
}

// s13: data linking one person's pre and post answers identifies the
// participant number and the survey wave. A file of one wave may take the
// wave from its SURVEY_TIME.
func (c *checker) s13() {
	byFile := map[string][]*kosac.Var{}
	for _, v := range c.cb.Vars {
		byFile[v.Rec.Get("FILE_NO")] = append(byFile[v.Rec.Get("FILE_NO")], v)
	}
	linkedFrom := map[string][]string{}
	for _, f := range c.cb.Files {
		if lf := strings.TrimSpace(f.Rec.Get("LINK_FILE_NO")); lf != "" {
			linkedFrom[lf] = append(linkedFrom[lf], f.Rec.Get("FILE_NO"))
		}
	}
	for _, f := range c.cb.Files {
		r := f.Rec
		no := r.Get("FILE_NO")
		role := func(want string) bool {
			return slices.ContainsFunc(byFile[no], func(v *kosac.Var) bool { return strings.TrimSpace(v.Rec.Get("KEY_ROLE")) == want })
		}
		timeVar := filled423(r.Get("TIME_VAR"))
		for _, name := range refNames423(r.Get("TIME_VAR")) {
			if c.vars[key{no, name}] == nil {
				c.add("S13", kosac.R, r, "TIME_VAR", name, "같은 파일의 변수명", "조사 시점 변수가 같은 파일에 없습니다.")
			}
		}
		wave := slices.ContainsFunc(byFile[no], func(v *kosac.Var) bool { return strings.TrimSpace(v.Rec.Get("STD_ID")) == "SURVEY.WAVE" })
		pre, post := surveyTimes423(r.Get("SURVEY_TIME"))
		partners := linkedFrom[no]
		if lf := strings.TrimSpace(r.Get("LINK_FILE_NO")); lf != "" {
			partners = append(partners, lf)
		}
		prepostLink := false
		for _, p := range partners {
			if pf := c.cb.File(p); pf != nil {
				a, b := surveyTimes423(pf.Rec.Get("SURVEY_TIME"))
				prepostLink = prepostLink || a || b
			}
		}
		prepostLink = len(partners) > 0 && (prepostLink || pre || post)
		combined := pre && post || timeVar || wave || role("조사 시점")
		if !combined && !prepostLink {
			continue
		}
		if !role("참여번호") {
			c.add("S13", kosac.R, r, "FILE_NO", no, "KEY_ROLE=참여번호 변수", "사전·사후 응답을 연결하는 파일에 참여번호 변수가 없습니다.")
		}
		hasTime := timeVar || role("조사 시점") || !(pre && post) && filled423(r.Get("SURVEY_TIME"))
		if !hasTime {
			c.add("S13", kosac.R, r, "TIME_VAR", r.Get("TIME_VAR"), "조사 시점 변수 또는 파일의 SURVEY_TIME", "사전·사후 응답을 연결하는 파일에서 조사 시점을 식별할 수 없습니다.")
		}
	}
	for _, v := range c.cb.Vars {
		if strings.TrimSpace(v.Rec.Get("STD_ID")) == "SURVEY.WAVE" && strings.TrimSpace(v.Rec.Get("KEY_ROLE")) != "조사 시점" {
			c.add("S13", kosac.W, v.Rec, "KEY_ROLE", v.Rec.Get("KEY_ROLE"), "조사 시점", "STD_ID=SURVEY.WAVE 변수의 KEY_ROLE이 조사 시점이 아닙니다.")
		}
	}
}

func refNames423(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == ',' || r == ';' || r == '+' })
}

// s14: the row key and the file links, resolved as the export resolves them.
func (c *checker) s14() {
	for _, f := range c.cb.Files {
		r := f.Rec
		no := r.Get("FILE_NO")
		exists := func(k, file string, names []string) {
			for _, name := range names {
				if c.vars[key{file, name}] == nil {
					c.add("S14", kosac.R, r, k, name, "FILE_NO "+file+"의 변수명", k+"의 변수가 해당 파일의 03_변수정의 행에 없습니다.")
				}
			}
		}
		uk, id := refNames423(r.Get("UNIQUE_KEY")), refNames423(r.Get("ID_VAR"))
		if len(uk) == 0 && len(id) == 0 {
			c.add("S14", kosac.R, r, "UNIQUE_KEY", "", "UNIQUE_KEY 또는 ID_VAR", "행 식별 키(UNIQUE_KEY 또는 ID_VAR)가 없습니다.")
		}
		exists("UNIQUE_KEY", no, uk)
		exists("ID_VAR", no, id)
		lf, lv, lt := r.Get("LINK_FILE_NO"), r.Get("LINK_VAR"), r.Get("LINK_TARGET_VAR")
		if blank(lf) && blank(lv) && blank(lt) {
			continue
		}
		for _, k := range []string{"LINK_FILE_NO", "LINK_VAR", "LINK_TARGET_VAR"} {
			if blank(r.Get(k)) {
				c.add("S14", kosac.R, r, k, "", "연결 설정 시 필수", "연결을 설정하면 LINK_FILE_NO, LINK_VAR, LINK_TARGET_VAR를 모두 적어야 합니다.")
			}
		}
		src, dst := refNames423(lv), refNames423(lt)
		exists("LINK_VAR", no, src)
		if !blank(lf) && c.cb.File(lf) == nil {
			c.add("S14", kosac.R, r, "LINK_FILE_NO", lf, "02_데이터파일의 FILE_NO", "LINK_FILE_NO의 파일이 02_데이터파일 시트에 없습니다.")
		} else if !blank(lf) {
			exists("LINK_TARGET_VAR", lf, dst)
		}
		if len(src) > 0 && len(dst) > 0 && len(src) != len(dst) {
			c.add("S14", kosac.R, r, "LINK_TARGET_VAR", strconv.Itoa(len(dst)), strconv.Itoa(len(src)), "LINK_VAR와 LINK_TARGET_VAR의 변수 개수가 다릅니다.")
		}
	}
}

// s15: UNIVERSE_EXPR grammar and its vars.
func (c *checker) s15() {
	for _, v := range c.cb.Vars {
		r := v.Rec
		expr := r.Get("UNIVERSE_EXPR")
		if !filled423(expr) {
			continue
		}
		names, err := parseUniverse423(expr)
		if err != nil {
			c.add("S15", kosac.R, r, "UNIVERSE_EXPR", err.Error(), "변수 = 값, 변수 in (값, …), AND·OR", "UNIVERSE_EXPR 문법이 맞지 않습니다.")
			continue
		}
		for _, name := range names {
			if c.vars[key{r.Get("FILE_NO"), name}] == nil {
				c.add("S15", kosac.R, r, "UNIVERSE_EXPR", name, "같은 파일의 변수", "UNIVERSE_EXPR가 참조한 변수가 같은 파일에 없습니다.")
			}
		}
	}
}

// codeSig423 is a var's code set as code=label pairs in sheet order.
func (c *checker) codeSig423(v *kosac.Var) string {
	var parts []string
	for _, cd := range c.codes[varKey(v.Rec)] {
		parts = append(parts, cd.Rec.Get("CODE")+"="+strings.TrimSpace(cd.Rec.Get("CODE_LABEL")))
	}
	slices.Sort(parts)
	return strings.Join(parts, "; ")
}

// s16: vars of one ITEM_KEY share a code set.
func (c *checker) s16() {
	first := map[string]*kosac.Var{}
	for _, v := range c.cb.Vars {
		k := strings.TrimSpace(v.Rec.Get("ITEM_KEY"))
		if k == "" {
			continue
		}
		if f := first[k]; f == nil {
			first[k] = v
		} else if a, b := c.codeSig423(v), c.codeSig423(f); a != b {
			c.add("S16", kosac.W, v.Rec, "ITEM_KEY", k, b, "같은 ITEM_KEY 변수 "+f.Rec.Get("VAR_NAME")+"와 코드셋이 다릅니다.")
		}
	}
}

// s17: vars of one SCALE subdomain are ordinal and share a code set.
func (c *checker) s17() {
	first := map[string]*kosac.Var{}
	for _, v := range c.cb.Vars {
		k := strings.TrimSpace(v.Rec.Get("SCALE"))
		if k == "" {
			continue
		}
		if ml := v.Rec.Get("MEAS_LEVEL"); ml != "서열" {
			c.add("S17", kosac.W, v.Rec, "MEAS_LEVEL", ml, "서열", "SCALE 하위 영역 변수의 측정수준이 서열이 아닙니다.")
		}
		if f := first[k]; f == nil {
			first[k] = v
		} else if a, b := c.codeSig423(v), c.codeSig423(f); a != b {
			c.add("S17", kosac.W, v.Rec, "SCALE", k, b, "같은 SCALE 하위 영역 변수 "+f.Rec.Get("VAR_NAME")+"와 코드셋이 다릅니다.")
		}
	}
}

// parseUniverse423 parses "VAR = v", "VAR != v", "VAR in (v, …)" joined by
// AND/OR, with optional grouping parentheses, and returns the vars in order.
func parseUniverse423(s string) ([]string, error) {
	toks, err := lex423(s)
	if err != nil {
		return nil, err
	}
	p := &uparser423{toks: toks}
	if err := p.expr(); err != nil {
		return nil, err
	}
	if p.pos < len(p.toks) {
		return nil, fmt.Errorf("조건 뒤에는 AND나 OR만 올 수 있습니다 (%q)", p.toks[p.pos])
	}
	return p.vars, nil
}

func lex423(s string) ([]string, error) {
	var toks []string
	rs := []rune(s)
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case unicode.IsSpace(r):
			i++
		case r == '(' || r == ')' || r == ',' || r == '=':
			toks = append(toks, string(r))
			i++
		case r == '!':
			if i+1 >= len(rs) || rs[i+1] != '=' {
				return nil, fmt.Errorf("%d번째 글자: ! 뒤에 =가 와야 합니다", i+1)
			}
			toks = append(toks, "!=")
			i += 2
		case r == '\'' || r == '"':
			j := i + 1
			for j < len(rs) && rs[j] != r {
				j++
			}
			if j >= len(rs) {
				return nil, fmt.Errorf("%d번째 글자: 따옴표가 닫히지 않았습니다", i+1)
			}
			toks = append(toks, string(rs[i:j+1]))
			i = j + 1
		default:
			j := i
			for j < len(rs) && !unicode.IsSpace(rs[j]) && !strings.ContainsRune("(),=!'\"", rs[j]) {
				j++
			}
			toks = append(toks, string(rs[i:j]))
			i = j
		}
	}
	return toks, nil
}

type uparser423 struct {
	toks []string
	pos  int
	vars []string
}

func (p *uparser423) peek() string {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return ""
}
func (p *uparser423) next() string { t := p.peek(); p.pos++; return t }
func (p *uparser423) kw(w string) bool {
	if strings.EqualFold(p.peek(), w) {
		p.pos++
		return true
	}
	return false
}

func (p *uparser423) expr() error {
	for {
		if err := p.term(); err != nil {
			return err
		}
		if !p.kw("AND") && !p.kw("OR") {
			return nil
		}
	}
}

func (p *uparser423) term() error {
	if p.peek() == "(" {
		p.pos++
		if err := p.expr(); err != nil {
			return err
		}
		if p.next() != ")" {
			return fmt.Errorf("괄호가 닫히지 않았습니다")
		}
		return nil
	}
	name := p.next()
	if !wordRe.MatchString(name) || strings.EqualFold(name, "AND") || strings.EqualFold(name, "OR") || strings.EqualFold(name, "IN") {
		return fmt.Errorf("변수 이름이 와야 합니다 (%q)", name)
	}
	if !slices.Contains(p.vars, name) {
		p.vars = append(p.vars, name)
	}
	switch op := p.next(); {
	case op == "=" || op == "!=":
		return p.value()
	case strings.EqualFold(op, "in"):
		if p.next() != "(" {
			return fmt.Errorf("%s in 뒤에 (가 와야 합니다", name)
		}
		for {
			if err := p.value(); err != nil {
				return err
			}
			switch p.next() {
			case ",":
			case ")":
				return nil
			default:
				return fmt.Errorf("in 목록은 쉼표로 나누고 )로 닫아야 합니다")
			}
		}
	default:
		return fmt.Errorf("%s 뒤에 =, != 또는 in이 와야 합니다", name)
	}
}

func (p *uparser423) value() error {
	v := p.next()
	if v == "" || strings.ContainsAny(v, "(),=") || v == "!=" || strings.EqualFold(v, "AND") || strings.EqualFold(v, "OR") {
		return fmt.Errorf("값이 없습니다")
	}
	return nil
}

package check

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/universe"
)

// studyLists are the 01 sheet's dropdown values.
var studyLists = map[string][]string{
	"RULE_VERSION":  {"v1.0"},
	"ANALYSIS_UNIT": {"개인", "학교", "기관", "기타"},
	"SAMPLING_TYPE": {"확률-단순무작위", "확률-층화", "확률-집락", "확률-다단계", "할당(비확률)", "편의(비확률)", "전수", "기타"},
	"SPLIT_BALLOT":  {"Y", "N"},
	"MODE":          {"온라인", "전화", "대면", "우편", "혼합", "기타"},
	"WEIGHT_YN":     {"Y", "N"},
	"SURVEY_TYPE":   {"연구조사", "사업성과"},
}

var (
	mrTypes   = []string{"보기별 열", "순위별 열"}
	fileUnits = []string{"개인", "학교", "기관", "기타"}
	// PII, SENSITIVE and REVERSE go into kosac attributes that the XSD
	// enumerates exactly, so the values are compared untrimmed.
	yn = []string{"Y", "N"}
)

var (
	// A number cell formatted with a thousands separator reads as "10,000";
	// assembly drops the commas.
	integerRe = regexp.MustCompile(`^(\d+|\d{1,3}(,\d{3})+)$`)
	decimalRe = regexp.MustCompile(`^\d+(\.\d+)?$`)
	dateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	yearRe    = regexp.MustCompile(`^\d{4}$`)
	// DATA_NO goes into codeBook@ID (xs:ID), whose form xmllint does not
	// check on the root.
	dataNoRe   = regexp.MustCompile(`^[\p{L}\p{N}._-]+$`)
	groupNoRe  = regexp.MustCompile(`^(문항|[Qq])\s*\d+`)
	wordRe     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	codeRe     = regexp.MustCompile(`^-?\d+$`)
	tokenRe    = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*`)
	instRe     = regexp.MustCompile(`학교\s*(ID|코드)|기관\s*코드`)
	noteYearRe = regexp.MustCompile(`(^|\D)\d{4}(\D|$)`)
)

func integer(s string) (int, bool) {
	if !integerRe.MatchString(s) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	return n, err == nil
}

func date(s string) (time.Time, bool) {
	t, err := time.Parse(time.DateOnly, s)
	return t, err == nil && dateRe.MatchString(s)
}

func unverifiedMsg(k string) string { return k + "의 값이 확인 불가로 적혀 있습니다." }

// c01: required 01 values. A condition whose value is "확인 불가" is not
// judged, so it requires nothing.
func (c *checker) c01() {
	st := c.cb.Study
	when := map[string]string{} // conditional KEY → the condition that holds
	if s := st.Get("SAMPLING_TYPE"); checkable(s) {
		if s == "확률-층화" || s == "할당(비확률)" {
			when["STRATA_VARS"] = "SAMPLING_TYPE=" + s
		}
		if s == "확률-집락" || s == "확률-다단계" {
			when["CLUSTER_VAR"] = "SAMPLING_TYPE=" + s
		}
		if strings.HasPrefix(s, "확률-") {
			when["CONTACT_N"] = "SAMPLING_TYPE=" + s
			when["RESPONSE_RATE"] = "SAMPLING_TYPE=" + s
		}
	}
	if checkable(st.Get("RESPONSE_RATE")) {
		when["RR_FORMULA"] = "RESPONSE_RATE 있음"
	}
	if n, ok := integer(st.Get("EXCLUDE_N")); ok && n >= 1 {
		when["EXCLUDE_RULE"] = "EXCLUDE_N ≥ 1"
	}
	if slices.ContainsFunc(c.cb.Vars, func(v *kosac.Var) bool { return v.Rec.Get("STD_ID") == "SCHOOL.ID" }) {
		when["SCHOOL_CODE_YEAR"] = "STD_ID=SCHOOL.ID인 변수 있음"
	}
	if st.Get("SURVEY_TYPE") == "사업성과" {
		when["BIZ_NAME"] = "SURVEY_TYPE=사업성과"
		when["BIZ_YEAR"] = "SURVEY_TYPE=사업성과"
	}
	for _, k := range kosac.StudyKeys {
		v := st.Get(k)
		switch {
		case unverified(v):
			c.add("C01", kosac.W, st, k, v, "", unverifiedMsg(k))
		case !blank(v):
		case kosac.StudyRequired[k] == "필수":
			c.add("C01", kosac.R, st, k, v, "필수", "필수 항목 "+k+"의 값이 비어 있습니다.")
		case when[k] != "":
			c.add("C01", kosac.R, st, k, v, when[k]+"일 때 필수", "조건부 필수 항목 "+k+"의 값이 비어 있습니다.")
		}
	}
}

// c02: 01 value formats.
func (c *checker) c02() {
	st := c.cb.Study
	for _, k := range kosac.StudyKeys {
		v := st.Get(k)
		if !checkable(v) {
			continue
		}
		switch k {
		case "FIELD_START", "FIELD_END":
			if _, ok := date(v); !ok {
				c.add("C02", kosac.R, st, k, v, "YYYY-MM-DD", k+"의 값이 실제 날짜(YYYY-MM-DD)가 아닙니다.")
			}
		case "TARGET_N", "VALID_N", "CONTACT_N", "EXCLUDE_N":
			if _, ok := integer(v); !ok {
				c.add("C02", kosac.R, st, k, v, "정수", k+"의 값은 정수여야 합니다.")
			}
		case "RESPONSE_RATE":
			if !decimalRe.MatchString(v) {
				c.add("C02", kosac.R, st, k, v, "수", k+"의 값은 수여야 합니다.")
			}
		case "BIZ_YEAR", "SCHOOL_CODE_YEAR":
			if !yearRe.MatchString(v) {
				c.add("C02", kosac.R, st, k, v, "YYYY", k+"의 값은 네 자리 연도(YYYY)여야 합니다.")
			}
		case "DATA_NO":
			if !dataNoRe.MatchString(v) {
				c.add("C02", kosac.R, st, k, v, "글자, 숫자, '.', '_', '-'", "DATA_NO에는 글자, 숫자, 마침표, 밑줄, 하이픈만 쓸 수 있습니다.")
			}
		default:
			if l := studyLists[k]; l != nil && !slices.Contains(l, v) {
				c.add("C02", kosac.R, st, k, v, strings.Join(l, ", "), k+"의 값이 양식 목록에 없습니다.")
			}
		}
	}
	start, ok1 := date(st.Get("FIELD_START"))
	end, ok2 := date(st.Get("FIELD_END"))
	if ok1 && ok2 && end.Before(start) {
		c.add("C02", kosac.R, st, "FIELD_END", st.Get("FIELD_END"), "FIELD_START 이후", "조사 종료일(FIELD_END)이 시작일(FIELD_START)보다 앞섭니다.")
	}
}

// c03: the weight.
func (c *checker) c03() {
	st := c.cb.Study
	wv := st.Get("WEIGHT_VAR")
	switch st.Get("WEIGHT_YN") {
	case "Y":
		if blank(wv) {
			c.add("C03", kosac.R, st, "WEIGHT_VAR", wv, "가중치 변수", "가중치를 적용했으면 WEIGHT_VAR의 값이 있어야 합니다.")
		} else if !unverified(wv) {
			if lv, ok := c.studyLevel(wv); !ok {
				c.add("C03", kosac.R, st, "WEIGHT_VAR", wv, "03_변수정의의 변수", "WEIGHT_VAR의 변수가 03_변수정의 시트에 없습니다.")
			} else if lv != "scale" {
				c.add("C03", kosac.R, st, "WEIGHT_VAR", wv, "연속", "가중치 변수의 측정수준은 연속이어야 합니다.")
			}
		}
		if wm := st.Get("WEIGHT_METHOD"); blank(wm) {
			c.add("C03", kosac.R, st, "WEIGHT_METHOD", wm, "가중치 산출 방법", "가중치를 적용했으면 WEIGHT_METHOD의 값이 있어야 합니다.")
		}
	case "N":
		if checkable(wv) {
			c.add("C03", kosac.R, st, "WEIGHT_VAR", wv, "빈 값", "가중치를 적용하지 않았으면 WEIGHT_VAR는 비어 있어야 합니다.")
		}
	default:
		// Assembly resolves the name whatever WEIGHT_YN says.
		if _, ok := c.studyLevel(wv); checkable(wv) && !ok {
			c.add("C03", kosac.R, st, "WEIGHT_VAR", wv, "03_변수정의의 변수", "WEIGHT_VAR의 변수가 03_변수정의 시트에 없습니다.")
		}
	}
}

// c04: the split-ballot, strata and cluster vars. The 01 sheet names no file,
// so a name in any file counts. Assembly resolves SPLIT_VAR whatever
// SPLIT_BALLOT says, so a written one must exist even under N.
func (c *checker) c04() {
	st := c.cb.Study
	split, sv := st.Get("SPLIT_BALLOT") == "Y", st.Get("SPLIT_VAR")
	switch lv, ok := c.studyLevel(sv); {
	case split && blank(sv):
		c.add("C04", kosac.R, st, "SPLIT_VAR", sv, "분할 변수", "분할 설문이면 SPLIT_VAR의 값이 있어야 합니다.")
	case !checkable(sv):
	case !ok:
		c.add("C04", kosac.R, st, "SPLIT_VAR", sv, "03_변수정의의 변수", "SPLIT_VAR의 변수가 03_변수정의 시트에 없습니다.")
	case split && lv != "nominal":
		c.add("C04", kosac.R, st, "SPLIT_VAR", sv, "명목", "분할 변수의 측정수준은 명목이어야 합니다.")
	}
	for _, k := range []string{"STRATA_VARS", "CLUSTER_VAR"} {
		v := st.Get(k)
		if !checkable(v) {
			continue
		}
		// kosac:sampling@clusterVar holds one var, so CLUSTER_VAR is one name.
		names := []string{strings.TrimSpace(v)}
		if k == "STRATA_VARS" {
			names = strings.FieldsFunc(v, func(r rune) bool { return r == ';' || r == ',' || unicode.IsSpace(r) })
		}
		for _, name := range names {
			if _, ok := c.studyLevel(name); !ok {
				c.add("C04", kosac.R, st, k, name, "03_변수정의의 변수", k+"에 적은 변수가 03_변수정의 시트에 없습니다.")
			}
		}
	}
}

// c05: the 02 sheet.
func (c *checker) c05() {
	n := len(c.cb.Files)
	seenNo, seenName := map[int]bool{}, map[string]bool{}
	for _, f := range c.cb.Files {
		r := f.Rec
		// Assembly's own test, so what passes here assembles. A "확인 불가"
		// FILE_NO fails it too: a file cannot be assembled without its number.
		no := r.Get("FILE_NO")
		i, _ := strconv.Atoi(no)
		if !kosac.ValidFileNo(no) || i > n || seenNo[i] {
			c.add("C05", kosac.R, r, "FILE_NO", no, fmt.Sprintf("1~%d", n), "FILE_NO는 1부터 빠짐없이 이어지는 번호여야 합니다.")
		}
		seenNo[i] = true
		if name := r.Get("FILE_NAME"); c.ref.UploadedFiles != nil && !unverified(name) {
			if !slices.Contains(c.ref.UploadedFiles, name) {
				c.add("C05", kosac.R, r, "FILE_NAME", name, "업로드한 파일명", "FILE_NAME의 파일이 업로드한 파일 중에 없습니다.")
			} else if seenName[name] {
				c.add("C05", kosac.R, r, "FILE_NAME", name, "파일마다 다른 파일", "같은 FILE_NAME이 두 번 이상 있습니다.")
			}
			seenName[name] = true
		}
		if ff := r.Get("FILE_FORMAT"); !unverified(ff) && ff != "CSV" {
			c.add("C05", kosac.R, r, "FILE_FORMAT", ff, "CSV", "FILE_FORMAT의 값은 CSV여야 합니다.")
		}
		if vn := r.Get("VALID_N"); checkable(vn) {
			if _, ok := integer(vn); !ok {
				c.add("C05", kosac.R, r, "VALID_N", vn, "정수", "VALID_N의 값은 정수여야 합니다.")
			}
		}
		if u := r.Get("UNIT"); checkable(u) && !slices.Contains(fileUnits, u) {
			c.add("C05", kosac.R, r, "UNIT", u, strings.Join(fileUnits, ", "), "UNIT의 값이 양식 목록에 없습니다.")
		}
		if iv := r.Get("ID_VAR"); !unverified(iv) {
			if _, ok := c.level(no, iv); !ok {
				c.add("C05", kosac.R, r, "ID_VAR", iv, "이 파일의 변수", "ID_VAR의 변수가 이 파일의 03_변수정의 행에 없습니다.")
			}
		}
		lv, lf := r.Get("LINK_VAR"), r.Get("LINK_FILE_NO")
		link := checkable(lv)
		if _, ok := c.level(no, lv); link && !ok {
			c.add("C05", kosac.R, r, "LINK_VAR", lv, "이 파일의 변수", "LINK_VAR의 변수가 이 파일의 03_변수정의 행에 없습니다.")
		}
		// Assembly resolves LINK_FILE_NO whenever it is written, LINK_VAR or not.
		switch {
		case unverified(lf) || blank(lf) && !link:
		case c.cb.File(lf) == nil:
			c.add("C05", kosac.R, r, "LINK_FILE_NO", lf, "02_데이터파일의 FILE_NO", "LINK_FILE_NO의 파일이 02_데이터파일 시트에 없습니다.")
		case link:
			if _, ok := c.level(lf, lv); !ok {
				c.add("C05", kosac.R, r, "LINK_VAR", lv, "연결 파일의 변수", "LINK_VAR와 같은 이름의 변수가 연결 파일에 없습니다.")
			}
		}
	}
}

// c06: 03 rows.
func (c *checker) c06() {
	seen := map[key]bool{}
	for _, v := range c.cb.Vars {
		r := v.Rec
		k := varKey(r)
		if seen[k] && !blank(k.name) {
			c.add("C06", kosac.R, r, "VAR_NAME", k.name, "파일 안에서 유일한 이름", "같은 파일에 VAR_NAME이 두 번 이상 있습니다.")
		}
		seen[k] = true
		// A "확인 불가" FILE_NO too: assembly cannot place a var without its
		// file. Naming a file C05 passes is enough; its spelling is checked there.
		if no := r.Get("FILE_NO"); c.cb.File(no) == nil {
			c.add("C06", kosac.R, r, "FILE_NO", no, "02_데이터파일의 FILE_NO", "FILE_NO의 파일이 02_데이터파일 시트에 없습니다.")
		}
		for _, rk := range []string{"VAR_NAME", "VAR_LABEL", "Q_TEXT", "VAR_GROUP", "MEAS_LEVEL", "PII", "SENSITIVE"} {
			if blank(r.Get(rk)) {
				c.add("C06", kosac.R, r, rk, r.Get(rk), "필수", "필수 열 "+rk+"의 값이 비어 있습니다.")
			}
		}
		for _, rk := range []string{"PII", "SENSITIVE"} {
			if x := r.Get(rk); checkable(x) && !slices.Contains(yn, x) {
				c.add("C06", kosac.R, r, rk, x, "Y, N", rk+"의 값은 Y나 N이어야 합니다.")
			}
		}
	}
}

// c07: measurement levels, and enough codes for a categorical var. Missing
// codes do not count, since they are not answers.
func (c *checker) c07() {
	for _, v := range c.cb.Vars {
		ml := v.Rec.Get("MEAS_LEVEL")
		if !checkable(ml) {
			continue
		}
		if _, ok := kosac.MeasLevels[ml]; !ok {
			c.add("C07", kosac.R, v.Rec, "MEAS_LEVEL", ml, "명목, 서열, 연속, 문자, 날짜", "MEAS_LEVEL의 값이 양식 목록에 없습니다.")
			continue
		}
		if _, valid := c.codesOf(v); (ml == "명목" || ml == "서열") && len(valid) < 2 {
			c.add("C07", kosac.R, v.Rec, "VAR_NAME", strconv.Itoa(len(valid)), "2 이상", "명목·서열 변수에는 결측 코드가 아닌 코드가 두 개 이상 있어야 합니다.")
		}
	}
}

// c08: var labels, one issue per condition.
func (c *checker) c08() {
	for _, v := range c.cb.Vars {
		r := v.Rec
		l := r.Get("VAR_LABEL")
		if !checkable(l) {
			continue
		}
		if n := utf8.RuneCountInString(l); n > 30 {
			c.add("C08", kosac.W, r, "VAR_LABEL", strconv.Itoa(n), "30 이하", "변수 라벨이 30자를 넘습니다.")
		}
		if l == r.Get("Q_TEXT") {
			c.add("C08", kosac.W, r, "VAR_LABEL", l, "문항과 다른 짧은 라벨", "변수 라벨이 문항 내용(Q_TEXT)과 같습니다.")
		}
		if l == r.Get("VAR_NAME") || wordRe.MatchString(l) {
			c.add("C08", kosac.W, r, "VAR_LABEL", l, "뜻을 알 수 있는 라벨", "변수 라벨이 영문 한 단어이거나 변수명과 같습니다.")
		}
	}
}

// c09: section names. A name repeats on every row of its section, so each
// distinct name is reported once, on its first row.
func (c *checker) c09() {
	seen := map[string]bool{}
	for _, v := range c.cb.Vars {
		r := v.Rec
		g := r.Get("VAR_GROUP")
		if !checkable(g) || seen[g] {
			continue
		}
		seen[g] = true
		if groupNoRe.MatchString(g) {
			c.add("C09", kosac.R, r, "VAR_GROUP", g, "영역 이름", "VAR_GROUP은 '문항 n'이나 'Q n' 같은 번호가 아니라 영역 이름이어야 합니다.")
		}
		if n := utf8.RuneCountInString(g); n > 20 {
			c.add("C09", kosac.R, r, "VAR_GROUP", strconv.Itoa(n), "20 이하", "VAR_GROUP은 20자 이하여야 합니다.")
		}
	}
}

// sameValue compares as the universe evaluator does: numbers by value.
func sameValue(a, b string) bool {
	x, err1 := strconv.ParseFloat(a, 64)
	y, err2 := strconv.ParseFloat(b, 64)
	return a == b || err1 == nil && err2 == nil && x == y
}

// c10: response universes. Values are compared only with a var that has
// answer codes, and range bounds are not, since a range may run past them.
// A missing code is a value every coded var can hold.
func (c *checker) c10() {
	for _, v := range c.cb.Vars {
		r := v.Rec
		u := r.Get("RESP_UNIVERSE")
		if !checkable(u) {
			continue
		}
		e, err := universe.Parse(u)
		if err != nil {
			c.add("C10", kosac.R, r, "RESP_UNIVERSE", err.Error(), "변수=값[,값…] 또는 변수=하한~상한, AND/OR로 결합", "응답대상 문법이 맞지 않습니다.")
			continue
		}
		for _, name := range e.Vars() {
			target := c.vars[key{r.Get("FILE_NO"), name}]
			if target == nil {
				if !kosac.IsSystemVar(name) {
					c.add("C10", kosac.R, r, "RESP_UNIVERSE", name, "같은 파일의 변수", "응답대상이 참조한 변수가 같은 파일에 없습니다.")
				}
				continue
			}
			all, valid := c.codesOf(target)
			if len(valid) == 0 {
				continue
			}
			var list []string
			for _, cd := range all {
				list = append(list, cd.Rec.Get("CODE"))
			}
			for _, val := range e.Values(name) {
				if !isMissingCode(val) && !slices.ContainsFunc(list, func(code string) bool { return sameValue(val, code) }) {
					c.add("C10", kosac.R, r, "RESP_UNIVERSE", name+"="+val, strings.Join(list, ", "), "응답대상의 값이 참조 변수의 코드값에 없습니다.")
				}
			}
		}
	}
}

// group is a question's columns: the same file, section and name prefix.
type group struct{ file, section, prefix string }

func groupOf(r kosac.Record) (group, bool) {
	p, ok := kosac.Prefix(r.Get("VAR_NAME"))
	return group{r.Get("FILE_NO"), r.Get("VAR_GROUP"), p}, ok
}

// c11: multiple-response columns. Only nominal vars must share the group's
// MR_TYPE, so a text column such as Q5_other does not break the group.
func (c *checker) c11() {
	first := map[group]*kosac.Var{}
	for _, v := range c.cb.Vars {
		r := v.Rec
		mt := r.Get("MR_TYPE")
		if checkable(mt) && !slices.Contains(mrTypes, mt) {
			c.add("C11", kosac.R, r, "MR_TYPE", mt, strings.Join(mrTypes, ", "), "MR_TYPE의 값이 양식 목록에 없습니다.")
		}
		if mt == "보기별 열" {
			_, valid := c.codesOf(v)
			for _, cd := range valid {
				if code := cd.Rec.Get("CODE"); code != "0" && code != "1" {
					c.add("C11", kosac.R, cd.Rec, "CODE", code, "0, 1", "보기별 열 변수의 코드는 0과 1이어야 합니다.")
				}
			}
		}
		g, ok := groupOf(r)
		if !ok || r.Get("MEAS_LEVEL") != "명목" || unverified(mt) {
			continue
		}
		if f := first[g]; f == nil {
			first[g] = v
		} else if fm := f.Rec.Get("MR_TYPE"); mt != fm {
			c.add("C11", kosac.R, r, "MR_TYPE", mt, fm, "같은 묶음의 명목 변수끼리 MR_TYPE이 다릅니다.")
		}
	}
}

// c12: reverse-coded vars. A reversed var with no plain ordinal sibling in
// its question suggests the whole scale was marked, not the reversed items.
func (c *checker) c12() {
	plain := map[group]bool{}
	for _, v := range c.cb.Vars {
		r := v.Rec
		if rv := r.Get("REVERSE"); r.Get("MEAS_LEVEL") == "서열" && rv != "Y" && !unverified(rv) {
			if g, ok := groupOf(r); ok {
				plain[g] = true
			}
		}
	}
	for _, v := range c.cb.Vars {
		r := v.Rec
		if rv := r.Get("REVERSE"); checkable(rv) && !slices.Contains(yn, rv) {
			c.add("C12", kosac.R, r, "REVERSE", rv, "Y, N", "REVERSE의 값은 비어 있거나 Y나 N이어야 합니다.")
		}
		if r.Get("REVERSE") != "Y" {
			continue
		}
		if ml := r.Get("MEAS_LEVEL"); checkable(ml) && ml != "서열" {
			c.add("C12", kosac.W, r, "MEAS_LEVEL", ml, "서열", "역코딩 변수(REVERSE=Y)의 측정수준이 서열이 아닙니다.")
		}
		if g, ok := groupOf(r); ok && !plain[g] {
			c.add("C12", kosac.W, r, "REVERSE", "Y", "같은 묶음에 역코딩하지 않은 서열 변수", "같은 묶음에 역코딩하지 않은 서열 변수가 없어 역코딩 표시를 확인해야 합니다.")
		}
	}
}

// c13: derived vars. Only tokens with a digit or "_" are taken for var names,
// so words such as PCA or NA are not.
func (c *checker) c13() {
	for _, v := range c.cb.Vars {
		r := v.Rec
		d := r.Get("DERIVATION")
		if blank(d) {
			continue
		}
		if q := r.Get("Q_TEXT"); checkable(q) && q != "파생" {
			c.add("C13", kosac.R, r, "Q_TEXT", q, "파생", "파생 변수의 Q_TEXT는 '파생'이어야 합니다.")
		}
		if unverified(d) {
			continue
		}
		for _, t := range tokenRe.FindAllString(d, -1) {
			if !strings.ContainsAny(t, "0123456789_") {
				continue
			}
			if _, ok := c.level(r.Get("FILE_NO"), t); !ok {
				c.add("C13", kosac.R, r, "DERIVATION", t, "같은 파일의 변수", "산출식에 적은 변수가 같은 파일에 없습니다.")
			}
		}
	}
}

// numbersOnly reports whether a label is only numbers (①, 3) and marks.
func numbersOnly(s string) bool {
	num := false
	for _, r := range s {
		switch {
		case unicode.IsNumber(r):
			num = true
		case unicode.IsSpace(r) || unicode.IsPunct(r):
		default:
			return false
		}
	}
	return num
}

// c14: 04 rows.
func (c *checker) c14() {
	seen := map[[3]string]bool{}
	for _, cd := range c.cb.Codes {
		r := cd.Rec
		code := r.Get("CODE")
		if !unverified(code) {
			k := [3]string{r.Get("FILE_NO"), r.Get("VAR_NAME"), code}
			if seen[k] {
				c.add("C14", kosac.R, r, "CODE", code, "변수 안에서 유일한 코드", "같은 변수에 같은 CODE가 두 번 이상 있습니다.")
			} else if !codeRe.MatchString(code) {
				c.add("C14", kosac.R, r, "CODE", code, "정수", "CODE의 값은 정수여야 합니다.")
			}
			seen[k] = true
		}
		if name := r.Get("VAR_NAME"); !unverified(name) && c.vars[varKey(r)] == nil {
			c.add("C14", kosac.R, r, "VAR_NAME", name, "03_변수정의의 변수", "코드의 변수가 같은 FILE_NO의 03_변수정의 행에 없습니다.")
		}
		switch l := r.Get("CODE_LABEL"); {
		case blank(l):
			c.add("C14", kosac.R, r, "CODE_LABEL", l, "필수", "CODE_LABEL의 값이 비어 있습니다.")
		case !unverified(l) && numbersOnly(l):
			c.add("C14", kosac.W, r, "CODE_LABEL", l, "뜻을 알 수 있는 라벨", "코드 라벨이 ①②③ 같은 기호나 숫자뿐입니다.")
		}
	}
}

// c15: missing-code labels.
func (c *checker) c15() {
	labels := c.ref.MissingLabels
	if labels == nil {
		labels = kosac.DefaultMissingLabels
	}
	for _, cd := range c.cb.Codes {
		r := cd.Rec
		allowed, ok := labels[r.Get("CODE")]
		if l := r.Get("CODE_LABEL"); ok && !unverified(l) && !slices.Contains(allowed, l) {
			c.add("C15", kosac.R, r, "CODE_LABEL", l, strings.Join(allowed, ", "), "결측 코드의 라벨이 규정 문구가 아닙니다.")
		}
	}
}

// c16: standard item IDs.
func (c *checker) c16() {
	if c.ref.StdItems == nil {
		return
	}
	for _, v := range c.cb.Vars {
		r := v.Rec
		id := r.Get("STD_ID")
		if !checkable(id) {
			continue
		}
		item, ok := c.ref.StdItems[id]
		if !ok {
			c.add("C16", kosac.R, r, "STD_ID", id, "표준문항 사전의 STD_ID", "STD_ID가 표준문항 사전에 없습니다.")
		} else if ml := r.Get("MEAS_LEVEL"); checkable(ml) && kosac.MeasLevels[ml] != item.MeasLevel {
			c.add("C16", kosac.R, r, "MEAS_LEVEL", ml, item.MeasLevel, "MEAS_LEVEL이 표준문항 사전의 측정수준과 다릅니다.")
		}
	}
}

// c17: identifiers. An institution code changes between years, so its NOTE
// must say which year's code system it follows.
func (c *checker) c17() {
	for _, v := range c.cb.Vars {
		r := v.Rec
		if ml := r.Get("MEAS_LEVEL"); r.Get("PII") == "Y" && (ml == "명목" || ml == "서열") {
			c.add("C17", kosac.W, r, "PII", "Y", "문자 또는 연속", "개인식별정보(PII=Y) 변수의 측정수준이 명목이나 서열입니다.")
		}
		if r.Get("STD_ID") != "SCHOOL.ID" && !instRe.MatchString(r.Get("VAR_LABEL")) {
			continue
		}
		if n := r.Get("NOTE"); !unverified(n) && !noteYearRe.MatchString(n) {
			c.add("C17", kosac.W, r, "NOTE", n, "코드 체계와 기준 연도", "기관 식별자 변수의 NOTE에 코드 체계의 기준 연도가 없습니다.")
		}
	}
}

// pagesBeyond reports whether a normalized page range runs past n pages.
func pagesBeyond(pages string, n int) bool {
	a, b, _ := strings.Cut(pages, "-")
	x, _ := strconv.Atoi(a)
	y, _ := strconv.Atoi(b)
	return x > n || y > n
}

// c18: the questionnaire, report and extra files. The questionnaire may
// instead give pages of the report.
func (c *checker) c18() {
	st := c.cb.Study
	report := strings.TrimSpace(st.Get("REPORT_FILE"))
	for _, k := range []string{"QUESTIONNAIRE", "REPORT_FILE", "EXTRA_FILES"} {
		v := st.Get(k)
		if !checkable(v) {
			continue
		}
		items := []string{v}
		if k == "EXTRA_FILES" {
			items = strings.Split(v, ";")
		}
		for _, it := range items {
			if it = strings.TrimSpace(it); it == "" {
				continue
			}
			pages, isFile := kosac.MaterialItem(it)
			switch {
			case isFile:
				if c.ref.UploadedFiles != nil && !slices.Contains(c.ref.UploadedFiles, it) {
					c.add("C18", kosac.R, st, k, it, "업로드한 파일명", k+"의 파일이 업로드한 파일 중에 없습니다.")
				}
			case k == "REPORT_FILE":
				c.add("C18", kosac.R, st, k, it, "파일명", "REPORT_FILE의 값이 파일명이 아닙니다.")
			case pages != "":
				if n, ok := c.ref.PDFPages[report]; ok && pagesBeyond(pages, n) {
					c.add("C18", kosac.R, st, k, it, fmt.Sprintf("1-%d", n), "쪽 범위가 보고서 PDF의 쪽 수를 넘습니다.")
				}
			default:
				c.add("C18", kosac.R, st, k, it, "파일명 또는 REPORT:p{쪽}", k+"의 값이 파일명도 보고서 쪽 범위도 아닙니다.")
			}
		}
	}
}

// varOwners is the rule that judges each 03 value; the rest are C06's.
var varOwners = map[string]string{
	"MEAS_LEVEL": "C07", "VAR_LABEL": "C08", "VAR_GROUP": "C09", "RESP_UNIVERSE": "C10", "MR_TYPE": "C11",
	"REVERSE": "C12", "DERIVATION": "C13", "STD_ID": "C16", "NOTE": "C17",
}

// unverifiedRows warns on every "확인 불가" cell of the 02–04 sheets, under
// the rule that judges that cell.
func (c *checker) unverifiedRows() {
	warn := func(rule string, owners map[string]string, r kosac.Record, keys []string) {
		for _, k := range keys {
			if v := r.Get(k); unverified(v) {
				c.add(cmp.Or(owners[k], rule), kosac.W, r, k, v, "", unverifiedMsg(k))
			}
		}
	}
	for _, f := range c.cb.Files {
		warn("C05", nil, f.Rec, kosac.FileKeys)
	}
	for _, v := range c.cb.Vars {
		warn("C06", varOwners, v.Rec, kosac.VarKeys)
	}
	for _, cd := range c.cb.Codes {
		warn("C14", nil, cd.Rec, kosac.CodeKeys)
	}
}

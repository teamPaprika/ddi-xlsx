package check

import (
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/teamPaprika/ddi-xlsx/kosac"
)

var leadingZeroCode = regexp.MustCompile(`^[+-]?0[0-9]|\.0+$`)
var timeMarkers = regexp.MustCompile(`\d{4}(?:[-./]\d{1,2}(?:[-./]\d{1,2})?)?`)
var variableRangeRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]*\s*[~～]\s*[A-Za-z][A-Za-z0-9_]*`)
var targetCountRe = regexp.MustCompile(`\s*\(([0-9]+)개\)$`)

var defaultAccessLevels = []string{"공개 가능", "내부 이용", "접근 제한", "별도 검토"}

func (c *checker) v4() {
	if c.cb.Template.Version.Value != "4.0" {
		return
	}
	for _, k := range []string{"SERIES_NAME", "STUDY_ROUND", "ACCESS_RESTRICTIONS", "CLEANING_LOG", "SOURCE_FILES"} {
		c.v4required("C01", c.cb.Study, k)
	}
	c.v4types()
	c.v4statusVars()
	c.v4access()
	c.v4sources()
	c.v4recodeStatus()
	c.v4missing()
	c.v4recodes()
	c.v4files()
	c.v4codes()
}
func (c *checker) v4required(rule string, r kosac.Record, keys ...string) {
	for _, k := range keys {
		if blank(r.Get(k)) {
			c.add(rule, kosac.R, r, k, "", "필수", "필수 항목이 비어 있습니다.")
		}
	}
}

// C19 follows the type/level combinations in the v2.1 request, including character codes.
func (c *checker) v4types() {
	for _, v := range c.cb.Vars {
		r := v.Rec
		typ, level, interval := r.Get("DATA_TYPE"), r.Get("MEAS_LEVEL"), r.Get("DDI_INTERVAL")
		c.v4required("C19", r, "DATA_TYPE", "DDI_INTERVAL")
		if !slices.Contains([]string{"숫자", "문자", "날짜", "일시", "확인 불가"}, typ) {
			c.add("C19", kosac.R, r, "DATA_TYPE", typ, "숫자, 문자, 날짜, 일시", "자료형이 v4.0 목록에 없습니다.")
		}
		if !kosac.V4Absent(interval) && interval != "discrete" && interval != "contin" && !unverified(interval) && interval != "확인 불가" {
			c.add("C19", kosac.R, r, "DDI_INTERVAL", interval, "discrete, contin, 해당 없음", "연속·이산 값이 올바르지 않습니다.")
		}
		_, valid := c.codesOf(v)
		characterCode := typ == "문자" && level == "명목" && interval == "discrete" && len(valid) > 0
		if (typ == "문자" || typ == "날짜" || typ == "일시") && !characterCode && (!kosac.V4Absent(level) || !kosac.V4Absent(interval)) {
			c.add("C19", kosac.R, r, "MEAS_LEVEL", level, "해당 없음", "문자·날짜·일시는 수량 측정변수로 취급하지 않습니다.")
		}
		if typ == "숫자" && (level == "명목" || level == "서열") && interval != "discrete" {
			c.add("C19", kosac.R, r, "DDI_INTERVAL", interval, "discrete", "숫자형 명목·서열 변수는 이산형이어야 합니다.")
		}
		if typ == "숫자" && (level == "등간" || level == "비율") && interval != "contin" && interval != "discrete" {
			c.add("C19", kosac.R, r, "DDI_INTERVAL", interval, "contin, discrete", "등간·비율 변수의 연속·이산 구분이 필요합니다.")
		}
		if interval == "contin" && len(valid) > 0 {
			c.add("C19", kosac.R, r, "DDI_INTERVAL", interval, "유효 코드 없는 연속형", "연속형 변수에 유효 코드가 열거되어 있습니다.")
		}
		if typ == "확인 불가" || interval == "확인 불가" {
			c.add("C19", kosac.W, r, "DATA_TYPE", typ, "담당자 확인", "자료형·연속/이산에 확인 불가 값이 남아 있습니다.")
		}
	}
}
func (c *checker) v4statusVars() {
	for _, v := range c.cb.Vars {
		r := v.Rec
		name := r.Get("MISSING_STATUS_VAR")
		if blank(name) || kosac.V4Absent(name) || unverified(name) {
			continue
		}
		status := c.vars[key{r.Get("FILE_NO"), name}]
		if status == nil {
			c.add("C20", kosac.R, r, "MISSING_STATUS_VAR", name, "같은 파일의 변수명", "결측 상태 변수가 같은 파일에 없습니다.")
			continue
		}
		if status.Rec.Get("MEAS_LEVEL") != "명목" {
			c.add("C20", kosac.R, r, "MISSING_STATUS_VAR", name, "명목", "결측 상태 변수는 명목형이어야 합니다.")
		}
		for _, cd := range c.codes[varKey(status.Rec)] {
			if !slices.Contains([]string{"0", "-9", "-8", "-7"}, cd.Rec.Get("CODE")) {
				c.add("C20", kosac.R, cd.Rec, "CODE", cd.Rec.Get("CODE"), "0, -9, -8, -7", "결측 상태 변수의 코드가 허용 집합 밖입니다.")
			}
		}
		if r.Get("DATA_TYPE") == "숫자" {
			c.add("C20", kosac.W, r, "MISSING_STATUS_VAR", name, "결측 코드 충돌 확인", "숫자형 변수의 별도 상태 변수가 필요한지 유효값 충돌을 확인하세요.")
		}
	}
}
func (c *checker) v4access() {
	levels := c.ref.AccessLevels
	if levels == nil {
		levels = defaultAccessLevels
	}
	for _, v := range c.cb.Vars {
		r := v.Rec
		x := r.Get("ACCESS_LEVEL")
		if !slices.Contains(levels, x) {
			c.add("C21", kosac.R, r, "ACCESS_LEVEL", x, strings.Join(levels, ", "), "접근 수준이 허용 목록에 없습니다.")
		} else if len(levels) > 0 && x == levels[0] && (r.Get("PII") == "Y" || r.Get("SENSITIVE") == "Y") {
			c.add("C21", kosac.W, r, "ACCESS_LEVEL", x, "접근 제한 검토", "준식별자·민감정보 변수에 가장 넓은 접근 수준이 지정되어 있습니다.")
		}
	}
}
func (c *checker) v4sources() {
	history := map[key]bool{}
	for _, r := range c.cb.Recodes {
		names, err := c.v4targets(r.Get("FILE_NO"), r.Get("TARGET_VAR"), r.Get("SOURCE_VAR"))
		if err == nil {
			for _, name := range names {
				history[key{r.Get("FILE_NO"), name}] = true
			}
		}
	}
	for _, v := range c.cb.Vars {
		r := v.Rec
		source := r.Get("SOURCE_VAR")
		c.v4required("C22", r, "SOURCE_VAR")
		if !blank(source) && source != r.Get("VAR_NAME") && !history[varKey(r)] {
			c.add("C22", kosac.R, r, "SOURCE_VAR", source, "05 재코딩 이력", "새 변수의 변환 이력이 없습니다.")
		}
		if derivation := r.Get("DERIVATION"); checkable(derivation) && !kosac.V4Absent(derivation) {
			mentioned := map[string]bool{}
			for _, span := range variableRangeRe.FindAllString(derivation, -1) {
				if names, err := c.v4targets(r.Get("FILE_NO"), span, ""); err == nil {
					for _, name := range names {
						mentioned[name] = true
					}
				}
			}
			source = variableRangeRe.ReplaceAllStringFunc(source, func(span string) string {
				return strings.Join(strings.Fields(span), "")
			})
			for _, name := range splitNames(source) {
				names, err := c.v4targets(r.Get("FILE_NO"), name, "")
				if err != nil {
					names = []string{name}
				}
				if !blank(name) && slices.ContainsFunc(names, func(variable string) bool {
					return !mentioned[variable] && !literalVarName(derivation, variable)
				}) {
					c.add("C22", kosac.R, r, "SOURCE_VAR", name, "DERIVATION에 언급된 원변수", "산출식에 원변수가 언급되지 않았습니다.")
				}
			}
		}
	}
}
func (c *checker) v4recodeStatus() {
	total, pending := 0, 0
	var first kosac.Record
	firstKey := ""
	check := func(r kosac.Record, k string) {
		total++
		x := r.Get(k)
		allowed := []string{"처리 완료", "확인 필요"}
		if k == "STATUS" {
			allowed = []string{"미착수", "확인 필요", "규칙 확정", "처리 완료", "검증 완료", "적용 제외"}
		}
		if !slices.Contains(allowed, x) {
			c.add("C23", kosac.R, r, k, x, strings.Join(allowed, ", "), "재코딩 상태가 허용 목록에 없습니다.")
		} else if x == "확인 필요" {
			pending++
			if first == nil {
				first, firstKey = r, k
			}
			c.add("C23", kosac.W, r, k, x, "담당자 검토", "확인이 필요한 재코딩 항목입니다.")
		} else if x != "처리 완료" {
			c.add("C23", kosac.W, r, k, x, "최종 제출 상태 별도 확인", "이력의 작성 상태를 수용했습니다. 최종 제출 가능 여부는 별도 확인이 필요합니다.")
		}
	}
	for _, v := range c.cb.Vars {
		check(v.Rec, "RECODE_STATUS")
	}
	for _, cd := range c.cb.Codes {
		check(cd.Rec, "RECODE_STATUS")
	}
	for _, r := range c.cb.Recodes {
		check(r, "STATUS")
	}
	limit := c.ref.PendingRecodeLimit
	if limit == 0 {
		limit = .1
	}
	if total > 0 && float64(pending)/float64(total) > limit {
		c.add("C23", kosac.R, first, firstKey, strconv.Itoa(pending), fmt.Sprintf("전체의 %.0f%% 이하", limit*100), "확인 필요 항목의 비율이 제출 상한을 넘습니다.")
	}
}
func (c *checker) v4missing() {
	for _, cd := range c.cb.Codes {
		r := cd.Rec
		flag, reason, code := r.Get("IS_MISSING"), r.Get("MISSING_REASON"), r.Get("CODE")
		negative := false
		if n, ok := new(big.Rat).SetString(code); ok {
			negative = n.Sign() < 0
		}
		if !slices.Contains([]string{"예", "아니오", "Y", "N"}, flag) {
			c.add("C24", kosac.R, r, "IS_MISSING", flag, "예, 아니오", "결측 여부를 명시해야 합니다.")
		}
		if kosac.DeclaredMissing(flag) != negative {
			c.add("C24", kosac.R, r, "IS_MISSING", flag, "음수 코드와 결측 선언 일치", "결측 여부와 코드의 부호가 일치하지 않습니다.")
		}
		if kosac.DeclaredMissing(flag) {
			allowed := map[string][]string{"-9": {"무응답"}, "-8": {"비해당", "해당 없음"}, "-7": {"모름·거절"}}[code]
			if !slices.Contains(allowed, reason) {
				c.add("C24", kosac.R, r, "MISSING_REASON", reason, strings.Join(allowed, ", "), "결측 사유가 코드의 뜻과 다릅니다.")
			}
		} else if !kosac.V4Absent(reason) {
			c.add("C24", kosac.R, r, "MISSING_REASON", reason, "해당 없음", "유효 코드의 결측 사유는 해당 없음이어야 합니다.")
		}
	}
}
func (c *checker) v4recodes() {
	types := c.ref.TransformTypes
	for _, r := range c.cb.Recodes {
		c.v4required("C25", r, "FILE_NO", "SOURCE_FILE", "SOURCE_VAR", "SOURCE_CODE_OR_CONDITION", "TARGET_VAR", "TARGET_CODE_OR_RULE", "TRANSFORM_TYPE", "BASIS", "STATUS", "RECORD_COUNT")
		if c.cb.File(r.Get("FILE_NO")) == nil {
			c.add("C25", kosac.R, r, "FILE_NO", r.Get("FILE_NO"), "02_데이터파일의 FILE_NO", "재코딩 이력이 없는 파일을 가리킵니다.")
		}
		if name := r.Get("SOURCE_FILE"); name == "(코드북 추가 매핑)" {
			c.add("C25", kosac.W, r, "SOURCE_FILE", name, "매핑 근거·실행 별도 검증", "코드북 추가 매핑의 출처 표기를 수용했습니다. 기준정보 대조와 매핑 실행은 별도 검증이 필요합니다.")
		} else if !blank(name) && !literalName(c.cb.Study.Get("SOURCE_FILES"), name) {
			c.add("C25", kosac.R, r, "SOURCE_FILE", name, "SOURCE_FILES에 기록한 파일", "수집 원본 파일 목록에 없는 파일입니다.")
		}
		if types != nil && !slices.Contains(types, r.Get("TRANSFORM_TYPE")) {
			c.add("C25", kosac.R, r, "TRANSFORM_TYPE", r.Get("TRANSFORM_TYPE"), strings.Join(types, ", "), "변환 유형이 허용 목록에 없습니다.")
		}
		if value := r.Get("RECORD_COUNT"); !blank(value) {
			if _, err := strconv.ParseUint(value, 10, 64); err != nil {
				c.add("C25", kosac.R, r, "RECORD_COUNT", value, "0 이상의 정수", "영향 사례 수는 0 이상의 정수여야 합니다.")
			}
		}
		target, source := r.Get("TARGET_VAR"), r.Get("SOURCE_VAR")
		nonDataTarget := target == "(코드북 04)" || target == "-" && source == "-" || target == "동일 변수" && source == "전체 숫자 열"
		if nonDataTarget && r.Get("TRANSFORM_TYPE") == "결측 코드화" {
			c.add("C25", kosac.R, r, "TARGET_VAR", target, "결측 코드화 대상 변수명", "결측 코드화는 실제 대상 변수를 지정해야 합니다.")
			continue
		}
		if target == "(코드북 04)" {
			if _, err := c.v4targets(r.Get("FILE_NO"), source, ""); err != nil {
				c.add("C25", kosac.R, r, "SOURCE_VAR", source, "코드북 수정 대상 변수", err.Error())
			}
			c.add("C25", kosac.W, r, "TARGET_VAR", target, "메타데이터 수정 별도 확인", "코드북 수정 이력으로 수용했습니다. 수정 결과·근거는 별도 확인이 필요합니다.")
			continue
		}
		if target == "-" && source == "-" || target == "동일 변수" && source == "전체 숫자 열" {
			c.add("C25", kosac.W, r, "TARGET_VAR", target, "검토·대상 집합 별도 확인", "검토 또는 대상 집합으로 적힌 이력을 수용했습니다. 대상 확정과 실행·재현은 별도 검증이 필요합니다.")
			continue
		}
		targets, err := c.v4targets(r.Get("FILE_NO"), r.Get("TARGET_VAR"), r.Get("SOURCE_VAR"))
		if err != nil {
			c.add("C25", kosac.R, r, "TARGET_VAR", r.Get("TARGET_VAR"), "변수명·나열·범위", err.Error())
			continue
		}
		if r.Get("TRANSFORM_TYPE") == "결측 코드화" {
			for _, missing := range []string{"-9", "-8", "-7"} {
				if literalName(r.Get("TARGET_CODE_OR_RULE"), missing) {
					for _, name := range targets {
						if !slices.ContainsFunc(c.codes[key{r.Get("FILE_NO"), name}], func(cd *kosac.Code) bool {
							return cd.Rec.Get("CODE") == missing && kosac.DeclaredMissing(cd.Rec.Get("IS_MISSING"))
						}) {
							c.add("C25", kosac.R, r, "TARGET_VAR", name, "04의 결측 코드 선언", "결측 코드화 대상에 해당 결측 코드가 선언되지 않았습니다.")
						}
					}
				}
			}
		}
	}
}
func (c *checker) v4files() {
	for _, f := range c.cb.Files {
		r := f.Rec
		no := r.Get("FILE_NO")
		vars := []*kosac.Var{}
		qcount := 0
		for _, v := range c.cb.Vars {
			if v.Rec.Get("FILE_NO") == no {
				vars = append(vars, v)
				if !blank(v.Rec.Get("Q_NO")) {
					qcount++
				}
			}
		}
		if count, err := strconv.Atoi(r.Get("VARIABLE_N")); err != nil || count != len(vars) {
			c.add("C26", kosac.R, r, "VARIABLE_N", r.Get("VARIABLE_N"), strconv.Itoa(len(vars)), "제출본 변수 수가 변수정의 행 수와 다릅니다.")
		}
		keytext := r.Get("UNIQUE_KEY")
		if at := strings.Index(keytext, "("); at >= 0 {
			keytext = keytext[:at]
		}
		names := splitNames(keytext)
		if len(names) == 0 {
			c.v4required("C26", r, "UNIQUE_KEY")
		}
		if time := r.Get("TIME_VAR"); blank(time) || kosac.V4Absent(time) {
			if len(names) != 1 || names[0] != r.Get("ID_VAR") {
				c.add("C26", kosac.R, r, "UNIQUE_KEY", keytext, r.Get("ID_VAR"), "단일 시점의 유일 키는 응답자 ID 변수와 같아야 합니다.")
			}
		}
		for _, name := range names {
			if c.vars[key{no, name}] == nil {
				c.add("C26", kosac.R, r, "UNIQUE_KEY", name, "같은 파일의 변수명", "유일 키 변수가 변수정의에 없습니다.")
			}
		}
		if name := r.Get("TIME_VAR"); !blank(name) && !kosac.V4Absent(name) {
			v := c.vars[key{no, name}]
			if v == nil || !slices.Contains([]string{"명목", "서열"}, v.Rec.Get("MEAS_LEVEL")) {
				c.add("C26", kosac.R, r, "TIME_VAR", name, "같은 파일의 명목·서열 변수", "시점 변수의 참조·측정수준이 올바르지 않습니다.")
			}
			timeText := r.Get("SURVEY_TIME")
			markers := map[string]bool{}
			for _, m := range timeMarkers.FindAllString(timeText, -1) {
				markers[m] = true
			}
			if len(markers) < 2 && !(strings.Contains(timeText, "사전") && strings.Contains(timeText, "사후")) && !(strings.Contains(timeText, "1차") && strings.Contains(timeText, "2차")) {
				c.add("C26", kosac.W, r, "SURVEY_TIME", r.Get("SURVEY_TIME"), "둘 이상의 조사 시점", "둘 이상의 조사 시점이 기술됐는지 확인하세요.")
			}
		}
		if qcount > 0 && qcount < len(vars) {
			c.add("C26", kosac.W, r, "FILE_NO", strconv.Itoa(qcount), "문항번호 작성 일관성", "일부 변수에만 문항번호가 기재되어 있습니다.")
		}
	}
}
func (c *checker) v4codes() {
	seen := map[key]map[string]string{}
	for _, cd := range c.cb.Codes {
		r := cd.Rec
		v := c.vars[varKey(r)]
		if v == nil || v.Rec.Get("DATA_TYPE") != "숫자" {
			continue
		}
		raw := r.Get("CODE")
		number, ok := new(big.Rat).SetString(raw)
		if !ok {
			continue
		}
		normalized := number.RatString()
		k := varKey(r)
		if seen[k] == nil {
			seen[k] = map[string]string{}
		}
		previous, exists := seen[k][normalized]
		if exists && previous != raw || leadingZeroCode.MatchString(raw) {
			c.add("C27", kosac.W, r, "CODE", raw, normalized, "숫자형 코드의 앞자리 0·소수점 표기를 정규화해 비교해야 합니다.")
		}
		seen[k][normalized] = raw
	}
}

func splitNames(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == ',' || r == '+' || r == '|' || unicode.IsSpace(r) })
}
func literalName(text, name string) bool {
	if name == "" {
		return false
	}
	pattern := `(^|[^\pL\pN_])` + regexp.QuoteMeta(name) + `($|[^\pL\pN_])`
	return regexp.MustCompile(pattern).MatchString(text)
}

// Variable identifiers are ASCII; Korean particles in prose are not part of the name.
func literalVarName(text, name string) bool {
	pattern := `(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(name) + `($|[^A-Za-z0-9_])`
	return regexp.MustCompile(pattern).MatchString(text)
}

// Ranges follow the declared variable order, never an invented numeric expansion.
func (c *checker) v4targets(fileNo, expression, source string) ([]string, error) {
	expression = strings.TrimSpace(expression)
	count := -1
	if match := targetCountRe.FindStringSubmatchIndex(expression); match != nil {
		var err error
		count, err = strconv.Atoi(expression[match[2]:match[3]])
		if err != nil {
			return nil, fmt.Errorf("재코딩 대상 개수를 읽을 수 없습니다: %s", expression)
		}
		expression = strings.TrimSpace(expression[:match[0]])
	}
	all := []string{}
	for _, v := range c.cb.Vars {
		if v.Rec.Get("FILE_NO") == fileNo {
			all = append(all, v.Rec.Get("VAR_NAME"))
		}
	}
	if expression == "(전체)" {
		if count >= 0 && count != len(all) {
			return nil, fmt.Errorf("재코딩 대상 개수가 설명과 다릅니다: 설명 %d개, 실제 %d개", count, len(all))
		}
		return all, nil
	}
	if expression == "동일 변수" || expression == "-" {
		expression = source
	}
	if strings.Contains(expression, "(") && strings.HasSuffix(expression, ")") {
		start := strings.LastIndex(expression, "(")
		inside := expression[start+1 : len(expression)-1]
		if _, list, ok := strings.Cut(inside, ":"); ok {
			expression = list
		}
	}
	names := []string{}
	for _, part := range strings.FieldsFunc(expression, func(r rune) bool { return r == ';' || r == ',' }) {
		part = strings.TrimSpace(part)
		rangeParts := strings.FieldsFunc(part, func(r rune) bool { return r == '~' || r == '～' })
		if len(rangeParts) == 2 {
			a, b := slices.Index(all, strings.TrimSpace(rangeParts[0])), slices.Index(all, strings.TrimSpace(rangeParts[1]))
			if a < 0 || b < a {
				return nil, fmt.Errorf("재코딩 대상 범위가 변수정의에 없습니다: %s", part)
			}
			names = append(names, all[a:b+1]...)
		} else if slices.Contains(all, part) {
			names = append(names, part)
		} else {
			return nil, fmt.Errorf("재코딩 대상 변수가 변수정의에 없습니다: %s", part)
		}
	}
	if count >= 0 && len(names) != count {
		return nil, fmt.Errorf("재코딩 대상 개수가 설명과 다릅니다: 설명 %d개, 실제 %d개", count, len(names))
	}
	return names, nil
}

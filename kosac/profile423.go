package kosac

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac/universe"
)

// The foundation profile (KOSAC DDI 프로필 20261105-r3) writes a v4.2.3
// workbook as plain DDI Codebook 2.5. Foundation-only facts go into notes
// typed "KOSAC:{KEY}" at the places the profile's section 6 lists; there is
// no kosac: namespace and no legacy encoding. The export writes only what the
// workbook states: a blank, "미확인…" or "확인 불가…" value is left out, a
// value outside the template's list is left out rather than replaced by a
// default (check S09 reports it), and a fact the profile cannot represent
// stops the export with every problem listed.

const (
	codeListName423 = "재단 표준코드표"
	codeListVer423  = "20261105-v2.3"
	agency423       = "KOSAC"
	stdVocab423     = "KOSAC-STD"
	stdSelector423  = "/codeBook/dataDscr/var/concept"
	dataKind423     = "설문자료" // the profile covers survey response data only
	weightNone423   = "가중치 미적용"
	dateFormat423   = "YYYY-MM-DD"
	timeFormat423   = "YYYY-MM-DD hh:mm:ss"
)

// Template list values and the DDI values the profile maps them to.
var (
	nature423 = map[string]string{"명목": "nominal", "서열": "ordinal", "등간": "interval", "비율": "ratio", "해당 없음": "other"}
	// KEY_ROLE "해당 없음" has no note; the row key goes to recGrp@keyvar.
	keyRole423      = map[string]string{"참여번호": "PART_NO", "실행단위 ID": "UNIT_ID", "연결 대상 참여번호": "LINK_PART_NO", "조사 시점": "WAVE", "가중치": "WEIGHT"}
	surveyType423   = map[string]string{"사업성과": "2", "연구조사": "3"}
	analysisUnit423 = map[string]string{"개인": "Individual", "학교": "Organization", "기관": "Organization", "기타": "Other"}
	sampling423     = map[string]string{
		"확률-단순무작위": "Probability.SimpleRandom", "확률-층화": "Probability.Stratified", "확률-집락": "Probability.Cluster",
		"확률-다단계": "Probability.Multistage", "할당(비확률)": "Nonprobability.Quota", "편의(비확률)": "Nonprobability.Availability",
		"전수": "TotalUniverseCompleteEnumeration", "기타": "Other",
	}
	mode423 = map[string]string{
		"온라인(웹·모바일)": "SelfAdministeredQuestionnaire.WebBased", "면접": "Interview.FaceToFace", "전화": "Interview.Telephone",
		"지면 자기기입": "SelfAdministeredQuestionnaire.Paper", "혼합": "MixedMode", "기타": "Other",
	}
	mrTypes423 = []string{"보기별 열", "순위별 열"}
)

// is423 selects the foundation profile, as the check does.
func (cb *Codebook) is423() bool { return strings.TrimSpace(cb.Template.Version.Value) == Profile423 }

// ---- values ----------------------------------------------------------------

// unconfirmed423 is a value the producer marked as not yet confirmed.
func unconfirmed423(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "미확인") || strings.HasPrefix(v, "확인 불가") || strings.HasPrefix(v, "확인불가")
}

// stated423 is v unless it is unconfirmed. Use it for free text, where
// "해당 없음" can be the text itself (a code label, a question).
func stated423(v string) string {
	if unconfirmed423(v) {
		return ""
	}
	return strings.TrimSpace(v)
}

// known423 is v unless it is unconfirmed or marked "해당 없음". Use it for
// study and file facts, references and codes.
func known423(v string) string {
	if V4Absent(v) {
		return ""
	}
	return stated423(v)
}

// refNames423 splits a list of var names as the check does.
func refNames423(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == ',' || r == ';' || r == '+' })
}

// items423 splits a list of free items (file names, codes) on ";", "," or a
// line break, so a single item may contain spaces.
func items423(s string) []string {
	var out []string
	for _, it := range strings.FieldsFunc(s, func(r rune) bool { return r == ';' || r == ',' || r == '\n' }) {
		if it = strings.TrimSpace(it); it != "" {
			out = append(out, it)
		}
	}
	return out
}

func note423(key, value string) *ddi.Node { return text("notes", value, "type", "KOSAC:"+key) }

// fileID423 is F + FILE_NO in at least two digits (1 → F01).
func fileID423(no string) string {
	n, err := strconv.Atoi(no)
	if err != nil {
		return "F" + no
	}
	return fmt.Sprintf("F%02d", n)
}

var isoDate423 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ncName423 reports whether s can be an xs:ID.
func ncName423(s string) bool {
	for i, r := range s {
		ok := unicode.IsLetter(r) || r == '_'
		if i > 0 {
			ok = ok || unicode.IsDigit(r) || r == '-' || r == '.' || unicode.In(r, unicode.Mn, unicode.Mc)
		}
		if !ok {
			return false
		}
	}
	return s != ""
}

// ---- missing codes ---------------------------------------------------------

type reservedCode423 struct {
	typ    string
	labels []string // normalized by missingLabel423
}

// reserved423 are the foundation's missing codes and the labels that confirm
// their meaning.
var reserved423 = map[string]reservedCode423{
	"-9": {"NO_ANSWER", []string{"무응답"}},
	"-8": {"NOT_APPLICABLE", []string{"비해당", "해당없음"}},
	"-7": {"DONT_KNOW_REFUSED", []string{"모름·거절", "모름", "거절", "응답거절", "응답하지않음"}},
}

// missingLike423 are labels that name a missing reason. On another code they
// leave its status unconfirmed: a negative or other code is not missing by
// its value, nor valid when its label says it is missing.
var missingLike423 = []string{"무응답", "비해당", "모름·거절", "응답하지않음", "결측", "결측값"}

// missingLabel423 drops a trailing "(…)" note, spaces, and spells the
// separators of "모름·거절" one way.
func missingLabel423(label string) string {
	base, _, _ := strings.Cut(label, "(")
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return -1
		case strings.ContainsRune("/ㆍ・•,", r):
			return '·'
		}
		return r
	}, base)
}

// Missing423 returns the missType of code and whether its missing status is
// confirmed. -9, -8 and -7 are missing (NO_ANSWER, NOT_APPLICABLE,
// DONT_KNOW_REFUSED) only when the label states that meaning; another code is
// valid ("", true) unless its label is blank, unconfirmed or names a missing
// reason. An unconfirmed status is never taken as valid.
func Missing423(code, label string) (string, bool) {
	label = strings.TrimSpace(label)
	if label == "" || unconfirmed423(label) {
		return "", false
	}
	base := missingLabel423(label)
	if m, ok := reserved423[strings.TrimSpace(code)]; ok {
		if slices.Contains(m.labels, base) {
			return m.typ, true
		}
		return "", false
	}
	if slices.Contains(missingLike423, base) {
		return "", false
	}
	return "", true
}

// ---- assembly --------------------------------------------------------------

// profile423 assembles one export. It reuses the builder's indexes and
// problem list; nothing of the legacy profile is written.
type profile423 struct {
	*builder
	weights    map[string]bool   // var IDs with wgt="wgt"
	fileWeight map[string]string // FILE_NO → its only weight var's ID
	keys       map[string]bool   // var IDs that identify or link rows
}

func (cb *Codebook) tree423() (*ddi.Node, error) {
	p := &profile423{builder: newBuilder(cb)}
	p.index()
	root := el("codeBook", "xmlns", ddi.Namespace, "version", "2.5")
	add(root, p.document(), p.study())
	for _, f := range cb.Files {
		add(root, p.file(f))
	}
	data := add(el("dataDscr"), p.groups()...)
	for _, v := range cb.Vars {
		add(data, p.variable(v))
	}
	add(root, data)
	add(root, p.materials()...)
	if len(p.problems) > 0 {
		return nil, fmt.Errorf("재단 v4.2.3 프로필로 변환할 수 없습니다: %s", strings.Join(p.problems, "; "))
	}
	return root, nil
}

// get is a 01 value the workbook states.
func (p *profile423) get(key string) string { return known423(p.cb.Study.Get(key)) }

// index checks what every element relies on and finds the weight and key vars.
func (p *profile423) index() {
	cb := p.cb
	if strings.TrimSpace(cb.Study.Get("TITLE")) != "" && p.get("TITLE") == "" {
		p.fail("TITLE: 조사명이 확정되지 않음")
	}
	seen := map[[2]string]bool{}
	for _, v := range cb.Vars {
		if strings.TrimSpace(v.Rec.Get("VAR_NAME")) == "" {
			p.fail("%s: VAR_NAME 없음", varLoc(v))
		}
		if k := varKey(v); seen[k] {
			p.fail("%s: 같은 파일에 같은 변수명이 두 번 있음", varLoc(v))
		} else {
			seen[k] = true
		}
	}
	for _, c := range cb.Codes {
		if k := [2]string{c.Rec.Get("FILE_NO"), c.Rec.Get("VAR_NAME")}; p.vars[k] == nil {
			p.fail("F%s/%s 코드 %s: 03_변수정의에 변수가 없음", k[0], k[1], c.Rec.Get("CODE"))
		}
	}

	p.keys = map[string]bool{}
	for _, f := range cb.Files {
		no := f.Rec.Get("FILE_NO")
		for _, k := range []string{"UNIQUE_KEY", "ID_VAR", "LINK_VAR"} {
			for _, name := range refNames423(known423(f.Rec.Get(k))) {
				if v := p.vars[[2]string{no, name}]; v != nil {
					p.keys[v.ID] = true
				}
			}
		}
	}
	for _, v := range cb.Vars {
		if keyRole423[strings.TrimSpace(v.Rec.Get("KEY_ROLE"))] != "" {
			p.keys[v.ID] = true
		}
	}

	// Weights exist only when WEIGHT_YN is a confirmed Y. A var is a weight
	// when WEIGHT_VAR names it or its KEY_ROLE says so.
	p.weights, p.fileWeight = map[string]bool{}, map[string]string{}
	if p.get("WEIGHT_YN") != "Y" {
		return
	}
	for _, name := range refNames423(p.get("WEIGHT_VAR")) {
		found := false
		for _, f := range cb.Files {
			if v := p.vars[[2]string{f.Rec.Get("FILE_NO"), name}]; v != nil {
				p.weights[v.ID], found = true, true
			}
		}
		if !found {
			p.fail("WEIGHT_VAR: 변수 %s 없음", name)
		}
	}
	for _, v := range cb.Vars {
		if strings.TrimSpace(v.Rec.Get("KEY_ROLE")) == "가중치" {
			p.weights[v.ID] = true
		}
	}
	// A file with several weights does not say which applies to which var,
	// so its vars get no wgt-var.
	count := map[string]int{}
	for _, v := range cb.Vars {
		if p.weights[v.ID] {
			no := v.Rec.Get("FILE_NO")
			count[no]++
			p.fileWeight[no] = v.ID
		}
	}
	for no, n := range count {
		if n > 1 {
			delete(p.fileWeight, no)
		}
	}
}

// refs resolves the var names of value in file no into XML IDs.
func (p *profile423) refs(no, key, value string) []string {
	var ids []string
	for _, name := range refNames423(known423(value)) {
		if v := p.vars[[2]string{no, name}]; v != nil {
			ids = append(ids, v.ID)
		} else {
			p.fail("F%s %s: 변수 %s 없음", no, key, name)
		}
	}
	return ids
}

// count is a quantity written as an integer, or "" when not stated.
func (p *profile423) count(where, v string) string {
	if v == "" {
		return ""
	}
	n := sourceCount(v)
	if n == "" {
		p.fail("%s %q: 0 이상의 정수가 아님", where, v)
	}
	return n
}

// labelled joins the stated values of keys as "label: value" lines.
func (p *profile423) labelled(keys ...string) string {
	var lines []string
	for _, k := range keys {
		if v := p.get(k); v != "" {
			lines = append(lines, fieldLabels[k]+": "+v)
		}
	}
	return strings.Join(lines, "\n")
}

// coded is a list value as written, with its DDI controlled-vocabulary term.
// A value outside the list is left out.
func coded(tag, value string, terms map[string]string, vocab string) *ddi.Node {
	term, ok := terms[value]
	if !ok {
		return nil
	}
	return add(el(tag), text("txt", value), text("concept", term, "vocab", vocab))
}

func (p *profile423) document() *ddi.Node {
	cbDate := p.get("CB_DATE")
	date := ""
	if isoDate423.MatchString(cbDate) {
		date = cbDate
	}
	cit := add(el("citation"),
		add(el("titlStmt"), text("titl", p.get("TITLE")+" 코드북")),
		add(el("prodStmt"), text("producer", p.get("CB_PRODUCER")), text("prodDate", cbDate, "date", date)),
		add(el("verStmt"), text("version", p.get("CB_VERSION")), note423("TEMPLATE", "KOSAC-SURVEY-META "+Profile423)))
	d := add(el("docDscr"), cit,
		add(el("controlledVocabUsed"), text("codeListName", codeListName423), text("codeListAgencyName", agency423),
			text("codeListVersionID", codeListVer423), add(el("usage"), text("selector", stdSelector423))))
	// CODE_SYSTEM names the code list of each var's values.
	var systems []string
	users := map[string][]string{}
	for _, v := range p.cb.Vars {
		if s := known423(v.Rec.Get("CODE_SYSTEM")); s != "" {
			if users[s] == nil {
				systems = append(systems, s)
			}
			users[s] = append(users[s], v.ID)
		}
	}
	for _, s := range systems {
		add(d, add(el("controlledVocabUsed"), text("codeListName", s),
			add(el("usage"), el("specificElements", "refs", strings.Join(users[s], " ")))))
	}
	return d
}

func (p *profile423) study() *ddi.Node {
	g := p.get
	cit := add(el("citation"),
		add(el("titlStmt"), text("titl", g("TITLE")), text("altTitl", g("PROJECT_TITLE")),
			text("IDNo", g("PROJECT_NO"), "agency", "KOSAC-PROJECT"), text("IDNo", g("DATA_NO"), "agency", agency423),
			text("IDNo", g("STUDY_ID"), "agency", "KOSAC-ID")),
		add(el("rspStmt"), text("AuthEnty", g("PI_ORG")), text("AuthEnty", g("PI_NAME"), "affiliation", g("PI_ORG")),
			text("othId", g("DEPT_CD"), "role", "주관부서")),
		add(el("prodStmt"), text("fundAg", g("FUNDER")), text("grantNo", g("PROGRAM_CD"), "agency", agency423, "role", "생산 사업")),
		add(el("serStmt"), text("serName", g("SERIES_NAME"))),
		add(el("verStmt"), note423("RULE_VERSION", RuleVersion423)))

	sum := add(el("sumDscr"), text("timePrd", g("TIME_PERIOD")))
	round, dated := g("STUDY_ROUND"), false
	for _, d := range [][2]string{{"FIELD_START", "start"}, {"FIELD_END", "end"}} {
		if date := g(d[0]); date != "" {
			add(sum, el("collDate", "date", date, "event", d[1], "cycle", round))
			dated = true
		}
	}
	if !dated {
		add(sum, el("collDate", "cycle", round))
	}
	add(sum, text("nation", g("NATION")), text("geogCover", g("GEO_COVER")),
		coded("anlyUnit", g("ANALYSIS_UNIT"), analysisUnit423, "DDI AnalysisUnit"),
		text("universe", g("UNIVERSE")), text("dataKind", dataKind423))
	info := add(el("stdyInfo"), add(el("subject"), p.keywords()...),
		text("abstract", g("PURPOSE"), "contentType", "purpose"), text("abstract", g("SUMMARY"), "contentType", "abstract"), sum)

	var sources []*ddi.Node
	for _, s := range items423(g("SOURCE_FILES")) {
		sources = append(sources, text("dataSrc", s))
	}
	coll := add(el("dataColl"), text("dataCollector", g("FIELD_ORG")),
		add(el("sampProc"), text("txt", g("SAMPLING_DESC")), text("txt", p.labelled("STRATA_VARS", "CLUSTER_VAR")),
			text("concept", sampling423[g("SAMPLING_TYPE")], "vocab", "DDI SamplingProcedure")),
		// POP_SOURCE is one text; referencePeriod takes only a date, so it is not split out.
		add(el("sampleFrame"), text("sampleFrameName", g("SAMPLE_FRAME")), text("txt", g("POP_SOURCE"))),
		add(el("targetSampleSize"), text("sampleSize", p.count("TARGET_N", g("TARGET_N")))),
		coded("collMode", g("MODE"), mode423, "DDI ModeOfCollection"), text("resInstru", g("TOOL")),
		add(el("sources"), sources...), p.weight(), text("cleanOps", p.labelled("EXCLUDE_N", "EXCLUDE_RULE")))
	method := add(el("method"), coll, add(el("anlyInfo"), text("respRate", p.labelled("RESPONSE_RATE", "RR_FORMULA", "CONTACT_N"))))
	access := add(el("dataAccs"), add(el("useStmt"), text("restrctn", g("ACCESS_RESTRICTIONS")), text("citReq", g("CITATION_REQ"))))

	study := add(el("stdyDscr"), cit, info, method, access, note423("SURVEY_TYPE", surveyType423[g("SURVEY_TYPE")]))
	for _, c := range items423(g("RELATED_PROGRAM_CD")) {
		add(study, note423("RELATED_PROGRAM", c))
	}
	for _, n := range [][2]string{{"PROGRAM_SUB", "SUB_PROGRAM_CD"}, {"BIZ_NAME", "BIZ_NAME"}, {"BIZ_YEAR", "BIZ_YEAR"},
		{"ORG_CD", "FIELD_ORG_CD"}, {"AREA_CD", "AREA_CD"}, {"TARGET_CD", "TARGET_CD"},
		{"SCHOOL_CODE_YEAR", "SCHOOL_CODE_YEAR"}, {"SCALE_EXCEPTION", "SCALE_EXCEPTION"}} {
		add(study, note423(n[0], g(n[1])))
	}
	return study
}

func (p *profile423) keywords() []*ddi.Node {
	var out []*ddi.Node
	for _, k := range strings.Split(p.get("KEYWORDS"), ";") {
		out = append(out, text("keyword", k))
	}
	return out
}

// weight states a confirmed WEIGHT_YN only; a blank or unconfirmed one is not N.
func (p *profile423) weight() *ddi.Node {
	switch p.get("WEIGHT_YN") {
	case "Y":
		return text("weight", strings.TrimSpace("가중치 적용\n"+p.labelled("WEIGHT_VAR", "WEIGHT_METHOD")))
	case "N":
		return text("weight", weightNone423)
	}
	return nil
}

func (p *profile423) file(f *File) *ddi.Node {
	g := func(k string) string { return known423(f.Rec.Get(k)) }
	no := f.Rec.Get("FILE_NO")
	where := "F" + no
	keyField := "UNIQUE_KEY"
	if g(keyField) == "" {
		keyField = "ID_VAR"
	}
	content := g("CONTENT")
	if unit := g("UNIT"); unit != "" {
		content = strings.TrimSpace(content + "\n분석 단위: " + unit)
	}
	cases := p.count(where+" VALID_N", g("VALID_N"))
	if cases == "" && len(p.cb.Files) == 1 {
		// With one file the 01 VALID_N is that file's record count.
		cases = sourceCount(p.get("VALID_N"))
	}
	ft := add(el("fileTxt"), text("fileName", g("FILE_NAME")), text("fileCont", content),
		add(el("fileStrc"), el("recGrp", "keyvar", strings.Join(p.refs(no, keyField, g(keyField)), " "))),
		add(el("dimensns"), text("caseQnty", cases), text("varQnty", p.count(where+" VARIABLE_N", g("VARIABLE_N")))),
		text("fileType", g("FILE_FORMAT")))
	return add(el("fileDscr", "ID", fileID423(no)), ft, p.fileLink(f),
		note423("SURVEY_TIME", g("SURVEY_TIME")),
		note423("TIME_VAR", strings.Join(p.refs(no, "TIME_VAR", g("TIME_VAR")), " ")),
		note423("NOTE", stated423(f.Rec.Get("NOTE"))))
}

// fileLink423 is the KOSAC:FILE_LINK JSON; the field order is the profile's.
type fileLink423 struct {
	TargetFile string   `json:"targetFile"`
	SourceVars []string `json:"sourceVars"`
	TargetVars []string `json:"targetVars"`
}

func (p *profile423) fileLink(f *File) *ddi.Node {
	g := func(k string) string { return known423(f.Rec.Get(k)) }
	no, target := f.Rec.Get("FILE_NO"), g("LINK_FILE_NO")
	src, dst := refNames423(g("LINK_VAR")), refNames423(g("LINK_TARGET_VAR"))
	if target == "" && len(src) == 0 && len(dst) == 0 {
		return nil
	}
	if _, ok := p.fileIdx[target]; !ok {
		p.fail("F%s LINK_FILE_NO %q: 연결 대상 파일이 없음", no, target)
		return nil
	}
	if len(src) == 0 || len(src) != len(dst) {
		p.fail("F%s: LINK_VAR %d개와 LINK_TARGET_VAR %d개가 같은 순서로 짝지어지지 않음", no, len(src), len(dst))
		return nil
	}
	s, d := p.refs(no, "LINK_VAR", g("LINK_VAR")), p.refs(target, "LINK_TARGET_VAR", g("LINK_TARGET_VAR"))
	if len(s) != len(src) || len(d) != len(dst) {
		return nil // refs reported the missing vars
	}
	b, err := json.Marshal(fileLink423{TargetFile: fileID423(target), SourceVars: s, TargetVars: d})
	if err != nil {
		p.fail("F%s FILE_LINK: %v", no, err)
		return nil
	}
	return note423("FILE_LINK", string(b))
}

// analysis reports whether a weight applies to v: a question or a derived
// value, not a weight or a row key.
func (p *profile423) analysis(v *Var) bool {
	g := v.Rec.Get
	return !p.weights[v.ID] && !p.keys[v.ID] && (stated423(g("Q_TEXT")) != "" || stated423(g("DERIVATION")) != "")
}

func (p *profile423) variable(v *Var) *ddi.Node {
	g := v.Rec.Get
	no := g("FILE_NO")
	interval := strings.TrimSpace(g("DDI_INTERVAL"))
	if interval != "discrete" && interval != "contin" {
		interval = ""
	}
	n := el("var", "ID", v.ID, "name", g("VAR_NAME"), "files", fileID423(no),
		"nature", nature423[strings.TrimSpace(g("MEAS_LEVEL"))], "intrvl", interval)
	if p.weights[v.ID] {
		n.Attrs = append(n.Attrs, ddi.Attr{Name: "wgt", Value: "wgt"})
	} else if w := p.fileWeight[no]; w != "" && p.analysis(v) {
		n.Attrs = append(n.Attrs, ddi.Attr{Name: "wgt-var", Value: w})
	}
	unit := ""
	if f := p.cb.File(no); f != nil {
		unit = known423(f.Rec.Get("UNIT"))
	}
	add(n, text("labl", stated423(g("VAR_LABEL"))), text("security", known423(g("ACCESS_LEVEL"))), text("anlysUnit", unit),
		add(el("qstn", "seqNo", known423(g("Q_NO"))), text("qstnLit", stated423(g("Q_TEXT")))),
		text("universe", stated423(g("RESP_UNIVERSE"))))
	for _, c := range p.codes[varKey(v)] {
		code, label := c.Rec.Get("CODE"), c.Rec.Get("CODE_LABEL")
		missType, ok := Missing423(code, label)
		if !ok {
			p.fail("%s 코드 %s: 라벨 %q로 유효·결측 여부를 확정할 수 없음", varLoc(v), code, label)
			continue
		}
		missing := "N"
		if missType != "" {
			missing = "Y"
		}
		// Codes stay strings as written, so leading zeros survive.
		add(n, add(el("catgry", "missing", missing, "missType", missType),
			text("catValu", code), text("labl", label), text("txt", stated423(c.Rec.Get("NOTE")))))
	}
	add(n, text("concept", known423(g("STD_ID")), "vocab", stdVocab423))

	// SOURCE_VAR: a var of this file is referenced by ID; any other source is
	// kept as written (원본 파일명과 변수명), never as an ID.
	var inside, outside []string
	for _, name := range items423(known423(g("SOURCE_VAR"))) {
		if name == g("VAR_NAME") {
			continue
		}
		if src := p.vars[[2]string{no, name}]; src != nil {
			inside = append(inside, src.ID)
		} else {
			outside = append(outside, name)
		}
	}
	add(n, add(el("derivation", "var", strings.Join(inside, " ")), text("drvdesc", stated423(g("DERIVATION")))))
	switch strings.TrimSpace(g("DATA_TYPE")) {
	case "숫자":
		add(n, el("varFormat", "type", "numeric"))
	case "문자":
		add(n, el("varFormat", "type", "character"))
	case "날짜":
		add(n, el("varFormat", "type", "character", "formatname", dateFormat423, "category", "date"))
	case "일시":
		add(n, el("varFormat", "type", "character", "formatname", timeFormat423, "category", "time"))
	}
	yes := func(key string) *ddi.Node {
		if strings.TrimSpace(g(key)) == "Y" {
			return note423(key, "Y")
		}
		return nil
	}
	return add(n,
		note423("KEY_ROLE", keyRole423[strings.TrimSpace(g("KEY_ROLE"))]),
		note423("UNIVERSE_EXPR", known423(g("UNIVERSE_EXPR"))),
		yes("REVERSE"), yes("PII"), yes("SENSITIVE"),
		note423("MISSING_STATUS_VAR", strings.Join(p.refs(no, "MISSING_STATUS_VAR", g("MISSING_STATUS_VAR")), " ")),
		note423("SOURCE_VAR", strings.Join(outside, "; ")),
		note423("NOTE", stated423(g("NOTE"))))
}

// ---- variable groups -------------------------------------------------------

type group423 struct {
	kind, label string
	path        string // SCALE path, for messages
	id          string
	vars        []string // member var IDs
	subs        []*group423
	notes       []*ddi.Node
}

func (g *group423) join(id string) {
	if !slices.Contains(g.vars, id) {
		g.vars = append(g.vars, id)
	}
}

// groupKinds is the output order of the groups.
var groupKinds = []string{"section", "version", "analysis", "multipleResp", "randomized"}

type groups423 struct {
	byKind map[string][]*group423
	index  map[string]*group423
}

// get returns the group of kind named label within scope, made on first use.
func (gs *groups423) get(kind, scope, label string) *group423 {
	k := kind + "\x00" + scope + "\x00" + label
	if g := gs.index[k]; g != nil {
		return g
	}
	g := &group423{kind: kind, label: label}
	gs.index[k] = g
	gs.byKind[kind] = append(gs.byKind[kind], g)
	return g
}

// groups writes the varGrp elements. Membership comes only from what each
// row states: VAR_GROUP (one section per file), ITEM_KEY (one version group
// per key across files), SCALE (analysis groups, items at the lowest level),
// MR_TYPE with Q_NO (one multipleResp group per question of a file), and for
// a split ballot the items whose UNIVERSE_EXPR names SPLIT_VAR.
func (p *profile423) groups() []*ddi.Node {
	gs := &groups423{byKind: map[string][]*group423{}, index: map[string]*group423{}}
	for _, v := range p.cb.Vars {
		g, no := v.Rec.Get, v.Rec.Get("FILE_NO")
		if s := known423(g("VAR_GROUP")); s != "" {
			gs.get("section", no, s).join(v.ID)
		}
		if k := known423(g("ITEM_KEY")); k != "" {
			gs.get("version", "", k).join(v.ID)
		}
		p.scale(gs, v)
		p.multipleResp(gs, v)
	}
	p.randomized(gs)
	for _, g := range gs.byKind["analysis"] {
		if len(g.vars) > 0 && len(g.subs) > 0 {
			p.fail("SCALE %s: 하위 영역이 있는 영역에 변수를 직접 연결함. 문항은 가장 아래 영역에만 연결하세요", g.path)
		}
	}

	// IDs: an ITEM_KEY is its group's ID when it is a free xs:ID; every other
	// group is G{n}, skipping the IDs already in the document.
	used := map[string]bool{}
	for _, v := range p.cb.Vars {
		used[v.ID] = true
	}
	for _, f := range p.cb.Files {
		used[fileID423(f.Rec.Get("FILE_NO"))] = true
	}
	for _, g := range gs.byKind["version"] {
		if ncName423(g.label) && !used[g.label] {
			g.id, used[g.label] = g.label, true
		}
	}
	n := 0
	var out []*ddi.Node
	for _, kind := range groupKinds {
		for _, g := range gs.byKind[kind] {
			for g.id == "" {
				n++
				if id := "G" + strconv.Itoa(n); !used[id] {
					g.id, used[id] = id, true
				}
			}
		}
	}
	for _, kind := range groupKinds {
		for _, g := range gs.byKind[kind] {
			var subs []string
			for _, s := range g.subs {
				subs = append(subs, s.id)
			}
			node := el("varGrp", "ID", g.id, "type", g.kind, "var", strings.Join(g.vars, " "), "varGrp", strings.Join(subs, " "))
			out = append(out, add(add(node, text("labl", g.label)), g.notes...))
		}
	}
	return out
}

// scale puts v in the lowest group of its SCALE path "영역 > 하위 영역"; each
// group lists its subgroups.
func (p *profile423) scale(gs *groups423, v *Var) {
	var parts []string
	for _, s := range strings.Split(known423(v.Rec.Get("SCALE")), ">") {
		if s = strings.TrimSpace(s); s != "" {
			parts = append(parts, s)
		}
	}
	var parent *group423
	for i, part := range parts {
		g := gs.get("analysis", strings.Join(parts[:i], ">"), part)
		g.path = strings.Join(parts[:i+1], " > ")
		if parent != nil && !slices.Contains(parent.subs, g) {
			parent.subs = append(parent.subs, g)
		}
		parent = g
	}
	if parent != nil {
		parent.join(v.ID)
	}
}

// multipleResp groups the columns of one question: same file, Q_NO and MR_TYPE.
func (p *profile423) multipleResp(gs *groups423, v *Var) {
	mr := known423(v.Rec.Get("MR_TYPE"))
	if mr == "" {
		return
	}
	if !slices.Contains(mrTypes423, mr) {
		p.fail("%s MR_TYPE %q: 보기별 열 또는 순위별 열이 아님", varLoc(v), mr)
		return
	}
	q := known423(v.Rec.Get("Q_NO"))
	if q == "" {
		// Q_NO is optional. The foundation sample numbers the columns of a
		// declared multiple-response question, for example QA22_1, QA22_2.
		name := v.Rec.Get("VAR_NAME")
		q = name
		if prefix, ok := Prefix(name); ok && fileNoRe.MatchString(strings.TrimPrefix(name, prefix+"_")) {
			q = prefix
		}
	}
	g := gs.get("multipleResp", strings.Join([]string{v.Rec.Get("FILE_NO"), v.Rec.Get("VAR_GROUP"), mr}, "\x00"), q)
	if g.notes == nil {
		g.notes = []*ddi.Node{note423("MR_TYPE", mr)}
	}
	g.join(v.ID)
}

// randomized groups, per file, the items asked by assignment: SPLIT_BALLOT is
// a confirmed Y and the item's UNIVERSE_EXPR conditions on SPLIT_VAR.
func (p *profile423) randomized(gs *groups423) {
	if p.get("SPLIT_BALLOT") != "Y" {
		return
	}
	split := p.get("SPLIT_VAR")
	if split == "" {
		p.fail("SPLIT_BALLOT=Y: SPLIT_VAR 없음")
		return
	}
	found := false
	for _, v := range p.cb.Vars {
		no := v.Rec.Get("FILE_NO")
		expr := known423(v.Rec.Get("UNIVERSE_EXPR"))
		names := exprNames423(expr)
		if expr == "" {
			// Older source codebooks put machine-readable conditions in RESP_UNIVERSE.
			// A prose mention of the assignment variable does not establish membership.
			if e, err := universe.Parse(known423(v.Rec.Get("RESP_UNIVERSE"))); err == nil {
				names = e.Vars()
			}
		}
		if v.Rec.Get("VAR_NAME") == split || !slices.Contains(names, split) {
			continue
		}
		found = true
		sv := p.vars[[2]string{no, split}]
		if sv == nil {
			p.fail("%s UNIVERSE_EXPR: 같은 파일에 SPLIT_VAR %s 없음", varLoc(v), split)
			continue
		}
		g := gs.get("randomized", no, "설문 분할("+split+")")
		if g.notes == nil {
			g.notes = []*ddi.Node{note423("SPLIT_VAR", sv.ID)}
		}
		g.join(v.ID)
	}
	if !found {
		p.fail("SPLIT_BALLOT=Y: UNIVERSE_EXPR에 SPLIT_VAR %s를 쓰는 문항이 없어 분할 문항을 정할 수 없음", split)
	}
}

// exprNames423 lists the var names of a UNIVERSE_EXPR: the words followed by
// =, != or in. Quoted values are skipped, so a value is never a name.
func exprNames423(expr string) []string {
	var toks []string
	rs := []rune(expr)
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case unicode.IsSpace(r):
			i++
		case r == '\'' || r == '"':
			j := i + 1
			for j < len(rs) && rs[j] != r {
				j++
			}
			toks = append(toks, "'")
			i = j + 1
		case r == '!' && i+1 < len(rs) && rs[i+1] == '=':
			toks = append(toks, "!=")
			i += 2
		case strings.ContainsRune("(),=!", r):
			toks = append(toks, string(r))
			i++
		default:
			j := i
			for j < len(rs) && !unicode.IsSpace(rs[j]) && !strings.ContainsRune("(),=!'\"", rs[j]) {
				j++
			}
			toks = append(toks, string(rs[i:j]))
			i = j
		}
	}
	var names []string
	for i := 0; i+1 < len(toks); i++ {
		if next := toks[i+1]; next == "=" || next == "!=" || strings.EqualFold(next, "in") {
			names = append(names, toks[i])
		}
	}
	return names
}

// materials lists the questionnaire, report and extra files by location.
func (p *profile423) materials() []*ddi.Node {
	var out []*ddi.Node
	for _, m := range [][2]string{{"QUESTIONNAIRE", "questionnaire"}, {"REPORT_FILE", "report"}, {"EXTRA_FILES", "other"}} {
		for _, uri := range strings.Split(p.get(m[0]), ";") {
			if uri = strings.TrimSpace(uri); uri != "" {
				out = append(out, el("otherMat", "level", "study", "type", m[1], "URI", uri))
			}
		}
	}
	return out
}

// IsProfile423DDI reports whether root declares the foundation profile in
// docDscr/citation/verStmt/notes[@type="KOSAC:TEMPLATE"].
func IsProfile423DDI(root *ddi.Node) bool {
	for _, d := range kids(root, "docDscr") {
		for _, c := range kids(d, "citation") {
			for _, v := range kids(c, "verStmt") {
				for _, n := range kids(v, "notes") {
					if attr(n, "type") == "KOSAC:TEMPLATE" && n.Text == "KOSAC-SURVEY-META "+Profile423 {
						return true
					}
				}
			}
		}
	}
	return false
}

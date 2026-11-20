package kosac_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/check"
)

// Synthetic v4.2.3 rows; no respondent records.

func p423Rec(sheet string, kv ...string) kosac.Record {
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

func p423Var(no, name string, kv ...string) *kosac.Var {
	base := []string{"FILE_NO", no, "VAR_NAME", name, "VAR_LABEL", name + " 라벨", "MEAS_LEVEL", "해당 없음", "DATA_TYPE", "문자", "PII", "N", "SENSITIVE", "N"}
	return &kosac.Var{Rec: p423Rec(kosac.SheetVars, append(base, kv...)...)}
}

func p423Code(no, name, code, label string) *kosac.Code {
	return &kosac.Code{Rec: p423Rec(kosac.SheetCodes, "FILE_NO", no, "VAR_NAME", name, "CODE", code, "CODE_LABEL", label)}
}

// p423Model is a pre/post codebook with a weight, a split ballot, a scale, an
// item key and a multiple-response question.
func p423Model() *kosac.Codebook {
	study := []string{"RULE_VERSION", "20261105-r3", "TITLE", "예시 조사", "PROJECT_TITLE", "예시 과제", "PROJECT_NO", "P-1",
		"PURPOSE", "목적", "SUMMARY", "내용", "FUNDER", "예시 재단", "PI_ORG", "예시 연구소", "PI_NAME", "연구책임자 A",
		"FIELD_ORG", "예시 조사기관", "FIELD_START", "2026-08-01", "FIELD_END", "2026-08-31", "KEYWORDS", "가;나;다",
		"QUESTIONNAIRE", "q.pdf", "REPORT_FILE", "r.pdf", "UNIVERSE", "대상", "POP_SOURCE", "명부 2026-07", "ANALYSIS_UNIT", "개인",
		"SAMPLING_TYPE", "편의(비확률)", "SAMPLING_DESC", "설명", "TARGET_N", "100", "VALID_N", "사전 100, 사후 90",
		"SPLIT_BALLOT", "Y", "SPLIT_VAR", "FORM", "MODE", "온라인(웹·모바일)", "EXCLUDE_N", "0",
		"WEIGHT_YN", "Y", "WEIGHT_VAR", "WT", "WEIGHT_METHOD", "사후 층화", "SURVEY_TYPE", "연구조사",
		"ACCESS_RESTRICTIONS", "내부 이용", "SOURCE_FILES", "raw_pre.csv; raw_post.csv", "DATA_NO", "문화-01-01", "STUDY_ID", "D0001-2024",
		"PROGRAM_CD", "C-2026-001", "RELATED_PROGRAM_CD", "R-1; R-2", "TOOL", "미확인: 플랫폼 확인 중", "SERIES_NAME", "해당 없음",
		"CB_PRODUCER", "예시 조사기관", "CB_DATE", "2026-11-20", "CB_VERSION", "1.0"}
	files := []*kosac.File{
		{Rec: p423Rec(kosac.SheetFiles, "FILE_NO", "1", "FILE_NAME", "pre.csv", "FILE_FORMAT", "CSV", "CONTENT", "사전 응답", "UNIT", "개인",
			"SURVEY_TIME", "사전", "UNIQUE_KEY", "PID", "VALID_N", "100", "VARIABLE_N", "8")},
		{Rec: p423Rec(kosac.SheetFiles, "FILE_NO", "2", "FILE_NAME", "post.csv", "FILE_FORMAT", "CSV", "CONTENT", "사후 응답", "UNIT", "개인",
			"SURVEY_TIME", "사후", "UNIQUE_KEY", "PID", "VALID_N", "90", "LINK_FILE_NO", "1", "LINK_VAR", "PID", "LINK_TARGET_VAR", "PID")},
	}
	var vars []*kosac.Var
	var codes []*kosac.Code
	for _, no := range []string{"1", "2"} {
		vars = append(vars,
			p423Var(no, "PID", "KEY_ROLE", "참여번호", "VAR_GROUP", "관리"),
			p423Var(no, "SEX", "VAR_LABEL", "성별", "Q_NO", "1", "Q_TEXT", "성별은?", "MEAS_LEVEL", "명목", "STD_ID", "PERSON.GENDER", "VAR_GROUP", "배경"),
			p423Var(no, "LVL", "VAR_LABEL", "학교급", "Q_NO", "2", "Q_TEXT", "학교급은?", "MEAS_LEVEL", "명목", "STD_ID", "SCHOOL.LEVEL", "VAR_GROUP", "배경"),
			p423Var(no, "FORM", "VAR_LABEL", "설문 유형", "MEAS_LEVEL", "명목", "VAR_GROUP", "관리"),
			p423Var(no, "Q1", "Q_NO", "3", "Q_TEXT", "만족하십니까?", "MEAS_LEVEL", "서열", "DATA_TYPE", "숫자", "ITEM_KEY", "SAT",
				"SCALE", "만족>전반", "REVERSE", "Y", "VAR_GROUP", "만족도"),
			p423Var(no, "Q2", "Q_NO", "4", "Q_TEXT", "추천하십니까?", "MEAS_LEVEL", "서열", "DATA_TYPE", "숫자",
				"SCALE", "만족 > 전반", "UNIVERSE_EXPR", "FORM = 1 AND LVL in ('02', '03')", "VAR_GROUP", "만족도"),
			p423Var(no, "MR5_1", "Q_NO", "5", "Q_TEXT", "경로: 학교", "MEAS_LEVEL", "명목", "MR_TYPE", "보기별 열", "VAR_GROUP", "경로"),
			p423Var(no, "MR5_2", "Q_NO", "5", "Q_TEXT", "경로: 지인", "MEAS_LEVEL", "명목", "MR_TYPE", "보기별 열", "VAR_GROUP", "경로"),
		)
		codes = append(codes,
			p423Code(no, "SEX", "1", "남"), p423Code(no, "SEX", "2", "여"), p423Code(no, "SEX", "-9", "무응답"),
			p423Code(no, "LVL", "02", "초등학교"), p423Code(no, "LVL", "03", "중학교"),
			p423Code(no, "FORM", "1", "A형"), p423Code(no, "FORM", "2", "B형"),
			p423Code(no, "Q1", "1", "불만"), p423Code(no, "Q1", "2", "만족"), p423Code(no, "Q1", "-8", "비해당"),
			p423Code(no, "Q2", "1", "불만"), p423Code(no, "Q2", "2", "만족"), p423Code(no, "Q2", "-8", "비해당"),
			p423Code(no, "MR5_1", "1", "선택"), p423Code(no, "MR5_1", "0", "미선택"),
			p423Code(no, "MR5_2", "1", "선택"), p423Code(no, "MR5_2", "0", "미선택"),
		)
	}
	vars = append(vars, p423Var("1", "WT", "VAR_LABEL", "가중치", "MEAS_LEVEL", "비율", "DATA_TYPE", "숫자", "DDI_INTERVAL", "contin",
		"KEY_ROLE", "가중치", "DERIVATION", "사후 층화 가중치", "VAR_GROUP", "관리"))
	cb := &kosac.Codebook{Template: kosac.TemplateMeta{Version: kosac.Field{Value: "4.2.3"}},
		Study: p423Rec(kosac.SheetStudy, study...), Files: files, Vars: vars, Codes: codes}
	if err := kosac.AssignIDs(cb, nil); err != nil {
		panic(err)
	}
	return cb
}

func p423Tree(t *testing.T, cb *kosac.Codebook) *ddi.Node {
	t.Helper()
	root, err := cb.Tree()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// p423All returns the elements at path below n, "/"-separated.
func p423All(n *ddi.Node, path string) []*ddi.Node {
	cur := []*ddi.Node{n}
	for _, name := range strings.Split(path, "/") {
		var next []*ddi.Node
		for _, c := range cur {
			for _, k := range c.Children {
				if k.Name == name {
					next = append(next, k)
				}
			}
		}
		cur = next
	}
	return cur
}

func p423Attr(n *ddi.Node, name string) string {
	v, _ := n.Attr(name)
	return v
}

// p423Notes are the texts of n's notes of KOSAC:{key}.
func p423Notes(n *ddi.Node, key string) []string {
	var out []string
	for _, c := range n.Children {
		if c.Name == "notes" && p423Attr(c, "type") == "KOSAC:"+key {
			out = append(out, c.Text)
		}
	}
	return out
}

func p423VarNode(root *ddi.Node, file, name string) *ddi.Node {
	for _, v := range p423All(root, "dataDscr/var") {
		if p423Attr(v, "name") == name && p423Attr(v, "files") == file {
			return v
		}
	}
	return nil
}

func p423Group(root *ddi.Node, typ, label string) *ddi.Node {
	for _, g := range p423All(root, "dataDscr/varGrp") {
		if p423Attr(g, "type") == typ && len(p423All(g, "labl")) > 0 && p423All(g, "labl")[0].Text == label {
			return g
		}
	}
	return nil
}

func p423Groups(root *ddi.Node, typ string) []*ddi.Node {
	var out []*ddi.Node
	for _, g := range p423All(root, "dataDscr/varGrp") {
		if p423Attr(g, "type") == typ {
			out = append(out, g)
		}
	}
	return out
}

func p423Want(t *testing.T, what, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

// A model the check passes without an R converts.
func TestProfile423CheckPassConverts(t *testing.T) {
	cb := p423Model()
	for _, i := range check.Check(cb, kosac.RefData{}) {
		if i.Level == kosac.R {
			t.Errorf("check: %s %s %s %s", i.Rule, i.Cell, i.Actual, i.Message)
		}
	}
	p423Tree(t, cb)
}

func TestProfile423Document(t *testing.T) {
	cb := p423Model()
	root := p423Tree(t, cb)
	id := func(no, name string) string { return cb.Var(no, name).ID }

	// No legacy encoding: no kosac: names, every note typed KOSAC:.
	kosac.Walk(root, func(n, _ *ddi.Node, path string) bool {
		if strings.HasPrefix(n.Name, "kosac:") || slices.ContainsFunc(n.Attrs, func(a ddi.Attr) bool {
			return strings.HasPrefix(a.Name, "xmlns:") || strings.HasPrefix(a.Name, "kosac:")
		}) {
			t.Errorf("%s: legacy extension", path)
		}
		if n.Name == "notes" && !strings.HasPrefix(p423Attr(n, "type"), "KOSAC:") {
			t.Errorf("%s: notes without a KOSAC: type", path)
		}
		return true
	})
	if is := check.ValidateXML(ddi.WriteXML(root), nil); len(is) > 0 {
		t.Errorf("ValidateXML: %v", is)
	}

	doc := p423All(root, "docDscr")[0]
	p423Want(t, "TEMPLATE", strings.Join(p423Notes(p423All(doc, "citation/verStmt")[0], "TEMPLATE"), ","), "KOSAC-SURVEY-META 4.2.3")
	vocab := p423All(doc, "controlledVocabUsed")[0]
	p423Want(t, "codeListName", p423All(vocab, "codeListName")[0].Text, "재단 표준코드표")
	p423Want(t, "codeListVersionID", p423All(vocab, "codeListVersionID")[0].Text, "20261105-v2.3")
	p423Want(t, "selector", p423All(vocab, "usage/selector")[0].Text, "/codeBook/dataDscr/var/concept")

	study := p423All(root, "stdyDscr")[0]
	p423Want(t, "RULE_VERSION", strings.Join(p423Notes(p423All(study, "citation/verStmt")[0], "RULE_VERSION"), ","), "20261105-r3")
	ids := map[string]string{}
	for _, n := range p423All(study, "citation/titlStmt/IDNo") {
		ids[p423Attr(n, "agency")] = n.Text
	}
	p423Want(t, "IDNo KOSAC", ids["KOSAC"], "문화-01-01")
	p423Want(t, "IDNo KOSAC-ID", ids["KOSAC-ID"], "D0001-2024")
	p423Want(t, "IDNo KOSAC-PROJECT", ids["KOSAC-PROJECT"], "P-1")
	grant := p423All(study, "citation/prodStmt/grantNo")[0]
	p423Want(t, "grantNo", grant.Text+"|"+p423Attr(grant, "agency")+"|"+p423Attr(grant, "role"), "C-2026-001|KOSAC|생산 사업")
	authors := p423All(study, "citation/rspStmt/AuthEnty")
	if len(authors) != 2 || authors[0].Text != "예시 연구소" || authors[1].Text != "연구책임자 A" || p423Attr(authors[1], "affiliation") != "예시 연구소" {
		t.Errorf("AuthEnty = %v", authors)
	}
	if len(p423All(study, "citation/serStmt")) != 0 {
		t.Error("SERIES_NAME 해당 없음 written")
	}
	p423Want(t, "dataCollector", p423All(study, "method/dataColl/dataCollector")[0].Text, "예시 조사기관")
	p423Want(t, "SURVEY_TYPE", strings.Join(p423Notes(study, "SURVEY_TYPE"), ","), "3")
	p423Want(t, "RELATED_PROGRAM", strings.Join(p423Notes(study, "RELATED_PROGRAM"), ","), "R-1,R-2")
	p423Want(t, "dataKind", p423All(study, "stdyInfo/sumDscr/dataKind")[0].Text, "설문자료")
	if c := p423All(study, "stdyInfo/sumDscr/anlyUnit/concept"); len(c) != 1 || c[0].Text != "Individual" || p423Attr(c[0], "vocab") != "DDI AnalysisUnit" {
		t.Errorf("anlyUnit concept = %v", c)
	}
	if c := p423All(study, "method/dataColl/collMode/concept"); len(c) != 1 || c[0].Text != "SelfAdministeredQuestionnaire.WebBased" {
		t.Errorf("collMode concept = %v", c)
	}
	if c := p423All(study, "method/dataColl/sampProc/concept"); len(c) != 1 || c[0].Text != "Nonprobability.Availability" {
		t.Errorf("sampProc concept = %v", c)
	}
	if len(p423All(study, "method/dataColl/resInstru")) != 0 {
		t.Error("unconfirmed TOOL written")
	}
	if src := p423All(study, "method/dataColl/sources/dataSrc"); len(src) != 2 {
		t.Errorf("dataSrc = %v", src)
	}
	p423Want(t, "weight", p423All(study, "method/dataColl/weight")[0].Text, "가중치 적용\n가중치 변수: WT\n가중치 산출 방법: 사후 층화")
	p423Want(t, "sampleSize", p423All(study, "method/dataColl/targetSampleSize/sampleSize")[0].Text, "100")

	files := p423All(root, "fileDscr")
	p423Want(t, "file IDs", p423Attr(files[0], "ID")+" "+p423Attr(files[1], "ID"), "F01 F02")
	p423Want(t, "keyvar", p423Attr(p423All(files[1], "fileTxt/fileStrc/recGrp")[0], "keyvar"), id("2", "PID"))
	p423Want(t, "caseQnty", p423All(files[0], "fileTxt/dimensns/caseQnty")[0].Text, "100")
	p423Want(t, "varQnty", p423All(files[0], "fileTxt/dimensns/varQnty")[0].Text, "8")
	p423Want(t, "fileCont", p423All(files[0], "fileTxt/fileCont")[0].Text, "사전 응답\n분석 단위: 개인")
	if len(p423Notes(files[0], "FILE_LINK")) != 0 {
		t.Error("F01 has a link")
	}
	var link struct {
		TargetFile string   `json:"targetFile"`
		SourceVars []string `json:"sourceVars"`
		TargetVars []string `json:"targetVars"`
	}
	if err := json.Unmarshal([]byte(p423Notes(files[1], "FILE_LINK")[0]), &link); err != nil {
		t.Fatal(err)
	}
	if link.TargetFile != "F01" || !slices.Equal(link.SourceVars, []string{id("2", "PID")}) || !slices.Equal(link.TargetVars, []string{id("1", "PID")}) {
		t.Errorf("FILE_LINK = %+v", link)
	}
	// The profile's key order, as in its example.
	p423Want(t, "FILE_LINK JSON", p423Notes(files[1], "FILE_LINK")[0],
		`{"targetFile":"F01","sourceVars":["`+id("2", "PID")+`"],"targetVars":["`+id("1", "PID")+`"]}`)
	p423Want(t, "SURVEY_TIME", strings.Join(p423Notes(files[1], "SURVEY_TIME"), ","), "사후")

	// Variables.
	sex := p423VarNode(root, "F01", "SEX")
	p423Want(t, "SEX nature", p423Attr(sex, "nature"), "nominal")
	p423Want(t, "SEX concept", p423All(sex, "concept")[0].Text+"|"+p423Attr(p423All(sex, "concept")[0], "vocab"), "PERSON.GENDER|KOSAC-STD")
	var cats []string
	for _, c := range p423All(sex, "catgry") {
		cats = append(cats, p423All(c, "catValu")[0].Text+":"+p423Attr(c, "missing")+":"+p423Attr(c, "missType"))
	}
	p423Want(t, "SEX catgry", strings.Join(cats, " "), "1:N: 2:N: -9:Y:NO_ANSWER")
	lvl := p423VarNode(root, "F01", "LVL")
	p423Want(t, "leading zero", p423All(lvl, "catgry/catValu")[0].Text, "02")
	q1 := p423VarNode(root, "F01", "Q1")
	p423Want(t, "Q1 REVERSE", strings.Join(p423Notes(q1, "REVERSE"), ","), "Y")
	p423Want(t, "Q1 codes kept", p423All(q1, "catgry/catValu")[0].Text, "1")
	p423Want(t, "Q1 varFormat", p423Attr(p423All(q1, "varFormat")[0], "type"), "numeric")
	p423Want(t, "Q1 qstn", p423Attr(p423All(q1, "qstn")[0], "seqNo")+"|"+p423All(q1, "qstn/qstnLit")[0].Text, "3|만족하십니까?")
	p423Want(t, "Q1 anlysUnit", p423All(q1, "anlysUnit")[0].Text, "개인")
	if len(p423Notes(q1, "PII")) != 0 {
		t.Error("PII=N written")
	}

	// Weights: WT is the weight; analysis vars of its file point at it; keys
	// and other files do not.
	wt := p423VarNode(root, "F01", "WT")
	p423Want(t, "WT wgt", p423Attr(wt, "wgt"), "wgt")
	p423Want(t, "WT KEY_ROLE", strings.Join(p423Notes(wt, "KEY_ROLE"), ","), "WEIGHT")
	p423Want(t, "Q1 wgt-var", p423Attr(q1, "wgt-var"), id("1", "WT"))
	p423Want(t, "PID wgt-var", p423Attr(p423VarNode(root, "F01", "PID"), "wgt-var"), "")
	p423Want(t, "PID KEY_ROLE", strings.Join(p423Notes(p423VarNode(root, "F01", "PID"), "KEY_ROLE"), ","), "PART_NO")
	p423Want(t, "F02 Q1 wgt-var", p423Attr(p423VarNode(root, "F02", "Q1"), "wgt-var"), "")

	// Groups.
	if s := p423Groups(root, "section"); len(s) != 8 {
		t.Errorf("sections = %d, want 4 per file", len(s))
	}
	version := p423Group(root, "version", "SAT")
	p423Want(t, "version ID", p423Attr(version, "ID"), "SAT")
	p423Want(t, "version var", p423Attr(version, "var"), id("1", "Q1")+" "+id("2", "Q1"))
	top, sub := p423Group(root, "analysis", "만족"), p423Group(root, "analysis", "전반")
	p423Want(t, "scale parent", p423Attr(top, "varGrp"), p423Attr(sub, "ID"))
	p423Want(t, "scale parent var", p423Attr(top, "var"), "")
	p423Want(t, "scale items", p423Attr(sub, "var"), strings.Join([]string{id("1", "Q1"), id("1", "Q2"), id("2", "Q1"), id("2", "Q2")}, " "))
	mr := p423Groups(root, "multipleResp")
	if len(mr) != 2 {
		t.Fatalf("multipleResp = %d, want one per file", len(mr))
	}
	p423Want(t, "MR var", p423Attr(mr[0], "var"), id("1", "MR5_1")+" "+id("1", "MR5_2"))
	p423Want(t, "MR_TYPE", strings.Join(p423Notes(mr[0], "MR_TYPE"), ","), "보기별 열")
	split := p423Groups(root, "randomized")
	if len(split) != 2 {
		t.Fatalf("randomized = %d, want one per file", len(split))
	}
	p423Want(t, "randomized var", p423Attr(split[0], "var"), id("1", "Q2"))
	p423Want(t, "SPLIT_VAR", strings.Join(p423Notes(split[0], "SPLIT_VAR"), ","), id("1", "FORM"))

	mats := p423All(root, "otherMat")
	if len(mats) != 2 || p423Attr(mats[0], "type") != "questionnaire" || p423Attr(mats[0], "URI") != "q.pdf" {
		t.Errorf("otherMat = %v", mats)
	}
}

// XML and JSON carry the same tree.
func TestProfile423JSON(t *testing.T) {
	root := p423Tree(t, p423Model())
	x, j := ddi.WriteXML(root), ddi.WriteJSON(root)
	for _, doc := range [][]byte{x, j} {
		parsed, _, err := ddi.Parse(doc)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(ddi.WriteXML(parsed), x) {
			t.Error("tree differs after reading back")
		}
	}
}

func TestProfile423XSD(t *testing.T) {
	cb := p423Model()
	cb.Vars[0].Rec.Set("CODE_SYSTEM", "예시 코드 2026")
	x, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	xsdtest.Validate(t, x)
}

// The profile has no inverse; kosac.Parse says so instead of guessing.
func TestProfile423ParseRejects(t *testing.T) {
	x, err := p423Model().XML()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kosac.Parse(x); err == nil || !strings.Contains(err.Error(), "v4.2.3") {
		t.Fatalf("Parse = %v", err)
	}
}

// The trimmed version selects the profile, and DeriveDefaults adds nothing.
func TestProfile423Dispatch(t *testing.T) {
	cb := p423Model()
	cb.Template.Version.Value = " 4.2.3 "
	codes := len(cb.Codes)
	if s := kosac.DeriveDefaults(cb, kosac.RefData{}); s != nil || len(cb.Codes) != codes || cb.Vars[0].Role != "" {
		t.Fatalf("DeriveDefaults changed the model: %v", s)
	}
	root := p423Tree(t, cb)
	if n := p423All(root, "docDscr/citation/verStmt/notes"); len(n) != 1 {
		t.Fatal("not the profile")
	}
}

// What the workbook does not confirm is left out, never defaulted.
func TestProfile423Unconfirmed(t *testing.T) {
	cb := p423Model()
	cb.Study.Set("WEIGHT_YN", "")
	cb.Study.Set("SPLIT_BALLOT", "미확인: 배정 방식 확인 중")
	cb.Study.Set("SURVEY_TYPE", "")
	cb.Study.Set("MODE", "온라인")            // outside the list
	cb.Vars[4].Rec.Set("MEAS_LEVEL", "연속") // outside the list
	cb.Vars[4].Rec.Set("KEY_ROLE", "응답자")  // outside the list
	root := p423Tree(t, cb)
	study := p423All(root, "stdyDscr")[0]
	if len(p423All(study, "method/dataColl/weight")) != 0 || len(p423All(study, "method/dataColl/collMode")) != 0 || len(p423Notes(study, "SURVEY_TYPE")) != 0 {
		t.Error("unconfirmed study value written")
	}
	kosac.Walk(root, func(n, _ *ddi.Node, path string) bool {
		if p423Attr(n, "wgt") != "" || p423Attr(n, "wgt-var") != "" {
			t.Errorf("%s: weight without WEIGHT_YN=Y", path)
		}
		return true
	})
	if len(p423Groups(root, "randomized")) != 0 {
		t.Error("randomized without SPLIT_BALLOT=Y")
	}
	q1 := p423VarNode(root, "F01", "Q1")
	if p423Attr(q1, "nature") != "" || len(p423Notes(q1, "KEY_ROLE")) != 0 {
		t.Error("unlisted value written")
	}
}

// A file with two weights does not say which applies, so no var points at one.
func TestProfile423SeveralWeights(t *testing.T) {
	cb := p423Model()
	cb.Vars = append(cb.Vars, p423Var("1", "WT2", "MEAS_LEVEL", "비율", "DATA_TYPE", "숫자", "KEY_ROLE", "가중치"))
	if err := kosac.AssignIDs(cb, nil); err != nil {
		t.Fatal(err)
	}
	root := p423Tree(t, cb)
	if p423Attr(p423VarNode(root, "F01", "Q1"), "wgt-var") != "" {
		t.Error("wgt-var with two weights")
	}
	if p423Attr(p423VarNode(root, "F01", "WT2"), "wgt") != "wgt" {
		t.Error("WT2 not a weight")
	}
}

// An ITEM_KEY that cannot be an xs:ID keeps its name in labl.
func TestProfile423ItemKeyID(t *testing.T) {
	cb := p423Model()
	for _, v := range cb.Vars {
		if v.Rec.Get("ITEM_KEY") != "" {
			v.Rec.Set("ITEM_KEY", "1 SAT")
		}
	}
	root := p423Tree(t, cb)
	g := p423Group(root, "version", "1 SAT")
	if g == nil || !strings.HasPrefix(p423Attr(g, "ID"), "G") {
		t.Fatalf("version group = %v", g)
	}
	if is := check.ValidateXML(ddi.WriteXML(root), nil); len(is) > 0 {
		t.Error(is)
	}
}

// Without Q_NO, the numbered columns of one declared question stay together,
// and a var without MR_TYPE never joins them.
func TestProfile423MultipleRespWithoutQNo(t *testing.T) {
	cb := p423Model()
	for _, v := range cb.Vars {
		if v.Rec.Get("MR_TYPE") != "" {
			v.Rec.Set("Q_NO", "")
		}
	}
	cb.Vars = append(cb.Vars, p423Var("1", "MR5_3", "Q_TEXT", "경로: 기타 응답"))
	if err := kosac.AssignIDs(cb, nil); err != nil {
		t.Fatal(err)
	}
	root := p423Tree(t, cb)
	mr := p423Groups(root, "multipleResp")
	if len(mr) != 2 {
		t.Fatalf("multipleResp = %d, want one per file", len(mr))
	}
	p423Want(t, "MR labl", p423All(mr[0], "labl")[0].Text, "MR5")
	p423Want(t, "MR var", p423Attr(mr[0], "var"), cb.Var("1", "MR5_1").ID+" "+cb.Var("1", "MR5_2").ID)
}

func TestProfile423SourceVar(t *testing.T) {
	cb := p423Model()
	q2 := cb.Var("1", "Q2")
	q2.Rec.Set("SOURCE_VAR", "Q1; raw_pre.csv:Q2_OLD")
	q2.Rec.Set("DERIVATION", "Q1 재척도화")
	root := p423Tree(t, cb)
	n := p423VarNode(root, "F01", "Q2")
	p423Want(t, "derivation@var", p423Attr(p423All(n, "derivation")[0], "var"), cb.Var("1", "Q1").ID)
	p423Want(t, "drvdesc", p423All(n, "derivation/drvdesc")[0].Text, "Q1 재척도화")
	p423Want(t, "SOURCE_VAR", strings.Join(p423Notes(n, "SOURCE_VAR"), ","), "raw_pre.csv:Q2_OLD")
}

// Facts the profile cannot represent stop the export and say why.
func TestProfile423Failures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		modify func(*kosac.Codebook)
		want   string
	}{
		{"reserved code meaning", func(cb *kosac.Codebook) { cb.Codes[2].Rec.Set("CODE_LABEL", "모름") }, "SEX 코드 -9"},
		{"missing label on other code", func(cb *kosac.Codebook) { cb.Codes[1].Rec.Set("CODE_LABEL", "무응답") }, "SEX 코드 2"},
		{"unconfirmed label", func(cb *kosac.Codebook) { cb.Codes[0].Rec.Set("CODE_LABEL", "미확인") }, "유효·결측 여부"},
		{"orphan code", func(cb *kosac.Codebook) { cb.Codes[0].Rec.Set("VAR_NAME", "NOPE") }, "03_변수정의에 변수가 없음"},
		{"duplicate var", func(cb *kosac.Codebook) { cb.Vars[1].Rec.Set("VAR_NAME", "PID") }, "같은 변수명"},
		{"row key", func(cb *kosac.Codebook) { cb.Files[0].Rec.Set("UNIQUE_KEY", "NOPE") }, "F1 UNIQUE_KEY: 변수 NOPE 없음"},
		{"link count", func(cb *kosac.Codebook) { cb.Files[1].Rec.Set("LINK_TARGET_VAR", "PID SEX") }, "짝지어지지"},
		{"link file", func(cb *kosac.Codebook) { cb.Files[1].Rec.Set("LINK_FILE_NO", "9") }, "연결 대상 파일이 없음"},
		{"case count", func(cb *kosac.Codebook) { cb.Files[0].Rec.Set("VALID_N", "약 100") }, "0 이상의 정수가 아님"},
		{"weight var", func(cb *kosac.Codebook) { cb.Study.Set("WEIGHT_VAR", "NOPE") }, "WEIGHT_VAR: 변수 NOPE 없음"},
		{"mr type", func(cb *kosac.Codebook) { cb.Vars[6].Rec.Set("MR_TYPE", "이분형") }, "MR_TYPE"},
		{"split items", func(cb *kosac.Codebook) {
			for _, v := range cb.Vars {
				v.Rec.Set("UNIVERSE_EXPR", "")
			}
		}, "분할 문항을 정할 수 없음"},
		{"split var", func(cb *kosac.Codebook) { cb.Study.Set("SPLIT_VAR", "") }, "SPLIT_VAR 없음"},
		{"scale level", func(cb *kosac.Codebook) { cb.Vars[5].Rec.Set("SCALE", "만족") }, "가장 아래 영역"},
		{"missing status var", func(cb *kosac.Codebook) { cb.Vars[4].Rec.Set("MISSING_STATUS_VAR", "NOPE") }, "MISSING_STATUS_VAR: 변수 NOPE 없음"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cb := p423Model()
			tc.modify(cb)
			_, err := cb.Tree()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Tree error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestMissing423(t *testing.T) {
	for _, tc := range []struct {
		code, label, typ string
		ok               bool
	}{
		{"-9", "무응답", "NO_ANSWER", true},
		{"-9", "무응답(응답 안 함)", "NO_ANSWER", true},
		{"-8", "비해당", "NOT_APPLICABLE", true},
		{"-8", "해당 없음", "NOT_APPLICABLE", true},
		{"-7", "모름·거절", "DONT_KNOW_REFUSED", true},
		{"-7", "모름/거절", "DONT_KNOW_REFUSED", true},
		{"-7", "응답하지 않음", "DONT_KNOW_REFUSED", true},
		{"-9", "모름", "", false},  // another code's meaning
		{"-8", "-8점", "", false}, // not a confirmed missing meaning
		{"-1", "불만", "", true},   // negative is not missing by value
		{"9", "미상", "", true},    // gender 9 unknown is a valid code
		{"9", "무응답", "", false},  // a missing reason on another code
		{"1", "", "", false},     // no label, no meaning
		{"1", "미확인", "", false},
		{"1", "확인 불가: 원자료 없음", "", false},
	} {
		typ, ok := kosac.Missing423(tc.code, tc.label)
		if typ != tc.typ || ok != tc.ok {
			t.Errorf("Missing423(%q, %q) = %q, %v; want %q, %v", tc.code, tc.label, typ, ok, tc.typ, tc.ok)
		}
	}
}

package kosac

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

func hex(c byte) string { return strings.Repeat(string(c), 64) }

func sv(file, name string, kv ...string) *Var {
	return &Var{Rec: rec(append([]string{"FILE_NO", file, "VAR_NAME", name}, kv...)...)}
}

func code(file, name, value, label, typ string) *Code {
	return &Code{Rec: rec("FILE_NO", file, "VAR_NAME", name, "CODE", value, "CODE_LABEL", label), Type: typ}
}

// smallModel is a two-file codebook with one of each group type, a weight,
// strata, a cleansing rule and a package. IDs are V1… in 03 order.
func smallModel() *Codebook {
	idx := sv("1", "IDX", "VAR_LABEL", "응답자 ID", "VAR_GROUP", "응답자 특성", "MEAS_LEVEL", "문자", "PII", "N", "SENSITIVE", "N")
	idx.Role = "ID"
	sq1 := sv("1", "SQ1", "VAR_LABEL", "성별", "Q_NO", "SQ1", "Q_TEXT", "귀하의 성별은?", "VAR_GROUP", "응답자 특성",
		"MEAS_LEVEL", "명목", "PII", "N", "SENSITIVE", "N")
	sq1.Analysis = Analysis{Role: "ITEM", DimMeasure: "DIM", AggRules: "COUNT;RATIO", PhysicalName: "sq1",
		LogicalName: "성별", IncludeYn: "Y", CodeSetID: "CODES2_1"}
	sq2 := sv("1", "SQ2", "VAR_LABEL", "연령", "VAR_GROUP", "응답자 특성", "MEAS_LEVEL", "연속")
	grid := func(name, row string) *Var {
		v := sv("1", name, "VAR_LABEL", "과학기술 인식: "+row, "Q_TEXT", "과학기술에 대해 어떻게 생각하십니까? ▶ ["+row+"]",
			"VAR_GROUP", "인식", "MEAS_LEVEL", "서열")
		v.GridGroup, v.RowLabel = "QA1_1", row
		return v
	}
	mr := func(name, label string) *Var {
		v := sv("1", name, "VAR_LABEL", label, "VAR_GROUP", "인식", "MEAS_LEVEL", "명목", "MR_TYPE", "보기별 열")
		v.MRGroup, v.NoneOption = "QA19", "Y"
		return v
	}
	rank := func(name, no string) *Var {
		v := sv("1", name, "VAR_LABEL", "경쟁력 있는 과학기술 분야 ("+no+"순위)", "VAR_GROUP", "인식", "MEAS_LEVEL", "명목", "MR_TYPE", "순위별 열")
		v.MRGroup, v.RankNo = "QA20", no
		return v
	}
	wt := sv("1", "WT", "VAR_LABEL", "가중치", "VAR_GROUP", "가중치", "MEAS_LEVEL", "연속")
	wt.Role = "WEIGHT"
	sq1b := sv("2", "SQ1", "VAR_LABEL", "성별", "VAR_GROUP", "응답자 특성", "MEAS_LEVEL", "명목")

	cb := &Codebook{
		Study: rec(
			"RULE_VERSION", "v1.0", "TITLE", "2024 과학기술 국민인식도 조사", "PURPOSE", "국민 인식 파악",
			"SUMMARY", "인식도 조사", "FIELD_ORG", "㈜서베이피플", "FIELD_START", "2024-10-04", "FIELD_END", "2024-10-28",
			"KEYWORDS", " 과학기술 관심도; 과학자본;", "QUESTIONNAIRE", "최종보고서 부록 수록 (PDF 336쪽~)",
			"REPORT_FILE", "최종보고서.pdf", "EXTRA_FILES", "부록1.xlsx", "UNIVERSE", "전국 만 14~69세 남녀",
			"POP_SOURCE", "주민등록 인구", "ANALYSIS_UNIT", "개인", "SAMPLING_TYPE", "확률-층화",
			"SAMPLING_DESC", "성·연령·지역 비례 층화", "STRATA_VARS", "SQ1; SQ2", "TARGET_N", "10,000",
			"VALID_N", "10000", "SPLIT_BALLOT", "N", "MODE", "온라인", "EXCLUDE_N", "0", "EDIT_METHOD", "검토",
			"WEIGHT_YN", "Y", "WEIGHT_VAR", "WT", "WEIGHT_METHOD", "사후층화", "SURVEY_TYPE", "연구조사",
			"DATA_NO", "문화-01-01",
		),
		Admin: StudyAdmin{CategoryCode: "문화", Round: Round{Year: "2024", WaveType: "main"}, WallaID: "ppbCL08Ex69QbEEy9ajj"},
		Files: []*File{
			{Rec: rec("FILE_NO", "1", "FILE_NAME", "data1.csv", "FILE_FORMAT", "CSV", "CONTENT", "일반 국민 응답",
				"UNIT", "개인", "ID_VAR", "IDX", "VALID_N", "10000"),
				SensLevel: "3", PackageName: "문화-01-01_2024_v1_data_1.csv", SHA256: hex('c'), Bytes: "2048"},
			{Rec: rec("FILE_NO", "2", "FILE_NAME", "data2.csv", "FILE_FORMAT", "CSV", "CONTENT", "추적 조사 응답",
				"UNIT", "개인", "ID_VAR", "walla_response_id", "VALID_N", "500"),
				SensLevel: "2", Cleansing: &Cleansing{AppliedBy: `kosac\hsham`, AppliedAt: "2026-11-02T10:50:00+09:00",
					RawFileRef: "F2_raw", CellsChanged: "3", RowsAffected: "3", LogFile: "cleansing.json",
					Rules: []CleansingRule{{ID: "CL-01", VarName: "SQ1", Action: "REPLACE", Condition: "SQ1=9", Rows: "3",
						Source: "MANUAL", Reason: "코드 오류", Evidence: "용역사 회신", Map: []MapEntry{{"9", "1"}}}}}},
		},
		Vars: []*Var{idx, sq1, sq2, grid("QA1_1", "가"), grid("QA1_2", "나"),
			mr("QA19_1", "STEM 학위·자격: 대학원 학위"), mr("QA19_2", "STEM 학위·자격: 학사"), mr("QA19_3", "STEM 학위·자격: 없음"),
			rank("QA20_1", "1"), rank("QA20_2", "2"), wt, sq1b},
		Codes: []*Code{
			code("1", "SQ1", "1", "남성", "VALID"), code("1", "SQ1", "2", "여성", ""), code("1", "SQ1", "-6", "이전 값", "MISSING"),
			code("1", "SQ1", "-7", "모르겠다", "DK"), code("1", "SQ1", "-8", "묻지 않음", "MISSING"),
			code("2", "SQ1", "1", "남성", "VALID"),
		},
		Package: &Package{ExportID: "exp_20261103_0001", CodebookVersion: "1", ApprovedBy: `kosac\hsham`,
			ApprovedAt: "2026-11-03T10:12:00+09:00", ExportedAt: "2026-11-03", Software: "Walla", SoftwareVersion: "2.4",
			Source: SourceSurvey{Kind: "upload", TemplateVersion: "3.2"},
			Files: []PackageFile{
				{Role: "codebook", Name: "문화-01-01_2024_v1_codebook.xml", Bytes: "999", SHA256: hex('a')},
				{Role: "data", FileRef: "F1", Name: "문화-01-01_2024_v1_data_1.csv", Bytes: "2048", SHA256: hex('c')},
				{Role: "report", Name: "문화-01-01_2024_v1_report.pdf", Bytes: "1234", SHA256: hex('d')},
				{Role: "questionnaire", Name: "문화-01-01_2024_v1_report.pdf", Pages: "336-"},
				{Role: "extra", Name: "문화-01-01_2024_v1_extra_1.xlsx", Bytes: "10", SHA256: hex('e')},
			},
			Validation: &Validation{PassedAt: "2026-11-01T15:40:00+09:00", Ruleset: "2026.10", WarningCount: "6", Notes: []string{"C08 145건"}},
		},
	}
	cb.Codes[0].StdCode, cb.Codes[0].StdSet = "M", "CC_GENDER"
	cb.Vars[4].Rec.Set("REVERSE", "N") // not in the template's list, but written as is so it reads back
	setIDs(cb)
	return cb
}

func setIDs(cb *Codebook) {
	for i, v := range cb.Vars {
		v.ID = fmt.Sprintf("V%d", i+1)
	}
}

// fullModel fills every optional slot of smallModel.
func fullModel() *Codebook {
	cb := smallModel()
	for _, kv := range [][2]string{
		{"PROJECT_TITLE", "2024 과학기술문화 사회조사연구 용역"}, {"PROJECT_NO", "20240415B4D-00"}, {"FUNDER", "한국과학창의재단"},
		{"PI_ORG", "서울대학교"}, {"PI_NAME", "임동균"}, {"TOOL", "확인 불가: 보고서 미기재"}, {"SAMPLE_FRAME", "틸리언 패널"},
		{"CLUSTER_VAR", "SQ2"}, {"SPLIT_BALLOT", "Y"}, {"SPLIT_VAR", "walla_row_no"}, {"CONTACT_N", "20,000"},
		{"RESPONSE_RATE", "45.5"}, {"RR_FORMULA", "AAPOR RR1"}, {"EXCLUDE_N", "3.0"}, {"EXCLUDE_RULE", "불성실 응답 제외"},
		{"SCHOOL_CODE_YEAR", "2024"}, {"BIZ_NAME", "과학문화 확산; 사업"}, {"BIZ_YEAR", "2024"}, {"EXTRA_FILES", "부록1.xlsx; 부록2.hwp"},
	} {
		cb.Study.Set(kv[0], kv[1])
	}
	cb.Admin.Round = Round{Year: "2024", WaveType: "pre", WaveNo: "1"}
	cb.Package.CodebookVersion, cb.Package.PreviousVersion = "2", "CB-문화-01-01-2024-pre1-v1"
	cb.Package.Source.PublishedVersion = "3"
	f := cb.Files[0]
	f.Rec.Set("LINK_VAR", "IDX")
	f.Rec.Set("LINK_FILE_NO", "2")
	f.Rec.Set("NOTE", "응답자 파일")
	f.LinkLevel, f.RawName, f.RawSHA256, f.RawBytes = "RESPONDENT", "data1_raw.csv", hex('f'), "2050"

	sq1 := cb.Vars[1]
	sq1.Synonyms = []string{"남녀", "성"}
	sq1.FK = &FK{CodeSet: "CC_GENDER", DimTable: "dw.t_common_code", KeyColumn: "standard_code", Mapping: "CODE",
		CodeSystem: "KOSAC", RefYear: "2026", LinkKey: "N", CodeSetVersion: "2026.2", ValidatedAt: "2026-11-03T10:05:00+09:00",
		MatchedRows: "10,000", UnmatchedRows: "0"}
	qa := cb.Vars[3]
	for _, kv := range [][2]string{{"RESP_UNIVERSE", "SQ1=1"}, {"DERIVATION", "역코딩"}, {"STD_ID", "SCI.TRUST"},
		{"NOTE", "비고"}, {"REVERSE", "Y"}, {"Q_NO", "QA1"}} {
		qa.Rec.Set(kv[0], kv[1])
	}
	qa.Scale = Scale{Min: "1", Max: "5", Dir: "NEG"}
	etc := sv("1", "SQ1_ETC", "VAR_LABEL", "성별 기타", "VAR_GROUP", "응답자 특성", "MEAS_LEVEL", "문자")
	etc.TextFormat, etc.ParentVar = "TEXT", "SQ1"
	born := sv("1", "BORN", "VAR_LABEL", "생일", "VAR_GROUP", "응답자 특성", "MEAS_LEVEL", "날짜", "PII", "Y")
	cb.Vars = append(cb.Vars, etc, born)
	c := code("1", "SQ1", "-9", "유효하지 않음", "NA")
	c.Rec.Set("NOTE", "응답 오류")
	cb.Codes = append(cb.Codes, c)
	setIDs(cb)
	return cb
}

func tree(t *testing.T, cb *Codebook) *ddi.Node {
	t.Helper()
	n, err := cb.Tree()
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// at follows path from n through the first child of each name; nil when absent.
func at(n *ddi.Node, path string) *ddi.Node {
	for _, name := range strings.Split(path, "/") {
		i := slices.IndexFunc(n.Children, func(c *ddi.Node) bool { return c.Name == name })
		if i < 0 {
			return nil
		}
		n = n.Children[i]
	}
	return n
}

func all(n *ddi.Node, name string) []*ddi.Node {
	var out []*ddi.Node
	for _, c := range n.Children {
		if c.Name == name {
			out = append(out, c)
		}
	}
	return out
}

func withAttr(ns []*ddi.Node, name, value string) *ddi.Node {
	for _, n := range ns {
		if v, _ := n.Attr(name); v == value {
			return n
		}
	}
	return nil
}

func frag(n *ddi.Node) string {
	if n == nil {
		return "<nil>"
	}
	return ddi.WriteFragment([]*ddi.Node{n})
}

func names(ns []*ddi.Node) []string {
	var out []string
	for _, n := range ns {
		out = append(out, n.Name)
	}
	return out
}

func walk(n *ddi.Node, f func(*ddi.Node)) {
	f(n)
	for _, c := range n.Children {
		walk(c, f)
	}
}

func eq(t *testing.T, what, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s:\n got %s\nwant %s", what, got, want)
	}
}

// checkPlacement asserts that under every DDI element the kosac children come
// after the DDI ones, in KosacChildren order, and kosac attributes come after
// the DDI ones in KosacAttrs order and only where the table allows them.
func checkPlacement(t *testing.T, n *ddi.Node) {
	t.Helper()
	if strings.HasPrefix(n.Name, "kosac:") {
		return
	}
	k, kosacSeen := 0, false
	for _, c := range n.Children {
		local, isKosac := strings.CutPrefix(c.Name, "kosac:")
		if !isKosac {
			if kosacSeen {
				t.Errorf("<%s>: DDI child <%s> after a kosac child", n.Name, c.Name)
			}
			checkPlacement(t, c)
			continue
		}
		kosacSeen = true
		i := slices.Index(KosacChildren[n.Name][k:], local)
		if i < 0 {
			t.Errorf("<%s>: <%s> is not allowed here or out of order", n.Name, c.Name)
			continue
		}
		k += i
	}
	k, kosacSeen = 0, false
	for _, a := range n.Attrs {
		if a.Name == "xmlns:kosac" {
			continue
		}
		local, isKosac := strings.CutPrefix(a.Name, "kosac:")
		if !isKosac {
			if kosacSeen {
				t.Errorf("<%s>: DDI attribute %s after a kosac attribute", n.Name, a.Name)
			}
			continue
		}
		kosacSeen = true
		i := slices.Index(KosacAttrs[n.Name][k:], local)
		if i < 0 {
			t.Errorf("<%s>: attribute %s is not allowed here or out of order", n.Name, a.Name)
			continue
		}
		k += i + 1
	}
}

func TestTreeSmall(t *testing.T) {
	cb := smallModel()
	root := tree(t, cb)

	eq(t, "root attrs", ddi.FormatAttrs(root.Attrs),
		`xmlns="ddi:codebook:2_5" xmlns:kosac="urn:kosac:survey-meta:3.3" version="2.5" ID="CB-문화-01-01-2024-v1"`)
	if got := names(root.Children); !slices.Equal(got, []string{"docDscr", "stdyDscr", "fileDscr", "fileDscr", "dataDscr", "otherMat", "otherMat", "otherMat"}) {
		t.Errorf("root children %v", got)
	}
	checkPlacement(t, root)
	walk(root, func(n *ddi.Node) {
		if len(n.Attrs) == 0 && n.Text == "" && n.Raw == "" && len(n.Children) == 0 {
			t.Errorf("empty <%s>", n.Name)
		}
		if n.Name == "rspStmt" || n.Name == "anlyInfo" {
			t.Errorf("<%s> written with no values for it", n.Name)
		}
	})

	t.Run("docDscr", func(t *testing.T) {
		eq(t, "docDscr", frag(at(root, "docDscr")), `<docDscr><citation><titlStmt><titl>2024 과학기술 국민인식도 조사 코드북 v1</titl></titlStmt>`+
			`<prodStmt><prodDate date="2026-11-03">2026-11-03</prodDate><software version="2.4">Walla</software></prodStmt></citation>`+
			`<kosac:package exportId="exp_20261103_0001" codebookVersion="1" approvedBy="kosac\hsham" approvedAt="2026-11-03T10:12:00+09:00" cleansed="Y">`+
			`<kosac:sourceSurvey wallaId="ppbCL08Ex69QbEEy9ajj" kind="upload" templateVersion="3.2"/>`+
			`<kosac:file role="codebook" name="문화-01-01_2024_v1_codebook.xml"/>`+
			`<kosac:file role="data" fileRef="F1" name="문화-01-01_2024_v1_data_1.csv" bytes="2048" sha256="`+hex('c')+`"/>`+
			`<kosac:file role="report" name="문화-01-01_2024_v1_report.pdf" bytes="1234" sha256="`+hex('d')+`"/>`+
			`<kosac:file role="questionnaire" name="문화-01-01_2024_v1_report.pdf" pages="336-"/>`+
			`<kosac:file role="extra" name="문화-01-01_2024_v1_extra_1.xlsx" bytes="10" sha256="`+hex('e')+`"/>`+
			`<kosac:validation passedAt="2026-11-01T15:40:00+09:00" ruleset="2026.10" warningCount="6"><kosac:validationNote>C08 145건</kosac:validationNote></kosac:validation>`+
			`</kosac:package></docDscr>`)
	})

	t.Run("stdyDscr", func(t *testing.T) {
		s := at(root, "stdyDscr")
		eq(t, "citation", frag(at(s, "citation")), `<citation><titlStmt><titl>2024 과학기술 국민인식도 조사</titl>`+
			`<IDNo agency="walla">ppbCL08Ex69QbEEy9ajj</IDNo></titlStmt><prodStmt><producer>㈜서베이피플</producer></prodStmt></citation>`)
		eq(t, "stdyInfo", frag(at(s, "stdyInfo")), `<stdyInfo><subject><keyword>과학기술 관심도</keyword><keyword>과학자본</keyword></subject>`+
			`<abstract contentType="purpose">국민 인식 파악</abstract><abstract contentType="abstract">인식도 조사</abstract>`+
			`<sumDscr><collDate event="start" date="2024-10-04"/><collDate event="end" date="2024-10-28"/>`+
			`<anlyUnit>개인</anlyUnit><universe clusion="I">전국 만 14~69세 남녀</universe></sumDscr></stdyInfo>`)
		eq(t, "method", frag(at(s, "method")), `<method><dataColl><sampProc>성·연령·지역 비례 층화</sampProc>`+
			`<targetSampleSize><sampleSize>10000</sampleSize></targetSampleSize><collMode>온라인</collMode>`+
			`<weight>사후층화</weight><cleanOps>검토</cleanOps>`+
			`<kosac:sampling type="확률-층화" strataVars="V2 V3"/><kosac:splitBallot yn="N"/><kosac:weight yn="Y" var="V11"/>`+
			`<kosac:excludeN>0</kosac:excludeN></dataColl></method>`)
		eq(t, "kosac:study", frag(at(s, "kosac:study")), `<kosac:study ruleVersion="v1.0" surveyType="연구조사" dataNo="문화-01-01" categoryCode="문화" validN="10000">`+
			`<kosac:round year="2024" waveType="main"/><kosac:popSource>주민등록 인구</kosac:popSource></kosac:study>`)
	})

	t.Run("fileDscr", func(t *testing.T) {
		fs := all(root, "fileDscr")
		eq(t, "F1", frag(fs[0]), `<fileDscr ID="F1"><fileTxt><fileName>문화-01-01_2024_v1_data_1.csv</fileName><fileCont>일반 국민 응답</fileCont>`+
			`<dimensns><caseQnty>10000</caseQnty><varQnty>17</varQnty></dimensns><fileType charset="UTF-8">CSV</fileType></fileTxt>`+
			`<kosac:file fileNo="1" unit="개인" sensLevel="3" sha256="`+hex('c')+`" bytes="2048" sourceName="data1.csv">`+
			`<kosac:idVar ref="V1"/><kosac:systemIdVar ref="S1"/></kosac:file></fileDscr>`)
		// SQ1 is in both files; the rule on file 2 names file 2's SQ1.
		eq(t, "F2", frag(fs[1]), `<fileDscr ID="F2"><fileTxt><fileName>data2.csv</fileName><fileCont>추적 조사 응답</fileCont>`+
			`<dimensns><caseQnty>500</caseQnty><varQnty>7</varQnty></dimensns><fileType charset="UTF-8">CSV</fileType></fileTxt>`+
			`<kosac:file fileNo="2" unit="개인" sensLevel="2"><kosac:idVar ref="S7"/><kosac:systemIdVar ref="S7"/></kosac:file>`+
			`<kosac:cleansing appliedBy="kosac\hsham" appliedAt="2026-11-02T10:50:00+09:00" rawFileRef="F2_raw" cellsChanged="3" rowsAffected="3" logFile="cleansing.json">`+
			`<kosac:rule id="CL-01" var="V12" action="REPLACE" condition="SQ1=9" rows="3" source="MANUAL" reason="코드 오류" evidence="용역사 회신">`+
			`<kosac:map><kosac:entry from="9" to="1"/></kosac:map></kosac:rule></kosac:cleansing></fileDscr>`)
	})

	dd := at(root, "dataDscr")
	vs := all(dd, "var")

	t.Run("varGrp", func(t *testing.T) {
		gs := all(dd, "varGrp")
		var got []string
		for _, g := range gs {
			got = append(got, frag(g))
		}
		want := []string{
			`<varGrp ID="G1" type="section" var="V1 V2 V3 V12"><labl>응답자 특성</labl></varGrp>`,
			`<varGrp ID="G2" type="section" var="V4 V5 V6 V7 V8 V9 V10"><labl>인식</labl></varGrp>`,
			`<varGrp ID="G3" type="section" var="V11"><labl>가중치</labl></varGrp>`,
			`<varGrp ID="G4" type="grid" var="V4 V5"><labl>과학기술에 대해 어떻게 생각하십니까?</labl></varGrp>`,
			`<varGrp ID="G5" type="multipleResp" var="V6 V7 V8" kosac:mrGroup="QA19"><labl>STEM 학위·자격</labl><kosac:noneOption yn="Y"/></varGrp>`,
			`<varGrp ID="G6" type="other" otherType="rank" var="V9 V10" kosac:mrGroup="QA20"><labl>경쟁력 있는 과학기술 분야</labl></varGrp>`,
		}
		if !slices.Equal(got, want) {
			t.Errorf("varGrp:\n got %s\nwant %s", strings.Join(got, "\n    "), strings.Join(want, "\n    "))
		}
	})

	t.Run("system vars", func(t *testing.T) {
		if got := names(dd.Children)[6:18]; slices.ContainsFunc(got, func(s string) bool { return s != "var" }) {
			t.Fatalf("dataDscr children %v", got)
		}
		for fi := range 2 {
			for si, s := range SystemVars {
				v := vs[fi*len(SystemVars)+si]
				id, file := fmt.Sprintf("S%d", fi*len(SystemVars)+si+1), fmt.Sprintf("F%d", fi+1)
				if attr(v, "ID") != id || attr(v, "name") != s.Name || attr(v, "files") != file || attr(v, "kosac:seq") != fmt.Sprint(si+1) {
					t.Errorf("%s: %s", id, ddi.FormatAttrs(v.Attrs))
				}
				if l := at(v, "labl"); l == nil || l.Text != s.Label {
					t.Errorf("%s labl: %s", id, frag(l))
				}
				if at(v, "catgry") != nil || at(v, "kosac:missingCodes") != nil {
					t.Errorf("%s has missing codes", id)
				}
			}
		}
		eq(t, "S7", frag(withAttr(vs, "ID", "S7")), `<var ID="S7" name="walla_response_id" files="F2" intrvl="discrete" kosac:seq="1" kosac:role="SYSTEM" kosac:measLevel="text" kosac:pii="N">`+
			`<location fileid="F2"/><labl>왈라 응답 ID</labl><varFormat type="character"/>`+
			`<kosac:analysis physicalName="walla_response_id" dimMeasure="KEY" includeYn="N"/></var>`)
		eq(t, "S4 format", frag(at(withAttr(vs, "ID", "S4"), "varFormat")), `<varFormat type="character" category="date"/>`)
		eq(t, "S2 intrvl", attr(withAttr(vs, "ID", "S2"), "intrvl"), "contin")
	})

	t.Run("survey vars", func(t *testing.T) {
		var order []string
		for _, v := range vs[12:] {
			order = append(order, attr(v, "ID")+"/"+attr(v, "kosac:seq"))
		}
		if want := []string{"V1/7", "V2/8", "V3/9", "V4/10", "V5/11", "V6/12", "V7/13", "V8/14", "V9/15", "V10/16", "V11/17", "V12/7"}; !slices.Equal(order, want) {
			t.Errorf("vars in order %v, want %v", order, want)
		}
		eq(t, "V2", frag(withAttr(vs, "ID", "V2")), `<var ID="V2" name="SQ1" files="F1" intrvl="discrete" kosac:seq="8" kosac:role="ITEM" kosac:measLevel="nominal" kosac:pii="N" kosac:sensitive="N">`+
			`<location fileid="F1"/><labl>성별</labl><qstn seqNo="SQ1"><qstnLit>귀하의 성별은?</qstnLit></qstn>`+
			`<catgry><catValu>1</catValu><labl>남성</labl><kosac:codeType>VALID</kosac:codeType><kosac:stdCode set="CC_GENDER">M</kosac:stdCode></catgry>`+
			`<catgry><catValu>2</catValu><labl>여성</labl></catgry>`+
			`<catgry missing="Y"><catValu>-6</catValu><labl>이전 값</labl><kosac:codeType>MISSING</kosac:codeType></catgry>`+
			`<catgry missing="Y"><catValu>-7</catValu><labl>모르겠다</labl><kosac:codeType>DK</kosac:codeType></catgry>`+
			`<catgry missing="Y"><catValu>-8</catValu><labl>묻지 않음</labl><kosac:codeType>MISSING</kosac:codeType></catgry>`+
			`<varFormat type="numeric"/><kosac:missingCodes>-8;-7;-6</kosac:missingCodes><kosac:codeSetId>CODES2_1</kosac:codeSetId>`+
			`<kosac:analysis physicalName="sq1" dimMeasure="DIM" aggRules="COUNT;RATIO" includeYn="Y" logicalName="성별"/></var>`)
		eq(t, "V4", frag(withAttr(vs, "ID", "V4")), `<var ID="V4" name="QA1_1" files="F1" intrvl="discrete" kosac:seq="10" kosac:measLevel="ordinal" kosac:rowLabel="가">`+
			`<location fileid="F1"/><labl>과학기술 인식: 가</labl><qstn><qstnLit>과학기술에 대해 어떻게 생각하십니까? ▶ [가]</qstnLit></qstn>`+
			`<varFormat type="numeric"/></var>`)
		eq(t, "V5 attrs", ddi.FormatAttrs(withAttr(vs, "ID", "V5").Attrs),
			`ID="V5" name="QA1_2" files="F1" intrvl="discrete" kosac:seq="11" kosac:measLevel="ordinal" kosac:reverse="N" kosac:rowLabel="나"`)
		eq(t, "V9", frag(withAttr(vs, "ID", "V9")), `<var ID="V9" name="QA20_1" files="F1" intrvl="discrete" kosac:seq="15" kosac:measLevel="nominal" kosac:mrType="순위별 열" kosac:mrGroup="QA20" kosac:rankNo="1">`+
			`<location fileid="F1"/><labl>경쟁력 있는 과학기술 분야 (1순위)</labl><varFormat type="numeric"/></var>`)
		eq(t, "V11", ddi.FormatAttrs(withAttr(vs, "ID", "V11").Attrs), `ID="V11" name="WT" wgt="wgt" files="F1" intrvl="contin" kosac:seq="17" kosac:role="WEIGHT" kosac:measLevel="scale"`)
		eq(t, "V12", frag(withAttr(vs, "ID", "V12")), `<var ID="V12" name="SQ1" files="F2" intrvl="discrete" kosac:seq="7" kosac:measLevel="nominal" kosac:cleansed="Y">`+
			`<location fileid="F2"/><labl>성별</labl><catgry><catValu>1</catValu><labl>남성</labl><kosac:codeType>VALID</kosac:codeType></catgry>`+
			`<varFormat type="numeric"/></var>`)
	})

	t.Run("otherMat", func(t *testing.T) {
		ms := all(root, "otherMat")
		eq(t, "report", frag(ms[0]), `<otherMat type="report" level="study" URI="문화-01-01_2024_v1_report.pdf"><labl>최종보고서.pdf</labl>`+
			`<kosac:sha256>`+hex('d')+`</kosac:sha256><kosac:bytes>1234</kosac:bytes></otherMat>`)
		eq(t, "questionnaire", frag(ms[1]), `<otherMat type="questionnaire" level="study" URI="문화-01-01_2024_v1_report.pdf">`+
			`<labl>최종보고서 부록 수록 (PDF 336쪽~)</labl><kosac:pages>336-</kosac:pages></otherMat>`)
		eq(t, "extra", frag(ms[2]), `<otherMat type="extra" level="study" URI="문화-01-01_2024_v1_extra_1.xlsx"><labl>부록1.xlsx</labl>`+
			`<kosac:sha256>`+hex('e')+`</kosac:sha256><kosac:bytes>10</kosac:bytes></otherMat>`)
	})

	t.Run("XML and JSON", func(t *testing.T) {
		x, err := cb.XML()
		if err != nil || string(x) != string(ddi.WriteXML(root)) {
			t.Errorf("XML: %v", err)
		}
		j, err := cb.JSON()
		if err != nil || string(j) != string(ddi.WriteJSON(root)) {
			t.Errorf("JSON: %v", err)
		}
	})
}

func TestTreeUnverified(t *testing.T) {
	cb := smallModel()
	cb.Study.Set("POP_SOURCE", "확인 불가: a")
	cb.Study.Set("FIELD_START", "확인 불가: b")
	cb.Study.Set("TOOL", "확인 불가:")
	cb.Files[1].Rec.Set("VALID_N", "확인 불가: c")
	cb.Vars[2].Rec.Set("RESP_UNIVERSE", "확인 불가: d")
	root := tree(t, cb)

	study := at(root, "stdyDscr/kosac:study")
	var got []string
	for _, u := range all(study, "kosac:unverified") {
		got = append(got, frag(u))
	}
	want := []string{
		`<kosac:unverified key="FIELD_START">b</kosac:unverified>`,
		`<kosac:unverified key="POP_SOURCE">a</kosac:unverified>`,
		`<kosac:unverified key="TOOL"/>`,
		`<kosac:unverified key="VALID_N" file="F2">c</kosac:unverified>`,
		`<kosac:unverified key="RESP_UNIVERSE" var="V3">d</kosac:unverified>`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("unverified:\n got %s\nwant %s", strings.Join(got, "\n    "), strings.Join(want, "\n    "))
	}
	eq(t, "popSource keeps the text", frag(at(study, "kosac:popSource")), `<kosac:popSource>확인 불가: a</kosac:popSource>`)
	eq(t, "resInstru keeps the text", frag(at(root, "stdyDscr/method/dataColl/resInstru")), `<resInstru>확인 불가:</resInstru>`)
	eq(t, "no start date", frag(at(root, "stdyDscr/stdyInfo/sumDscr/collDate")), `<collDate event="end" date="2024-10-28"/>`)
	eq(t, "no caseQnty", frag(at(all(root, "fileDscr")[1], "fileTxt/dimensns")), `<dimensns><varQnty>7</varQnty></dimensns>`)
	eq(t, "universe keeps the text", frag(at(withAttr(all(at(root, "dataDscr"), "var"), "ID", "V3"), "universe")), `<universe clusion="I">확인 불가: d</universe>`)
}

// TestRebuildFromNormalized stands in for the fixed point until Parse exists:
// Parse can only return what the XML holds, the normalized text, so a value
// derived from a cell must come out the same from the normalized cell.
func TestRebuildFromNormalized(t *testing.T) {
	cb := smallModel()
	cb.Package.Files = cb.Package.Files[:2] // URIs come from the cells
	cb.Vars[8].Rec.Set("VAR_LABEL", "경쟁력 있는 과학기술 분야\n (1순위)")
	cb.Vars[9].Rec.Set("VAR_LABEL", "경쟁력 있는 과학기술 분야\n (2순위)")
	cb.Vars[5].Rec.Set("VAR_LABEL", " STEM 학위·자격:\n   대학원")
	cb.Vars[6].Rec.Set("VAR_LABEL", " STEM 학위·자격:\n   학사")
	cb.Vars[3].Rec.Set("Q_TEXT", "과학기술에 대해\n    어떻게 생각하십니까? ▶ [가]")
	cb.Vars[4].Rec.Set("Q_TEXT", "과학기술에 대해\n    어떻게 생각하십니까? ▶ [나]")
	cb.Vars[0].Rec.Set("VAR_GROUP", " 응답자\n   특성")
	cb.Study.Set("REPORT_FILE", " 최종\n   보고서.pdf ")
	cb.Study.Set("QUESTIONNAIRE", "최종보고서 부록 수록\n    (PDF 336쪽~)")
	cb.Study.Set("EXTRA_FILES", " 부록\n  1.xlsx ;부록2.hwp")
	cb.Study.Set("KEYWORDS", "과학\n  기술; 자본")
	cb.Codes[4].Rec.Set("CODE", " -8\n ")
	first, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}

	norm := func(r Record, keys ...string) {
		for _, k := range keys {
			r.Set(k, ddi.NormalizeText(r.Get(k)))
		}
	}
	joined := func(r Record, key string) {
		var items []string
		for _, it := range strings.Split(r.Get(key), ";") {
			if it = ddi.NormalizeText(it); it != "" {
				items = append(items, it)
			}
		}
		r.Set(key, strings.Join(items, "; "))
	}
	for _, v := range cb.Vars {
		norm(v.Rec, "VAR_LABEL", "Q_TEXT", "VAR_GROUP")
	}
	for _, c := range cb.Codes {
		norm(c.Rec, "CODE")
	}
	norm(cb.Study, "REPORT_FILE", "QUESTIONNAIRE")
	joined(cb.Study, "EXTRA_FILES")
	joined(cb.Study, "KEYWORDS")
	second, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		a, b := strings.Split(string(first), "\n"), strings.Split(string(second), "\n")
		for i := range min(len(a), len(b)) {
			if a[i] != b[i] {
				t.Errorf("rebuild differs at line %d:\n first %s\nsecond %s", i+1, a[i], b[i])
			}
		}
	}
}

func TestTreeErrors(t *testing.T) {
	for _, c := range []struct {
		name  string
		mod   func(*Codebook)
		names []string
	}{
		{"empty TITLE", func(cb *Codebook) { cb.Study.Set("TITLE", " ") }, []string{"TITLE"}},
		{"var without ID", func(cb *Codebook) { cb.Vars[0].ID = "" }, []string{"IDX"}},
		{"unknown WEIGHT_VAR", func(cb *Codebook) { cb.Study.Set("WEIGHT_VAR", "NOPE") }, []string{"WEIGHT_VAR", "NOPE"}},
		{"cleansing rule on an unknown var", func(cb *Codebook) { cb.Files[1].Cleansing.Rules[0].VarName = "GHOST" }, []string{"GHOST"}},
		{"several at once", func(cb *Codebook) {
			cb.Study.Set("STRATA_VARS", "SQ1; QX")
			cb.Files[0].Rec.Set("LINK_FILE_NO", "9")
			cb.Vars[5].ParentVar = "NOPE2"
		}, []string{"QX", "LINK_FILE_NO", "NOPE2"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			cb := smallModel()
			c.mod(cb)
			_, err := cb.Tree()
			if err == nil {
				t.Fatal("no error")
			}
			for _, n := range c.names {
				if !strings.Contains(err.Error(), n) {
					t.Errorf("error does not name %s: %v", n, err)
				}
			}
			if _, err := cb.XML(); err == nil {
				t.Error("XML: no error")
			}
		})
	}
}

// Native is written whether or not datajam has filled the Package yet.
func TestTreeNative(t *testing.T) {
	cb := smallModel()
	cb.Native, cb.Package = true, nil
	eq(t, "no package", frag(at(tree(t, cb), "docDscr")),
		`<docDscr><kosac:package><kosac:sourceSurvey wallaId="ppbCL08Ex69QbEEy9ajj" kind="native"/></kosac:package></docDscr>`)

	cb = smallModel()
	cb.Native, cb.Package.Source.Kind = true, ""
	eq(t, "package without a kind", attr(at(tree(t, cb), "docDscr/kosac:package/kosac:sourceSurvey"), "kind"), "native")

	cb = smallModel()
	cb.Native, cb.Package.Source.Kind = true, "upload"
	if _, err := cb.Tree(); err == nil || !strings.Contains(err.Error(), "upload") {
		t.Errorf("native with kind upload: %v", err)
	}
}

// A bad FILE_NO is reported once with how many vars use it, not once per var.
func TestTreeErrorFileNoOnce(t *testing.T) {
	cb := smallModel()
	for _, v := range cb.Vars {
		if v.Rec.Get("FILE_NO") == "1" {
			v.Rec.Set("FILE_NO", "01")
		}
	}
	_, err := cb.Tree()
	if err == nil {
		t.Fatal("no error")
	}
	if got := strings.Count(err.Error(), "FILE_NO 01"); got != 1 || !strings.Contains(err.Error(), "변수 11개") {
		t.Errorf("FILE_NO 01 named %d times, want once with 변수 11개: %v", got, err)
	}
}

func TestValidFileNo(t *testing.T) {
	for no, want := range map[string]bool{"1": true, "12": true, "01": false, "+1": false, "0": false, " 1": false, "": false, "1.0": false} {
		if got := ValidFileNo(no); got != want {
			t.Errorf("ValidFileNo(%q) = %v, want %v", no, got, want)
		}
	}
}

// "p.336" also ends like a file extension; assembly reads pages first, as C18
// does.
func TestMaterialPagesFirst(t *testing.T) {
	cb := smallModel()
	cb.Package = nil
	cb.Study.Set("QUESTIONNAIRE", "부록 수록 p.336")
	cb.Study.Set("EXTRA_FILES", "부록 수록 p.12")
	ms := all(tree(t, cb), "otherMat")
	eq(t, "questionnaire", frag(ms[1]), `<otherMat type="questionnaire" level="study" URI="최종보고서.pdf">`+
		`<labl>부록 수록 p.336</labl><kosac:pages>336</kosac:pages></otherMat>`)
	eq(t, "extra", frag(ms[2]), `<otherMat type="extra" level="study"><labl>부록 수록 p.12</labl></otherMat>`)
}

func TestAuthEnty(t *testing.T) {
	for _, c := range []struct{ org, name, want string }{
		{"서울대학교", "임동균", `<rspStmt><AuthEnty affiliation="서울대학교">임동균</AuthEnty></rspStmt>`},
		{"", "임동균", `<rspStmt><AuthEnty affiliation="">임동균</AuthEnty></rspStmt>`},
		{"서울대학교", "", `<rspStmt><AuthEnty>서울대학교</AuthEnty></rspStmt>`},
	} {
		cb := smallModel()
		cb.Study.Set("PI_ORG", c.org)
		cb.Study.Set("PI_NAME", c.name)
		eq(t, c.org+"/"+c.name, frag(at(tree(t, cb), "stdyDscr/citation/rspStmt")), c.want)
	}
}

func TestCodebookID(t *testing.T) {
	for _, c := range []struct {
		name string
		mod  func(*Codebook)
		id   string
	}{
		{"main", func(*Codebook) {}, "CB-문화-01-01-2024-v1"},
		{"pre", func(cb *Codebook) { cb.Admin.Round.WaveType, cb.Admin.Round.WaveNo = "pre", "2" }, "CB-문화-01-01-2024-pre2-v1"},
		{"no DATA_NO", func(cb *Codebook) { cb.Study.Set("DATA_NO", "") }, ""},
		{"no wave type", func(cb *Codebook) { cb.Admin.Round.WaveType = "" }, ""},
		{"no version", func(cb *Codebook) { cb.Package.CodebookVersion = "" }, ""},
		{"no package", func(cb *Codebook) { cb.Package = nil }, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			cb := smallModel()
			c.mod(cb)
			eq(t, "ID", attr(tree(t, cb), "ID"), c.id)
		})
	}

	cb := smallModel()
	cb.Package = nil
	if d := at(tree(t, cb), "docDscr"); d != nil {
		t.Errorf("nil Package wrote %s", frag(d))
	}
	cb = smallModel()
	cb.Package.CodebookVersion = ""
	if got := names(at(tree(t, cb), "docDscr").Children); !slices.Equal(got, []string{"kosac:package"}) {
		t.Errorf("no CodebookVersion: docDscr children %v", got)
	}
}

func TestProjectionValidates(t *testing.T) {
	for name, cb := range map[string]*Codebook{"small": smallModel(), "full": fullModel()} {
		t.Run(name, func(t *testing.T) {
			root := tree(t, cb)
			checkPlacement(t, root)
			p := ddi.WriteXML(DDIProjection(root))
			if strings.Contains(string(p), "kosac:") {
				t.Errorf("projection keeps kosac:\n%s", p)
			}
			if !strings.Contains(string(ddi.WriteXML(root)), "kosac:") {
				t.Error("projection changed the tree it was given")
			}
			xsdtest.Validate(t, p)
		})
	}
}

func TestLabelPrefix(t *testing.T) {
	for _, c := range []struct {
		labels []string
		want   string
	}{
		{[]string{"과학기술분야의 업무는 …", "과학기술과 관련하여 …"}, ""},
		{[]string{"STEM 학위·자격: 대학원 학위", "STEM 학위·자격: 학사"}, "STEM 학위·자격"},
		{[]string{"경쟁력 있는 과학기술 분야 (1순위)", "경쟁력 있는 과학기술 분야 (2순위)"}, "경쟁력 있는 과학기술 분야"},
		{[]string{"학위·자격", "학위·경력"}, "학위"},
		{[]string{"과", "관"}, ""}, // share a leading UTF-8 byte, not a rune
		{nil, ""},
	} {
		if got := LabelPrefix(c.labels); got != c.want {
			t.Errorf("LabelPrefix(%q) = %q, want %q", c.labels, got, c.want)
		}
	}
}

func TestGridStem(t *testing.T) {
	for _, c := range []struct {
		in, stem, row string
		ok            bool
	}{
		{"과학기술에 대해 어떻게 생각하십니까? ▶ [가] ", "과학기술에 대해 어떻게 생각하십니까?", "가", true},
		{" 문항▶[ 행 1 ]", "문항", "행 1", true},
		{" 그냥 문항 ", "그냥 문항", "", false},
	} {
		stem, row, ok := GridStem(c.in)
		if stem != c.stem || row != c.row || ok != c.ok {
			t.Errorf("GridStem(%q) = %q, %q, %v", c.in, stem, row, ok)
		}
	}
}

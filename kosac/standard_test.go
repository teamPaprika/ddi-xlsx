package kosac

import (
	"bytes"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

func standardDocument(t *testing.T, cb *Codebook) *ddi.Node {
	t.Helper()
	x, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	root, w, err := ddi.Parse(x)
	if err != nil || len(w) > 0 {
		t.Fatalf("parse: %v %v", err, w)
	}
	xsdtest.Validate(t, x)
	Walk(root, func(n, _ *ddi.Node, path string) bool {
		if isKosac(n.Name) {
			t.Errorf("extension at %s", path)
		}
		for _, a := range n.Attrs {
			if isKosac(a.Name) || a.Name == "xmlns:kosac" {
				t.Errorf("extension at %s/@%s", path, a.Name)
			}
		}
		if n.Name == "notes" && attr(n, "type") == v4NoteType {
			t.Errorf("private JSON note at %s", path)
		}
		return true
	})
	j, err := cb.JSON()
	if err != nil {
		t.Fatal(err)
	}
	back, err := ddi.ParseJSON(j)
	if err != nil {
		t.Fatal(err)
	}
	if d := ddi.Diff(root, back); d != "" {
		t.Fatal(d)
	}
	if !bytes.Equal(x, ddi.WriteXML(back)) {
		t.Fatal("XML/JSON roundtrip differs")
	}
	return root
}
func TestStandardOnlyExport(t *testing.T) {
	for _, cb := range []*Codebook{smallModel(), fullModel()} {
		root := standardDocument(t, cb)
		vars := kids(child(root, "dataDscr"), "var")
		if len(vars) != len(cb.Vars) {
			t.Fatalf("%d variables, want %d source variables", len(vars), len(cb.Vars))
		}
		for _, n := range vars {
			if strings.HasPrefix(attr(n, "name"), "walla_") {
				t.Fatal("fabricated Walla column")
			}
		}
		if at(root, "stdyDscr/method/dataColl/dataCollector").Text != cb.Study.Get("FIELD_ORG") {
			t.Fatal("collector mapping lost")
		}
	}
}
func TestStandardV4NativeFields(t *testing.T) {
	cb := &Codebook{Template: TemplateMeta{Version: Field{Value: "4.0"}}, Study: rec("TITLE", "매핑 검증", "VALID_N", "1 (중복 없음)"),
		Files: []*File{{Rec: rec("FILE_NO", "1", "FILE_NAME", "input.csv", "FILE_FORMAT", "CSV", "VALID_N", "1", "VARIABLE_N", "4")}}}
	levels := []string{"명목", "서열", "등간", "비율"}
	nature := []string{"nominal", "ordinal", "interval", "ratio"}
	for i, level := range levels {
		name := []string{"gender", "rank", "score", "age"}[i]
		cb.Vars = append(cb.Vars, &Var{ID: []string{"V1", "V2", "V3", "V4"}[i], Rec: rec("FILE_NO", "1", "VAR_NAME", name, "VAR_LABEL", name, "MEAS_LEVEL", level, "DATA_TYPE", "숫자", "DDI_INTERVAL", "discrete", "ACCESS_LEVEL", "내부 이용", "PII", "N", "SENSITIVE", "N", "RECODE_STATUS", "확인 필요")})
	}
	cb.Codes = []*Code{{Rec: rec("FILE_NO", "1", "VAR_NAME", "gender", "CODE", "01", "CODE_LABEL", "남자", "IS_MISSING", "아니오")}, {Rec: rec("FILE_NO", "1", "VAR_NAME", "gender", "CODE", "-8", "CODE_LABEL", "해당 없음", "IS_MISSING", "예", "MISSING_REASON", "해당 없음")}}
	cb.Recodes = []Record{rec("FILE_NO", "1", "SOURCE_FILE", "original.csv", "SOURCE_VAR", "sex", "TARGET_VAR", "gender", "TRANSFORM_TYPE", "문자 코드화", "SOURCE_CODE_OR_CONDITION", "남자", "TARGET_CODE_OR_RULE", "01", "BASIS", "원문 설문지", "STATUS", "확인 필요", "RECORD_COUNT", "1")}
	root := standardDocument(t, cb)
	vars := kids(child(root, "dataDscr"), "var")
	for i, n := range vars {
		if attr(n, "nature") != nature[i] || attr(n, "intrvl") != "discrete" {
			t.Fatalf("wrong measurement %v", n.Attrs)
		}
	}
	cats := kids(vars[0], "catgry")
	if textOf(cats[0], "catValu") != "01" || attr(cats[1], "missing") != "Y" || attr(cats[1], "missType") != "해당 없음" {
		t.Fatal("code or missing reason changed")
	}
	method := at(root, "stdyDscr/method")
	instructions := kids(method, "codingInstructions")
	if len(instructions) != 1 || attr(instructions[0], "relatedProcesses") == "" {
		t.Fatal("recode process not connected")
	}
	if len(kids(instructions[0], "command")) != 0 {
		t.Fatal("invented executable command")
	}
	if !strings.Contains(textOf(instructions[0], "txt"), "original.csv") {
		t.Fatal("source history lost")
	}
	if cb.Study.Get("VALID_N") != "1 (중복 없음)" || cb.Codes[0].Rec.Get("CODE") != "01" {
		t.Fatal("source model changed")
	}
	cb.Codes[0].Rec.Set("IS_MISSING", "확인 불가")
	if _, err := cb.XML(); err == nil {
		t.Fatal("unconfirmed missingness became the DDI default N")
	}
}

func TestStandardDoesNotAssertMalformedCountOrSelfHash(t *testing.T) {
	cb := smallModel()
	cb.Study.Set("TARGET_N", "10,00")
	cb.Package.Files = append(cb.Package.Files, PackageFile{Role: "codebook", Name: "self.xml", Bytes: "9999", SHA256: "circular-self-hash"})
	root := standardDocument(t, cb)
	coll := at(root, "stdyDscr/method/dataColl")
	if child(coll, "targetSampleSize") != nil {
		t.Fatal("malformed count asserted as sampleSize")
	}
	x, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(x, []byte("10,00")) {
		t.Fatal("original malformed count lost")
	}
	if bytes.Contains(x, []byte("circular-self-hash")) {
		t.Fatal("self hash written into hashed document")
	}
	if cb.Package.Files[len(cb.Package.Files)-1].SHA256 != "circular-self-hash" {
		t.Fatal("source model changed")
	}
}

func TestLegacyNativeUpgradesToStandard(t *testing.T) {
	cb := fullModel()
	cb.Native, cb.Package.Source.Kind = true, "native"
	old, err := cb.profileTree()
	if err != nil {
		t.Fatal(err)
	}
	imported, err := Parse(ddi.WriteXML(old))
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.SourceSystemVars) == 0 {
		t.Fatal("fixture has no imported systems")
	}
	standardDocument(t, imported)
}
func TestImportedSystemLinksFollowRenumbering(t *testing.T) {
	cb := smallModel()
	cb.SourceSystemVars = []*ddi.Node{
		add(el("var", "ID", "V1", "name", "source_a", "files", "F1"), add(el("qstn", "ID", "SOURCE_Q"), text("qstnLit", "기록 출처"))),
		el("var", "ID", "V2", "name", "source_b", "files", "F1", "qstn", "SOURCE_Q", "wgt-var", "V1"),
	}
	root := standardDocument(t, cb)
	vars := kids(child(root, "dataDscr"), "var")
	if attr(vars[1], "qstn") != attr(child(vars[0], "qstn"), "ID") || attr(vars[1], "wgt-var") != attr(vars[0], "ID") {
		t.Fatal("source links broken")
	}
	if attr(cb.SourceSystemVars[0], "ID") != "V1" {
		t.Fatal("source ID modified")
	}
}

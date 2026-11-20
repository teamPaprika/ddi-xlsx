package template

import (
	"reflect"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
)

func TestProfile423Workbook(t *testing.T) {
	raw := Blank()
	raw[0] ^= 255
	cb, issues := read(t, Blank(), ReadOptions{})
	if cb == nil || len(issues) != 0 || cb.Template.Version.Value != kosac.Profile423 || cb.Study.Get("RULE_VERSION") != "20261105-r3" {
		t.Fatalf("blank: %v", issues)
	}
	cb.Study.Set("STUDY_ID", "D0001-2024")
	cb.Vars = []*kosac.Var{{Rec: kosac.Record{"VAR_NAME": {Value: "Q1"}, "ITEM_KEY": {Value: "D0001.Q1"}}}}
	cb.Codes = []*kosac.Code{{Rec: kosac.Record{"CODE": {Value: "01"}}}}
	cb.Supplemental = map[string][][]string{"10_미확인사유": {{"항목", "사유"}, {"문항", "미확인"}}}
	x, err := Write(cb)
	if err != nil {
		t.Fatal(err)
	}
	again, issues := read(t, x, ReadOptions{})
	if again == nil || len(issues) != 0 || len(again.Vars) != 1 || len(again.Codes) != 1 || again.Vars[0].Rec.Get("ITEM_KEY") != "D0001.Q1" || again.Codes[0].Rec.Get("CODE") != "01" || again.Study.Get("STUDY_ID") != "D0001-2024" || !reflect.DeepEqual(again.Supplemental, cb.Supplemental) {
		t.Fatalf("roundtrip: %v", issues)
	}
	cb.Template.Version.Value = "4.0"
	if _, err := Write(cb); err == nil {
		t.Fatal("lossy downgrade")
	}
	rejected, issues := read(t, x, ReadOptions{AllowedVersions: []string{"4.0"}})
	if rejected != nil || len(issues) == 0 {
		t.Fatal("version filter")
	}
}

// The 2024 sample identifies multiple-response columns by numbered names and
// records the split condition in RESP_UNIVERSE. Q_NO and UNIVERSE_EXPR are blank.
func TestProfile423OptionalQuestionNumber(t *testing.T) {
	cb, _ := read(t, Blank(), ReadOptions{})
	cb.Study.Set("TITLE", "합성 조사")
	cb.Study.Set("SPLIT_BALLOT", "Y")
	cb.Study.Set("SPLIT_VAR", "Ballot")
	cb.Files = []*kosac.File{{Rec: kosac.Record{"FILE_NO": {Value: "1"}, "UNIQUE_KEY": {Value: "ID"}}}}
	for _, name := range []string{"ID", "Ballot", "Q1_1", "Q1_2"} {
		r := kosac.Record{"FILE_NO": {Value: "1"}, "VAR_NAME": {Value: name}}
		if strings.HasPrefix(name, "Q1_") {
			r.Set("MR_TYPE", "순위별 열")
			r.Set("RESP_UNIVERSE", "Ballot=1")
		}
		cb.Vars = append(cb.Vars, &kosac.Var{Rec: r})
	}
	if err := kosac.AssignIDs(cb, nil); err != nil {
		t.Fatal(err)
	}
	root, err := cb.Tree()
	if err != nil {
		t.Fatal(err)
	}
	members := map[string]string{}
	kosac.Walk(root, func(n, _ *ddi.Node, _ string) bool {
		if n.Name == "varGrp" {
			kind, _ := n.Attr("type")
			members[kind], _ = n.Attr("var")
		}
		return true
	})
	for _, kind := range []string{"multipleResp", "randomized"} {
		if members[kind] != "V3 V4" {
			t.Fatalf("%s members %q", kind, members[kind])
		}
	}
}

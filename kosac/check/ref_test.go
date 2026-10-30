package check

import (
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
)

// noXSD leaves only the Go checks.
func noXSD(Schema, []byte) ([]string, error) { return nil, nil }

func TestRef(t *testing.T) {
	for _, c := range []struct {
		name, path, actual string
		edit               func(root *ddi.Node)
	}{
		{"varGrp var", "dataDscr/varGrp[@ID='G1']/@var", "V99", func(r *ddi.Node) { setAttr(at(r, "dataDscr/varGrp"), "var", "V1 V99") }},
		{"var files", "dataDscr/var[@ID='V1']/@files", "F9", func(r *ddi.Node) { setAttr(varOf(r, "V1"), "files", "F9") }},
		{"idVar ref", "fileDscr[@ID='F1']/kosac:file/kosac:idVar/@ref", "V99", func(r *ddi.Node) {
			setAttr(at(r, "fileDscr/kosac:file/kosac:idVar"), "ref", "V99")
		}},
		{"rule var", "fileDscr[@ID='F1']/kosac:cleansing/kosac:rule/@var", "V99", func(r *ddi.Node) {
			setAttr(at(r, "fileDscr/kosac:cleansing/kosac:rule"), "var", "V99")
		}},
		{"unverified file", "stdyDscr/kosac:study/kosac:unverified/@file", "F9", func(r *ddi.Node) {
			setAttr(at(r, "stdyDscr/kosac:study/kosac:unverified"), "file", "F9")
		}},
		{"duplicate var ID", "dataDscr/var[@ID='V1']/@ID", "V1", func(r *ddi.Node) {
			d := at(r, "dataDscr")
			d.Children = append(d.Children, varOf(r, "V1"))
		}},
		{"codeBook ID", "codeBook/@ID", "1abc", func(r *ddi.Node) { setAttr(r, "ID", "1abc") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := engineTree(t)
			if got := ValidateXML(ddi.WriteXML(root), noXSD); len(got) > 0 {
				t.Fatalf("unedited:\n%s", issueSums(got))
			}
			c.edit(root)
			got := ValidateXML(ddi.WriteXML(root), noXSD)
			if len(got) != 1 || got[0].Rule != "REF" || got[0].Level != kosac.R || got[0].Path != c.path ||
				got[0].Actual != c.actual || got[0].Message == "" {
				t.Errorf("want one R REF at %s for %s, got:\n%s", c.path, c.actual, issueSums(got))
			}
		})
	}
}

// Assembly keeps a survey var ID it is given, so one that is also a system
// var's S{n} reaches the XML; only validation sees the clash.
func TestRefSystemVarClash(t *testing.T) {
	cb := engineModel()
	cb.Vars[1].ID = "S1"
	b, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	got := ValidateXML(b, noXSD)
	if len(got) != 1 || got[0].Rule != "REF" || got[0].Actual != "S1" {
		t.Errorf("got:\n%s", issueSums(got))
	}
}

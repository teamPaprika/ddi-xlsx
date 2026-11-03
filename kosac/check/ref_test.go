package check

import (
	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"testing"
)

func noXSD(Schema, []byte) ([]string, error) { return nil, nil }
func TestRef(t *testing.T) {
	for _, c := range []struct {
		name, path, actual string
		edit               func(*ddi.Node)
	}{
		{"group", "dataDscr/varGrp[@ID='G1']/@var", "V99", func(r *ddi.Node) { setAttr(at(r, "dataDscr/varGrp"), "var", "V1 V99") }},
		{"file", "dataDscr/var[@ID='V1']/@files", "F9", func(r *ddi.Node) { setAttr(varOf(r, "V1"), "files", "F9") }},
		{"derivation", "dataDscr/var[@ID='V1']/derivation/@var", "V99", func(r *ddi.Node) {
			n := varOf(r, "V1")
			n.Children = append(n.Children, &ddi.Node{Name: "derivation", Attrs: []ddi.Attr{{Name: "var", Value: "V99"}}})
		}},
		{"process", "stdyDscr/method/codingInstructions[@ID='CI1']/@relatedProcesses", "P99", func(r *ddi.Node) { setAttr(at(r, "stdyDscr/method/codingInstructions"), "relatedProcesses", "P99") }},
		{"link", "dataDscr/var[@ID='V1']/notes/Link/@refs", "MISSING", func(r *ddi.Node) {
			n := varOf(r, "V1")
			n.Children = append(n.Children, &ddi.Node{Name: "notes", Children: []*ddi.Node{{Name: "Link", Attrs: []ddi.Attr{{Name: "refs", Value: "MISSING"}}, Text: "source"}}})
		}},
		{"duplicate", "dataDscr/var[@ID='V1']/@ID", "V1", func(r *ddi.Node) { d := at(r, "dataDscr"); d.Children = append(d.Children, varOf(r, "V1")) }},
		{"root ID", "codeBook/@ID", "1abc", func(r *ddi.Node) { setAttr(r, "ID", "1abc") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := engineTree(t)
			if got := ValidateXML(ddi.WriteXML(root), noXSD); len(got) > 0 {
				t.Fatal(issueSums(got))
			}
			c.edit(root)
			got := ValidateXML(ddi.WriteXML(root), noXSD)
			if len(got) != 1 || got[0].Rule != "REF" || got[0].Level != kosac.R || got[0].Path != c.path || got[0].Actual != c.actual {
				t.Fatalf("want %s %s: %s", c.path, c.actual, issueSums(got))
			}
		})
	}
}
func TestNoInventedSystemVarClash(t *testing.T) {
	cb := engineModel()
	cb.Vars[1].ID = "S1"
	cb.SourceSystemVars = []*ddi.Node{{Name: "var", Attrs: []ddi.Attr{{Name: "ID", Value: "S1"}, {Name: "name", Value: "source_id"}, {Name: "files", Value: "F1"}}, Children: []*ddi.Node{{Name: "varFormat", Attrs: []ddi.Attr{{Name: "type", Value: "character"}}}}}}
	x, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	if got := ValidateXML(x, noXSD); len(got) > 0 {
		t.Fatal(issueSums(got))
	}
}

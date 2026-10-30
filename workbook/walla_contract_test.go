package workbook

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

func TestWallaContractRoundTrip(t *testing.T) {
	xml := readFile(t, "walla_contract_v1", "survey.xml")
	json := readFile(t, "walla_contract_v1", "survey.json")
	if d := ddi.Diff(parse(t, xml), parse(t, json)); d != "" {
		t.Fatal(d)
	}
	for _, input := range []struct {
		name string
		data []byte
	}{{"XML", xml}, {"JSON", json}} {
		t.Run(input.name, func(t *testing.T) {
			xlsx, warnings, err := ToXLSX(input.data)
			if err != nil || len(warnings) != 0 {
				t.Fatalf("ToXLSX: warnings=%q err=%v", warnings, err)
			}
			xo, jo, warnings, err := FromXLSX(xlsx)
			if err != nil || len(warnings) != 0 {
				t.Fatalf("FromXLSX: warnings=%q err=%v", warnings, err)
			}
			for _, out := range [][]byte{xo, jo} {
				if d := ddi.Diff(parse(t, input.data), parse(t, out)); d != "" {
					t.Error(d)
				}
			}
			xsdtest.Validate(t, xo)
		})
	}
}

func TestWallaContractAdditionalMetadata(t *testing.T) {
	root := parse(t, readFile(t, "walla_contract_v1", "survey.xml"))
	root.Attrs = append(root.Attrs, ddi.Attr{Name: "xmlns:consumer", Value: "urn:ddi-xlsx:test"})
	v := findVar(t, root, "V9")
	for _, child := range v.Children {
		switch child.Name {
		case "catgry":
			for _, c := range child.Children {
				if c.Name == "catValu" && (c.Text == "1" || c.Text == "2") {
					c.Text = "0" + c.Text
				}
			}
			if missing, _ := child.Attr("missing"); missing == "Y" {
				child.Attrs = append(child.Attrs, ddi.Attr{Name: "missType", Value: "legacy"})
			}
		case "varFormat":
			child.Attrs = []ddi.Attr{{Name: "type", Value: "character"}}
		}
	}
	for _, c := range root.Children {
		if c.Name == "stdyDscr" {
			c.Children = append(c.Children, &ddi.Node{Name: "consumer:metadata", Text: "001",
				Attrs: []ddi.Attr{{Name: "consumer:source", Value: "synthetic"}}})
		}
	}
	in := bytes.Replace(ddi.WriteXML(root), []byte("<codeBook"), []byte("<!-- synthetic comment --><codeBook"), 1)
	xlsx, warnings, err := ToXLSX(in)
	if err != nil || len(warnings) != 1 || !strings.Contains(warnings[0], "comment dropped") {
		t.Fatalf("ToXLSX: warnings=%q err=%v", warnings, err)
	}
	xo, jo, warnings, err := FromXLSX(xlsx)
	if err != nil || !slices.ContainsFunc(warnings, func(s string) bool {
		return strings.Contains(s, "<consumer:metadata> is not allowed in <stdyDscr>")
	}) {
		t.Fatalf("FromXLSX: warnings=%q err=%v", warnings, err)
	}
	for _, out := range [][]byte{xo, jo} {
		if d := ddi.Diff(root, parse(t, out)); d != "" {
			t.Error(d)
		}
		if bytes.Contains(out, []byte("synthetic comment")) {
			t.Error("dropped comment survived")
		}
	}
}

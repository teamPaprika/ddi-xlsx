package kosac

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

func kosacXSD() string { return filepath.Join(xsdtest.Root(), "kosac", "xsd", "kosac-3.3.xsd") }

func varOf(root *ddi.Node, id string) *ddi.Node {
	return withAttr(all(at(root, "dataDscr"), "var"), "ID", id)
}

func setAttr(n *ddi.Node, name, value string) {
	for i := range n.Attrs {
		if n.Attrs[i].Name == name {
			n.Attrs[i].Value = value
			return
		}
	}
	n.Attrs = append(n.Attrs, ddi.Attr{Name: name, Value: value})
}

func TestKosacProjection(t *testing.T) {
	root := tree(t, smallModel())
	before := string(ddi.WriteXML(root))
	p := KosacProjection(root)

	eq(t, "root", p.Name+" "+ddi.FormatAttrs(p.Attrs), `kosac:projection xmlns:kosac="urn:kosac:survey-meta:3.3"`)
	place := withAttr(p.Children, "path", "dataDscr/var[@ID='V2']")
	if place == nil {
		t.Fatalf("no place for V2 in %v", p.Children)
	}
	eq(t, "V2 place attrs", ddi.FormatAttrs(place.Attrs),
		`path="dataDscr/var[@ID='V2']" kosac:seq="8" kosac:role="ITEM" kosac:measLevel="nominal" kosac:pii="N" kosac:sensitive="N"`)
	if got := names(place.Children); !slices.Equal(got, []string{"kosac:missingCodes", "kosac:codeSetId", "kosac:analysis"}) {
		t.Errorf("V2 place children %v", got)
	}
	var paths []string
	for _, c := range p.Children {
		paths = append(paths, attr(c, "path"))
	}
	for _, want := range []string{"docDscr", "stdyDscr", "stdyDscr/method/dataColl", "fileDscr[@ID='F2']",
		"dataDscr/varGrp[@ID='G5']", "dataDscr/var[@ID='V2']/catgry", "otherMat"} {
		if !slices.Contains(paths, want) {
			t.Errorf("no place %q in %v", want, paths)
		}
	}
	if slices.ContainsFunc(p.Children, func(c *ddi.Node) bool { return attr(c, "path") == "dataDscr/var[@ID='V1']/location" }) {
		t.Error("a place for an element with no kosac content")
	}
	if got := string(ddi.WriteXML(root)); got != before {
		t.Error("projection changed the tree it was given")
	}

	setAttr(root, "xmlns:kosac", "urn:other")
	eq(t, "declared as the document does", ddi.FormatAttrs(KosacProjection(root).Attrs), `xmlns:kosac="urn:other"`)
}

func TestKosacXSDCompiles(t *testing.T) {
	empty := ddi.WriteXML(KosacProjection(&ddi.Node{Name: "codeBook", Attrs: []ddi.Attr{{Name: "xmlns:kosac", Value: Namespace}}}))
	if p := xsdtest.ValidateSchema(t, kosacXSD(), empty); p != nil {
		t.Errorf("empty projection:\n%s", strings.Join(p, "\n"))
	}
}

func TestKosacXSDAcceptsEngineOutput(t *testing.T) {
	for name, cb := range map[string]*Codebook{"small": smallModel(), "full": fullModel(), "unverified": unverifiedModel()} {
		t.Run(name, func(t *testing.T) {
			if p := xsdtest.ValidateSchema(t, kosacXSD(), ddi.WriteXML(KosacProjection(tree(t, cb)))); p != nil {
				t.Errorf("%s", strings.Join(p, "\n"))
			}
		})
	}
}

func TestKosacXSDRejects(t *testing.T) {
	for _, c := range []struct {
		name, want string // want is a word of the expected problem
		edit       func(root *ddi.Node)
	}{
		{"role", "BOSS", func(r *ddi.Node) { setAttr(varOf(r, "V2"), "kosac:role", "BOSS") }},
		{"fk mapping", "'X'", func(r *ddi.Node) { setAttr(at(varOf(r, "V2"), "kosac:fk"), "mapping", "X") }},
		{"file sha256", "xyz", func(r *ddi.Node) { setAttr(at(r, "fileDscr/kosac:file"), "sha256", "xyz") }},
		{"excludeN text", "많음", func(r *ddi.Node) { at(r, "stdyDscr/method/dataColl/kosac:excludeN").Text = "많음" }},
		{"undeclared attribute", "foo", func(r *ddi.Node) { setAttr(varOf(r, "V2"), "kosac:foo", "1") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := tree(t, fullModel())
			c.edit(root)
			p := xsdtest.ValidateSchema(t, kosacXSD(), ddi.WriteXML(KosacProjection(root)))
			if !slices.ContainsFunc(p, func(s string) bool { return strings.Contains(s, c.want) }) {
				t.Errorf("no problem naming %s:\n%s", c.want, strings.Join(p, "\n"))
			}
		})
	}
}

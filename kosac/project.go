package kosac

import (
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

func isKosac(name string) bool { return strings.HasPrefix(name, "kosac:") }

// DDIProjection returns a copy of root without kosac elements, kosac
// attributes and the xmlns:kosac declaration, which the official DDI 2.5 XSD
// accepts in no position. Raw content is copied as is: assembly never writes
// it.
func DDIProjection(root *ddi.Node) *ddi.Node {
	n := &ddi.Node{Name: root.Name, Text: root.Text, Raw: root.Raw}
	for _, a := range root.Attrs {
		if !isKosac(a.Name) && a.Name != "xmlns:kosac" {
			n.Attrs = append(n.Attrs, a)
		}
	}
	for _, c := range root.Children {
		if !isKosac(c.Name) {
			n.Children = append(n.Children, DDIProjection(c))
		}
	}
	return n
}

// KosacProjection returns <kosac:projection> with one <kosac:place> per DDI
// element that holds kosac content: its kosac children as they are (shared
// with root, not copied) and its kosac attributes moved onto the place, whose
// @path names the element as Issue.Path does. The DDI 2.5 XSD has no slot for
// these nodes, so they are validated against the kosac XSD apart from their
// DDI context, which the placement check covers. The prefix is declared as the
// document declares it, so a document that binds kosac to another URI fails
// the kosac schema.
func KosacProjection(root *ddi.Node) *ddi.Node {
	p := &ddi.Node{Name: "kosac:projection"}
	if v, ok := root.Attr("xmlns:kosac"); ok {
		p.Attrs = []ddi.Attr{{Name: "xmlns:kosac", Value: v}}
	}
	Walk(root, func(n, _ *ddi.Node, path string) bool {
		if isKosac(n.Name) {
			return false // its parent's place holds it whole
		}
		place := &ddi.Node{Name: "kosac:place", Attrs: []ddi.Attr{{Name: "path", Value: path}}}
		for _, a := range n.Attrs {
			if isKosac(a.Name) {
				place.Attrs = append(place.Attrs, a)
			}
		}
		for _, c := range n.Children {
			if isKosac(c.Name) {
				place.Children = append(place.Children, c)
			}
		}
		if len(place.Attrs) > 1 || len(place.Children) > 0 {
			p.Children = append(p.Children, place)
		}
		return true
	})
	return p
}

// Walk calls fn on root and every element below it in document order, with
// its parent (nil for root) and its path as Issue.Path writes it: the root is
// "codeBook", any other element is named from below the root with the ID of
// each element that has one, as dataDscr/var[@ID='V12']. When fn returns
// false, Walk skips the element's children.
func Walk(root *ddi.Node, fn func(n, parent *ddi.Node, path string) bool) {
	if fn(root, nil, "codeBook") {
		walkBelow(root, "", fn)
	}
}

func walkBelow(n *ddi.Node, path string, fn func(n, parent *ddi.Node, path string) bool) {
	for _, c := range n.Children {
		cp := c.Name
		if id, ok := c.Attr("ID"); ok {
			cp += "[@ID='" + id + "']"
		}
		if path != "" {
			cp = path + "/" + cp
		}
		if fn(c, n, cp) {
			walkBelow(c, cp, fn)
		}
	}
}

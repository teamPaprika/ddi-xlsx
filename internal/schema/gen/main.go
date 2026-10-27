// Command gen reads the DDI Codebook XSD and writes the schema package's lookup tables.
//
// Usage: gen <codebook.xsd> <out.go>
package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"go/format"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
)

const xsdNS = "http://www.w3.org/2001/XMLSchema"

type xnode struct {
	XMLName xml.Name
	Attr    []xml.Attr `xml:",any,attr"`
	Kids    []*xnode   `xml:",any"`
}

func (n *xnode) attr(name string) string {
	for _, a := range n.Attr {
		if a.Name.Space == "" && a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// dropAnnotations removes xs:annotation subtrees, whose xhtml documentation is not content model.
func dropAnnotations(n *xnode) {
	kept := n.Kids[:0]
	for _, k := range n.Kids {
		if k.XMLName.Space == xsdNS && k.XMLName.Local == "annotation" {
			continue
		}
		dropAnnotations(k)
		kept = append(kept, k)
	}
	n.Kids = kept
}

type decl struct {
	name string
	typ  *xnode // nil when the type is simple or outside this schema
}

type gen struct {
	types, groups, attrGroups map[string]*xnode
	queue                     []decl
	seen                      map[decl]bool
	repeatable                map[string]bool
	children                  map[string]map[string]int
	anyChild                  map[string]bool
	enums                     map[string]map[string][]string
}

func repeats(n *xnode) bool {
	m := n.attr("maxOccurs")
	if m == "unbounded" {
		return true
	}
	v, err := strconv.Atoi(m)
	return err == nil && v > 1
}

// declare records an element declaration so its own content model is indexed under its name.
func (g *gen) declare(el *xnode) {
	name := el.attr("name")
	if _, ok := g.repeatable[name]; !ok {
		g.repeatable[name] = false
	}
	d := decl{name: name, typ: g.types[el.attr("type")]}
	for _, k := range el.Kids {
		if k.XMLName.Local == "complexType" {
			d.typ = k
		}
	}
	if d.typ != nil && !g.seen[d] {
		g.seen[d] = true
		g.queue = append(g.queue, d)
	}
}

// walk numbers the element particles under n into parent's content model.
// pos is the next free position; step is 0 inside a choice, whose members all
// share the choice's position. rep is true when an enclosing particle repeats.
// It returns the next free position.
func (g *gen) walk(n *xnode, parent string, pos, step int, rep bool) int {
	rep = rep || repeats(n)
	switch n.XMLName.Local {
	case "complexType", "complexContent", "restriction", "sequence":
		for _, k := range n.Kids {
			pos = g.walk(k, parent, pos, step, rep)
		}
	case "extension":
		if base := g.types[n.attr("base")]; base != nil {
			pos = g.walk(base, parent, pos, step, rep)
		}
		for _, k := range n.Kids {
			pos = g.walk(k, parent, pos, step, rep)
		}
	case "choice":
		for _, k := range n.Kids {
			g.walk(k, parent, pos, 0, rep)
		}
		pos += step
	case "group":
		ref := n.attr("ref")
		// The XSD's default namespace is the DDI target namespace, so a prefixed
		// ref names a foreign (xhtml, dc) group whose content model is not indexed.
		if strings.Contains(ref, ":") {
			g.anyChild[parent] = true
			break
		}
		for _, k := range g.groups[ref].Kids {
			pos = g.walk(k, parent, pos, step, rep)
		}
	case "any":
		g.anyChild[parent] = true
	case "element":
		name := n.attr("ref")
		if name == "" {
			name = n.attr("name")
			g.declare(n)
		}
		if g.children[parent] == nil {
			g.children[parent] = map[string]int{}
		}
		if _, ok := g.children[parent][name]; !ok {
			g.children[parent][name] = pos
		}
		g.repeatable[name] = g.repeatable[name] || rep
		pos += step
	}
	return pos
}

// attrs records, for element el, the enumerated values of each attribute that
// n declares or inherits. A derived type's own declarations come after its
// base's, so a redeclared or prohibited attribute replaces the inherited one.
func (g *gen) attrs(n *xnode, el string) {
	switch n.XMLName.Local {
	case "complexType", "complexContent", "simpleContent":
		for _, k := range n.Kids {
			g.attrs(k, el)
		}
	case "extension", "restriction":
		if base := g.types[n.attr("base")]; base != nil {
			g.attrs(base, el)
		}
		for _, k := range n.Kids {
			g.attrs(k, el)
		}
	case "attributeGroup":
		if grp := g.attrGroups[n.attr("ref")]; grp != nil {
			n = grp
		}
		for _, k := range n.Kids {
			g.attrs(k, el)
		}
	case "attribute":
		// A ref, such as xml:lang, names an attribute declared elsewhere.
		name := n.attr("name")
		if name == "" {
			break
		}
		var vals []string
		for _, st := range n.Kids {
			if st.XMLName.Local != "simpleType" || n.attr("use") == "prohibited" {
				continue
			}
			for _, r := range st.Kids {
				for _, e := range r.Kids {
					if r.XMLName.Local == "restriction" && e.XMLName.Local == "enumeration" {
						vals = append(vals, e.attr("value"))
					}
				}
			}
		}
		if vals == nil {
			delete(g.enums[el], name)
			break
		}
		if g.enums[el] == nil {
			g.enums[el] = map[string][]string{}
		}
		g.enums[el][name] = vals
	}
}

func run(in, out string) error {
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	var root xnode
	if err := xml.Unmarshal(data, &root); err != nil {
		return err
	}
	dropAnnotations(&root)

	g := &gen{
		types:      map[string]*xnode{},
		groups:     map[string]*xnode{},
		attrGroups: map[string]*xnode{},
		seen:       map[decl]bool{},
		repeatable: map[string]bool{},
		children:   map[string]map[string]int{},
		anyChild:   map[string]bool{},
		enums:      map[string]map[string][]string{},
	}
	for _, k := range root.Kids {
		switch k.XMLName.Local {
		case "complexType":
			g.types[k.attr("name")] = k
		case "group":
			g.groups[k.attr("name")] = k
		case "attributeGroup":
			g.attrGroups[k.attr("name")] = k
		}
	}
	for _, k := range root.Kids {
		if k.XMLName.Local == "element" {
			g.declare(k)
		}
	}
	// Local declarations found while walking join the queue, so it grows as it is drained.
	for i := 0; i < len(g.queue); i++ {
		d := g.queue[i]
		g.walk(d.typ, d.name, 0, 1, false)
		g.attrs(d.typ, d.name)
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by \"go run ./gen %s %s\"; DO NOT EDIT.\n\npackage schema\n\n", in, out)
	b.WriteString("var repeatable = map[string]bool{\n")
	for _, k := range slices.Sorted(maps.Keys(g.repeatable)) {
		fmt.Fprintf(&b, "%q: %t,\n", k, g.repeatable[k])
	}
	b.WriteString("}\n\nvar children = map[string]map[string]int{\n")
	for _, p := range slices.Sorted(maps.Keys(g.children)) {
		fmt.Fprintf(&b, "%q: {", p)
		for _, c := range slices.Sorted(maps.Keys(g.children[p])) {
			fmt.Fprintf(&b, "%q: %d, ", c, g.children[p][c])
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n\nvar anyChild = map[string]bool{\n")
	for _, k := range slices.Sorted(maps.Keys(g.anyChild)) {
		fmt.Fprintf(&b, "%q: true,\n", k)
	}
	b.WriteString("}\n\nvar attrEnums = map[string]map[string][]string{\n")
	for _, el := range slices.Sorted(maps.Keys(g.enums)) {
		if len(g.enums[el]) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%q: {", el)
		for _, a := range slices.Sorted(maps.Keys(g.enums[el])) {
			fmt.Fprintf(&b, "%q: {", a)
			for _, v := range g.enums[el][a] {
				fmt.Fprintf(&b, "%q, ", v)
			}
			b.WriteString("}, ")
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n")

	src, err := format.Source(b.Bytes())
	if err != nil {
		return err
	}
	return os.WriteFile(out, src, 0o644)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: gen <codebook.xsd> <out.go>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

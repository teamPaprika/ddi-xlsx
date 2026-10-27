package workbook

import (
	"cmp"
	"slices"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/schema"
)

// sortChildren puts the children from each source together in the order of
// parent's content model. The sort is stable, so children that share a position
// keep the order they were gathered in. A child the model does not name takes
// the position of the child before it in its own source, or -1 at the head of
// it. Inheriting across sources would tie it to a column or row neighbour that
// the next conversion no longer puts before it, so the order would not be a
// fixed point.
func sortChildren(parent string, sources ...[]*ddi.Node) []*ddi.Node {
	type keyed struct {
		pos int
		n   *ddi.Node
	}
	var ks []keyed
	for _, src := range sources {
		pos := -1
		for _, c := range src {
			if p, ok := schema.Position(parent, c.Name); ok {
				pos = p
			}
			ks = append(ks, keyed{pos, c})
		}
	}
	slices.SortStableFunc(ks, func(a, b keyed) int { return cmp.Compare(a.pos, b.pos) })
	out := make([]*ddi.Node, len(ks))
	for i, k := range ks {
		out[i] = k.n
	}
	return out
}

// unraw turns n's Raw back into children when it holds only elements. It is
// for var, catgry, varGrp and dataDscr only: their content is element-only and
// the reader re-sorts their children into schema order, so the sibling order or
// the spacing that kept n as written carries nothing to keep. Text between the
// children is content, so then n stays as written. ns holds the xmlns:*
// declarations above n.
func unraw(n *ddi.Node, ns []ddi.Attr) []string {
	if n.Raw == "" {
		return nil
	}
	kids, warns, err := ddi.ParseFragmentWithWarnings(n.Raw, append(slices.Clip(ns), n.Attrs...))
	if err != nil {
		// Raw has been checked to be well-formed already, so this is the one
		// way it can fail as a fragment.
		return nil
	}
	n.Children, n.Raw = kids, ""
	// Undeclared prefixes are left to the parse that kept n as written, which
	// read the same names. A comment it kept, though, is dropped only now.
	return slices.DeleteFunc(warns, func(w string) bool { return strings.HasPrefix(w, "prefix ") })
}

// resorted reports whether the reader re-sorts the children of an element
// named child in one named parent, as it puts the element together from its
// row and the rows and cells that add to it.
func resorted(parent, child string) bool {
	return parent == "dataDscr" && (child == "var" || child == "varGrp") || parent == "var" && child == "catgry"
}

// sortNested sorts the children of each element below n that resorted names.
// Typed in n's cell, such an element skips that sort; the next conversion
// gives it a row, which would sort it then, so the output would not be a
// fixed point. Sorting a sorted list changes nothing.
func sortNested(n *ddi.Node) {
	for _, c := range n.Children {
		if resorted(n.Name, c.Name) {
			c.Children = sortChildren(c.Name, c.Children)
			sortNested(c)
		}
	}
}

// dedupe drops repeated warnings, keeping the first of each.
func dedupe(warns []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range warns {
		if !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

// group moves every child next to the first earlier sibling of its name,
// throughout the tree. JSON holds one key per name, so this is the only order
// both outputs can carry.
func group(n *ddi.Node) {
	var order []string
	byName := map[string][]*ddi.Node{}
	for _, c := range n.Children {
		group(c)
		if _, ok := byName[c.Name]; !ok {
			order = append(order, c.Name)
		}
		byName[c.Name] = append(byName[c.Name], c)
	}
	n.Children = n.Children[:0]
	for _, name := range order {
		n.Children = append(n.Children, byName[name]...)
	}
}

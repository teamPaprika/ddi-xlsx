package ddi

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Equal reports whether a and b have the same names, the same attributes in
// any order, the same Text and Raw, and equal children in the same order.
func Equal(a, b *Node) bool { return Diff(a, b) == "" }

// Diff returns the path of the first difference between a and b, such as
// "codeBook/dataDscr[3]/var[0]: text \"a\" != \"b\"", or "" when they are Equal.
func Diff(a, b *Node) string {
	if a == nil || b == nil {
		if a == b {
			return ""
		}
		return "one node is nil"
	}
	return diff(a, b, a.Name)
}

func diff(a, b *Node, path string) string {
	switch {
	case a.Name != b.Name:
		return fmt.Sprintf("%s: name %q != %q", path, a.Name, b.Name)
	case !slices.Equal(sortAttrs(a.Attrs), sortAttrs(b.Attrs)):
		return fmt.Sprintf("%s: attrs %q != %q", path, a.Attrs, b.Attrs)
	case a.Text != b.Text:
		return fmt.Sprintf("%s: text %q != %q", path, a.Text, b.Text)
	case a.Raw != b.Raw:
		return fmt.Sprintf("%s: raw %q != %q", path, a.Raw, b.Raw)
	}
	for i := range min(len(a.Children), len(b.Children)) {
		c := a.Children[i]
		if d := diff(c, b.Children[i], fmt.Sprintf("%s/%s[%d]", path, c.Name, i)); d != "" {
			return d
		}
	}
	if len(a.Children) != len(b.Children) {
		return fmt.Sprintf("%s: children %d != %d", path, len(a.Children), len(b.Children))
	}
	return ""
}

func sortAttrs(attrs []Attr) []Attr {
	s := slices.Clone(attrs)
	slices.SortFunc(s, func(x, y Attr) int {
		return cmp.Or(strings.Compare(x.Name, y.Name), strings.Compare(x.Value, y.Value))
	})
	return s
}

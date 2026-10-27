// Package schema answers content-model questions about DDI Codebook 2.5 elements
// from tables generated from the vendored XSD.
package schema

import (
	"slices"
	"strings"
)

//go:generate go run ./gen ../../xsd/ddi-codebook-2.5/codebook.xsd table.go

// IsRepeatable reports whether name can occur more than once under any parent.
// Names the XSD does not declare (foreign elements such as xhtml:p) count as
// repeatable, since nothing bounds them.
func IsRepeatable(name string) bool {
	r, known := repeatable[name]
	return r || !known
}

// Position returns child's sort position in parent's content model. Members of
// one choice share a position.
func Position(parent, child string) (int, bool) {
	p, ok := children[parent][child]
	return p, ok
}

// Allowed reports whether child may appear under parent. Unknown parents allow
// anything. A parent whose model includes a foreign group or xs:any also allows
// any prefixed (non-DDI) child, but not DDI elements outside its model.
func Allowed(parent, child string) bool {
	if _, known := repeatable[parent]; !known {
		return true
	}
	if _, ok := children[parent][child]; ok {
		return true
	}
	// ponytail: xs:any is treated like a foreign group. The schema's only xs:any
	// sits in abstract abstractTextType, which every concrete type restricts away;
	// split anyChild into "any DDI child" and "foreign child" if that changes.
	return anyChild[parent] && strings.Contains(child, ":")
}

// AttrEnum returns the values the XSD enumerates for attribute attr of element,
// in XSD order, or nil when it enumerates none.
func AttrEnum(element, attr string) []string {
	return slices.Clone(attrEnums[element][attr])
}

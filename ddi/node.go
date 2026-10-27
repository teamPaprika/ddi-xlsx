// Package ddi holds the DDI Codebook 2.5 document tree and reads it from XML.
package ddi

// Namespace is the DDI Codebook 2.5 namespace URI.
const Namespace = "ddi:codebook:2_5"

// Attr is one attribute. Name keeps its prefix as written: "xml:lang",
// "xsi:schemaLocation", "xmlns:xsi".
type Attr struct{ Name, Value string }

// Node is one element. At most one of Text, Children and Raw is set.
type Node struct {
	Name     string // unprefixed for DDI elements; foreign elements keep their prefix ("xhtml:p")
	Attrs    []Attr // input order
	Text     string // set only when the node has no element children and Raw == ""
	Children []*Node
	// Raw is the verbatim inner XML of an element whose content cannot be split
	// into text or children without losing meaning or order.
	Raw string
}

// Attr returns the value of the attribute named name.
func (n *Node) Attr(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

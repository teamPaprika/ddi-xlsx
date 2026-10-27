package ddi

import (
	"fmt"
	"strings"
)

var (
	textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	// Literal tabs and line breaks in a value would be turned into spaces by
	// attribute-value normalization when the XML is read back.
	attrEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;",
		"\t", "&#9;", "\n", "&#10;", "\r", "&#13;")
)

// WriteXML writes root as a canonical document: an XML declaration, two-space
// indentation, element-only content one child per line, text and kept-as-written
// content inline, and a trailing newline. Raw is not re-parsed here: it enters
// a tree only through ParseXML or ParseFragment, which check it is well-formed.
func WriteXML(root *Node) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	writeNode(&b, root, "\n")
	b.WriteString("\n")
	return []byte(b.String())
}

// WriteFragment writes each node on its own line with no indentation inside it.
func WriteFragment(nodes []*Node) string {
	var b strings.Builder
	for i, n := range nodes {
		if i > 0 {
			b.WriteString("\n")
		}
		writeNode(&b, n, "")
	}
	return b.String()
}

// FormatAttrs writes attrs as name="value" pairs separated by spaces.
func FormatAttrs(attrs []Attr) string {
	var b strings.Builder
	writeAttrs(&b, attrs)
	return strings.TrimPrefix(b.String(), " ")
}

// ParseAttrs reads name="value" or name='value' pairs, in order.
func ParseAttrs(s string) ([]Attr, error) {
	// The only way to end the wrapper as one element is through the trailing
	// "/>", so content smuggled into s shows up as a second node or an error.
	nodes, err := ParseFragment("<x "+s+"/>", nil)
	if err != nil {
		return nil, fmt.Errorf("attributes %q: %w", s, err)
	}
	if len(nodes) != 1 {
		return nil, fmt.Errorf("attributes %q: not a list of name=\"value\" pairs", s)
	}
	return nodes[0].Attrs, nil
}

// writeNode writes n. nl is the line break plus indentation that precedes n,
// or "" to write it compact.
func writeNode(b *strings.Builder, n *Node, nl string) {
	b.WriteString("<" + n.Name)
	writeAttrs(b, n.Attrs)
	// Cleaning can empty a value, so the empty-element form is decided after it.
	raw, text := clean(n.Raw), clean(n.Text)
	switch {
	case raw != "":
		b.WriteString(">" + raw)
	case text != "":
		b.WriteString(">" + textEscaper.Replace(text))
	case len(n.Children) > 0:
		b.WriteString(">")
		inner := nl
		if nl != "" {
			inner += "  "
		}
		for _, c := range n.Children {
			b.WriteString(inner)
			writeNode(b, c, inner)
		}
		b.WriteString(nl)
	default:
		b.WriteString("/>")
		return
	}
	b.WriteString("</" + n.Name + ">")
}

func writeAttrs(b *strings.Builder, attrs []Attr) {
	for _, a := range attrs {
		b.WriteString(" " + a.Name + `="` + attrEscaper.Replace(clean(a.Value)) + `"`)
	}
}

// clean removes what XML 1.0 forbids: C0 controls other than tab, LF and CR,
// U+FFFE, U+FFFF, and bytes that are not UTF-8 (which is how a lone surrogate
// reaches a Go string).
func clean(s string) string {
	// ToValidUTF8 first: strings.Map reports invalid bytes as U+FFFD, which it
	// could not tell apart from a real U+FFFD.
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0xFFFE || r == 0xFFFF {
			return -1
		}
		return r
	}, strings.ToValidUTF8(s, ""))
}

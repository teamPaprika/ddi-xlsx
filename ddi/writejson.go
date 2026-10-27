package ddi

import (
	"fmt"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/internal/schema"
)

// WriteJSON writes root as {"codeBook": …} in the layout of
// JSON.stringify(x, null, 2) plus a trailing newline. Attributes become "@name"
// keys, text "#text" (or a bare string when there are no attributes), kept-as-written
// content "#xml", and children are grouped by name in first-seen order. A name
// is an array when the schema lets it repeat, or when it does repeat here.
func WriteJSON(root *Node) []byte {
	var b strings.Builder
	writeObject(&b, nil, "", "", []*Node{root}, "\n")
	b.WriteString("\n")
	return []byte(b.String())
}

// writeValue writes n. nl is the line break plus indentation of the line n starts on.
func writeValue(b *strings.Builder, n *Node, nl string) {
	// The same content WriteXML writes, so XML and JSON carry the same tree.
	raw, text, children := clean(n.Raw), clean(n.Text), n.Children
	if raw != "" || text != "" {
		children = nil
	}
	if len(n.Attrs) == 0 && raw == "" && len(children) == 0 {
		writeString(b, text)
		return
	}
	writeObject(b, n.Attrs, raw, text, children, nl)
}

func writeObject(b *strings.Builder, attrs []Attr, raw, text string, children []*Node, nl string) {
	inner := nl + "  "
	sep := "{"
	member := func(key string) {
		b.WriteString(sep + inner)
		writeString(b, key)
		b.WriteString(": ")
		sep = ","
	}
	for _, a := range attrs {
		member("@" + a.Name)
		writeString(b, clean(a.Value))
	}
	if raw != "" {
		member("#xml")
		writeString(b, raw)
	} else if text != "" {
		member("#text")
		writeString(b, text)
	}
	var names []string
	groups := map[string][]*Node{}
	for _, c := range children {
		if groups[c.Name] == nil {
			names = append(names, c.Name)
		}
		groups[c.Name] = append(groups[c.Name], c)
	}
	for _, name := range names {
		member(name)
		g := groups[name]
		// A name the schema bounds to one still repeats in a document that breaks
		// the schema; an array keeps every occurrence.
		if len(g) == 1 && !schema.IsRepeatable(name) {
			writeValue(b, g[0], inner)
			continue
		}
		open := "["
		for _, c := range g {
			b.WriteString(open + inner + "  ")
			writeValue(b, c, inner+"  ")
			open = ","
		}
		b.WriteString(inner + "]")
	}
	if sep == "{" {
		b.WriteString("{}")
		return
	}
	b.WriteString(nl + "}")
}

// writeString escapes as JSON.stringify does: only the quote, the backslash and
// C0 controls.
func writeString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 {
				fmt.Fprintf(b, `\u%04x`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
}

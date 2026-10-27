package ddi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// ParseJSON reads a document in the layout WriteJSON writes. It also takes a
// scalar where an array is expected, which is how Walla exports write it.
func ParseJSON(data []byte) (*Node, error) {
	// json.Decoder.Token keeps object keys in input order; decoding into a map would not.
	d := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	errTop := errors.New(`JSON must be an object with the single key "codeBook"`)
	tok, err := d.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, errTop
	}
	// The wrapper object sits one level above codeBook, the root at depth 0.
	top, err := readElement(d, "", tok, -1)
	if err != nil {
		return nil, err
	}
	if tok, err := d.Token(); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("unexpected %v after the JSON object", tok)
		}
		return nil, err
	}
	if len(top.Attrs) > 0 || top.Text != "" || top.Raw != "" || len(top.Children) != 1 || top.Children[0].Name != "codeBook" {
		return nil, errTop
	}
	root := top.Children[0]
	xmlns, _ := root.Attr("xmlns")
	if err := checkNamespace(xmlns); err != nil {
		return nil, err
	}
	return rebuild(root, nil)
}

// readElements reads the value of key name: an array is a run of siblings,
// anything else one element. depth is the depth of those elements.
func readElements(d *json.Decoder, name string, depth int) ([]*Node, error) {
	tok, err := d.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('[') {
		n, err := readElement(d, name, tok, depth)
		return []*Node{n}, err
	}
	var nodes []*Node
	for d.More() {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		if tok == json.Delim('[') {
			return nil, fmt.Errorf("%s: array inside an array", name)
		}
		n, err := readElement(d, name, tok, depth)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	_, err = d.Token()
	return nodes, err
}

// readElement reads the element named name whose value starts with tok. Raw
// holds "#xml" as written; rebuild checks and classifies it.
func readElement(d *json.Decoder, name string, tok json.Token, depth int) (*Node, error) {
	if depth > MaxDepth {
		return nil, fmt.Errorf("%s: nested more than %d elements deep (at offset %d)", name, MaxDepth, d.InputOffset())
	}
	n := &Node{Name: name}
	if tok != json.Delim('{') {
		s, err := jsonString(name, tok)
		n.Text = normalize(s)
		return n, err
	}
	seen := map[string]bool{}
	for d.More() {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		key := tok.(string)
		if seen[key] {
			return nil, fmt.Errorf("%s: key %q repeated", name, key)
		}
		seen[key] = true
		if key != "#text" && key != "#xml" && !strings.HasPrefix(key, "@") {
			c, err := readElements(d, key, depth+1)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, c...)
			continue
		}
		if tok, err = d.Token(); err != nil {
			return nil, err
		}
		s, err := jsonString(name+" "+key, tok)
		if err != nil {
			return nil, err
		}
		switch key {
		case "#text":
			n.Text = normalize(s)
		case "#xml":
			n.Raw = s
		default:
			n.Attrs = append(n.Attrs, Attr{key[1:], s})
		}
	}
	_, err := d.Token()
	return n, err
}

func jsonString(where string, tok json.Token) (string, error) {
	if s, ok := tok.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("%s: want a string, not %v", where, tok)
}

// normalize makes text what reading it back from WriteXML gives, which cleans
// before it trims.
func normalize(s string) string { return NormalizeText(clean(s)) }

// rebuild passes n's name, attributes and #xml through the XML reader, so a
// name or attribute that XML cannot hold is an error rather than broken output,
// prefixes resolve as in XML, and #xml is checked and classified exactly as
// ParseXML would. ns holds the xmlns:* declarations of n's ancestors.
func rebuild(n *Node, ns []Attr) (*Node, error) {
	switch {
	case n.Raw != "" && (n.Text != "" || len(n.Children) > 0):
		return nil, fmt.Errorf("%s: #xml cannot sit next to text or child elements", n.Name)
	case n.Text != "" && len(n.Children) > 0:
		return nil, fmt.Errorf("%s: text next to child elements must be written as #xml", n.Name)
	}
	for _, a := range n.Attrs {
		if strings.HasPrefix(a.Name, "xmlns:") {
			ns = append(slices.Clip(ns), a)
		}
	}
	nodes, err := ParseFragment("<"+n.Name+" "+FormatAttrs(n.Attrs)+">"+clean(n.Raw)+"</"+n.Name+">", ns)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", n.Name, err)
	}
	if len(nodes) != 1 || !sameAttrNames(nodes[0].Attrs, n.Attrs) {
		return nil, fmt.Errorf("%s: not an element name with its attributes and content", n.Name)
	}
	m := nodes[0]
	if n.Raw != "" {
		return m, nil
	}
	m.Text = n.Text
	for _, c := range n.Children {
		c, err := rebuild(c, ns)
		if err != nil {
			return nil, err
		}
		m.Children = append(m.Children, c)
	}
	return m, nil
}

// sameAttrNames catches a name or attribute name that smuggled in more
// attributes, such as the key `labl x="1"`.
func sameAttrNames(a, b []Attr) bool {
	return slices.EqualFunc(a, b, func(x, y Attr) bool { return x.Name == y.Name })
}

package ddi

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

const xmlSpace = " \t\n\r"

// MaxDepth is how deeply elements may nest, counting the root as 0. Real
// codebooks stay under about 20; without a bound, a crafted file costs time
// and output that grow faster than its size.
const MaxDepth = 256

// toLF turns CRLF and lone CR into LF, the line ending an XML reader reports,
// so a value equals what reading its written form gives back.
var toLF = strings.NewReplacer("\r\n", "\n", "\r", "\n")

// ParseXML reads a well-formed DDI Codebook 2.5 document. Comments, processing
// instructions and the DOCTYPE are dropped, each with a warning, unless they sit
// inside a kept-as-written element, whose Raw holds them.
func ParseXML(data []byte) (*Node, []string, error) {
	// A byte order mark would otherwise read as text before the root.
	return parse(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), false)
}

// ParseFragment reads a sequence of elements as children of a wrapper that
// declares the DDI default namespace plus the xmlns:* declarations in ns, so a
// fragment may use the prefixes its document root declared. The elements are
// classified exactly as ParseXML classifies them; comments and processing
// instructions are dropped silently.
func ParseFragment(fragment string, ns []Attr) ([]*Node, error) {
	nodes, _, err := ParseFragmentWithWarnings(fragment, ns)
	return nodes, err
}

// ParseFragmentWithWarnings is ParseFragment plus the warnings ParseXML would
// give for the fragment, such as a dropped comment or an undeclared prefix.
func ParseFragmentWithWarnings(fragment string, ns []Attr) ([]*Node, []string, error) {
	var b strings.Builder
	b.WriteString(`<x xmlns="` + Namespace + `"`)
	for _, a := range ns {
		if strings.HasPrefix(a.Name, "xmlns:") {
			b.WriteString(" " + a.Name + `="`)
			xml.EscapeText(&b, []byte(a.Value))
			b.WriteString(`"`)
		}
	}
	b.WriteString(">" + fragment + "</x>")
	root, warns, err := parse([]byte(b.String()), true)
	if err != nil {
		return nil, nil, err
	}
	return root.Children, warns, nil
}

// NormalizeText trims surrounding whitespace and the indentation of every line
// after the first, which exporters such as NADA add around text values.
func NormalizeText(s string) string {
	lines := strings.Split(strings.Trim(toLF.Replace(s), xmlSpace), "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.TrimLeft(lines[i], xmlSpace)
	}
	return strings.Join(lines, "\n")
}

type frame struct {
	node  *Node
	qname string            // as written, to match the end tag
	ns    map[string]string // prefix -> URI declared on this element
	texts []string          // texts[i] is the character data after the i-th child
	text  strings.Builder
	start int64 // input offset of the inner XML
	warns []string
}

// parse builds the tree. In fragment mode the root is a wrapper whose children
// are the result, so it is neither namespace-checked nor classified.
func parse(data []byte, fragment bool) (*Node, []string, error) {
	// RawToken keeps prefixes as written; Token would turn them into URIs we
	// would have to map back.
	d := xml.NewDecoder(bytes.NewReader(data))
	// The decoder's own error names a Go field. Converting other encodings
	// would take a second dependency, so the user converts the file instead.
	var encErr error
	d.CharsetReader = func(enc string, _ io.Reader) (io.Reader, error) {
		encErr = fmt.Errorf("only UTF-8 input is supported (the file declares encoding %q)", enc)
		return nil, encErr
	}
	var stack []*frame
	var root *Node
	var warns []string
	warn := func(line int, format string, a ...any) {
		msg := fmt.Sprintf(format+" at line %d", append(a, line)...)
		if len(stack) == 0 {
			warns = append(warns, msg)
		} else {
			top := stack[len(stack)-1]
			top.warns = append(top.warns, msg)
		}
	}
	// Namespace-aware readers reject a prefix that nothing binds. The warning
	// is not tied to a frame, since content kept as written still carries it.
	// XML itself binds "xml", and "xmlns" only declares.
	undeclared := func(line int, n xml.Name, what string) {
		if n.Space != "" && n.Space != "xml" && n.Space != "xmlns" && resolve(stack, n.Space) == "" {
			warns = append(warns, fmt.Sprintf("prefix %q of %s is not declared at line %d", n.Space, what, line))
		}
	}
	for {
		off := d.InputOffset()
		line, _ := d.InputPos()
		tok, err := d.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			if encErr != nil {
				return nil, nil, encErr
			}
			return nil, nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if root != nil {
				return nil, nil, &xml.SyntaxError{Msg: "more than one root element", Line: line}
			}
			f := &frame{qname: qname(t.Name), ns: map[string]string{}, start: d.InputOffset(), node: &Node{}}
			// The decoder does not check this well-formedness rule. A map keeps
			// an element with very many attributes from costing quadratic time.
			seen := make(map[string]bool, len(t.Attr))
			for _, a := range t.Attr {
				name := qname(a.Name)
				if seen[name] {
					return nil, nil, &xml.SyntaxError{Msg: "attribute " + name + " is repeated", Line: line}
				}
				seen[name] = true
				if a.Name.Space == "xmlns" {
					f.ns[a.Name.Local] = a.Value
				} else if name == "xmlns" {
					f.ns[""] = a.Value
				}
				f.node.Attrs = append(f.node.Attrs, Attr{name, a.Value})
			}
			stack = append(stack, f)
			if len(stack)-1 > MaxDepth {
				return nil, nil, &xml.SyntaxError{Msg: fmt.Sprintf("<%s> is nested more than %d elements deep", f.qname, MaxDepth), Line: line}
			}
			undeclared(line, t.Name, "<"+f.qname+">")
			for _, a := range t.Attr {
				undeclared(line, a.Name, qname(a.Name))
			}
			uri := resolve(stack, t.Name.Space)
			f.node.Name = t.Name.Local
			if t.Name.Space != "" && uri != Namespace {
				f.node.Name = f.qname
			}
			if len(stack) == 1 && !fragment {
				if err := checkNamespace(uri); err != nil {
					return nil, nil, err
				}
				if t.Name.Local != "codeBook" {
					return nil, nil, fmt.Errorf("the root element is <%s>, not <codeBook>", f.qname)
				}
				// DDI names lose their prefix in the tree, so the root must make
				// DDI the default namespace for them to stay DDI when written. The
				// prefix declaration stays, since Raw content still uses the prefix.
				// An unprefixed root has it already: checkNamespace saw to that.
				switch def, ok := f.ns[""]; {
				case !ok:
					f.node.Attrs = append([]Attr{{"xmlns", Namespace}}, f.node.Attrs...)
				case def != Namespace:
					return nil, nil, fmt.Errorf("the root <%s> declares the default namespace %q; "+
						"DDI names are written without a prefix, so it must be absent or %q", f.qname, def, Namespace)
				}
			}
		case xml.EndElement:
			if len(stack) == 0 || stack[len(stack)-1].qname != qname(t.Name) {
				return nil, nil, &xml.SyntaxError{Msg: "unexpected end element </" + qname(t.Name) + ">", Line: line}
			}
			f := stack[len(stack)-1]
			f.texts = append(f.texts, f.text.String())
			stack = stack[:len(stack)-1]
			if fragment && len(stack) == 0 {
				for _, s := range f.texts {
					if strings.Trim(s, xmlSpace) != "" {
						return nil, nil, &xml.SyntaxError{Msg: "text outside elements", Line: line}
					}
				}
				root = f.node
				warns = append(warns, f.warns...)
				break
			}
			if classify(f, data[f.start:off]) {
				f.warns = nil // kept in Raw, so not dropped
			}
			if len(stack) == 0 {
				root = f.node
				warns = append(warns, f.warns...)
				break
			}
			p := stack[len(stack)-1]
			p.node.Children = append(p.node.Children, f.node)
			p.texts = append(p.texts, p.text.String())
			p.text.Reset()
			p.warns = append(p.warns, f.warns...)
		case xml.CharData:
			if len(stack) == 0 {
				if strings.Trim(string(t), xmlSpace) != "" {
					return nil, nil, &xml.SyntaxError{Msg: "text outside the root element", Line: line}
				}
				break
			}
			f := stack[len(stack)-1]
			f.text.Write(t)
		case xml.Comment:
			warn(line, "comment dropped")
		case xml.ProcInst:
			if t.Target != "xml" {
				warn(line, "processing instruction <?%s?> dropped", t.Target)
			}
		case xml.Directive:
			// The only directive a well-formed document can hold; the decoder
			// returns its internal subset as part of it.
			warn(line, "DOCTYPE dropped")
		}
	}
	if len(stack) > 0 {
		line, _ := d.InputPos()
		return nil, nil, &xml.SyntaxError{Msg: "unexpected EOF: <" + stack[len(stack)-1].qname + "> is not closed", Line: line}
	}
	if root == nil {
		return nil, nil, errors.New("no root element")
	}
	return root, warns, nil
}

func checkNamespace(uri string) error {
	switch uri {
	case Namespace:
		return nil
	case "http://www.icpsr.umich.edu/DDI":
		return errors.New("DDI 1.2.2 is not supported; convert the file to DDI 2.5 first")
	}
	return fmt.Errorf("not a DDI Codebook 2.5 document (namespace %q)", uri)
}

// classify sets the node's Text or Raw from its content and reports whether it
// kept the content as written.
func classify(f *frame, inner []byte) bool {
	n := f.node
	if len(n.Children) == 0 {
		n.Text = NormalizeText(f.texts[0])
		return false
	}
	if !keepRaw(n.Children, f.texts) {
		return false
	}
	n.Children = nil
	n.Raw = toLF.Replace(string(inner))
	return true
}

func keepRaw(children []*Node, texts []string) bool {
	for i, s := range texts {
		if strings.Trim(s, xmlSpace) != "" {
			return true
		}
		// Layout between children always breaks the line; a bare space between
		// two children, as in <emph>a</emph> <emph>b</emph>, is content.
		if i > 0 && i < len(texts)-1 && s != "" && !strings.Contains(s, "\n") {
			return true
		}
	}
	// Children are grouped by name later, which would reorder a name that
	// comes back after a different one.
	seen := map[string]bool{}
	for i, c := range children {
		if seen[c.Name] && children[i-1].Name != c.Name {
			return true
		}
		seen[c.Name] = true
	}
	return false
}

func resolve(stack []*frame, prefix string) string {
	for i := len(stack) - 1; i >= 0; i-- {
		if uri, ok := stack[i].ns[prefix]; ok {
			return uri
		}
	}
	return ""
}

func qname(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

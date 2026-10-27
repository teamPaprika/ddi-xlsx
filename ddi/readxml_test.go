package ddi

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

func doc(inner string) []byte {
	return []byte(`<codeBook xmlns="ddi:codebook:2_5">` + inner + `</codeBook>`)
}

func mustParse(t *testing.T, data []byte) *Node {
	t.Helper()
	root, warns, err := ParseXML(data)
	if err != nil {
		t.Fatalf("ParseXML: %v", err)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %q", warns)
	}
	return root
}

func dump(n *Node) string {
	var b strings.Builder
	var walk func(*Node)
	walk = func(n *Node) {
		fmt.Fprintf(&b, "%s %v %q %q;", n.Name, n.Attrs, n.Text, n.Raw)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func TestNormalizeText(t *testing.T) {
	for in, want := range map[string]string{
		"\n          1\n        ": "1",
		"a\n   b\n":               "a\nb",
		"x":                       "x",
		"a  \n\t b":               "a  \nb",
		"\u00a0a":                 "\u00a0a",
		"a\r\nb\rc":               "a\nb\nc",
	} {
		if got := NormalizeText(in); got != want {
			t.Errorf("NormalizeText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIndentedQstnIsStructured(t *testing.T) {
	root := mustParse(t, doc("<qstn>\n  <qstnLit>\n    Q?\n  </qstnLit>\n  <preQTxt>pre</preQTxt>\n</qstn>"))
	q := root.Children[0]
	if q.Raw != "" || len(q.Children) != 2 || q.Children[0].Text != "Q?" || q.Children[1].Text != "pre" {
		t.Fatalf("qstn = %s", dump(q))
	}
}

func TestKeptAsWritten(t *testing.T) {
	for in, raw := range map[string]string{
		`<abstract>a <emph>b</emph></abstract>`:                                       `a <emph>b</emph>`,
		`<abstract><emph>a</emph> <emph>b</emph></abstract>`:                          `<emph>a</emph> <emph>b</emph>`,
		`<qstn><qstnLit>a</qstnLit><ivuInstr>i</ivuInstr><qstnLit>b</qstnLit></qstn>`: `<qstnLit>a</qstnLit><ivuInstr>i</ivuInstr><qstnLit>b</qstnLit>`,
	} {
		n := mustParse(t, doc(in)).Children[0]
		if n.Raw != raw || n.Text != "" || n.Children != nil {
			t.Errorf("%s: got %s, want Raw %q", in, dump(n), raw)
		}
	}
	// Whitespace with a newline between children is layout, and a newline-free space
	// outside the children is not "between" them.
	n := mustParse(t, doc("<qstn> <qstnLit>a</qstnLit>\n<preQTxt>p</preQTxt></qstn>")).Children[0]
	if n.Raw != "" || len(n.Children) != 2 {
		t.Errorf("structured qstn: got %s", dump(n))
	}
}

func TestRawCRLFNormalized(t *testing.T) {
	n := mustParse(t, doc("<abstract>a\r\n<emph>b\r\nc</emph></abstract>")).Children[0]
	if n.Raw != "a\n<emph>b\nc</emph>" {
		t.Fatalf("Raw = %q", n.Raw)
	}
}

func TestNamespaces(t *testing.T) {
	root := mustParse(t, []byte(`<codeBook xmlns="ddi:codebook:2_5" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="ddi:codebook:2_5 codebook.xsd" xml:lang="en" xmlns:xhtml="http://www.w3.org/1999/xhtml"><stdyDscr><xhtml:p>x</xhtml:p></stdyDscr></codeBook>`))
	want := []Attr{
		{"xmlns", "ddi:codebook:2_5"},
		{"xmlns:xsi", "http://www.w3.org/2001/XMLSchema-instance"},
		{"xsi:schemaLocation", "ddi:codebook:2_5 codebook.xsd"},
		{"xml:lang", "en"},
		{"xmlns:xhtml", "http://www.w3.org/1999/xhtml"},
	}
	if !reflect.DeepEqual(root.Attrs, want) {
		t.Errorf("root attrs = %v", root.Attrs)
	}
	if v, ok := root.Attr("xml:lang"); !ok || v != "en" {
		t.Errorf(`Attr("xml:lang") = %q, %v`, v, ok)
	}
	if p := root.Children[0].Children[0]; p.Name != "xhtml:p" || p.Text != "x" {
		t.Errorf("foreign child = %s", dump(p))
	}

	root = mustParse(t, []byte(`<ddi:codeBook xmlns:ddi="ddi:codebook:2_5" version="2.5"><ddi:stdyDscr/><ddi:abstract><ddi:emph>x</ddi:emph> <ddi:emph>y</ddi:emph></ddi:abstract></ddi:codeBook>`))
	if root.Name != "codeBook" || root.Children[0].Name != "stdyDscr" || root.Children[1].Name != "abstract" {
		t.Errorf("prefixed DDI = %s", dump(root))
	}
	if want := []Attr{{"xmlns", "ddi:codebook:2_5"}, {"xmlns:ddi", "ddi:codebook:2_5"}, {"version", "2.5"}}; !reflect.DeepEqual(root.Attrs, want) {
		t.Errorf("prefixed root attrs = %v", root.Attrs)
	}
	// Raw keeps the prefix as written, so it must still resolve against the root attrs.
	raw := root.Children[1].Raw
	if raw != `<ddi:emph>x</ddi:emph> <ddi:emph>y</ddi:emph>` {
		t.Fatalf("prefixed Raw = %q", raw)
	}
	if _, err := ParseFragment("<abstract>"+raw+"</abstract>", root.Attrs); err != nil {
		t.Errorf("re-parse Raw: %v", err)
	}
	if nodes, err := ParseFragment(raw, root.Attrs); err != nil || len(nodes) != 2 || nodes[0].Name != "emph" {
		t.Errorf("Raw children = %v %v", nodes, err)
	}
}

func TestRejectsOtherVersions(t *testing.T) {
	for in, want := range map[string]string{
		`<codeBook xmlns="http://www.icpsr.umich.edu/DDI"/>`: "DDI 1.2.2 is not supported; convert the file to DDI 2.5 first",
		`<codeBook xmlns="urn:x"/>`:                          `not a DDI Codebook 2.5 document (namespace "urn:x")`,
		`<codeBook/>`:                                        `not a DDI Codebook 2.5 document (namespace "")`,
		`<var xmlns="ddi:codebook:2_5"/>`:                    `the root element is <var>, not <codeBook>`,
		`<ddi:codeBook xmlns:ddi="ddi:codebook:2_5" xmlns="urn:x"/>`: `the root <ddi:codeBook> declares the default namespace "urn:x"; ` +
			`DDI names are written without a prefix, so it must be absent or "ddi:codebook:2_5"`,
		`<?xml version="1.0" encoding="EUC-KR"?><codeBook xmlns="ddi:codebook:2_5"/>`: `only UTF-8 input is supported (the file declares encoding "EUC-KR")`,
	} {
		_, _, err := ParseXML([]byte(in))
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", in, err, want)
		}
	}
}

func TestRejectsMalformed(t *testing.T) {
	for _, in := range []string{
		``,
		`<codeBook xmlns="ddi:codebook:2_5">`,
		`<codeBook xmlns="ddi:codebook:2_5"></stdyDscr>`,
		`<codeBook xmlns="ddi:codebook:2_5"/><codeBook xmlns="ddi:codebook:2_5"/>`,
		`x<codeBook xmlns="ddi:codebook:2_5"/>`,
		`<codeBook xmlns="ddi:codebook:2_5"><a>&bogus;</a></codeBook>`,
	} {
		if _, _, err := ParseXML([]byte(in)); err == nil {
			t.Errorf("%q: want error", in)
		}
	}
	// A UTF-8 byte order mark is not content.
	mustParse(t, append([]byte("\xef\xbb\xbf"), doc("")...))
	for _, enc := range []string{"UTF-8", "utf-8"} {
		mustParse(t, append([]byte(`<?xml version="1.0" encoding="`+enc+`"?>`), doc("")...))
	}
	_, _, err := ParseXML(doc(`<var ID="V1" name="a" ID="V2"/>`))
	if err == nil || !strings.Contains(err.Error(), "attribute ID is repeated") {
		t.Errorf("repeated attribute: err = %v", err)
	}
}

// nest wraps n <a> elements inside one another.
func nest(n int) string { return strings.Repeat("<a>", n) + strings.Repeat("</a>", n) }

func TestDepthLimit(t *testing.T) {
	// The root is depth 0, so its stdyDscr child plus n nested elements reach depth n+1.
	mustParse(t, doc("<stdyDscr>"+nest(199)+"</stdyDscr>"))
	if _, _, err := ParseXML(doc("<stdyDscr>" + nest(299) + "</stdyDscr>")); err == nil ||
		!strings.Contains(err.Error(), "line 1: <a> is nested more than 256 elements deep") {
		t.Errorf("depth 300: err = %v", err)
	}
	if _, err := ParseFragment(nest(200), nil); err != nil {
		t.Errorf("fragment depth 200: %v", err)
	}
	if _, err := ParseFragment(nest(300), nil); err == nil || !strings.Contains(err.Error(), "nested more than 256") {
		t.Errorf("fragment depth 300: err = %v", err)
	}
}

func TestFragmentWarnings(t *testing.T) {
	ns := []Attr{{"xmlns:x", "urn:x"}}
	nodes, warns, err := ParseFragmentWithWarnings(`<a/><!--c--><b><?pi?></b><x:c x:d="1" xml:lang="en"/>`, ns)
	if err != nil || len(nodes) != 3 {
		t.Fatalf("nodes = %v, err = %v", nodes, err)
	}
	if want := []string{"comment dropped at line 1", "processing instruction <?pi?> dropped at line 1"}; !reflect.DeepEqual(warns, want) {
		t.Errorf("warnings = %q, want %q", warns, want)
	}
	_, warns, err = ParseFragmentWithWarnings("<y:e/>\n<f y:g=\"1\"/>", ns)
	want := []string{`prefix "y" of <y:e> is not declared at line 1`, `prefix "y" of y:g is not declared at line 2`}
	if err != nil || !reflect.DeepEqual(warns, want) {
		t.Errorf("undeclared prefixes: warnings = %q, err = %v; want %q", warns, err, want)
	}
}

func TestWarnings(t *testing.T) {
	const decl = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	for _, in := range []string{
		decl + `<codeBook xmlns="ddi:codebook:2_5"><stdyDscr><!-- zzz --><citation/></stdyDscr></codeBook>`,
		decl + `<codeBook xmlns="ddi:codebook:2_5"><titl>a<?zzz q?>b</titl></codeBook>`,
		decl + `<!DOCTYPE zzz><codeBook xmlns="ddi:codebook:2_5"/>`,
	} {
		root, warns, err := ParseXML([]byte(in))
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if len(warns) != 1 {
			t.Errorf("%s: warnings = %q, want 1", in, warns)
		}
		if s := dump(root); strings.Contains(s, "zzz") {
			t.Errorf("%s: tree holds dropped item: %s", in, s)
		}
	}
	// A comment inside a kept-as-written element stays in Raw, so nothing is dropped.
	root, warns, err := ParseXML(doc(`<abstract>a <!--c--><emph>b</emph></abstract>`))
	if err != nil || len(warns) != 0 || root.Children[0].Raw != `a <!--c--><emph>b</emph>` {
		t.Errorf("comment in Raw: %v %q %s", err, warns, dump(root))
	}
}

func TestParseFragment(t *testing.T) {
	nodes, err := ParseFragment("<location fileid=\"F1\"/>\n<sumStat type=\"vald\">580</sumStat>", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 || nodes[0].Name != "location" || nodes[1].Name != "sumStat" || nodes[1].Text != "580" {
		t.Fatalf("nodes = %v", nodes)
	}
	if v, _ := nodes[0].Attr("fileid"); v != "F1" {
		t.Errorf("fileid = %q", v)
	}

	// The wrapper is not classified: repeated top-level names stay separate nodes.
	nodes, err = ParseFragment(`<p>a</p><list/><p>b</p>`, nil)
	if err != nil || len(nodes) != 3 {
		t.Errorf("top-level repeats: %v %v", nodes, err)
	}

	// Children classify exactly like ParseXML.
	nodes, err = ParseFragment(`<abstract>a <emph>b</emph></abstract>`, nil)
	if err != nil || len(nodes) != 1 || nodes[0].Raw != `a <emph>b</emph>` {
		t.Errorf("raw child: %v %v", nodes, err)
	}

	ns := []Attr{{"xmlns", "ignored"}, {"xmlns:xhtml", "http://www.w3.org/1999/xhtml"}, {"xmlns:ddi", Namespace}}
	nodes, err = ParseFragment(`<xhtml:p>x</xhtml:p><ddi:emph>y</ddi:emph>`, ns)
	if err != nil || len(nodes) != 2 || nodes[0].Name != "xhtml:p" || nodes[1].Name != "emph" {
		t.Errorf("namespaced fragment: %v %v", nodes, err)
	}

	for _, in := range []string{`<a>`, `<a></b>`, `x<a/>`, `</x><y/>`} {
		if _, err := ParseFragment(in, nil); err == nil {
			t.Errorf("%q: want error", in)
		}
	}
}

func TestKofacParses(t *testing.T) {
	for _, name := range []string{"kofac_2024_public", "kofac_2025_edu"} {
		data, err := os.ReadFile(filepath.Join(xsdtest.Root(), "testdata", name, "survey.xml"))
		if err != nil {
			t.Fatal(err)
		}
		root := mustParse(t, data)
		v1 := find(root, func(n *Node) bool { id, _ := n.Attr("ID"); return n.Name == "var" && id == "V1" })
		if v1 == nil {
			t.Fatalf("%s: V1 not found", name)
		}
		if v, _ := v1.Attr("name"); v != "response_id" {
			t.Errorf("%s: V1 name = %q", name, v)
		}
	}
}

func find(n *Node, match func(*Node) bool) *Node {
	if match(n) {
		return n
	}
	for _, c := range n.Children {
		if f := find(c, match); f != nil {
			return f
		}
	}
	return nil
}

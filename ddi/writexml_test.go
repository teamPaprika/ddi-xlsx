package ddi

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

var kofacNames = []string{"kofac_2024_public", "kofac_2025_edu"}

func readKofac(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(xsdtest.Root(), "testdata", name, "survey.xml"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLayout(t *testing.T) {
	v := &Node{Name: "var", Attrs: []Attr{{"ID", "V1"}, {"name", "x"}}, Children: []*Node{
		{Name: "location", Attrs: []Attr{{"fileid", "F1"}}},
		{Name: "labl"},
		{Name: "txt", Raw: "a <emph>b</emph>"},
	}}
	root := &Node{Name: "codeBook", Attrs: []Attr{{"xmlns", Namespace}, {"version", "2.5"}}, Children: []*Node{
		{Name: "docDscr", Children: []*Node{{Name: "citation", Children: []*Node{{Name: "titlStmt", Children: []*Node{
			{Name: "titl", Text: `A & B <c> "d" 'e'`},
		}}}}}},
		{Name: "dataDscr", Children: []*Node{v}},
	}}
	want := `<?xml version="1.0" encoding="UTF-8"?>
<codeBook xmlns="ddi:codebook:2_5" version="2.5">
  <docDscr>
    <citation>
      <titlStmt>
        <titl>A &amp; B &lt;c&gt; "d" 'e'</titl>
      </titlStmt>
    </citation>
  </docDscr>
  <dataDscr>
    <var ID="V1" name="x">
      <location fileid="F1"/>
      <labl/>
      <txt>a <emph>b</emph></txt>
    </var>
  </dataDscr>
</codeBook>
`
	if got := string(WriteXML(root)); got != want {
		t.Errorf("WriteXML:\n%s\nwant:\n%s", got, want)
	}

	frag := WriteFragment([]*Node{v, {Name: "labl", Text: "a<b"}})
	if want := `<var ID="V1" name="x"><location fileid="F1"/><labl/><txt>a <emph>b</emph></txt></var>` + "\n" + `<labl>a&lt;b</labl>`; frag != want {
		t.Errorf("WriteFragment = %q, want %q", frag, want)
	}
}

func TestAttrEscapes(t *testing.T) {
	if got, want := FormatAttrs([]Attr{{"x", "a\"b\tc\nd&<>"}}), `x="a&quot;b&#9;c&#10;d&amp;&lt;&gt;"`; got != want {
		t.Errorf("FormatAttrs = %s, want %s", got, want)
	}
	if got := FormatAttrs(nil); got != "" {
		t.Errorf("FormatAttrs(nil) = %q", got)
	}
	for _, attrs := range [][]Attr{
		nil,
		{{"files", "F1"}, {"dcml", "0"}},
		{{"x", "a\"b\tc\nd&<>"}, {"y", "cr\r'q'  "}, {"xml:lang", "ko"}, {"xsi:schemaLocation", "ddi:codebook:2_5 codebook.xsd"}},
	} {
		got, err := ParseAttrs(FormatAttrs(attrs))
		if err != nil || !reflect.DeepEqual(got, attrs) {
			t.Errorf("ParseAttrs(FormatAttrs(%q)) = %q, %v", attrs, got, err)
		}
	}

	got, err := ParseAttrs(` a="1"  b='2' xml:lang="en"` + "\n")
	if want := []Attr{{"a", "1"}, {"b", "2"}, {"xml:lang", "en"}}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("ParseAttrs = %q, %v", got, err)
	}
	for _, in := range []string{`a=1`, `a`, `a="1" a="2"`, `a="&bogus;"`, `a="1"/><y b="2"`, `a="1">t</x><x`, `a="1`} {
		if got, err := ParseAttrs(in); err == nil {
			t.Errorf("ParseAttrs(%q) = %q, want error", in, got)
		}
	}
}

func TestForbiddenCharsDropped(t *testing.T) {
	// "\xed\xa0\x80" is U+D800 encoded as UTF-8: the nearest a Go string gets to
	// a lone surrogate. U+FFFD is a real character and must survive.
	const dirty = "a\x01b\xed\xa0\x80c￾￿\t�\U0001F600"
	const kept = "abc\t�\U0001F600"
	root := &Node{Name: "codeBook", Attrs: []Attr{{"xmlns", Namespace}, {"x", dirty}}, Children: []*Node{
		{Name: "titl", Text: dirty},
		{Name: "txt", Raw: dirty + "<emph>e</emph>"},
	}}
	out := WriteXML(root)
	got, _, err := ParseXML(out)
	if err != nil {
		t.Fatalf("ParseXML(WriteXML): %v\n%s", err, out)
	}
	want := &Node{Name: "codeBook", Attrs: []Attr{{"xmlns", Namespace}, {"x", kept}}, Children: []*Node{
		{Name: "titl", Text: kept},
		{Name: "txt", Raw: kept + "<emph>e</emph>"},
	}}
	if d := Diff(got, want); d != "" {
		t.Errorf("%s\n%s", d, out)
	}
	if s := FormatAttrs([]Attr{{"x", dirty}}); s != `x="abc&#9;`+"�\U0001F600"+`"` {
		t.Errorf("FormatAttrs = %q", s)
	}
}

func TestFixedPoint(t *testing.T) {
	for _, name := range kofacNames {
		t1 := mustParse(t, readKofac(t, name))
		x1 := WriteXML(t1)
		t2 := mustParse(t, x1)
		if d := Diff(t1, t2); d != "" || !Equal(t1, t2) {
			t.Fatalf("%s: tree changed after writing: %s", name, d)
		}
		if x2 := WriteXML(t2); !bytes.Equal(x2, x1) {
			t.Errorf("%s: second write differs from the first", name)
		}
	}
}

// A CR in text would be written literally and read back as LF, so canonical
// text must never hold one.
func TestFixedPointCR(t *testing.T) {
	t1 := mustParse(t, doc(`<titl>a&#13;b</titl>`))
	x1 := WriteXML(t1)
	t2 := mustParse(t, x1)
	if d := Diff(t1, t2); d != "" || !Equal(t1, t2) {
		t.Fatalf("tree changed after writing: %s", d)
	}
	if x2 := WriteXML(t2); !bytes.Equal(x2, x1) {
		t.Errorf("second write differs: %q != %q", x2, x1)
	}
}

func TestRoundTripValidates(t *testing.T) {
	xsdtest.Validate(t, WriteXML(mustParse(t, readKofac(t, "kofac_2025_edu"))))
}

// TestFragmentRoundTrip checks ParseFragment(WriteFragment(ns), nil) == ns for the
// children of every element in both KOFAC documents and for hand-built edge cases.
func TestFragmentRoundTrip(t *testing.T) {
	check := func(label string, ns []*Node) {
		t.Helper()
		frag := WriteFragment(ns)
		got, err := ParseFragment(frag, nil)
		if err != nil {
			t.Fatalf("%s: ParseFragment: %v\n%s", label, err, frag)
		}
		if len(got) != len(ns) {
			t.Fatalf("%s: %d nodes, want %d\n%s", label, len(got), len(ns), frag)
		}
		for i := range ns {
			if d := Diff(ns[i], got[i]); d != "" || !Equal(ns[i], got[i]) {
				t.Fatalf("%s: %s\n%s", label, d, frag)
			}
		}
	}
	check("edge cases", []*Node{
		{Name: "labl", Attrs: []Attr{{"xml:lang", "ko"}, {"x", "a\"b\tc\nd&<>'\r"}}, Text: "a & b <c> \"d\"\n'e' ]]>"},
		{Name: "labl"},
		{Name: "txt", Raw: "a &amp; <emph>b</emph>\n<emph>c</emph>"},
		{Name: "qstn", Children: []*Node{{Name: "qstnLit", Text: "Q?"}, {Name: "ivuInstr", Text: ""}}},
	})
	for _, name := range kofacNames {
		root := mustParse(t, readKofac(t, name))
		check(name+" root", []*Node{root})
		var walk func(*Node)
		walk = func(n *Node) {
			if len(n.Children) > 0 {
				check(name+" "+n.Name, n.Children)
			}
			for _, c := range n.Children {
				walk(c)
			}
		}
		walk(root)
	}
}

func TestEqualAndDiff(t *testing.T) {
	a := &Node{Name: "var", Attrs: []Attr{{"ID", "V1"}, {"name", "x"}}, Children: []*Node{{Name: "labl", Text: "L"}, {Name: "txt", Raw: "r"}}}
	b := &Node{Name: "var", Attrs: []Attr{{"name", "x"}, {"ID", "V1"}}, Children: []*Node{{Name: "labl", Text: "L"}, {Name: "txt", Raw: "r"}}}
	if !Equal(a, b) || Diff(a, b) != "" {
		t.Fatalf("attribute order must not matter: %s", Diff(a, b))
	}
	if !Equal(nil, nil) || Equal(a, nil) || Equal(nil, a) {
		t.Error("nil handling")
	}
	for _, c := range []struct {
		edit func(n *Node)
		want string
	}{
		{func(n *Node) { n.Name = "varx" }, "var: name"},
		{func(n *Node) { n.Attrs[0].Value = "V2" }, "var: attrs"},
		{func(n *Node) { n.Attrs = append(n.Attrs, Attr{"ID", "V1"}) }, "var: attrs"},
		{func(n *Node) { n.Children[0].Text = "M" }, "var/labl[0]: text"},
		{func(n *Node) { n.Children[1].Raw = "s" }, "var/txt[1]: raw"},
		{func(n *Node) { n.Children = n.Children[:1] }, "var: children"},
		{func(n *Node) { n.Children[0], n.Children[1] = n.Children[1], n.Children[0] }, "var/labl[0]: name"},
	} {
		m := clone(a)
		c.edit(m)
		if d := Diff(a, m); !strings.HasPrefix(d, c.want) || Equal(a, m) {
			t.Errorf("Diff = %q, want prefix %q", d, c.want)
		}
	}
}

func clone(n *Node) *Node {
	c := *n
	c.Attrs = append([]Attr(nil), n.Attrs...)
	c.Children = nil
	for _, ch := range n.Children {
		c.Children = append(c.Children, clone(ch))
	}
	return &c
}

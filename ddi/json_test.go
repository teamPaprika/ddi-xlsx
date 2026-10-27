package ddi

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

// runNode runs a Node.js script with stdin and returns what it prints. It skips
// when node is not installed.
func runNode(t *testing.T, script string, stdin []byte) []byte {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	cmd := exec.Command("node", "-e", script)
	cmd.Stdin = bytes.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node: %v", err)
	}
	return out
}

func mustParseJSON(t *testing.T, data []byte) *Node {
	t.Helper()
	root, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	return root
}

func TestJSONShape(t *testing.T) {
	root := mustParse(t, doc(`<var ID="V1" name="x"><labl>x</labl><location fileid="F1"/><catValu>x</catValu>`+
		`<IDNo agency="w">f</IDNo><txt>a <emph>b</emph></txt><anlyInfo/></var><catValu>1</catValu><catValu>2</catValu>`))
	// catValu is not repeatable, but two of them still need an array to survive.
	want := `{
  "codeBook": {
    "@xmlns": "ddi:codebook:2_5",
    "var": [
      {
        "@ID": "V1",
        "@name": "x",
        "labl": [
          "x"
        ],
        "location": [
          {
            "@fileid": "F1"
          }
        ],
        "catValu": "x",
        "IDNo": [
          {
            "@agency": "w",
            "#text": "f"
          }
        ],
        "txt": [
          {
            "#xml": "a <emph>b</emph>"
          }
        ],
        "anlyInfo": ""
      }
    ],
    "catValu": [
      "1",
      "2"
    ]
  }
}
`
	if got := string(WriteJSON(root)); got != want {
		t.Errorf("WriteJSON:\n%s\nwant:\n%s", got, want)
	}
}

func TestJSONStringEscapesMatchNode(t *testing.T) {
	// U+2028 and U+2029 stay unescaped, as JSON.stringify leaves them.
	const s = "<>&    \t\n\r\"\\/é😀"
	got := string(WriteJSON(&Node{Name: "codeBook", Children: []*Node{{Name: "titl", Text: s}}}))
	want := string(runNode(t, `process.stdout.write(JSON.stringify(require("fs").readFileSync(0, "utf8")))`, []byte(s)))
	if !strings.Contains(got, want) {
		t.Errorf("WriteJSON = %s, want it to contain %s", got, want)
	}

	// XML cannot hold U+0001, so JSON drops it too and both carry the same text.
	got = string(WriteJSON(&Node{Name: "codeBook", Children: []*Node{{Name: "titl", Text: "a\x01b"}}}))
	if strings.Contains(got, `\u0001`) || strings.Contains(got, "\x01") || !strings.Contains(got, `"ab"`) {
		t.Errorf("WriteJSON kept U+0001: %q", got)
	}
}

// TestJSONLayoutMatchesNode checks the whole layout, not just strings, against
// JSON.stringify(x, null, 2).
func TestJSONLayoutMatchesNode(t *testing.T) {
	for _, name := range kofacNames {
		j := WriteJSON(mustParse(t, readKofac(t, name)))
		want := runNode(t, `const s = require("fs").readFileSync(0, "utf8"); process.stdout.write(JSON.stringify(JSON.parse(s), null, 2) + "\n")`, j)
		if !bytes.Equal(j, want) {
			t.Errorf("%s: WriteJSON differs from JSON.stringify", name)
		}
	}
}

func TestWallaJSONMatchesWallaXML(t *testing.T) {
	for _, name := range kofacNames {
		data, err := os.ReadFile(filepath.Join(xsdtest.Root(), "testdata", name, "survey.json"))
		if err != nil {
			t.Fatal(err)
		}
		if d := Diff(mustParseJSON(t, data), mustParse(t, readKofac(t, name))); d != "" {
			t.Errorf("%s: %s", name, d)
		}
	}
}

func TestJSONFixedPoint(t *testing.T) {
	inputs := map[string][]byte{
		"edge cases": []byte(`<codeBook xmlns="ddi:codebook:2_5" xmlns:xhtml="http://www.w3.org/1999/xhtml" xmlns:ddi="ddi:codebook:2_5" xml:lang="ko" a="t&#9;n&#10;r&#13;q&quot;">
  <stdyDscr><xhtml:p>x</xhtml:p><abstract>a <ddi:emph>b</ddi:emph> &amp; &lt;c&gt;</abstract><abstract><emph>a</emph> <emph>b</emph></abstract>
  <qstn><qstnLit>a</qstnLit><ivuInstr>i</ivuInstr><qstnLit>b</qstnLit></qstn><catValu/><catValu>  01
     x "y" \ é</catValu><labl xml:lang="en"/></stdyDscr></codeBook>`),
	}
	for _, name := range kofacNames {
		inputs[name] = readKofac(t, name)
	}
	for name, x := range inputs {
		t1 := mustParse(t, x)
		j1 := WriteJSON(t1)
		t2 := mustParseJSON(t, j1)
		if d := Diff(t1, t2); d != "" {
			t.Fatalf("%s: tree changed after writing JSON: %s\n%s", name, d, j1)
		}
		if j2 := WriteJSON(t2); !bytes.Equal(j2, j1) {
			t.Errorf("%s: second write differs from the first:\n%s\n%s", name, j1, j2)
		}
	}
}

func TestJSONRawIsReclassified(t *testing.T) {
	got := mustParseJSON(t, []byte(`{"codeBook":{"@xmlns":"ddi:codebook:2_5","stdyDscr":{"stdyInfo":[{"abstract":[{"#xml":"<emph>a</emph>"}]}]}}}`))
	want := mustParse(t, doc(`<stdyDscr><stdyInfo><abstract><emph>a</emph></abstract></stdyInfo></stdyDscr>`))
	if d := Diff(got, want); d != "" {
		t.Fatal(d)
	}
	if a := got.Children[0].Children[0].Children[0]; a.Raw != "" || len(a.Children) != 1 || a.Children[0].Name != "emph" {
		t.Errorf("abstract = %s", dump(a))
	}

	// A prefix the root declares resolves inside #xml, as it does in XML.
	got = mustParseJSON(t, []byte(`{"codeBook":{"@xmlns":"ddi:codebook:2_5","@xmlns:d":"ddi:codebook:2_5","abstract":{"#xml":"a <d:emph>b</d:emph>"},"d:titl":"y"}}`))
	if a := got.Children[0]; a.Raw != "a <d:emph>b</d:emph>" {
		t.Errorf("abstract = %s", dump(a))
	}
	if n := got.Children[1]; n.Name != "titl" {
		t.Errorf("prefixed DDI name = %s", dump(n))
	}
}

func TestJSONRootNamespace(t *testing.T) {
	for in, want := range map[string]string{
		`{"codeBook":{}}`: `not a DDI Codebook 2.5 document (namespace "")`,
		`{"codeBook":""}`: `not a DDI Codebook 2.5 document (namespace "")`,
		`{"codeBook":{"@xmlns":"http://www.icpsr.umich.edu/DDI"}}`: "DDI 1.2.2 is not supported; convert the file to DDI 2.5 first",
		`{"codeBook":{"@xmlns":"urn:x"}}`:                          `not a DDI Codebook 2.5 document (namespace "urn:x")`,
	} {
		if _, err := ParseJSON([]byte(in)); err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", in, err, want)
		}
	}
}

func TestJSONRejects(t *testing.T) {
	const ns = `"@xmlns":"ddi:codebook:2_5"`
	for _, in := range []string{
		``,
		`[]`,
		`{}`,
		`{"stdyDscr":{` + ns + `}}`,
		`{"codeBook":{` + ns + `},"x":""}`,
		`{"codeBook":[{` + ns + `},{` + ns + `}]}`,
		`{"codeBook":{` + ns + `}} {}`,
		`{"codeBook":{` + ns + `,"labl":null}}`,
		`{"codeBook":{` + ns + `,"labl":1}}`,
		`{"codeBook":{` + ns + `,"labl":[["x"]]}}`,
		`{"codeBook":{` + ns + `,"labl":"a","labl":"b"}}`,
		`{"codeBook":{` + ns + `,"labl":{"@a":"1","@a":"2"}}}`,
		`{"codeBook":{` + ns + `,"labl":{"#text":"a","emph":"b"}}}`,
		`{"codeBook":{` + ns + `,"labl":{"#xml":"a","emph":"b"}}}`,
		`{"codeBook":{` + ns + `,"labl":{"#xml":"a","#text":"b"}}}`,
		`{"codeBook":{` + ns + `,"labl":{"#xml":"</labl><labl>"}}}`,
		`{"codeBook":{` + ns + `,"labl":{"#xml":"<emph>"}}}`,
		`{"codeBook":{` + ns + `,"labl x=\"1\"":""}}`,
		`{"codeBook":{` + ns + `,"labl":{"@a=\"1\" b":"2"}}}`,
		`{"codeBook":{` + ns + `,"a b":""}}`,
		`{"codeBook":{` + ns + `,"":""}}`,
		`{"codeBook":{` + ns + `,"#comment":""}}`,
	} {
		if root, err := ParseJSON([]byte(in)); err == nil {
			t.Errorf("%s: want error, got %s", in, dump(root))
		}
	}
}

func TestJSONDepthLimit(t *testing.T) {
	// nested gives codeBook n levels of {"a": …} below it.
	nested := func(n int) []byte {
		return []byte(`{"codeBook":{"@xmlns":"ddi:codebook:2_5",` + strings.Repeat(`"a":{`, n) + strings.Repeat("}", n) + "}}")
	}
	if _, err := ParseJSON(nested(200)); err != nil {
		t.Errorf("depth 200: %v", err)
	}
	if _, err := ParseJSON(nested(300)); err == nil || !strings.Contains(err.Error(), "a: nested more than 256 elements deep") {
		t.Errorf("depth 300: err = %v", err)
	}
}

func TestParseDetectsFormat(t *testing.T) {
	for _, in := range []string{
		"\xef\xbb\xbf \n" + `{"codeBook":{"@xmlns":"ddi:codebook:2_5","titl":"x"}}`,
		"\xef\xbb\xbf \n" + `<codeBook xmlns="ddi:codebook:2_5"><titl>x</titl></codeBook>`,
	} {
		root, _, err := Parse([]byte(in))
		if err != nil || len(root.Children) != 1 || root.Children[0].Text != "x" {
			t.Errorf("%q: %v", in, err)
		}
	}
	for _, in := range []string{"", "\xef\xbb\xbf", " codeBook", "[]"} {
		if _, _, err := Parse([]byte(in)); err == nil || err.Error() != "input is neither XML nor JSON" {
			t.Errorf("%q: err = %v", in, err)
		}
	}
}

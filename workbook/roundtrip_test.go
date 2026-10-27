package workbook

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

var update = flag.Bool("update", false, "rewrite testdata/kofac_*/canonical.{xml,json}")

func readFile(t *testing.T, elem ...string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{xsdtest.Root(), "testdata"}, elem...)...))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// convert runs in through ToXLSX and FromXLSX.
func convert(t *testing.T, in []byte) (xo, jo []byte, w []string) {
	t.Helper()
	x, _, err := ToXLSX(in)
	if err != nil {
		t.Fatal(err)
	}
	xo, jo, w, err = FromXLSX(x)
	if err != nil {
		t.Fatal(err)
	}
	return xo, jo, w
}

func parse(t *testing.T, in []byte) *ddi.Node {
	t.Helper()
	root, _, err := ddi.Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// roundTrip converts in and checks what holds for every input: the output XML
// has in's tree, the JSON has the same tree as the XML, and converting either
// output again gives both outputs back byte for byte.
func roundTrip(t *testing.T, in []byte) (xo, jo []byte, w []string) {
	t.Helper()
	xo, jo, w = convert(t, in)
	got := parse(t, xo)
	if d := ddi.Diff(parse(t, in), got); d != "" {
		t.Errorf("input vs output XML: %s", d)
	}
	gotJSON, err := ddi.ParseJSON(jo)
	if err != nil {
		t.Fatal(err)
	}
	if d := ddi.Diff(got, gotJSON); d != "" {
		t.Errorf("output XML vs JSON: %s", d)
	}
	for _, out := range [][]byte{xo, jo} {
		xo2, jo2, _ := convert(t, out)
		if !bytes.Equal(xo2, xo) || !bytes.Equal(jo2, jo) {
			t.Errorf("converting the output again changes it (from %.1s…)", out)
		}
	}
	return xo, jo, w
}

func findVar(t *testing.T, root *ddi.Node, id string) *ddi.Node {
	t.Helper()
	for _, dd := range root.Children {
		if dd.Name != "dataDscr" {
			continue
		}
		for _, v := range dd.Children {
			if got, _ := v.Attr("ID"); v.Name == "var" && got == id {
				return v
			}
		}
	}
	t.Fatalf("no var %s", id)
	return nil
}

func names(nodes []*ddi.Node) []string {
	var out []string
	for _, n := range nodes {
		out = append(out, n.Name)
	}
	return out
}

func TestRoundTripKofac(t *testing.T) {
	for _, name := range []string{"kofac_2024_public", "kofac_2025_edu"} {
		for _, input := range []string{"survey.xml", "survey.json"} {
			t.Run(name+"/"+input, func(t *testing.T) {
				xo, jo, w := roundTrip(t, readFile(t, name, input))
				if len(w) > 0 {
					t.Errorf("warnings: %q", w)
				}
				xsdtest.Validate(t, xo)
				golden := filepath.Join(xsdtest.Root(), "testdata", name, "canonical")
				// Only the XML input writes the goldens, so the JSON input is still
				// compared against them.
				if *update && input == "survey.xml" {
					if err := os.WriteFile(golden+".xml", xo, 0o644); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(golden+".json", jo, 0o644); err != nil {
						t.Fatal(err)
					}
				}
				for ext, got := range map[string][]byte{".xml": xo, ".json": jo} {
					if want := readFile(t, name, "canonical"+ext); !bytes.Equal(got, want) {
						t.Errorf("output differs from canonical%s", ext)
					}
				}
			})
		}
	}
}

func TestRoundTripArchive(t *testing.T) {
	in := readFile(t, "archive", "nada_like.xml")
	_, warns, err := ToXLSX(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "comment") {
		t.Errorf("ToXLSX warnings = %q, want one about the comment", warns)
	}
	xo, _, w := roundTrip(t, in)
	if len(w) > 0 {
		t.Errorf("FromXLSX warnings: %q", w)
	}
	xsdtest.Validate(t, xo)
}

func TestMisorderedComesBackInSchemaOrder(t *testing.T) {
	xo, _, _ := convert(t, readFile(t, "archive", "misordered.xml"))
	v := findVar(t, parse(t, xo), "V1")
	if got, want := names(v.Children), []string{"labl", "catgry", "varFormat", "notes"}; !slices.Equal(got, want) {
		t.Errorf("var children = %q, want %q", got, want)
	}
}

// An element the schema does not place keeps the place of the one before it in
// its own cell or outline, and a second conversion leaves the order alone.
func TestChildOrderIsAFixedPoint(t *testing.T) {
	for _, tc := range []struct {
		dataDscr string
		inVar    bool
		want     []string
	}{
		{`<var ID="V1"><location/><foo/><labl>a</labl></var>`, true, []string{"location", "foo", "labl"}},
		{`<var ID="V1"><foo/><location/><labl>a</labl></var>`, true, []string{"foo", "location", "labl"}},
		{`<foo/><nCubeGrp/><var ID="V1"/>`, false, []string{"foo", "nCubeGrp", "var"}},
		{`<var ID="V1"><sumStat type="vald">1</sumStat><location/></var>`, true, []string{"location", "sumStat"}},
	} {
		xo, _, _ := convert(t, codebook(tc.dataDscr))
		root := parse(t, xo)
		n := root.Children[0]
		if tc.inVar {
			n = findVar(t, root, "V1")
		}
		if got := names(n.Children); !slices.Equal(got, tc.want) {
			t.Errorf("%s: %s children = %q, want %q", tc.dataDscr, n.Name, got, tc.want)
		}
		if xo2, _, _ := convert(t, xo); !bytes.Equal(xo2, xo) {
			t.Errorf("%s: second conversion changes the output:\n%s\n%s", tc.dataDscr, xo, xo2)
		}
	}
}

func TestPrefixedDDIInXMLCells(t *testing.T) {
	// The qstn is kept as written only because ddi:qstnLit and qstnLit are one
	// name, which the xml cell can tell only with the root's xmlns:ddi.
	roundTrip(t, []byte(`<ddi:codeBook xmlns:ddi="ddi:codebook:2_5"><ddi:dataDscr><ddi:var ID="V1"><ddi:qstn>`+
		"<ddi:qstnLit>a</ddi:qstnLit>\n<ddi:ivuInstr>i</ddi:ivuInstr>\n<qstnLit xmlns=\"ddi:codebook:2_5\">b</qstnLit>"+
		`</ddi:qstn></ddi:var></ddi:dataDscr></ddi:codeBook>`))
}

func TestNastyStrings(t *testing.T) {
	const nasty = `A & B <c> "d" 'e' 😀 한글`
	text := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(nasty)
	attr := strings.ReplaceAll(text, `"`, "&quot;")
	in := []byte(`<codeBook xmlns="ddi:codebook:2_5"><stdyDscr><citation><titlStmt><titl>` + text +
		`</titl></titlStmt></citation><stdyInfo><abstract>` + text + `<emph>x</emph></abstract></stdyInfo></stdyDscr>` +
		`<dataDscr><var ID="V1" name="v1"><labl>` + text + `</labl><catgry missType="` + attr + `"><catValu>1</catValu><labl>` +
		text + `</labl></catgry><notes type="x">` + text + `</notes></var></dataDscr></codeBook>`)
	if got := findVar(t, parse(t, in), "V1").Children[0].Text; got != nasty {
		t.Fatalf("input labl = %q", got)
	}
	xo, _, _ := roundTrip(t, in)
	xsdtest.Validate(t, xo)
	roundTrip(t, ddi.WriteJSON(parse(t, in)))
}

func TestLargeCodebook(t *testing.T) {
	if testing.Short() {
		t.Skip("large codebook")
	}
	var b strings.Builder
	b.WriteString(`<codeBook xmlns="ddi:codebook:2_5"><stdyDscr><citation><titlStmt><titl>Large</titl></titlStmt></citation></stdyDscr><dataDscr>`)
	for v := 1; v <= 3358; v++ {
		fmt.Fprintf(&b, `<var ID="V%d" name="v%d" files="F1"><location fileid="F1"/><labl>Variable %d</labl>`, v, v, v)
		for _, s := range []string{"vald", "invd", "min", "max"} {
			fmt.Fprintf(&b, `<sumStat type="%s">%d</sumStat>`, s, v)
		}
		for c := 1; c <= 10; c++ {
			fmt.Fprintf(&b, `<catgry><catValu>%d</catValu><labl>Category %d</labl><catStat type="freq">%d</catStat></catgry>`, c, c, v*c)
		}
		b.WriteString(`<varFormat type="numeric"/></var>`)
	}
	b.WriteString(`</dataDscr></codeBook>`)
	in := []byte(b.String())
	start := time.Now()
	x, _, err := ToXLSX(in)
	if err != nil {
		t.Fatal(err)
	}
	mid := time.Now()
	xo, _, _, err := FromXLSX(x)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	t.Logf("ToXLSX %v, FromXLSX %v", mid.Sub(start), time.Since(mid))
	if elapsed > 30*time.Second {
		t.Errorf("round trip took %v, want under 30s", elapsed)
	}
	if d := ddi.Diff(parse(t, in), parse(t, xo)); d != "" {
		t.Error(d)
	}
}

// var, catgry, varGrp and dataDscr hold only elements, and the reader re-sorts
// their children, so an order or a spacing that would keep one as written is
// split into rows and cells instead.
func TestMergePointParentsAreSplit(t *testing.T) {
	t.Run("varGrp after the vars", func(t *testing.T) {
		root := parse(t, readFile(t, "kofac_2025_edu", "survey.xml"))
		d := slices.IndexFunc(root.Children, func(n *ddi.Node) bool { return n.Name == "dataDscr" })
		dd := root.Children[d]
		i := slices.IndexFunc(dd.Children, func(n *ddi.Node) bool { return n.Name == "var" }) - 1
		if i < 0 || dd.Children[i].Name != "varGrp" {
			t.Fatalf("dataDscr children = %q, want a varGrp right before the first var", names(dd.Children))
		}
		g := dd.Children[i]
		dd.Children = append(slices.Delete(dd.Children, i, i+1), g)
		in := ddi.WriteXML(root)
		if parse(t, in).Children[d].Raw == "" {
			t.Fatal("moving the varGrp did not keep dataDscr as written")
		}
		xo, jo, _ := convert(t, in)
		for ext, got := range map[string][]byte{".xml": xo, ".json": jo} {
			if want := readFile(t, "kofac_2025_edu", "canonical"+ext); !bytes.Equal(got, want) {
				t.Errorf("output differs from canonical%s", ext)
			}
		}
		xsdtest.Validate(t, xo)
	})
	t.Run("labl, qstn, labl", func(t *testing.T) {
		in := codebook(`<var ID="V1"><labl>a</labl><qstn><qstnLit>q</qstnLit></qstn><labl level="x">b</labl></var>`)
		if findVar(t, parse(t, in), "V1").Raw == "" {
			t.Fatal("the input var is not kept as written")
		}
		xo, _, _ := convert(t, in)
		if got, want := names(findVar(t, parse(t, xo), "V1").Children), []string{"labl", "labl", "qstn"}; !slices.Equal(got, want) {
			t.Errorf("var children = %q, want %q", got, want)
		}
		if xo2, _, _ := convert(t, xo); !bytes.Equal(xo2, xo) {
			t.Errorf("second conversion changes the output:\n%s\n%s", xo, xo2)
		}
	})
	t.Run("a comment between children", func(t *testing.T) {
		_, w, err := ToXLSX(codebook(`<varGrp ID="G1"/><var ID="V1"/><!-- c --><varGrp ID="G2"/>`))
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"dataDscr: comment dropped at line 1"}; !slices.Equal(w, want) {
			t.Errorf("warnings = %q, want %q", w, want)
		}
	})
}

// Text between the children of a var, catgry, varGrp or dataDscr is content,
// so that element stays as written in one cell.
func TestMixedTextAtMergePointsIsKept(t *testing.T) {
	for _, dataDscr := range []string{
		`x<var ID="V1"/>`,
		`<var ID="V1"><labl>a</labl>x<notes>n</notes></var>`,
		`<var ID="V1"/><var ID="V2"><catgry>x<catValu>1</catValu></catgry></var>`,
		`<varGrp ID="G1"><labl>a</labl>x</varGrp>`,
	} {
		t.Run(dataDscr, func(t *testing.T) { roundTrip(t, codebook(dataDscr)) })
	}
}

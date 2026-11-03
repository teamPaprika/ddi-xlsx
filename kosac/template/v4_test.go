package template

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/workbook"
)

func v4Sample(t *testing.T, year string) []byte {
	t.Helper()
	b, e := os.ReadFile(filepath.Join("..", "..", "testdata", "kosac", "v40_sample_"+year+".xlsx"))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func compareV4Records(t *testing.T, a, b *kosac.Codebook) {
	t.Helper()
	if !maps.Equal(values(a.Study), values(b.Study)) {
		t.Fatal("study values changed")
	}
	ar, br := []kosac.Record{}, []kosac.Record{}
	for _, f := range a.Files {
		ar = append(ar, f.Rec)
	}
	for _, f := range b.Files {
		br = append(br, f.Rec)
	}
	for _, v := range a.Vars {
		ar = append(ar, v.Rec)
	}
	for _, v := range b.Vars {
		br = append(br, v.Rec)
	}
	for _, c := range a.Codes {
		ar = append(ar, c.Rec)
	}
	for _, c := range b.Codes {
		br = append(br, c.Rec)
	}
	ar = append(ar, a.Recodes...)
	br = append(br, b.Recodes...)
	if len(ar) != len(br) {
		t.Fatalf("records %d -> %d", len(ar), len(br))
	}
	for i := range ar {
		if !maps.Equal(values(ar[i]), values(br[i])) {
			t.Fatalf("record %d changed: %v -> %v", i, values(ar[i]), values(br[i]))
		}
	}
}
func TestV4OfficialTemplates(t *testing.T) {
	layout := kosac.LayoutFor("4.0")
	if len(layout.Study)+len(layout.Files)+len(layout.Vars)+len(layout.Codes)+len(kosac.RecodeKeys) != 105 {
		t.Fatal("105 fields not covered")
	}
	for _, tc := range []struct {
		year                        string
		files, vars, codes, recodes int
	}{{"2024", 1, 221, 935, 40}, {"2025", 2, 404, 1659, 50}} {
		t.Run(tc.year, func(t *testing.T) {
			cb, is := read(t, v4Sample(t, tc.year), ReadOptions{})
			if cb == nil || len(is) != 0 {
				t.Fatalf("read: %v", is)
			}
			if len(cb.Files) != tc.files || len(cb.Vars) != tc.vars || len(cb.Codes) != tc.codes || len(cb.Recodes) != tc.recodes {
				t.Fatalf("counts %d %d %d %d", len(cb.Files), len(cb.Vars), len(cb.Codes), len(cb.Recodes))
			}
			if cb.Vars[0].Rec["VAR_NAME"].Loc.Cell != "B3" || cb.Recodes[0]["FILE_NO"].Loc.Cell != "A3" {
				t.Fatal("first data row skipped")
			}
			x, e := Write(cb)
			if e != nil {
				t.Fatal(e)
			}
			again, is := read(t, x, ReadOptions{})
			if len(is) > 0 || again == nil {
				t.Fatalf("reread %v", is)
			}
			compareV4Records(t, cb, again)
		})
	}
}
func TestV4BlankAndDowngrade(t *testing.T) {
	b := Blank()
	if !bytes.Equal(b, blankV4) {
		t.Fatal("blank must be official v4")
	}
	b[0] ^= 255
	if Blank()[0] == b[0] {
		t.Fatal("shared bytes")
	}
	cb, is := read(t, Blank(), ReadOptions{})
	if cb == nil || len(is) > 0 || cb.Template.Version.Value != "4.0" {
		t.Fatalf("v4 blank: %v", is)
	}
	cb.Study.Set("SERIES_NAME", "example")
	cb.Template.Version.Value = "3.2"
	if _, e := Write(cb); e == nil {
		t.Fatal("silent v4 downgrade")
	}
}
func TestV4StructureAndTextCodes(t *testing.T) {
	for _, sheet := range []string{kosac.SheetRecodes, kosac.SheetMeta} {
		t.Run(sheet, func(t *testing.T) {
			f := open(t, v4Sample(t, "2024"))
			if e := f.DeleteSheet(sheet); e != nil {
				t.Fatal(e)
			}
			cb, is := read(t, save(t, f), ReadOptions{})
			if cb != nil || len(is) == 0 {
				t.Fatal("missing sheet accepted")
			}
		})
	}
	f := open(t, v4Sample(t, "2024"))
	if e := f.SetCellStr(kosac.SheetCodes, "C3", "01"); e != nil {
		t.Fatal(e)
	}
	cb, is := read(t, save(t, f), ReadOptions{})
	if cb == nil || len(is) > 0 {
		t.Fatal(is)
	}
	out, e := Write(cb)
	if e != nil {
		t.Fatal(e)
	}
	again, is := read(t, out, ReadOptions{})
	if again == nil || len(is) > 0 || again.Codes[0].Rec.Get("CODE") != "01" {
		t.Fatal("leading zero lost")
	}
}
func TestV4StandardXMLExamples(t *testing.T) {
	for _, year := range []string{"2024", "2025"} {
		t.Run(year, func(t *testing.T) {
			cb, is := read(t, v4Sample(t, year), ReadOptions{})
			if cb == nil || len(is) > 0 {
				t.Fatal(is)
			}
			kosac.DeriveDefaults(cb, kosac.RefData{})
			if err := kosac.AssignIDs(cb, nil); err != nil {
				t.Fatal(err)
			}
			before, err := json.Marshal(cb)
			if err != nil {
				t.Fatal(err)
			}
			source, err := cb.XML()
			if err != nil {
				t.Fatal(err)
			}
			decoder := xml.NewDecoder(bytes.NewReader(source))
			for {
				_, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			xsdtest.Validate(t, source)
			if strings.Contains(string(source), "kosac:") || strings.Contains(string(source), "KOFAC-SURVEY-META-v4.0") {
				t.Fatal("private metadata exported")
			}
			root, w, err := ddi.Parse(source)
			if err != nil || len(w) > 0 {
				t.Fatalf("parse %v %v", err, w)
			}
			var vars, codes, recodes int
			var outputVars []*ddi.Node
			kosac.Walk(root, func(n, _ *ddi.Node, _ string) bool {
				switch n.Name {
				case "var":
					vars++
					outputVars = append(outputVars, n)
				case "catgry":
					codes++
				case "codingInstructions":
					recodes++
				}
				return true
			})
			if vars != len(cb.Vars) || codes != len(cb.Codes) || recodes != len(cb.Recodes) {
				t.Fatalf("output counts: %d %d %d", vars, codes, recodes)
			}
			jsonOut, err := cb.JSON()
			if err != nil {
				t.Fatal(err)
			}
			back, err := ddi.ParseJSON(jsonOut)
			if err != nil {
				t.Fatal(err)
			}
			if d := ddi.Diff(root, back); d != "" {
				t.Fatal(d)
			}
			if !bytes.Equal(source, ddi.WriteXML(back)) {
				t.Fatal("XML/JSON roundtrip differs")
			}
			after, err := json.Marshal(cb)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("export modified source model", err)
			}
			verifyV4Variables(t, cb, outputVars)
			editable, warns, err := workbook.ToXLSX(source)
			if err != nil || len(warns) > 0 {
				t.Fatalf("DDI workbook: %v %v", err, warns)
			}
			xmlBack, jsonBack, warns, err := workbook.FromXLSX(editable)
			if err != nil || len(warns) > 0 {
				t.Fatalf("DDI workbook roundtrip: %v %v", err, warns)
			}
			// Workbook columns normalize attribute order; content and child order stay exact.
			roundRoot, _, err := ddi.Parse(xmlBack)
			if err != nil {
				t.Fatal(err)
			}
			if difference := ddi.Diff(root, roundRoot); difference != "" {
				t.Fatal(difference)
			}
			jsonRoot, err := ddi.ParseJSON(jsonBack)
			if err != nil {
				t.Fatal(err)
			}
			if difference := ddi.Diff(root, jsonRoot); difference != "" {
				t.Fatal(difference)
			}
			xsdtest.Validate(t, xmlBack)

			x, err := Write(cb)
			if err != nil {
				t.Fatal(err)
			}
			restored, is := read(t, x, ReadOptions{})
			if restored == nil || len(is) > 0 {
				t.Fatal(is)
			}
			compareV4Records(t, cb, restored)
		})
	}
}
func TestLegacyV4MetadataRead(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "kosac", "legacy", "v40_profile.xml"))
	if err != nil {
		t.Fatal(err)
	}
	cb, err := kosac.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Template.Version.Value != "4.0" || len(cb.Recodes) != 1 || cb.Vars[0].Rec.Get("SOURCE_VAR") != "raw_id" {
		t.Fatal("legacy source fields lost")
	}
	out, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	xsdtest.Validate(t, out)
	if strings.Contains(string(out), "kosac:") {
		t.Fatal("legacy import wrote an extension")
	}
	for _, tc := range []string{"json", "reference", "duplicate"} {
		t.Run(tc, func(t *testing.T) {
			root, _, err := ddi.Parse(source)
			if err != nil {
				t.Fatal(err)
			}
			var study, note *ddi.Node
			for _, n := range root.Children {
				if n.Name == "stdyDscr" {
					study = n
				}
			}
			for _, n := range study.Children {
				subject, _ := n.Attr("subject")
				if n.Name == "notes" && subject == kosac.SheetFiles {
					note = n
					break
				}
			}
			if note == nil {
				t.Fatal("fixture missing legacy metadata")
			}
			switch tc {
			case "json":
				note.Text = "{bad json"
			case "reference":
				note.Text = strings.Replace(note.Text, `"fileNo":"1"`, `"fileNo":"999"`, 1)
			case "duplicate":
				study.Children = append(study.Children, note)
			}
			if _, err := kosac.Parse(ddi.WriteXML(root)); err == nil {
				t.Fatal("damaged legacy metadata accepted")
			}
		})
	}
}

// Verify actual values, order and missing flags, not only record counts.
func verifyV4Variables(t *testing.T, cb *kosac.Codebook, out []*ddi.Node) {
	t.Helper()
	attr := func(n *ddi.Node, key string) string { v, _ := n.Attr(key); return v }
	children := func(n *ddi.Node, tag string) []*ddi.Node {
		var out []*ddi.Node
		for _, c := range n.Children {
			if c.Name == tag {
				out = append(out, c)
			}
		}
		return out
	}
	codes := map[[2]string][]*kosac.Code{}
	for _, c := range cb.Codes {
		k := [2]string{c.Rec.Get("FILE_NO"), c.Rec.Get("VAR_NAME")}
		codes[k] = append(codes[k], c)
	}
	for i, v := range cb.Vars {
		n := out[i]
		if attr(n, "ID") != v.ID || attr(n, "name") != v.Rec.Get("VAR_NAME") || attr(n, "files") != "F"+v.Rec.Get("FILE_NO") {
			t.Fatal("variable order/identity differs", i)
		}
		nature := map[string]string{"명목": "nominal", "서열": "ordinal", "등간": "interval", "비율": "ratio"}[v.Rec.Get("MEAS_LEVEL")]
		if attr(n, "nature") != nature {
			t.Fatal("nature differs", i)
		}
		cats := children(n, "catgry")
		source := codes[[2]string{v.Rec.Get("FILE_NO"), v.Rec.Get("VAR_NAME")}]
		if len(cats) != len(source) {
			t.Fatal("category count differs", i)
		}
		for j, c := range source {
			values := children(cats[j], "catValu")
			labels := children(cats[j], "labl")
			missing := "N"
			if kosac.DeclaredMissing(c.Rec.Get("IS_MISSING")) {
				missing = "Y"
			}
			if len(values) != 1 || values[0].Text != ddi.NormalizeText(c.Rec.Get("CODE")) || len(labels) != 1 || labels[0].Text != ddi.NormalizeText(c.Rec.Get("CODE_LABEL")) || attr(cats[j], "missing") != missing {
				t.Fatalf("category changed: %s/%s/%s", v.Rec.Get("FILE_NO"), v.Rec.Get("VAR_NAME"), c.Rec.Get("CODE"))
			}
		}
	}
}

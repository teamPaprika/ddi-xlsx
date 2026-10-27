package workbook

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"maps"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
	"github.com/xuri/excelize/v2"
)

// fromFile saves f and reads it back with FromXLSX.
func fromFile(t *testing.T, f *excelize.File) ([]byte, []string, error) {
	t.Helper()
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	xo, _, w, err := FromXLSX(buf.Bytes())
	return xo, w, err
}

// at returns the address of the cell in column header on Excel row row.
func at(t *testing.T, f *excelize.File, sheet, header string, row int) string {
	t.Helper()
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	c := slices.Index(rows[0], header)
	if c < 0 {
		t.Fatalf("%s has no column %s", sheet, header)
	}
	cell, _ := excelize.CoordinatesToCellName(c+1, row)
	return cell
}

// rowsWhere returns the Excel rows whose column header holds val.
func rowsWhere(t *testing.T, f *excelize.File, sheet, header, val string) []int {
	t.Helper()
	var out []int
	for i, r := range records(t, f, sheet) {
		if r[header] == val {
			out = append(out, i+2)
		}
	}
	if len(out) == 0 {
		t.Fatalf("no %s row with %s=%s", sheet, header, val)
	}
	return out
}

func set(t *testing.T, f *excelize.File, sheet, header string, row int, v string) {
	t.Helper()
	if err := f.SetCellStr(sheet, at(t, f, sheet, header, row), v); err != nil {
		t.Fatal(err)
	}
}

func TestHandEdit(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	set(t, f, varSheet, "label", rowsWhere(t, f, varSheet, "name", "Q8_r1")[0], "바뀐 라벨 & <ok>")
	set(t, f, varSheet, "xml", rowsWhere(t, f, varSheet, "ID", "V17")[0], `<location fileid="F1"/><notes>n</notes>`)

	cats := rowsWhere(t, f, catSheet, "varID", "V17")
	row := cats[len(cats)-1] + 1
	if err := f.InsertRows(catSheet, row, 1); err != nil {
		t.Fatal(err)
	}
	for h, v := range map[string]string{"varID": "V17", "code": "9", "label": "새 범주", "missing": "Y"} {
		set(t, f, catSheet, h, row, v)
	}

	// keyword sits in stdyDscr/stdyInfo/subject, and stdyInfo comes right after citation.
	doc := records(t, f, docSheet)
	type line struct {
		level         int
		element, text string
	}
	level := func(i int) int { l, _ := strconv.Atoi(doc[i]["level"]); return l }
	var add []line
	s := slices.IndexFunc(doc, func(r map[string]string) bool { return r["element"] == "stdyDscr" })
	pos := s + 1
	for i := s + 1; i < len(doc) && level(i) > 1; i++ {
		if doc[i]["element"] == "subject" {
			pos, add = i+1, []line{{level(i) + 1, "keyword", "과학"}}
			break
		}
	}
	if add == nil {
		if doc[pos]["element"] == "citation" {
			for pos++; pos < len(doc) && level(pos) > 2; pos++ {
			}
		}
		add = []line{{2, "stdyInfo", ""}, {3, "subject", ""}, {4, "keyword", "과학"}}
	}
	if err := f.InsertRows(docSheet, pos+2, len(add)); err != nil {
		t.Fatal(err)
	}
	for i, l := range add {
		set(t, f, docSheet, "level", pos+2+i, strconv.Itoa(l.level))
		set(t, f, docSheet, "element", pos+2+i, l.element)
		set(t, f, docSheet, "text", pos+2+i, l.text)
	}

	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	xsdtest.Validate(t, xo)
	for _, want := range []string{"<labl>바뀐 라벨 &amp; &lt;ok&gt;</labl>", "<keyword>과학</keyword>"} {
		if !strings.Contains(string(xo), want) {
			t.Errorf("output lacks %s", want)
		}
	}
	v := findVar(t, parse(t, xo), "V17")
	if !slices.ContainsFunc(v.Children, func(c *ddi.Node) bool {
		m, _ := c.Attr("missing")
		return c.Name == "catgry" && m == "Y" && len(c.Children) > 1 && c.Children[0].Name == "catValu" &&
			c.Children[0].Text == "9" && c.Children[1].Text == "새 범주"
	}) {
		t.Errorf("V17 has no <catgry missing=\"Y\"> with code 9")
	}
	got := names(v.Children)
	if f, n := slices.Index(got, "varFormat"), slices.Index(got, "notes"); f < 0 || n < f {
		t.Errorf("V17 children = %q, want notes after varFormat", got)
	}
}

func TestEditErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(t *testing.T, f *excelize.File) string // returns the edited cell
	}{
		{"unknown varID", func(t *testing.T, f *excelize.File) string {
			f.SetCellStr(catSheet, "A2", "V99999")
			return catSheet + "!A2"
		}},
		{"broken xml cell", func(t *testing.T, f *excelize.File) string {
			cell := at(t, f, varSheet, "xml", 2)
			f.SetCellStr(varSheet, cell, "<labl>")
			return varSheet + "!" + cell
		}},
		{"duplicate ID", func(t *testing.T, f *excelize.File) string {
			id, _ := f.GetCellValue(varSheet, "A2")
			f.SetCellStr(varSheet, "A3", id)
			return varSheet + "!A3"
		}},
		{"level jump", func(t *testing.T, f *excelize.File) string {
			doc := records(t, f, docSheet)
			i := slices.IndexFunc(doc[:len(doc)-1], func(r map[string]string) bool { return r["level"] == "1" })
			if doc[i+1]["level"] != "2" {
				t.Fatalf("row after %v is %v", doc[i], doc[i+1])
			}
			cell := at(t, f, docSheet, "level", i+3)
			f.SetCellStr(docSheet, cell, "3")
			return docSheet + "!" + cell
		}},
		{"no level-0 row", func(t *testing.T, f *excelize.File) string {
			f.SetCellStr(docSheet, "A2", "1")
			return docSheet + "!A2"
		}},
		{"broken attributes cell", func(t *testing.T, f *excelize.File) string {
			cell := at(t, f, varSheet, "attributes", 2)
			f.SetCellStr(varSheet, cell, `files="F1`)
			return varSheet + "!" + cell
		}},
		{"questionID without question", func(t *testing.T, f *excelize.File) string {
			row := rowsWhere(t, f, varSheet, "ID", "V17")[0]
			set(t, f, varSheet, "question", row, "")
			return varSheet + "!" + at(t, f, varSheet, "questionID", row)
		}},
		{"attribute in its column and the attributes cell", func(t *testing.T, f *excelize.File) string {
			cell := at(t, f, varSheet, "attributes", 2)
			f.SetCellStr(varSheet, cell, `name="other"`)
			return varSheet + "!" + cell
		}},
		{"markup in an element name", func(t *testing.T, f *excelize.File) string {
			row := rowsWhere(t, f, docSheet, "element", "titl")[0]
			set(t, f, docSheet, "element", row, `titl x="1"`)
			return docSheet + "!" + at(t, f, docSheet, "element", row)
		}},
		{"text-only xml cell beside column children", func(t *testing.T, f *excelize.File) string {
			row := rowsWhere(t, f, varSheet, "ID", "V17")[0]
			set(t, f, varSheet, "xml", row, "just text")
			return varSheet + "!" + at(t, f, varSheet, "xml", row)
		}},
		{"var ID reused as a group ID", func(t *testing.T, f *excelize.File) string {
			id, _ := f.GetCellValue(varSheet, "A2")
			f.SetCellStr(grpSheet, "A2", id)
			return grpSheet + "!A2"
		}},
		{"Variables without dataDscr", func(t *testing.T, f *excelize.File) string {
			f.RemoveRow(docSheet, rowsWhere(t, f, docSheet, "element", "dataDscr")[0])
			return varSheet + "!A2"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := kofac(t, "kofac_2025_edu")
			cell := tc.edit(t, f)
			_, _, err := fromFile(t, f)
			if err == nil || !strings.HasPrefix(err.Error(), cell+": ") {
				t.Errorf("err = %v, want it to start with %s: ", err, cell)
			}
		})
	}
}

func TestNumericAndCRLFCells(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	varID, _ := f.GetCellValue(catSheet, "A2")
	if err := f.SetCellInt(catSheet, at(t, f, catSheet, "code", 2), 7); err != nil {
		t.Fatal(err)
	}
	set(t, f, varSheet, "question", rowsWhere(t, f, varSheet, "ID", "V17")[0], "a\r\nb")
	// The trailing spaces of an inner line would not survive writing and reading the XML.
	set(t, f, varSheet, "notes", rowsWhere(t, f, varSheet, "ID", "V17")[0], "n1  \r\n\r\n  n2")
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<qstnLit>a\nb</qstnLit>", "<notes>n1</notes>\n      <notes>n2</notes>\n"} {
		if !strings.Contains(string(xo), want) {
			t.Errorf("output lacks %q", want)
		}
	}
	v := findVar(t, parse(t, xo), varID)
	i := slices.IndexFunc(v.Children, func(c *ddi.Node) bool { return c.Name == "catgry" })
	if c := v.Children[i].Children[0]; c.Name != "catValu" || c.Text != "7" {
		t.Errorf("%s first catgry starts with <%s>%s, want <catValu>7", varID, c.Name, c.Text)
	}
}

func TestUserColumnsAndBlankRows(t *testing.T) {
	in := readFile(t, "kofac_2025_edu", "survey.xml")
	f := open(t, in)
	rows, err := f.GetRows(varSheet)
	if err != nil {
		t.Fatal(err)
	}
	label, end := slices.Index(rows[0], "label"), len(rows[0])
	for r, row := range rows {
		if label < len(row) {
			cell, _ := excelize.CoordinatesToCellName(end+1, r+1)
			f.SetCellStr(varSheet, cell, row[label])
		}
	}
	col, _ := excelize.ColumnNumberToName(label + 1)
	if err := f.RemoveCol(varSheet, col); err != nil {
		t.Fatal(err)
	}
	mine, _ := excelize.ColumnNumberToName(end + 1)
	f.SetCellStr(varSheet, mine+"1", "my notes")
	f.SetCellStr(varSheet, mine+"3", "check this one")
	for _, sheet := range []string{varSheet, catSheet} {
		if err := f.InsertRows(sheet, 4, 1); err != nil {
			t.Fatal(err)
		}
	}
	xo, w, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if d := ddi.Diff(parse(t, in), parse(t, xo)); d != "" {
		t.Error(d)
	}
	if len(w) != 1 || !strings.Contains(w[0], "my notes") {
		t.Errorf("warnings = %q, want one about my notes", w)
	}
}

func TestPlacementWarnings(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	row := rowsWhere(t, f, varSheet, "ID", "V17")[0]
	set(t, f, varSheet, "xml", row, `<location fileid="F1"/><catValu>1</catValu>`)
	set(t, f, grpSheet, "vars", rowsWhere(t, f, grpSheet, "ID", "VG1")[0], "V17 V99999")
	dd := rowsWhere(t, f, docSheet, "element", "dataDscr")[0]
	if err := f.InsertRows(docSheet, dd+1, 1); err != nil {
		t.Fatal(err)
	}
	set(t, f, docSheet, "level", dd+1, "2")
	set(t, f, docSheet, "element", dd+1, "keyword")
	_, w, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		varSheet + "!" + at(t, f, varSheet, "xml", row) + ": <catValu>",
		grpSheet + "!" + at(t, f, grpSheet, "vars", rowsWhere(t, f, grpSheet, "ID", "VG1")[0]) + ": V99999",
		docSheet + "!" + at(t, f, docSheet, "element", dd+1) + ": <keyword>",
	}
	if len(w) != len(want) {
		t.Fatalf("warnings = %q, want %d", w, len(want))
	}
	for _, p := range want {
		if !slices.ContainsFunc(w, func(s string) bool { return strings.HasPrefix(s, p) }) {
			t.Errorf("no warning starting with %q in %q", p, w)
		}
	}
}

func TestNonContiguousSiblingsAreGrouped(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	// docDscr holds citation, notes, notes; a notes row before citation splits them.
	row := rowsWhere(t, f, docSheet, "element", "docDscr")[0] + 1
	if err := f.InsertRows(docSheet, row, 1); err != nil {
		t.Fatal(err)
	}
	set(t, f, docSheet, "level", row, "2")
	set(t, f, docSheet, "element", row, "notes")
	set(t, f, docSheet, "text", row, "first")
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	xo, jo, _, err := FromXLSX(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	root := parse(t, xo)
	gotJSON, err := ddi.ParseJSON(jo)
	if err != nil {
		t.Fatal(err)
	}
	if d := ddi.Diff(root, gotJSON); d != "" {
		t.Errorf("XML vs JSON: %s", d)
	}
	if got, want := names(root.Children[0].Children), []string{"notes", "notes", "notes", "citation"}; !slices.Equal(got, want) {
		t.Errorf("docDscr children = %q, want %q", got, want)
	}
}

func TestCategoryRowWithOnlyVarID(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	cats := rowsWhere(t, f, catSheet, "varID", "V17")
	row := cats[len(cats)-1] + 1
	if err := f.InsertRows(catSheet, row, 1); err != nil {
		t.Fatal(err)
	}
	set(t, f, catSheet, "varID", row, "V17")
	set(t, f, catSheet, "varName", row, "Q8_r1")
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	var got []*ddi.Node
	for _, c := range findVar(t, parse(t, xo), "V17").Children {
		if c.Name == "catgry" {
			got = append(got, c)
		}
	}
	if last := got[len(got)-1]; len(got) != len(cats)+1 || len(last.Attrs)+len(last.Children) > 0 || last.Text != "" {
		t.Errorf("V17 has %d catgry, the last %+v; want %d with an empty last one", len(got), last, len(cats)+1)
	}
}

// Without its table, a sheet's header row is row 1.
func TestHeaderRowMustBeFirst(t *testing.T) {
	for _, sheet := range []string{varSheet, catSheet, grpSheet, docSheet} {
		t.Run(sheet, func(t *testing.T) {
			f := kofac(t, "kofac_2025_edu")
			if err := f.DeleteTable("ddi" + sheet); err != nil {
				t.Fatal(err)
			}
			if err := f.InsertRows(sheet, 1, 1); err != nil {
				t.Fatal(err)
			}
			if err := f.SetCellStr(sheet, "A1", "KOFAC 2025 codebook"); err != nil {
				t.Fatal(err)
			}
			if _, _, err := fromFile(t, f); err == nil || !strings.HasPrefix(err.Error(), sheet+"!A1: ") {
				t.Errorf("err = %v, want it to start with %s!A1: ", err, sheet)
			}
		})
	}
	f := kofac(t, "kofac_2025_edu")
	col, _ := excelize.ColumnNumberToName(slices.Index(varHeaders, "xml") + 1)
	if err := f.RemoveCol(varSheet, col); err != nil {
		t.Fatal(err)
	}
	_, w, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(w, func(s string) bool { return strings.HasPrefix(s, varSheet+"!A1: ") && strings.Contains(s, "xml") }) {
		t.Errorf("warnings = %q, want one at %s!A1 about the xml column", w, varSheet)
	}
}

func TestDeclaredZipSizeIsBounded(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateRaw(&zip.FileHeader{Name: "docProps/app.xml", Method: zip.Store, CompressedSize64: 1, UncompressedSize64: 1 << 30})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, _, _, err = FromXLSX(buf.Bytes())
	runtime.ReadMemStats(&after)
	if err == nil {
		t.Error("FromXLSX accepted a zip entry that declares 1 GiB")
	}
	if n := after.TotalAlloc - before.TotalAlloc; n > 64<<20 {
		t.Errorf("FromXLSX allocated %d bytes for a %d-byte file", n, buf.Len())
	}
}

// deep is a codebook whose deepest element is at depth n, below stdyDscr.
func deep(n int) []byte {
	return []byte(`<codeBook xmlns="ddi:codebook:2_5"><stdyDscr>` + strings.Repeat("<a>", n-1) + strings.Repeat("</a>", n-1) +
		`</stdyDscr></codeBook>`)
}

func TestDocumentDepthLimit(t *testing.T) {
	roundTrip(t, deep(200))
	// Document rows sit at Excel row level+2, and each row may go only one level deeper.
	f := open(t, deep(200))
	for level := 201; level <= 300; level++ {
		set(t, f, docSheet, "level", level+2, strconv.Itoa(level))
		set(t, f, docSheet, "element", level+2, "a")
	}
	if _, _, err := fromFile(t, f); err == nil || err.Error() != docSheet+"!A259: level 257 is deeper than 256" {
		t.Errorf("err = %v, want %s!A259: level 257 is deeper than 256", err, docSheet)
	}
}

func TestSpacedXMLCellIsSplit(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	set(t, f, varSheet, "xml", rowsWhere(t, f, varSheet, "ID", "V17")[0], `<location fileid="F1"/> <sumStat type="vald">1</sumStat>`)
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	v := findVar(t, parse(t, xo), "V17")
	if got := names(v.Children); v.Raw != "" || !slices.Contains(got, "location") || !slices.Contains(got, "sumStat") {
		t.Errorf("V17 children = %q, Raw = %q; want location and sumStat as children", got, v.Raw)
	}
}

func TestCellWarnings(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	v17 := rowsWhere(t, f, varSheet, "ID", "V17")[0]
	set(t, f, varSheet, "xml", v17, `<location fileid="F1"/><!-- check --><y:notes>n</y:notes>`)
	titl := rowsWhere(t, f, docSheet, "element", "titl")[0]
	set(t, f, docSheet, "text", titl, "a<?pi x?>b")
	set(t, f, docSheet, "kind", titl, "xml")
	if err := f.InsertRows(docSheet, titl+1, 1); err != nil {
		t.Fatal(err)
	}
	level, _ := f.GetCellValue(docSheet, at(t, f, docSheet, "level", titl))
	set(t, f, docSheet, "level", titl+1, level)
	set(t, f, docSheet, "element", titl+1, "z:extra")
	_, w, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		varSheet + "!" + at(t, f, varSheet, "xml", v17) + ": comment dropped at line 1",
		varSheet + "!" + at(t, f, varSheet, "xml", v17) + `: prefix "y" of <y:notes> is not declared at line 1`,
		docSheet + "!" + at(t, f, docSheet, "text", titl) + ": processing instruction <?pi?> dropped at line 1",
		docSheet + "!" + at(t, f, docSheet, "element", titl+1) + `: prefix "z" of <z:extra> is not declared at line 1`,
	} {
		if !slices.Contains(w, want) {
			t.Errorf("no warning %q in %q", want, w)
		}
	}
}

func TestTitleRowAboveTable(t *testing.T) {
	in := readFile(t, "kofac_2025_edu", "survey.xml")
	f := open(t, in)
	for _, sheet := range f.GetSheetList() {
		if err := f.InsertRows(sheet, 1, 1); err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellStr(sheet, "A1", "KOFAC 2025 codebook"); err != nil {
			t.Fatal(err)
		}
		if tables, err := f.GetTables(sheet); err != nil || len(tables) != 1 || !strings.HasPrefix(tables[0].Range, "A2:") {
			t.Fatalf("%s tables = %+v (%v), want one starting at A2", sheet, tables, err)
		}
	}
	xo, w, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if !ddi.Equal(parse(t, in), parse(t, xo)) {
		t.Error(ddi.Diff(parse(t, in), parse(t, xo)))
	}
	if len(w) > 0 {
		t.Errorf("warnings = %q", w)
	}
}

func TestRowsTypedBelowTable(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	rows, err := f.GetRows(catSheet)
	if err != nil {
		t.Fatal(err)
	}
	row := len(rows) + 1
	tables, err := f.GetTables(catSheet)
	if err != nil {
		t.Fatal(err)
	}
	if end, _ := excelize.CoordinatesToCellName(len(catHeaders), row-1); len(tables) != 1 || tables[0].Range != "A1:"+end {
		t.Fatalf("Categories tables = %+v, want one ending at %s", tables, end)
	}
	set(t, f, catSheet, "varID", row, "V17")
	set(t, f, catSheet, "code", row, "9")
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	v := findVar(t, parse(t, xo), "V17")
	if !slices.ContainsFunc(v.Children, func(c *ddi.Node) bool {
		return c.Name == "catgry" && len(c.Children) == 1 && c.Children[0].Text == "9"
	}) {
		t.Error("V17 has no catgry with code 9")
	}
}

func TestNoTableFallsBackToRow1(t *testing.T) {
	in := readFile(t, "kofac_2025_edu", "survey.xml")
	f := open(t, in)
	for _, sheet := range f.GetSheetList() {
		if err := f.DeleteTable("ddi" + sheet); err != nil {
			t.Fatal(err)
		}
	}
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if d := ddi.Diff(parse(t, in), parse(t, xo)); d != "" {
		t.Error(d)
	}
}

// Text between child elements is content, so a var whose xml cell holds it
// comes back as written.
func TestMixedTextXMLCellIsKept(t *testing.T) {
	f := open(t, codebook(`<var ID="V1"/>`))
	const cell = `<labl>a</labl>x<notes>n</notes>`
	set(t, f, varSheet, "xml", 2, cell)
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if v := findVar(t, parse(t, xo), "V1"); v.Raw != cell {
		t.Errorf("V1 Raw = %q, want %q", v.Raw, cell)
	}
	set(t, f, varSheet, "label", 2, "l")
	want := varSheet + "!" + at(t, f, varSheet, "xml", 2) + ": <var> holds text between its child elements, " +
		"so this cell holds all its content, and its other columns and rows must be empty"
	if _, _, err := fromFile(t, f); err == nil || err.Error() != want {
		t.Errorf("err = %v, want %s", err, want)
	}
}

// An xml cell counts depth from its own element, so the reader checks the
// depth of the tree it builds; otherwise its output would be too deep to read.
func TestOutputDepthLimit(t *testing.T) {
	nest := func(n int) string { return strings.Repeat("<a>", n) + strings.Repeat("</a>", n) }
	f := open(t, codebook(`<var ID="V1"/>`))
	// var sits at depth 2, so its content may nest 254 deep.
	set(t, f, varSheet, "xml", 2, nest(254))
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	parse(t, xo)
	set(t, f, varSheet, "xml", 2, nest(255))
	want := "codeBook/dataDscr[1]/var[1]/a[1]/…/a[1]: elements nest more than 256 deep"
	if _, _, err := fromFile(t, f); err == nil || err.Error() != want {
		t.Errorf("err = %v, want %s", err, want)
	}
	// Text between the children keeps var as written, so its Raw is counted.
	set(t, f, varSheet, "xml", 2, "x"+nest(254))
	if xo, _, err = fromFile(t, f); err != nil {
		t.Fatal(err)
	}
	parse(t, xo)
	set(t, f, varSheet, "xml", 2, "x"+nest(255))
	want = "codeBook/dataDscr[1]/var[1]: elements nest more than 256 deep"
	if _, _, err := fromFile(t, f); err == nil || err.Error() != want {
		t.Errorf("err = %v, want %s", err, want)
	}
}

// An element kept as written is parsed again when it is split, and that parse
// must not repeat the warning the first one gave.
func TestPrefixWarningOnce(t *testing.T) {
	const content = `<labl>a</labl><y:notes>n</y:notes><labl>b</labl>`
	prefix := func(w []string) int {
		n := 0
		for _, s := range w {
			if strings.Contains(s, `prefix "y"`) {
				n++
			}
		}
		return n
	}
	x, w, err := ToXLSX(codebook(`<var ID="V1">` + content + `</var>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != 1 || prefix(w) != 1 {
		t.Errorf("ToXLSX warnings = %q, want one about prefix y", w)
	}
	f, err := excelize.OpenReader(bytes.NewReader(x))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	set(t, f, varSheet, "xml", 2, content)
	set(t, f, varSheet, "label", 2, "")
	if _, w, err = fromFile(t, f); err != nil {
		t.Fatal(err)
	}
	// <y:notes> is also not allowed in <var>, which is a different warning.
	if prefix(w) != 1 {
		t.Errorf("FromXLSX warnings = %q, want one about prefix y", w)
	}
}

// Table parts are read straight from the zip, and one larger than any real
// table part is skipped unread while the others still count.
func TestTablePartSizeIsBounded(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, part := range map[string]string{
		"xl/tables/table1.xml": `<table name="ddiVariables" ref="A3:M9"/>`,
		// Well-formed and named like a real table: only its 2 MiB size skips it.
		"xl/tables/table2.xml": `<table name="ddiCategories" ref="A5:H9"/>` + strings.Repeat(" ", 2<<20),
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	heads := tableHeads(buf.Bytes())
	runtime.ReadMemStats(&after)
	if want := map[string]tableRows{"ddiVariables": {head: 3}}; !maps.Equal(heads, want) {
		t.Errorf("tableHeads = %v, want %v", heads, want)
	}
	if n := after.TotalAlloc - before.TotalAlloc; n > 4<<20 {
		t.Errorf("tableHeads allocated %d bytes", n)
	}
}

// withTableRef returns the workbook x with the range of the table named name
// set to ref.
func withTableRef(t *testing.T, x []byte, name, ref string) []byte {
	t.Helper()
	return withTablePart(t, x, name, func(data []byte) []byte {
		return regexp.MustCompile(`ref="[^"]*"`).ReplaceAll(data, []byte(`ref="`+ref+`"`))
	})
}

// withTablePart returns the workbook x with edit applied to the part of the
// table named name.
func withTablePart(t *testing.T, x []byte, name string, edit func([]byte) []byte) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(x), int64(len(x)))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	found := false
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(zf.Name, "xl/tables/") && bytes.Contains(data, []byte(`name="`+name+`"`)) {
			data = edit(data)
			found = true
		}
		w, err := zw.Create(zf.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("no table %s", name)
	}
	return buf.Bytes()
}

func TestKeyHeaderMissingFromTableHeader(t *testing.T) {
	// A header row past the last filled row has no columns at all.
	x, _, err := ToXLSX(readFile(t, "kofac_2025_edu", "survey.xml"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = FromXLSX(withTableRef(t, x, "ddiGroups", "A500:G501"))
	want := grpSheet + "!A500: header row 500 has no ID column; the header row is the first row of the table ddiGroups, or row 1 without it"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %s", err, want)
	}

	// Below a title row, the table's header row is the one checked.
	f := kofac(t, "kofac_2025_edu")
	if err := f.InsertRows(varSheet, 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellStr(varSheet, "A3", "Identifier"); err != nil {
		t.Fatal(err)
	}
	want = varSheet + "!A3: header row 3 has no ID column; the header row is the first row of the table ddiVariables, or row 1 without it"
	if _, _, err := fromFile(t, f); err == nil || err.Error() != want {
		t.Errorf("err = %v, want %s", err, want)
	}
}

func TestAutoIDSkipsUsed(t *testing.T) {
	f := openTemplate(t)
	for i, id := range []string{"", "V1", ""} {
		set(t, f, varSheet, "ID", i+2, id)
		set(t, f, varSheet, "name", i+2, string(rune('a'+i)))
	}
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := varIDs(t, xo), map[string]string{"a": "V2", "b": "V1", "c": "V3"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("var IDs = %v, want %v", got, want)
	}

	f = openTemplate(t)
	set(t, f, varSheet, "name", 2, "a")
	set(t, f, varSheet, "name", 3, "b")
	set(t, f, grpSheet, "ID", 2, "V2")
	set(t, f, grpSheet, "ID", 3, "VG1")
	set(t, f, grpSheet, "type", 4, "section")
	xo, _, err = fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := varIDs(t, xo), map[string]string{"a": "V1", "b": "V3"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("var IDs = %v, want %v", got, want)
	}
	if !strings.Contains(string(xo), `<varGrp ID="VG2" type="section"`) {
		t.Errorf("the blank group ID is not VG2:\n%s", xo)
	}
}

func TestVarNameLink(t *testing.T) {
	// Two vars named x and one named y, none with an ID.
	base := func(t *testing.T) *excelize.File {
		f := openTemplate(t)
		for i, name := range []string{"x", "x", "y"} {
			set(t, f, varSheet, "name", i+2, name)
		}
		return f
	}
	for _, tc := range []struct {
		name, varName, col string
	}{
		{"ambiguous name", "x", "varName"},
		{"unknown name", "z", "varName"},
		{"neither varID nor varName", "", "varID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := base(t)
			set(t, f, catSheet, "varName", 2, tc.varName)
			set(t, f, catSheet, "code", 2, "1")
			cell := catSheet + "!" + at(t, f, catSheet, tc.col, 2)
			if _, _, err := fromFile(t, f); err == nil || !strings.HasPrefix(err.Error(), cell+": ") {
				t.Errorf("err = %v, want it to start with %s: ", err, cell)
			}
		})
	}

	t.Run("varID wins over varName", func(t *testing.T) {
		f := base(t)
		set(t, f, catSheet, "varID", 2, "V2")
		set(t, f, catSheet, "varName", 2, "y")
		xo, _, err := fromFile(t, f)
		if err != nil {
			t.Fatal(err)
		}
		if got := names(findVar(t, parse(t, xo), "V2").Children); !slices.Equal(got, []string{"catgry"}) {
			t.Errorf("V2 children = %q, want one catgry", got)
		}
	})

	t.Run("only varName is an empty catgry", func(t *testing.T) {
		f := base(t)
		set(t, f, catSheet, "varName", 2, "y")
		xo, _, err := fromFile(t, f)
		if err != nil {
			t.Fatal(err)
		}
		v := findVar(t, parse(t, xo), "V3")
		if got := names(v.Children); !slices.Equal(got, []string{"catgry"}) || len(v.Children[0].Attrs)+len(v.Children[0].Children) > 0 {
			t.Errorf("V3 children = %q, want one empty catgry", got)
		}
	})

	t.Run("Groups.vars", func(t *testing.T) {
		f := base(t)
		set(t, f, grpSheet, "vars", 2, "V1 y x zz")
		xo, w, err := fromFile(t, f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(xo), `var="V1 V3 x zz"`) {
			t.Errorf("vars not resolved to V1 V3 x zz:\n%s", xo)
		}
		cell := grpSheet + "!" + at(t, f, grpSheet, "vars", 2)
		var got []string
		for _, s := range w {
			if strings.HasPrefix(s, cell+": ") {
				got = append(got, s)
			}
		}
		if len(got) != 2 || !strings.Contains(got[0], " x ") || !strings.Contains(got[1], " zz ") {
			t.Errorf("warnings at %s = %q, want one about x and one about zz", cell, got)
		}
	})
}

// qstn/@ID is an xs:ID too, so an assigned ID skips a typed questionID.
func TestAutoIDSkipsQuestionID(t *testing.T) {
	f := openTemplate(t)
	set(t, f, varSheet, "name", 2, "a")
	set(t, f, varSheet, "question", 2, "q")
	set(t, f, varSheet, "questionID", 2, "V1")
	xo, _, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if got := varIDs(t, xo)["a"]; got != "V2" {
		t.Errorf("var a ID = %s, want V2", got)
	}
	xsdtest.Validate(t, xo)
}

// An ID only in the attributes cell would get an assigned ID as well, so the
// reader asks for it in the ID column instead.
func TestIDOnlyInAttributes(t *testing.T) {
	for _, sheet := range []string{varSheet, grpSheet} {
		f := openTemplate(t)
		set(t, f, sheet, "attributes", 2, `ID="X1"`)
		cell := sheet + "!" + at(t, f, sheet, "attributes", 2)
		_, _, err := fromFile(t, f)
		if err == nil || !strings.HasPrefix(err.Error(), cell+": ") || !strings.Contains(err.Error(), "ID column") {
			t.Errorf("%s: err = %v, want one at %s that names the ID column", sheet, err, cell)
		}
	}
}

// Excel's Total Row adds a last table row of a label and SUBTOTALs, which is
// no content. A row typed below it is.
func TestTotalsRowIsSkipped(t *testing.T) {
	x, _, err := ToXLSX(codebook(`<varGrp ID="G1" var="V1"/><var ID="V1"/><var ID="V2"/>`))
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(x))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	refs := map[string][2]string{} // table name -> its range, then with the totals row
	var below int
	for _, sheet := range f.GetSheetList() {
		tables, err := f.GetTables(sheet)
		if err != nil || len(tables) != 1 {
			t.Fatalf("%s tables = %+v (%v)", sheet, tables, err)
		}
		last, end, err := excelize.CellNameToCoordinates(strings.SplitN(tables[0].Range, ":", 2)[1])
		if err != nil {
			t.Fatal(err)
		}
		lastCell, _ := excelize.CoordinatesToCellName(last, end+1)
		for cell, v := range map[string]string{"A" + strconv.Itoa(end+1): "Total", lastCell: "2"} {
			if err := f.SetCellStr(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
		refs[tables[0].Name] = [2]string{tables[0].Range, "A1:" + lastCell}
		if sheet == varSheet {
			below = end + 2
		}
	}
	set(t, f, varSheet, "ID", below, "V3")
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	x = buf.Bytes()
	for name, ref := range refs {
		x = withTablePart(t, x, name, func(data []byte) []byte {
			// The first ref is the table's own; its autoFilter leaves the totals row out.
			return bytes.Replace(data, []byte(`ref="`+ref[0]+`"`), []byte(`ref="`+ref[1]+`" totalsRowCount="1"`), 1)
		})
	}
	xo, _, w, err := FromXLSX(x)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) > 0 {
		t.Errorf("warnings = %q", w)
	}
	want := codebook(`<varGrp ID="G1" var="V1"/><var ID="V1"/><var ID="V2"/><var ID="V3"/>`)
	if d := ddi.Diff(parse(t, want), parse(t, xo)); d != "" {
		t.Error(d)
	}
}

// Pasting skips Excel's validation, so the reader warns about a value the
// dropdown does not list, and keeps it.
func TestValueOutsideDropdownWarns(t *testing.T) {
	f := open(t, codebook(`<var ID="V1" intrvl="discrete"/>`))
	set(t, f, varSheet, "intrvl", 2, "weird")
	xo, w, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	cell := varSheet + "!" + at(t, f, varSheet, "intrvl", 2)
	if len(w) != 1 || !strings.HasPrefix(w[0], cell+": ") {
		t.Errorf("warnings = %q, want one at %s", w, cell)
	}
	if v := findVar(t, parse(t, xo), "V1"); fmt.Sprint(v.Attrs) != fmt.Sprint([]ddi.Attr{{Name: "ID", Value: "V1"}, {Name: "intrvl", Value: "weird"}}) {
		t.Errorf("V1 attributes = %v, want intrvl kept as weird", v.Attrs)
	}
}

// A var, catgry, varGrp or dataDscr typed in a cell is split and sorted as its
// own row would be, so the output converts back to itself.
func TestTypedMergePointsAreAFixedPoint(t *testing.T) {
	inDataDscr := func(content string) func(t *testing.T, f *excelize.File) {
		return func(t *testing.T, f *excelize.File) {
			row := rowsWhere(t, f, docSheet, "element", "dataDscr")[0]
			set(t, f, docSheet, "text", row, content)
			set(t, f, docSheet, "kind", row, "xml")
			if err := f.RemoveRow(varSheet, 2); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, tc := range []struct {
		name string
		edit func(t *testing.T, f *excelize.File)
	}{
		{"catgry in a var's xml cell", func(t *testing.T, f *excelize.File) {
			set(t, f, varSheet, "xml", 2, `<catgry><labl>a</labl><catValu>1</catValu><labl>b</labl></catgry>`)
		}},
		{"dataDscr kept as xml", inDataDscr(
			`<varGrp ID="G1"/><var ID="V2"><qstn><qstnLit>q</qstnLit></qstn><labl>l</labl></var><varGrp ID="G2"/>`)},
		{"var before varGrp in a dataDscr cell", inDataDscr(`<var ID="V2"/><varGrp ID="G1"/>`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := open(t, codebook(`<var ID="V1"/>`))
			tc.edit(t, f)
			xo, _, err := fromFile(t, f)
			if err != nil {
				t.Fatal(err)
			}
			if xo2, _, _ := convert(t, xo); !bytes.Equal(xo2, xo) {
				t.Errorf("converting the output again changes it:\n%s\nthen\n%s", xo, xo2)
			}
		})
	}
}

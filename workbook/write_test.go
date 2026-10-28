package workbook

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/internal/schema"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
	"github.com/xuri/excelize/v2"
)

func open(t *testing.T, input []byte) *excelize.File {
	t.Helper()
	x, _, err := ToXLSX(input)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(x))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func kofac(t *testing.T, name string) *excelize.File {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(xsdtest.Root(), "testdata", name, "survey.xml"))
	if err != nil {
		t.Fatal(err)
	}
	return open(t, data)
}

func codebook(dataDscr string) []byte {
	return []byte(`<codeBook xmlns="ddi:codebook:2_5"><dataDscr>` + dataDscr + `</dataDscr></codeBook>`)
}

// records returns the rows below the header as header -> value maps.
func records(t *testing.T, f *excelize.File, sheet string) []map[string]string {
	t.Helper()
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]string
	for _, r := range rows[1:] {
		m := map[string]string{}
		for i, h := range rows[0] {
			if i < len(r) {
				m[h] = r[i]
			}
		}
		out = append(out, m)
	}
	return out
}

func where(recs []map[string]string, col, val string) []map[string]string {
	var out []map[string]string
	for _, r := range recs {
		if r[col] == val {
			out = append(out, r)
		}
	}
	return out
}

func checkFields(t *testing.T, what string, got, want map[string]string) {
	t.Helper()
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s %s = %q, want %q", what, k, got[k], v)
		}
	}
}

func TestSheetsAndHeaders(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	want := map[string][]string{
		"Variables": {"ID", "name", "label", "question", "questionID", "questionRef", "responseDomainType", "intrvl", "format",
			"formatSchema", "formatCategory", "notes", "files", "locations", "attributes", "xml"},
		"Categories": {"varID", "varName", "code", "label", "missing", "freq", "attributes", "xml"},
		"Groups":     {"ID", "type", "name", "label", "vars", "attributes", "xml"},
		"Document":   {"level", "element", "attributes", "text", "kind"},
	}
	if got := f.GetSheetList(); !slices.Equal(got, []string{"Variables", "Categories", "Groups", "Document"}) {
		t.Fatalf("sheets = %q", got)
	}
	for sheet, headers := range want {
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(rows[0], headers) {
			t.Errorf("%s headers = %q, want %q", sheet, rows[0], headers)
		}
	}
}

func TestKofacVariableRow(t *testing.T) {
	rows := where(records(t, kofac(t, "kofac_2025_edu"), "Variables"), "name", "Q8_r1")
	if len(rows) != 1 {
		t.Fatalf("%d rows named Q8_r1", len(rows))
	}
	checkFields(t, "Q8_r1", rows[0], map[string]string{
		"ID":                 "V17",
		"label":              "초 Q1 과학 학습배경: 나는 과학에 대해 잘 알고 있다.",
		"question":           "1. 여러분은 다음 내용에 얼마나 동의하나요?\n(해당란에 체크 해주세요)",
		"questionID":         "QS8",
		"responseDomainType": "category",
		"intrvl":             "discrete",
		"format":             "numeric",
		"formatSchema":       "",
		"notes":              "",
		"files":              "F1",
		"questionRef":        "QS8",
		"locations":          "F1",
		"attributes":         "",
		"xml":                "",
	})
}

func TestKofacCategories(t *testing.T) {
	rows := where(records(t, kofac(t, "kofac_2025_edu"), "Categories"), "varID", "V17")
	if len(rows) != 6 {
		t.Fatalf("%d category rows for V17, want 6", len(rows))
	}
	checkFields(t, "first", rows[0], map[string]string{"code": "1", "label": "전혀 아니다(①)", "missing": "", "xml": ""})
	checkFields(t, "last", rows[5], map[string]string{
		"varName": "Q8_r1", "code": "-7", "missing": "Y", "label": "현재 게시본에 없는 값", "attributes": "", "xml": "",
	})
}

func TestKofacGroups(t *testing.T) {
	rows := where(records(t, kofac(t, "kofac_2025_edu"), "Groups"), "ID", "VG1")
	if len(rows) != 1 {
		t.Fatalf("%d rows with ID VG1", len(rows))
	}
	checkFields(t, "VG1", rows[0], map[string]string{
		"type": "grid", "name": "Q8", "vars": "V17 V18 V19 V20 V21", "label": "초 Q1 과학 학습배경", "attributes": "", "xml": "",
	})
}

func TestDocumentOutline(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	rows, err := f.GetRows("Document")
	if err != nil {
		t.Fatal(err)
	}
	if got := rows[1]; !slices.Equal(got, []string{"0", "codeBook", `xmlns="ddi:codebook:2_5" version="2.5" ID="CB1"`}) {
		t.Errorf("row 2 = %q", got) // GetRows drops the trailing empty text and kind
	}
	recs := records(t, f, "Document")
	titl := where(recs, "element", "titl")
	if len(titl) == 0 || titl[0]["level"] != "4" {
		t.Errorf("titl rows = %v, want level 4 first", titl)
	}
	for _, name := range []string{"var", "varGrp", "catgry"} {
		if n := len(where(recs, "element", name)); n > 0 {
			t.Errorf("%d %s rows in Document", n, name)
		}
	}
	if dd := where(recs, "element", "dataDscr"); len(dd) != 1 || dd[0]["level"] != "1" {
		t.Errorf("dataDscr rows = %v", dd)
	}
	style, err := f.GetCellStyle("Document", "B3")
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := f.GetStyle(style); s.Alignment == nil || s.Alignment.Indent != 1 {
		t.Errorf("Document!B3 (level 1) alignment = %+v, want indent 1", s.Alignment)
	}
}

func TestTextFormatOnColumns(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		for c, h := range rows[0] {
			cell, _ := excelize.CoordinatesToCellName(c+1, len(rows)+5)
			id, err := f.GetCellStyle(sheet, cell)
			if err != nil {
				t.Fatal(err)
			}
			s, _ := f.GetStyle(id)
			if s.NumFmt != 49 {
				t.Errorf("%s!%s NumFmt = %d, want 49", sheet, cell, s.NumFmt)
			}
			wrap := h == "question" || h == "notes" || h == "xml" || h == "text"
			if got := s.Alignment != nil && s.Alignment.WrapText; got != wrap {
				t.Errorf("%s!%s (%s) wraps = %v, want %v", sheet, cell, h, got, wrap)
			}
		}
	}
	panes, err := f.GetPanes("Categories")
	if err != nil {
		t.Fatal(err)
	}
	if !panes.Freeze || panes.YSplit != 1 {
		t.Errorf("Categories panes = %+v, want row 1 frozen", panes)
	}
}

func TestRejects(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        []string
	}{
		{"var without ID", string(codebook(`<var name="x"/>`)), []string{"var", "ID"}},
		{"var with empty ID", string(codebook(`<var ID="" name="x"/>`)), []string{"var", "ID"}},
		{"varGrp without ID", string(codebook(`<varGrp type="grid"/>`)), []string{"varGrp", "ID"}},
		{"two dataDscr", `<codeBook xmlns="ddi:codebook:2_5"><dataDscr/><dataDscr/></codeBook>`, []string{"dataDscr"}},
		{"var ID repeats", string(codebook(`<var ID="V1"/><var ID="V1"/>`)), []string{"dataDscr/var[2]: ID V1 repeats"}},
		{"var and varGrp share an ID", string(codebook(`<varGrp ID="V1"/><var ID="V1"/>`)), []string{"dataDscr/var[1]: ID V1 repeats"}},
		// A catgry kept as written takes itself and every later catgry into
		// the var's xml cell, which then overflows.
		{"text between catgry children, too long for the var's cell", string(codebook(`<var ID="V1"/><var ID="V2">` +
			`<catgry>x<catValu>1</catValu></catgry><catgry><catValu>2</catValu><labl>` + strings.Repeat("l", cellLimit-10) +
			`</labl></catgry></var>`)),
			[]string{"dataDscr/var[2]/catgry[1]: <catgry> holds text between its child elements"}},
		// Kept as written, its content would not fit in one cell.
		{"text between children, too long for a cell", string(codebook(`<var ID="V1"/><var ID="V2"><labl>a</labl>x<notes>` +
			strings.Repeat("n", cellLimit) + `</notes></var>`)),
			[]string{"dataDscr/var[2]: <var> holds text between its child elements"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ToXLSX([]byte(tc.input))
			for _, w := range tc.want {
				if err == nil || !strings.Contains(err.Error(), w) {
					t.Errorf("err = %v, want it to mention %s", err, w)
				}
			}
		})
	}
}

func TestCellTooLong(t *testing.T) {
	ok := codebook(`<var ID="V1"><labl>` + strings.Repeat("가", 32767) + `</labl></var>`)
	if _, _, err := ToXLSX(ok); err != nil {
		t.Errorf("32767 characters: %v", err)
	}
	// Excel counts UTF-16 units, so 16,384 emoji are 32,768 characters.
	long := codebook(`<var ID="V1"><labl>` + strings.Repeat("😀", 16384) + `</labl></var>`)
	if _, _, err := ToXLSX(long); err == nil || !strings.HasPrefix(err.Error(), "Variables!C2: ") {
		t.Errorf("err = %v, want it to start with Variables!C2: ", err)
	}
}

// TestSplitRules checks that what a column cannot hold without loss, or
// without changing order when it is read back column-first, goes to the cells.
func TestSplitRules(t *testing.T) {
	for _, tc := range []struct {
		name string
		vars string
		want map[string]string
		cats []map[string]string
	}{
		{
			name: "two labl",
			vars: `<var ID="V1"><labl>a</labl><labl level="x">b</labl></var>`,
			want: map[string]string{"label": "", "xml": "<labl>a</labl>\n<labl level=\"x\">b</labl>"},
		},
		{
			name: "empty labl",
			vars: `<var ID="V1"><labl/></var>`,
			want: map[string]string{"label": "", "xml": "<labl/>"},
		},
		{
			name: "qstn with another attribute",
			vars: `<var ID="V1"><qstn ID="Q1" seqNo="1"><qstnLit>q</qstnLit></qstn></var>`,
			want: map[string]string{"question": "", "questionID": "", "xml": `<qstn ID="Q1" seqNo="1"><qstnLit>q</qstnLit></qstn>`},
		},
		{
			name: "qstn with preQTxt",
			vars: `<var ID="V1"><qstn><preQTxt>p</preQTxt><qstnLit>q</qstnLit></qstn></var>`,
			want: map[string]string{"question": "", "xml": `<qstn><preQTxt>p</preQTxt><qstnLit>q</qstnLit></qstn>`},
		},
		{
			name: "qstn modeled without attributes",
			vars: `<var ID="V1"><qstn><qstnLit>q</qstnLit></qstn></var>`,
			want: map[string]string{"question": "q", "questionID": "", "xml": ""},
		},
		{
			name: "varFormat with text",
			vars: `<var ID="V1"><varFormat type="numeric">x</varFormat></var>`,
			want: map[string]string{"format": "", "xml": `<varFormat type="numeric">x</varFormat>`},
		},
		{
			name: "varFormat without attributes",
			vars: `<var ID="V1"><varFormat/></var>`,
			want: map[string]string{"format": "", "xml": "<varFormat/>"},
		},
		{
			name: "varFormat modeled",
			vars: `<var ID="V1"><varFormat schema="other" type="character" category="date"/></var>`,
			want: map[string]string{"format": "character", "formatSchema": "other", "formatCategory": "date", "xml": ""},
		},
		{
			name: "notes column takes only the leading plain notes",
			vars: `<var ID="V1"><notes>a</notes><notes>b</notes><notes type="t">c</notes><notes>d</notes></var>`,
			want: map[string]string{"notes": "a\nb", "xml": "<notes type=\"t\">c</notes>\n<notes>d</notes>"},
		},
		{
			name: "multi-line note",
			vars: "<var ID=\"V1\"><notes>x\ny</notes></var>",
			want: map[string]string{"notes": "", "xml": "<notes>x\ny</notes>"},
		},
		{
			name: "attribute values a column cannot hold",
			vars: "<var ID=\"V1\" name=\"\" intrvl=\" d\" files=\"F1\"/>",
			want: map[string]string{"ID": "V1", "name": "", "intrvl": "", "files": "F1", "attributes": `name="" intrvl=" d"`},
		},
		{
			name: "catStat",
			vars: `<var ID="V1">` +
				`<catgry missing="Y"><catValu>1</catValu><catStat type="percent">5</catStat><catStat type="freq">3</catStat></catgry>` +
				`<catgry><catValu>2</catValu><catStat type="freq">3</catStat><catStat type="freq" wgtd="wgtd">2.5</catStat></catgry>` +
				`<catgry><catValu>3</catValu><catStat type="freq">1</catStat><catStat type="freq">2</catStat></catgry>` +
				`</var>`,
			cats: []map[string]string{
				{"code": "1", "missing": "Y", "freq": "", "xml": "<catStat type=\"percent\">5</catStat>\n<catStat type=\"freq\">3</catStat>"},
				{"code": "2", "freq": "3", "xml": `<catStat type="freq" wgtd="wgtd">2.5</catStat>`},
				{"code": "3", "freq": "", "xml": "<catStat type=\"freq\">1</catStat>\n<catStat type=\"freq\">2</catStat>"},
			},
		},
		{
			name: "catgry kept as written ends the Categories rows",
			vars: `<var ID="V1"><catgry><catValu>1</catValu></catgry><catgry>x<catValu>2</catValu></catgry><catgry><catValu>3</catValu></catgry></var>`,
			want: map[string]string{"xml": "<catgry>x<catValu>2</catValu></catgry>\n<catgry><catValu>3</catValu></catgry>"},
			cats: []map[string]string{{"code": "1", "attributes": "", "xml": ""}},
		},
		{
			name: "var with a space between children is split",
			vars: `<var ID="V1" name="n"><labl>a</labl> <catgry><catValu>1</catValu></catgry></var>`,
			want: map[string]string{"name": "n", "label": "a", "xml": ""},
			cats: []map[string]string{{"code": "1", "xml": ""}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := open(t, codebook(tc.vars))
			vars := records(t, f, "Variables")
			if len(vars) != 1 {
				t.Fatalf("%d Variables rows", len(vars))
			}
			checkFields(t, "var", vars[0], tc.want)
			cats := records(t, f, "Categories")
			if len(cats) != len(tc.cats) {
				t.Fatalf("%d Categories rows, want %d", len(cats), len(tc.cats))
			}
			for i, want := range tc.cats {
				checkFields(t, "catgry", cats[i], want)
			}
		})
	}
}

func TestSheetsAreTables(t *testing.T) {
	for name, f := range map[string]*excelize.File{
		"kofac":       kofac(t, "kofac_2025_edu"),
		"header only": open(t, codebook(`<var ID="V1"/>`)),
	} {
		for _, sheet := range f.GetSheetList() {
			tables, err := f.GetTables(sheet)
			if err != nil {
				t.Fatal(err)
			}
			if len(tables) != 1 || tables[0].Name != "ddi"+sheet {
				t.Errorf("%s %s tables = %+v, want one named ddi%s", name, sheet, tables, sheet)
				continue
			}
			rows, err := f.GetRows(sheet)
			if err != nil {
				t.Fatal(err)
			}
			// A table has at least one row below its header.
			end, _ := excelize.CoordinatesToCellName(len(rows[0]), max(len(rows), 2))
			if got := tables[0].Range; got != "A1:"+end {
				t.Errorf("%s %s table range = %s, want A1:%s", name, sheet, got, end)
			}
		}
	}
}

func TestDropdowns(t *testing.T) {
	f := kofac(t, "kofac_2025_edu")
	want := map[string]map[string][]string{
		varSheet: {
			"intrvl":             schema.AttrEnum("var", "intrvl"),
			"responseDomainType": schema.AttrEnum("qstn", "responseDomainType"),
			"format":             schema.AttrEnum("varFormat", "type"),
			"formatSchema":       schema.AttrEnum("varFormat", "schema"),
			"formatCategory":     schema.AttrEnum("varFormat", "category"),
		},
		catSheet: {"missing": schema.AttrEnum("catgry", "missing")},
		grpSheet: {"type": schema.AttrEnum("varGrp", "type")},
		docSheet: {},
	}
	for sheet, cols := range want {
		dvs, err := f.GetDataValidations(sheet)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, dv := range dvs {
			got[dv.Sqref] = dv.Formula1
			if dv.Type != "list" || dv.ErrorStyle == nil || *dv.ErrorStyle != "warning" || !dv.ShowErrorMessage {
				t.Errorf("%s!%s validation = %+v, want a list that warns", sheet, dv.Sqref, dv)
			}
		}
		if len(got) != len(cols) {
			t.Errorf("%s validations = %q, want %d", sheet, got, len(cols))
		}
		for col, vals := range cols {
			if len(vals) == 0 {
				t.Fatalf("%s.%s has no enumeration", sheet, col)
			}
			c := strings.TrimSuffix(at(t, f, sheet, col, 2), "2")
			ref := c + "2:" + c + "1048576"
			if want := `"` + strings.Join(vals, ",") + `"`; got[ref] != want {
				t.Errorf("%s!%s list = %s, want %s", sheet, ref, got[ref], want)
			}
		}
	}
}

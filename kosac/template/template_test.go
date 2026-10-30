package template

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/xuri/excelize/v2"
)

var update = flag.Bool("update", false, "rebuild blank_v3.2.xlsx from the sample")

const (
	guideSheet   = "00_작성안내"
	exampleSheet = "예시_설명"
	blankNote    = "임시 내장본, 원본 교체 예정"
)

func sample(t *testing.T) []byte {
	t.Helper()
	x, err := os.ReadFile(filepath.Join("..", "..", "testdata", "kosac", "v32_sample_2024.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func open(t *testing.T, x []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(x))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func save(t *testing.T, f *excelize.File) []byte {
	t.Helper()
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// edit returns the sample after fn changed it.
func edit(t *testing.T, fn func(f *excelize.File) error) []byte {
	t.Helper()
	f := open(t, sample(t))
	if err := fn(f); err != nil {
		t.Fatal(err)
	}
	return save(t, f)
}

func read(t *testing.T, x []byte, opt ReadOptions) (*kosac.Codebook, []kosac.Issue) {
	t.Helper()
	cb, issues, err := Read(x, opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, is := range issues {
		if is.Rule != "S05" || is.Message == "" || is.Sheet == "" {
			t.Errorf("issue without rule, sheet or message: %+v", is)
		}
	}
	return cb, issues
}

func values(r kosac.Record) map[string]string {
	m := map[string]string{}
	for k, f := range r {
		m[k] = f.Value
	}
	return m
}

func TestReadSample(t *testing.T) {
	cb, issues := read(t, sample(t), ReadOptions{})
	if len(issues) > 0 {
		t.Fatalf("issues: %+v", issues)
	}
	if len(cb.Files) != 1 || len(cb.Vars) != 214 || len(cb.Codes) != 828 {
		t.Errorf("files %d, vars %d, codes %d", len(cb.Files), len(cb.Vars), len(cb.Codes))
	}
	if got := cb.Study.Get("VALID_N"); got != "10000" {
		t.Errorf("VALID_N = %q", got)
	}
	if got := cb.Study["POP_SOURCE"].Loc; got != (kosac.Loc{Sheet: kosac.SheetStudy, Cell: "E19"}) {
		t.Errorf("POP_SOURCE at %+v", got)
	}
	if got := cb.Vars[0].Rec["VAR_NAME"]; got != (kosac.Field{Value: "IDX", Loc: kosac.Loc{Sheet: kosac.SheetVars, Cell: "B4"}}) {
		t.Errorf("first VAR_NAME = %+v", got)
	}
	if cb.Template.ID.Value != "KOFAC-SURVEY-META" || cb.Template.Version.Value != "3.2" {
		t.Errorf("template %+v", cb.Template)
	}
	if len(cb.Study) != len(kosac.StudyKeys) || len(cb.Files[0].Rec) != len(kosac.FileKeys) ||
		len(cb.Vars[0].Rec) != len(kosac.VarKeys) || len(cb.Codes[0].Rec) != len(kosac.CodeKeys) {
		t.Error("a record does not hold every KEY of its sheet")
	}
}

// The engine never reads the rule standards from a file, so its constants
// must match the template they were copied from.
func TestKeysMatchSample(t *testing.T) {
	f := open(t, sample(t))
	rows, err := f.GetRows(kosac.SheetStudy)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	req := map[string]string{}
	for _, r := range rows[1:45] {
		keys = append(keys, r[0])
		req[r[0]] = r[3]
	}
	if !slices.Equal(keys, kosac.StudyKeys) || !maps.Equal(req, kosac.StudyRequired) {
		t.Errorf("01 keys or 구분 differ from the sample")
	}
	for sheet, want := range map[string][]string{kosac.SheetFiles: kosac.FileKeys, kosac.SheetVars: kosac.VarKeys, kosac.SheetCodes: kosac.CodeKeys} {
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(rows[0], want) {
			t.Errorf("%s row 1 = %q", sheet, rows[0])
		}
	}
}

func TestReadS05(t *testing.T) {
	setMeta := func(cell, v string) func(f *excelize.File) error {
		return func(f *excelize.File) error { return f.SetCellStr(kosac.SheetMeta, cell, v) }
	}
	for _, c := range []struct {
		name  string
		edit  func(f *excelize.File) error
		opt   ReadOptions
		want  kosac.Issue // only Level, Sheet, Cell, Actual are compared
		count int         // issues expected; 0 means any number
	}{
		{"no 04 sheet", func(f *excelize.File) error { return f.DeleteSheet(kosac.SheetCodes) }, ReadOptions{},
			kosac.Issue{Level: kosac.R, Sheet: kosac.SheetCodes}, 1},
		{"extra 03 key", func(f *excelize.File) error { return f.SetCellStr(kosac.SheetVars, "P1", "FOO") }, ReadOptions{},
			kosac.Issue{Level: kosac.R, Sheet: kosac.SheetVars, Cell: "P1", Actual: "FOO"}, 1},
		{"no TOOL row", func(f *excelize.File) error { return f.RemoveRow(kosac.SheetStudy, 31) }, ReadOptions{},
			kosac.Issue{Level: kosac.R, Sheet: kosac.SheetStudy}, 1},
		{"KOSAC id", setMeta("B1", "KOSAC-SURVEY-META"), ReadOptions{},
			kosac.Issue{Level: kosac.W, Sheet: kosac.SheetMeta, Cell: "B1", Actual: "KOSAC-SURVEY-META"}, 1},
		{"unknown id", setMeta("B1", "X"), ReadOptions{},
			kosac.Issue{Level: kosac.R, Sheet: kosac.SheetMeta, Cell: "B1", Actual: "X"}, 1},
		{"version 3.1", setMeta("B2", "3.1"), ReadOptions{},
			kosac.Issue{Level: kosac.R, Sheet: kosac.SheetMeta, Cell: "B2", Actual: "3.1"}, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			cb, issues := read(t, edit(t, c.edit), c.opt)
			if len(issues) != c.count {
				t.Fatalf("issues: %+v", issues)
			}
			got := issues[0]
			if got.Level != c.want.Level || got.Sheet != c.want.Sheet || got.Cell != c.want.Cell || got.Actual != c.want.Actual {
				t.Errorf("issue %+v, want %+v", got, c.want)
			}
			if (cb == nil) != (c.want.Level == kosac.R) {
				t.Errorf("model nil = %v with a %s issue", cb == nil, c.want.Level)
			}
		})
	}
	t.Run("version 3.1 allowed", func(t *testing.T) {
		cb, issues := read(t, edit(t, setMeta("B2", "3.1")), ReadOptions{AllowedVersions: []string{"3.1"}})
		if cb == nil || len(issues) > 0 {
			t.Errorf("issues: %+v", issues)
		}
	})
	t.Run("not a zip", func(t *testing.T) {
		if _, _, err := Read([]byte("not a workbook"), ReadOptions{}); err == nil {
			t.Error("no error")
		}
	})
}

func TestReadEditedCells(t *testing.T) {
	x := edit(t, func(f *excelize.File) error {
		code := "yyyy/mm/dd"
		date, err := f.NewStyle(&excelize.Style{CustomNumFmt: &code})
		if err != nil {
			return err
		}
		thousands, err := f.NewStyle(&excelize.Style{NumFmt: 3})
		if err != nil {
			return err
		}
		// Korean Excel writes these built-in IDs for dates typed as
		// "2024년 10월 4일"; excelize shows them as the bare serial.
		korean, err := f.NewStyle(&excelize.Style{NumFmt: 31})
		if err != nil {
			return err
		}
		korean57, err := f.NewStyle(&excelize.Style{NumFmt: 57})
		if err != nil {
			return err
		}
		return errors.Join(
			f.SetCellValue(kosac.SheetStudy, "E12", time.Date(2024, 10, 4, 0, 0, 0, 0, time.UTC)),
			f.SetCellStyle(kosac.SheetStudy, "E12", "E12", date),
			f.SetCellValue(kosac.SheetStudy, "E13", time.Date(2024, 10, 28, 0, 0, 0, 0, time.UTC)),
			f.SetCellStyle(kosac.SheetStudy, "E13", "E13", korean),
			f.SetCellValue(kosac.SheetVars, "O4", time.Date(2024, 10, 5, 0, 0, 0, 0, time.UTC)),
			f.SetCellStyle(kosac.SheetVars, "O4", "O4", korean57),
			// Text in a date-formatted cell is still text.
			f.SetCellStr(kosac.SheetStudy, "E27", "45569"),
			f.SetCellStyle(kosac.SheetStudy, "E27", "E27", date),
			f.SetCellValue(kosac.SheetStudy, "E26", 10000),
			f.SetCellStyle(kosac.SheetStudy, "E26", "E26", thousands),
			f.SetCellStr(kosac.SheetVars, "B1", "VAR_NAME "),
			f.InsertRows(kosac.SheetVars, 10, 2),
			f.SetCellStr(kosac.SheetVars, "O220", "메모"),
		)
	})
	// The edits must show the way a contractor sees them, or the test proves nothing.
	f := open(t, x)
	if v, _ := f.GetCellValue(kosac.SheetStudy, "E12"); v != "2024/10/04" {
		t.Fatalf("E12 shows %q", v)
	}
	if v, _ := f.GetCellValue(kosac.SheetStudy, "E26"); v != "10,000" {
		t.Fatalf("E26 shows %q", v)
	}

	cb, issues := read(t, x, ReadOptions{})
	if len(issues) > 0 {
		t.Fatalf("issues: %+v", issues)
	}
	for _, c := range []struct{ got, want string }{
		{cb.Study.Get("FIELD_START"), "2024-10-04"},
		{cb.Study.Get("FIELD_END"), "2024-10-28"},
		{cb.Vars[0].Rec.Get("NOTE"), "2024-10-05"},
		{cb.Study.Get("VALID_N"), "45569"},
	} {
		if c.got != c.want {
			t.Errorf("read %q, want %q", c.got, c.want)
		}
	}
	if got := cb.Study.Get("TARGET_N"); got != "10,000" {
		t.Errorf("TARGET_N = %q", got)
	}
	if len(cb.Vars) != 215 {
		t.Fatalf("%d vars", len(cb.Vars))
	}
	if got := cb.Vars[6].Rec["VAR_NAME"].Loc.Cell; got != "B12" {
		t.Errorf("var after the blank rows at %s", got)
	}
	last := cb.Vars[214].Rec
	if last.Get("VAR_NAME") != "" || last["NOTE"] != (kosac.Field{Value: "메모", Loc: kosac.Loc{Sheet: kosac.SheetVars, Cell: "O220"}}) {
		t.Errorf("NOTE-only row: %+v", last)
	}
}

func TestWriteRoundTrip(t *testing.T) {
	cb, _ := read(t, sample(t), ReadOptions{})
	x, err := Write(cb)
	if err != nil {
		t.Fatal(err)
	}
	back, issues := read(t, x, ReadOptions{})
	if len(issues) > 0 {
		t.Fatalf("issues: %+v", issues)
	}
	if !maps.Equal(values(cb.Study), values(back.Study)) {
		t.Error("01 values differ")
	}
	if len(back.Files) != len(cb.Files) || len(back.Vars) != len(cb.Vars) || len(back.Codes) != len(cb.Codes) {
		t.Fatalf("files %d, vars %d, codes %d", len(back.Files), len(back.Vars), len(back.Codes))
	}
	for i := range cb.Files {
		if !maps.Equal(values(cb.Files[i].Rec), values(back.Files[i].Rec)) {
			t.Errorf("02 row %d differs", i)
		}
	}
	for i := range cb.Vars {
		if !maps.Equal(values(cb.Vars[i].Rec), values(back.Vars[i].Rec)) {
			t.Errorf("03 row %d differs", i)
		}
	}
	for i := range cb.Codes {
		if !maps.Equal(values(cb.Codes[i].Rec), values(back.Codes[i].Rec)) {
			t.Errorf("04 row %d differs", i)
		}
	}
}

// A draft is compared and stored by its bytes, so the same model must give
// the same file.
func TestWriteDeterministic(t *testing.T) {
	sampleCB, _ := read(t, sample(t), ReadOptions{})
	export, err := os.ReadFile(filepath.Join("..", "..", "testdata", "walla_synthetic", "survey.xml"))
	if err != nil {
		t.Fatal(err)
	}
	nativeCB, _, err := kosac.FromWallaDDI(export, kosac.WallaOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for name, cb := range map[string]*kosac.Codebook{"sample": sampleCB, "native": nativeCB} {
		t.Run(name, func(t *testing.T) {
			first, err := Write(cb)
			if err != nil {
				t.Fatal(err)
			}
			for range 4 {
				again, err := Write(cb)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(first, again) {
					t.Fatal("two writes of one model differ")
				}
			}
		})
	}
}

func dataValidations(t *testing.T, f *excelize.File, sheet string) []string {
	t.Helper()
	dvs, err := f.GetDataValidations(sheet)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range dvs {
		out = append(out, d.Sqref+" "+d.Type+" "+d.Formula1)
	}
	return out
}

func autoFilters(t *testing.T, x []byte) []string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(x), int64(len(x)))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`<autoFilter ref="([^"]+)"`)
	var refs []string
	for _, zf := range z.File {
		if !strings.HasPrefix(zf.Name, "xl/worksheets/") {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllSubmatch(b, -1) {
			refs = append(refs, string(m[1]))
		}
	}
	slices.Sort(refs)
	return refs
}

func TestBlank(t *testing.T) {
	b := Blank()
	b[0] ^= 0xff
	if Blank()[0] == b[0] {
		t.Fatal("Blank shares its bytes with callers")
	}
	b = Blank()

	cb, issues := read(t, b, ReadOptions{})
	if cb == nil || len(issues) > 0 {
		t.Fatalf("issues: %+v", issues)
	}
	if len(cb.Files)+len(cb.Vars)+len(cb.Codes) > 0 {
		t.Errorf("rows left: files %d, vars %d, codes %d", len(cb.Files), len(cb.Vars), len(cb.Codes))
	}

	f, s := open(t, b), open(t, sample(t))
	if want := []string{guideSheet, kosac.SheetStudy, kosac.SheetFiles, kosac.SheetVars, kosac.SheetCodes, kosac.SheetMeta}; !slices.Equal(f.GetSheetList(), want) {
		t.Errorf("sheets %q", f.GetSheetList())
	}
	if title, _ := f.GetCellValue(guideSheet, "A1"); strings.Contains(title, "작성예시") || strings.Contains(title, "2024") {
		t.Errorf("guide title %q", title)
	}
	meta, err := f.GetRows(kosac.SheetMeta)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(meta, func(r []string) bool { return slices.Equal(r, []string{"NOTE", blankNote}) }) {
		t.Errorf("_meta rows %q", meta)
	}

	wantDV := map[string][]string{
		kosac.SheetStudy: {"E2", "E20", "E21", "E28", "E30", "E38", "E42"},
		kosac.SheetFiles: {"C4:C23", "E4:E23"},
		kosac.SheetVars:  {"G4:G3003", "I4:I3003", "J4:J3003", "L4:L3003", "M4:M3003", "N4:N3003"},
		kosac.SheetCodes: nil,
	}
	for sheet, refs := range wantDV {
		got := dataValidations(t, f, sheet)
		if !slices.Equal(got, dataValidations(t, s, sheet)) {
			t.Errorf("%s data validations differ from the sample: %q", sheet, got)
		}
		var gotRefs []string
		for _, g := range got {
			gotRefs = append(gotRefs, strings.Fields(g)[0])
		}
		if !slices.Equal(gotRefs, refs) {
			t.Errorf("%s data validation ranges %q", sheet, gotRefs)
		}
	}

	if got := autoFilters(t, b); !slices.Equal(got, []string{"A2:E15003", "A2:I23", "A2:O3003"}) {
		t.Errorf("autoFilters %q", got)
	}
	filterNames := map[string]string{}
	for _, d := range f.GetDefinedName() {
		if d.Name == "_xlnm._FilterDatabase" {
			filterNames[d.Scope] = d.RefersTo
		}
	}
	if want := map[string]string{
		kosac.SheetFiles: "'02_데이터파일'!$A$2:$I$23",
		kosac.SheetVars:  "'03_변수정의'!$A$2:$O$3003",
		kosac.SheetCodes: "'04_코드값'!$A$2:$E$15003",
	}; !maps.Equal(filterNames, want) {
		t.Errorf("filter names %q", filterNames)
	}

	for row := 2; row <= 45; row++ {
		v, err := f.GetCellValue(kosac.SheetStudy, fmt.Sprintf("E%d", row))
		if err != nil {
			t.Fatal(err)
		}
		if want := map[bool]string{true: "v1.0"}[row == 2]; v != want {
			t.Errorf("01 E%d = %q", row, v)
		}
	}
}

// TestBlankRegenerates rebuilds the embedded blank from the sample, so the
// blank can be made again when the sample changes.
func TestBlankRegenerates(t *testing.T) {
	if !*update {
		t.Skip("run with -update to rebuild blank_v3.2.xlsx")
	}
	f := open(t, sample(t))
	// A blank keeps RULE_VERSION's only list value, so a contractor need not pick it.
	for row := 2; row <= 45; row++ {
		if k, _ := f.GetCellValue(kosac.SheetStudy, fmt.Sprintf("A%d", row)); k == "RULE_VERSION" {
			continue
		}
		if err := f.SetCellValue(kosac.SheetStudy, fmt.Sprintf("E%d", row), nil); err != nil {
			t.Fatal(err)
		}
	}
	// SetCellValue keeps the style, so the rows keep their colors.
	for _, sheet := range []string{kosac.SheetFiles, kosac.SheetVars, kosac.SheetCodes} {
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		for i := 3; i < len(rows); i++ {
			for c := range rows[i] {
				cell, _ := excelize.CoordinatesToCellName(c+1, i+1)
				if err := f.SetCellValue(sheet, cell, nil); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if err := f.DeleteSheet(exampleSheet); err != nil {
		t.Fatal(err)
	}
	// "[작성예시] 2024 … — 재단 조사 코드북 v3.2 (…)": the part after the dash is the template's own title.
	title, err := f.GetCellValue(guideSheet, "A1")
	if err != nil {
		t.Fatal(err)
	}
	_, general, ok := strings.Cut(title, " — ")
	if !strings.HasPrefix(title, "[작성예시]") || !ok {
		t.Fatalf("guide title %q", title)
	}
	if err := f.SetCellStr(guideSheet, "A1", general); err != nil {
		t.Fatal(err)
	}
	meta, err := f.GetRows(kosac.SheetMeta)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow(kosac.SheetMeta, fmt.Sprintf("A%d", len(meta)+1), &[]string{"NOTE", blankNote}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("blank_v3.2.xlsx", save(t, f), 0o644); err != nil {
		t.Fatal(err)
	}
}

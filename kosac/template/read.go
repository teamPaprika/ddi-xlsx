// Package template reads and writes the foundation's v3.2 and v4.0 workbooks.
package template

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/xuri/excelize/v2"
)

const (
	templateID = "KOFAC-SURVEY-META"
	// The foundation accepts this spelling with a warning.
	altTemplateID = "KOSAC-SURVEY-META"
	// Rows 2 and 3 of the 02-04 sheets hold Korean names and guidance.
	firstDataRow = 4
)

type ReadOptions struct {
	AllowedVersions []string // Empty accepts v3.2 and v4.0; their layouts remain distinct.
}

// Read selects the v3.2 or v4.0 layout from _meta before reading records. The model is nil when the structure check (S05)
// rejects the workbook; the issues say why. The error is for a file that is
// not a workbook or unzips past the size limit.
func Read(xlsx []byte, opt ReadOptions) (*kosac.Codebook, []kosac.Issue, error) {
	f, err := excelize.OpenReader(bytes.NewReader(xlsx), unzipLimits(len(xlsx)))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	r := &reader{f: f, dateStyle: map[int]bool{}}
	if p, err := f.GetWorkbookProps(); err == nil && p.Date1904 != nil {
		r.date1904 = *p.Date1904
	}

	cb := &kosac.Codebook{}
	if rows, ok, err := r.rows(kosac.SheetMeta); err != nil {
		return nil, nil, err
	} else if ok {
		cb.Template = r.meta(rows, opt.AllowedVersions)
	}
	r.version = strings.TrimSpace(cb.Template.Version.Value)
	layout := kosac.LayoutFor(r.version)
	if rows, ok, err := r.rows(kosac.SheetStudy); err != nil {
		return nil, nil, err
	} else if ok {
		cb.Study = r.study(rows)
	}
	tables := map[string][]kosac.Record{}
	for _, t := range []struct {
		sheet string
		keys  []string
	}{{kosac.SheetFiles, layout.Files}, {kosac.SheetVars, layout.Vars}, {kosac.SheetCodes, layout.Codes}} {
		rows, ok, err := r.rows(t.sheet)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			tables[t.sheet] = r.table(t.sheet, t.keys, rows)
		}
	}
	if r.version == "4.0" {
		if rows, ok, err := r.rows(kosac.SheetRecodes); err != nil {
			return nil, nil, err
		} else if ok {
			cb.Recodes = r.table(kosac.SheetRecodes, kosac.RecodeKeys, rows)
		}
	}

	// A rejected structure stops here, so no rule runs on a misread sheet.
	if slices.ContainsFunc(r.issues, func(i kosac.Issue) bool { return i.Level == kosac.R }) {
		return nil, r.issues, nil
	}
	for _, rec := range tables[kosac.SheetFiles] {
		cb.Files = append(cb.Files, &kosac.File{Rec: rec})
	}
	for _, rec := range tables[kosac.SheetVars] {
		cb.Vars = append(cb.Vars, &kosac.Var{Rec: rec})
	}
	for _, rec := range tables[kosac.SheetCodes] {
		cb.Codes = append(cb.Codes, &kosac.Code{Rec: rec})
	}
	return cb, r.issues, nil
}

type reader struct {
	version   string
	f         *excelize.File
	date1904  bool
	dateStyle map[int]bool // by style index
	issues    []kosac.Issue
}

func (r *reader) add(level kosac.Level, sheet, cell, actual, expected, msg string) {
	r.issues = append(r.issues, kosac.Issue{
		Rule: "S05", Level: level, Sheet: sheet, Cell: cell, Actual: actual, Expected: expected, Message: msg,
	})
}

// rows returns a sheet's cells as displayed, with dates as YYYY-MM-DD, and
// whether the sheet exists.
func (r *reader) rows(sheet string) ([][]string, bool, error) {
	if i, _ := r.f.GetSheetIndex(sheet); i < 0 {
		r.add(kosac.R, sheet, "", "", sheet, fmt.Sprintf("%s 시트가 없습니다.", sheet))
		return nil, false, nil
	}
	shown, err := r.f.GetRows(sheet)
	if err != nil {
		return nil, false, err
	}
	raw, err := r.f.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, false, err
	}
	// Every number is checked, not only those whose shown text differs from
	// the raw one: excelize shows the bare serial for the locale-dependent
	// date formats Korean Excel writes.
	for i, row := range shown {
		for c := range row {
			if i < len(raw) && c < len(raw[i]) {
				if d, ok := r.date(sheet, c, i, raw[i][c]); ok {
					row[c] = d
				}
			}
		}
	}
	return shown, true, nil
}

// date returns the cell at 0-based (col, row) as YYYY-MM-DD when its number
// format shows a date. A contractor who types a date gets whatever format
// their Excel picks, but the template asks for YYYY-MM-DD.
func (r *reader) date(sheet string, col, row int, raw string) (string, bool) {
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return "", false
	}
	cell, _ := excelize.CoordinatesToCellName(col+1, row+1)
	s, err := r.f.GetCellStyle(sheet, cell)
	if err != nil {
		return "", false
	}
	isDate, seen := r.dateStyle[s]
	if !seen {
		st, err := r.f.GetStyle(s)
		isDate = err == nil && isDateFormat(st)
		r.dateStyle[s] = isDate
	}
	if !isDate {
		return "", false
	}
	// Text that looks like a number shows as typed whatever the cell's format.
	if ct, err := r.f.GetCellType(sheet, cell); err != nil || ct != excelize.CellTypeNumber && ct != excelize.CellTypeUnset {
		return "", false
	}
	t, err := excelize.ExcelDateToTime(n, r.date1904)
	if err != nil {
		return "", false
	}
	return t.Format(time.DateOnly), true
}

// dateNumFmts are the built-in number formats that show a date, as Korean
// Excel defines the locale-dependent ones (27-36, 50-58).
var dateNumFmts = map[int]bool{
	14: true, 15: true, 16: true, 17: true, 22: true,
	27: true, 28: true, 29: true, 30: true, 31: true, 34: true, 35: true, 36: true,
	50: true, 51: true, 52: true, 53: true, 54: true, 55: true, 56: true, 57: true, 58: true,
}

func isDateFormat(s *excelize.Style) bool {
	if s.CustomNumFmt == nil {
		return dateNumFmts[s.NumFmt]
	}
	return isDateCode(*s.CustomNumFmt)
}

// isDateCode reports whether a custom format code has a year or day token
// outside quoted text, escapes and [...] sections. A month alone does not
// count, since "m" is minutes too.
func isDateCode(code string) bool {
	for i := 0; i < len(code); i++ {
		switch code[i] {
		case '"', '[':
			end := byte('"')
			if code[i] == '[' {
				end = ']'
			}
			j := strings.IndexByte(code[i+1:], end)
			if j < 0 {
				return false
			}
			i += j + 1
		case '\\', '_', '*': // the next character is literal
			i++
		case 'y', 'Y', 'd', 'D':
			return true
		}
	}
	return false
}

// study reads the 01 sheet, which runs down: one KEY per row, its value in the
// 입력값 column.
func (r *reader) study(rows [][]string) kosac.Record {
	const sheet = kosac.SheetStudy
	keyCol, valCol := -1, -1
	if len(rows) > 0 {
		for c, h := range rows[0] {
			switch h = strings.TrimSpace(h); {
			case h == "KEY" && keyCol < 0:
				keyCol = c
			case h == "입력값" && valCol < 0:
				valCol = c
			}
		}
	}
	for _, h := range []struct {
		name string
		col  int
	}{{"KEY", keyCol}, {"입력값", valCol}} {
		if h.col < 0 {
			r.add(kosac.R, sheet, "", "", h.name, fmt.Sprintf("%s 시트 1행에 %s 머리글이 없습니다.", sheet, h.name))
		}
	}
	if keyCol < 0 || valCol < 0 {
		return nil
	}
	rec := kosac.Record{}
	for i := 1; i < len(rows); i++ {
		k := strings.TrimSpace(cellAt(rows[i], keyCol))
		_, dup := rec[k]
		switch {
		case k == "":
		case !slices.Contains(kosac.LayoutFor(r.version).Study, k):
			r.add(kosac.R, sheet, cellName(keyCol, i), k, "", fmt.Sprintf("%s 시트의 KEY %s는 양식에 없습니다.", sheet, k))
		case dup:
			r.add(kosac.R, sheet, cellName(keyCol, i), k, "", fmt.Sprintf("%s 시트에 KEY %s 행이 두 번 이상 있습니다.", sheet, k))
		default:
			rec[k] = kosac.Field{Value: cellAt(rows[i], valCol), Loc: kosac.Loc{Sheet: sheet, Cell: cellName(valCol, i)}}
		}
	}
	for _, k := range kosac.LayoutFor(r.version).Study {
		if _, ok := rec[k]; !ok {
			r.add(kosac.R, sheet, "", "", k, fmt.Sprintf("%s 시트에 KEY %s 행이 없습니다.", sheet, k))
		}
	}
	return rec
}

// table reads KEYs in row 1 and values from the selected layout's data row.
// Every record holds every KEY of the sheet.
func (r *reader) table(sheet string, keys []string, rows [][]string) []kosac.Record {
	cols := r.columns(sheet, keys, rows)
	var recs []kosac.Record
	for i := kosac.LayoutFor(r.version).FirstDataRow - 1; i < len(rows); i++ {
		rec, blank := kosac.Record{}, true
		for k, c := range cols {
			v := cellAt(rows[i], c)
			blank = blank && strings.TrimSpace(v) == ""
			rec[k] = kosac.Field{Value: v, Loc: kosac.Loc{Sheet: sheet, Cell: cellName(c, i)}}
		}
		// A person leaves blank rows between entries; a row with any KEY cell
		// filled is a row, so its empty required cells get reported.
		if !blank {
			recs = append(recs, rec)
		}
	}
	return recs
}

// columns maps each KEY in row 1 to its 0-based column. The KEY set must be
// the template's, in any order.
func (r *reader) columns(sheet string, keys []string, rows [][]string) map[string]int {
	cols := map[string]int{}
	if len(rows) > 0 {
		for c, h := range rows[0] {
			h = strings.TrimSpace(h)
			_, dup := cols[h]
			switch {
			case h == "":
			case !slices.Contains(keys, h):
				r.add(kosac.R, sheet, cellName(c, 0), h, "", fmt.Sprintf("%s 시트 1행의 KEY %s는 양식에 없습니다.", sheet, h))
			case dup:
				r.add(kosac.R, sheet, cellName(c, 0), h, "", fmt.Sprintf("%s 시트 1행에 KEY %s 열이 두 번 이상 있습니다.", sheet, h))
			default:
				cols[h] = c
			}
		}
	}
	for _, k := range keys {
		if _, ok := cols[k]; !ok {
			r.add(kosac.R, sheet, "", "", k, fmt.Sprintf("%s 시트 1행에 KEY %s 열이 없습니다.", sheet, k))
		}
	}
	return cols
}

// meta reads _meta (KEY in A, value in B) and checks the template's identity.
// Unknown keys are ignored.
func (r *reader) meta(rows [][]string, allowed []string) kosac.TemplateMeta {
	const sheet = kosac.SheetMeta
	var m kosac.TemplateMeta
	for i, row := range rows {
		f := kosac.Field{Value: cellAt(row, 1), Loc: kosac.Loc{Sheet: sheet, Cell: cellName(1, i)}}
		switch strings.TrimSpace(cellAt(row, 0)) {
		case "TEMPLATE_ID":
			m.ID = f
		case "TEMPLATE_VERSION":
			m.Version = f
		case "CREATED":
			m.Created = f
		}
	}
	switch id := strings.TrimSpace(m.ID.Value); id {
	case templateID:
	case altTemplateID:
		r.add(kosac.W, sheet, m.ID.Loc.Cell, id, templateID,
			fmt.Sprintf("TEMPLATE_ID가 %s로 적혀 있어 %s로 보고 받습니다.", id, templateID))
	default:
		r.add(kosac.R, sheet, m.ID.Loc.Cell, id, templateID,
			fmt.Sprintf("TEMPLATE_ID %q는 재단 코드북 양식의 ID가 아닙니다.", id))
	}
	if len(allowed) == 0 {
		allowed = []string{"3.2", "4.0"}
	}
	if v := strings.TrimSpace(m.Version.Value); !slices.Contains(allowed, v) {
		r.add(kosac.R, sheet, m.Version.Loc.Cell, v, strings.Join(allowed, ", "),
			fmt.Sprintf("TEMPLATE_VERSION %q는 받을 수 있는 양식 버전이 아닙니다.", v))
	}
	return m
}

func cellAt(row []string, c int) string {
	if c < len(row) {
		return row[c]
	}
	return ""
}

// cellName names the cell at 0-based (col, row).
func cellName(col, row int) string {
	n, _ := excelize.CoordinatesToCellName(col+1, row+1)
	return n
}

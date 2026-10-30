package template

import (
	"bytes"

	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/xuri/excelize/v2"
)

// Write fills a copy of the blank template with cb's template values, so the
// colors, lists, guidance rows and filter ranges stay the template's. Values
// the template has no column for (analysis meta, FK, operator input) are not
// written. Reading the result gives cb's record values back at new cells.
func Write(cb *kosac.Codebook) ([]byte, error) {
	f, err := excelize.OpenReader(bytes.NewReader(blank))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// The blank's own layout says where each KEY goes.
	r := &reader{f: f, dateStyle: map[int]bool{}}

	rows, _, err := r.rows(kosac.SheetStudy)
	if err != nil {
		return nil, err
	}
	// Cells are set in KEY order, never map order: each new string joins the
	// shared string table in the order it is set, so map order would change
	// the bytes from one write to the next.
	cells := r.study(rows)
	for _, k := range kosac.StudyKeys {
		at, ok := cells[k]
		if !ok {
			continue
		}
		// Every cell is set, even to empty, or the blank's list default would
		// read back in place of an empty value.
		v := cb.Study.Get(k)
		if v == "" {
			err = f.SetCellValue(at.Loc.Sheet, at.Loc.Cell, nil)
		} else {
			err = f.SetCellStr(at.Loc.Sheet, at.Loc.Cell, v)
		}
		if err != nil {
			return nil, err
		}
	}

	var files, vars, codes []kosac.Record
	for _, x := range cb.Files {
		files = append(files, x.Rec)
	}
	for _, x := range cb.Vars {
		vars = append(vars, x.Rec)
	}
	for _, x := range cb.Codes {
		codes = append(codes, x.Rec)
	}
	for _, t := range []struct {
		sheet string
		keys  []string
		recs  []kosac.Record
	}{{kosac.SheetFiles, kosac.FileKeys, files}, {kosac.SheetVars, kosac.VarKeys, vars}, {kosac.SheetCodes, kosac.CodeKeys, codes}} {
		rows, _, err := r.rows(t.sheet)
		if err != nil {
			return nil, err
		}
		cols := r.columns(t.sheet, t.keys, rows)
		for i, rec := range t.recs {
			for _, k := range t.keys {
				c, ok := cols[k]
				if !ok {
					continue
				}
				if v := rec.Get(k); v != "" {
					if err := f.SetCellStr(t.sheet, cellName(c, firstDataRow-1+i), v); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

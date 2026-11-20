package template

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/xuri/excelize/v2"
)

// Write fills a copy of the blank template with cb's template values, so the
// colors, lists, guidance rows and filter ranges stay the template's. Values
// the template has no column for (analysis meta, FK, operator input) are not
// written. Reading the result gives cb's record values back at new cells.
func Write(cb *kosac.Codebook) ([]byte, error) {
	if cb == nil {
		return nil, fmt.Errorf("코드북이 없습니다")
	}
	version := strings.TrimSpace(cb.Template.Version.Value)
	if version == "" {
		version = "4.0"
	}
	if version != "3.2" && version != "4.0" && version != kosac.Profile423 {
		return nil, fmt.Errorf("지원하지 않는 양식 버전: %s", version)
	}
	if version == "3.2" && cb.HasV4Fields() {
		return nil, fmt.Errorf("v4.0 항목을 v3.2 양식에 쓰면 유실됩니다")
	}
	layout := kosac.LayoutFor(version)
	if err := checkOutputFields(cb, layout, version); err != nil {
		return nil, err
	}
	source := blankV4
	if version == kosac.Profile423 {
		source = blank423
	}
	if version == "3.2" {
		source = blank
	}
	f, err := excelize.OpenReader(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// The blank's own layout says where each KEY goes.
	r := &reader{f: f, dateStyle: map[int]bool{}, version: version}

	rows, _, err := r.rows(kosac.SheetStudy)
	if err != nil {
		return nil, err
	}
	if cb.Template.Created.Value != "" {
		if err := f.SetCellStr(kosac.SheetMeta, "B3", cb.Template.Created.Value); err != nil {
			return nil, err
		}
	}
	// Cells are set in KEY order, never map order: each new string joins the
	// shared string table in the order it is set, so map order would change
	// the bytes from one write to the next.
	cells := r.study(rows)
	for _, k := range layout.Study {
		at, ok := cells[k]
		if !ok {
			continue
		}
		// Every cell is set, even to empty, or the blank's list default would
		// read back in place of an empty value.
		v := cb.Study.Get(k)
		if version == kosac.Profile423 && k == "RULE_VERSION" {
			if v != "" && v != kosac.RuleVersion423 {
				return nil, fmt.Errorf("v4.2.3 양식의 고정 규칙 버전은 %s입니다", kosac.RuleVersion423)
			}
			v = kosac.RuleVersion423
		}
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
	}{{kosac.SheetFiles, layout.Files, files}, {kosac.SheetVars, layout.Vars, vars}, {kosac.SheetCodes, layout.Codes, codes}, {kosac.SheetRecodes, kosac.RecodeKeys, cb.Recodes}} {
		if t.sheet == kosac.SheetRecodes && version != "4.0" {
			continue
		}
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
					if err := f.SetCellStr(t.sheet, cellName(c, layout.FirstDataRow-1+i), v); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	for _, sheet := range slices.Sorted(maps.Keys(cb.Supplemental)) {
		if sheet != "10_미확인사유" && sheet != "11_변환내역" {
			return nil, fmt.Errorf("지원하지 않는 보조 시트: %s", sheet)
		}
		if _, err := f.NewSheet(sheet); err != nil {
			return nil, err
		}
		for i, row := range cb.Supplemental[sheet] {
			for j, value := range row {
				if err := f.SetCellStr(sheet, cellName(j, i), value); err != nil {
					return nil, err
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

func checkOutputFields(cb *kosac.Codebook, layout kosac.Layout, version string) error {
	if len(cb.Recodes) > 0 && version != "4.0" || len(cb.Supplemental) > 0 && version != kosac.Profile423 {
		return fmt.Errorf("양식 %s에 이력 시트를 기록할 수 없습니다", version)
	}
	// Only fields from a different supported layout represent a lossy version change.
	alternate := kosac.LayoutFor(kosac.Profile423)
	if version == kosac.Profile423 {
		alternate = kosac.LayoutFor("4.0")
	}
	check := func(r kosac.Record, keys, other []string) error {
		for _, key := range other {
			if r.Get(key) != "" && !slices.Contains(keys, key) {
				return fmt.Errorf("양식 %s에 %s 항목이 없습니다", version, key)
			}
		}
		return nil
	}
	if err := check(cb.Study, layout.Study, alternate.Study); err != nil {
		return err
	}
	for _, f := range cb.Files {
		if err := check(f.Rec, layout.Files, alternate.Files); err != nil {
			return err
		}
	}
	for _, v := range cb.Vars {
		if err := check(v.Rec, layout.Vars, alternate.Vars); err != nil {
			return err
		}
	}
	for _, c := range cb.Codes {
		if err := check(c.Rec, layout.Codes, alternate.Codes); err != nil {
			return err
		}
	}
	return nil
}

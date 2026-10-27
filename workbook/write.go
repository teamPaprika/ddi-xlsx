package workbook

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"unicode/utf16"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/schema"
	"github.com/xuri/excelize/v2"
)

// cellLimit is the most characters (UTF-16 units) an Excel cell holds.
// excelize truncates longer values without an error.
const cellLimit = 32767

// textFormat is the built-in "@" number format. On a column it keeps values
// typed into new cells as text, so a code like 01 stays 01.
const textFormat = 49

type sheet struct {
	name    string
	headers []string
	rows    [][]string
}

func (s *sheet) add(r map[string]string) {
	row := make([]string, len(s.headers))
	for i, h := range s.headers {
		row[i] = r[h]
	}
	s.rows = append(s.rows, row)
}

// ToXLSX converts a DDI Codebook 2.5 document, XML or JSON, to a workbook. The
// warnings are those from reading the document.
func ToXLSX(input []byte) ([]byte, []string, error) {
	root, warns, err := ddi.Parse(input)
	if err != nil {
		return nil, nil, err
	}
	sheets, more, err := buildSheets(root)
	if err != nil {
		return nil, nil, err
	}
	out, err := writeSheets(sheets)
	if err != nil {
		return nil, nil, err
	}
	return out, dedupe(append(warns, more...)), nil
}

func buildSheets(root *ddi.Node) ([]*sheet, []string, error) {
	vars := &sheet{name: varSheet, headers: varHeaders}
	cats := &sheet{name: catSheet, headers: catHeaders}
	grps := &sheet{name: grpSheet, headers: grpHeaders}
	doc := &sheet{name: docSheet, headers: docHeaders}
	var dataDscr *ddi.Node
	for _, c := range root.Children {
		if c.Name == "dataDscr" {
			if dataDscr != nil {
				return nil, nil, errors.New("codeBook has more than one dataDscr; only one is supported")
			}
			dataDscr = c
		}
	}
	var warns []string
	if dataDscr != nil {
		var err error
		if warns, err = splitMergePoints(dataDscr, root.Attrs, "dataDscr"); err != nil {
			return nil, nil, err
		}
		seen := map[string]int{}
		ids := map[string]bool{}
		for _, c := range dataDscr.Children {
			seen[c.Name]++
			var r map[string]string
			switch c.Name {
			case "var":
				var cs []*ddi.Node
				r, cs = varRow(c)
				if err := rawCatgryFits(c, r["xml"], fmt.Sprintf("dataDscr/var[%d]", seen["var"])); err != nil {
					return nil, nil, err
				}
				name, _ := c.Attr("name")
				for _, cg := range cs {
					cr := catRow(cg)
					cr["varID"], cr["varName"] = r["ID"], name
					cats.add(cr)
				}
				vars.add(r)
			case "varGrp":
				r = grpRow(c)
				grps.add(r)
			default:
				continue
			}
			// Categories and Groups refer to rows by ID, and the ID column cannot
			// hold an empty or space-padded value.
			if r["ID"] == "" {
				return nil, nil, fmt.Errorf("dataDscr/%s[%d]: ID is missing, empty or padded with spaces", c.Name, seen[c.Name])
			}
			// A repeated ID would attach Categories rows to the wrong var, and var
			// and varGrp IDs are both xs:ID.
			if ids[r["ID"]] {
				return nil, nil, fmt.Errorf("dataDscr/%s[%d]: ID %s repeats", c.Name, seen[c.Name], r["ID"])
			}
			ids[r["ID"]] = true
		}
	}
	outline(doc, root, 0)
	return []*sheet{vars, cats, grps, doc}, warns, nil
}

// splitMergePoints applies unraw to n and below it to each var, catgry and
// varGrp, the parents the reader re-sorts. path names n in errors and warnings.
func splitMergePoints(n *ddi.Node, ns []ddi.Attr, path string) ([]string, error) {
	warns := unraw(n, ns)
	// Still kept as written, n goes whole into one cell. Said here, the error
	// names the cause instead of only the cell.
	if len(n.Raw) > cellLimit && utf16Len(n.Raw) > cellLimit {
		return nil, fmt.Errorf("%s: <%s> holds text between its child elements, so it is kept as written in one cell, "+
			"and its %d characters are more than the %d a cell can hold", path, n.Name, utf16Len(n.Raw), cellLimit)
	}
	for i, w := range warns {
		warns[i] = path + ": " + w
	}
	ns = append(slices.Clip(ns), n.Attrs...)
	seen := map[string]int{}
	for _, c := range n.Children {
		seen[c.Name]++
		if resorted(n.Name, c.Name) {
			w, err := splitMergePoints(c, ns, fmt.Sprintf("%s/%s[%d]", path, c.Name, seen[c.Name]))
			if err != nil {
				return nil, err
			}
			warns = append(warns, w...)
		}
	}
	return warns, nil
}

// rawCatgryFits names the cause when v's xml cell is too long and v has a
// catgry kept as written: that catgry and every later one go to the cell
// instead of the Categories sheet. path names v.
func rawCatgryFits(v *ddi.Node, cell, path string) error {
	if len(cell) <= cellLimit || utf16Len(cell) <= cellLimit {
		return nil
	}
	k := 0
	for _, c := range v.Children {
		if c.Name != "catgry" {
			continue
		}
		if k++; c.Raw != "" {
			return fmt.Errorf("%s/catgry[%d]: <catgry> holds text between its child elements, so it and the catgry after it "+
				"are kept in the xml cell of their var, whose %d characters are more than the %d a cell can hold",
				path, k, utf16Len(cell), cellLimit)
		}
	}
	return nil
}

func writeSheets(sheets []*sheet) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	text, err := f.NewStyle(&excelize.Style{NumFmt: textFormat})
	if err != nil {
		return nil, err
	}
	wrap, err := f.NewStyle(&excelize.Style{NumFmt: textFormat, Alignment: &excelize.Alignment{WrapText: true}})
	if err != nil {
		return nil, err
	}
	head, err := f.NewStyle(&excelize.Style{NumFmt: textFormat, Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	indents := map[int]int{0: text}
	indent := func(level int) (int, error) {
		if id, ok := indents[level]; ok {
			return id, nil
		}
		id, err := f.NewStyle(&excelize.Style{NumFmt: textFormat, Alignment: &excelize.Alignment{Indent: level}})
		indents[level] = id
		return id, err
	}
	for i, s := range sheets {
		if i == 0 {
			err = f.SetSheetName(f.GetSheetName(0), s.name)
		} else {
			_, err = f.NewSheet(s.name)
		}
		if err != nil {
			return nil, err
		}
		// The stream writer keeps a NADA-size codebook (tens of thousands of
		// rows) to seconds; it needs column settings before the first row.
		sw, err := f.NewStreamWriter(s.name)
		if err != nil {
			return nil, err
		}
		header := make([]any, len(s.headers))
		for c, h := range s.headers {
			style, width := text, float64(defaultWidth)
			if wrapCols[h] {
				style = wrap
			}
			if w, ok := widths[h]; ok {
				width = w
			}
			if err := sw.SetColStyle(c+1, c+1, style); err != nil {
				return nil, err
			}
			if err := sw.SetColWidth(c+1, c+1, width); err != nil {
				return nil, err
			}
			header[c] = h
		}
		if err := sw.SetPanes(&excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
			return nil, err
		}
		if err := sw.SetRow("A1", header, excelize.RowOpts{StyleID: head}); err != nil {
			return nil, err
		}
		for r, row := range s.rows {
			vals := make([]any, len(row))
			for c, v := range row {
				if len(v) > cellLimit && utf16Len(v) > cellLimit {
					cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
					return nil, fmt.Errorf("%s!%s: %d characters, more than the %d a cell can hold", s.name, cell, utf16Len(v), cellLimit)
				}
				// A nil value writes no cell at all; the column style still applies.
				if v != "" {
					vals[c] = v
				}
			}
			if s.name == docSheet {
				level, _ := strconv.Atoi(row[0])
				id, err := indent(level)
				if err != nil {
					return nil, err
				}
				vals[1] = excelize.Cell{StyleID: id, Value: row[1]}
			}
			cell, _ := excelize.CoordinatesToCellName(1, r+2)
			if err := sw.SetRow(cell, vals); err != nil {
				return nil, err
			}
		}
		// A table keeps the header row out of a sort and lets the reader find
		// the header below a title row. Its style leaves the bold header visible.
		end, _ := excelize.CoordinatesToCellName(len(s.headers), len(s.rows)+1)
		if err := sw.AddTable(&excelize.Table{Range: "A1:" + end, Name: tableName(s.name), StyleName: "TableStyleLight1"}); err != nil {
			return nil, err
		}
		if err := addDropdowns(f, s); err != nil {
			return nil, err
		}
		if err := sw.Flush(); err != nil {
			return nil, err
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// addDropdowns lists the XSD's values in each column whose attribute the XSD
// enumerates. Another value only warns: the input may hold one already, and
// the workbook keeps it. It must run before the sheet's stream writer flushes.
func addDropdowns(f *excelize.File, s *sheet) error {
	for _, src := range attrSources[s.name] {
		for _, c := range src.cols {
			vals := schema.AttrEnum(src.element, c.attr)
			if vals == nil {
				continue
			}
			col, _ := excelize.ColumnNumberToName(slices.Index(s.headers, c.col) + 1)
			dv := excelize.NewDataValidation(true)
			dv.Sqref = fmt.Sprintf("%s2:%s%d", col, col, excelize.TotalRows)
			if err := dv.SetDropList(vals); err != nil {
				return err
			}
			dv.SetError(excelize.DataValidationErrorStyleWarning, "Not a DDI value",
				"DDI 2.5 does not list this value. It is kept as typed.")
			if err := f.AddDataValidation(s.name, dv); err != nil {
				return err
			}
		}
	}
	return nil
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

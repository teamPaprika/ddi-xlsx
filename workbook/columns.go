// Package workbook converts a DDI Codebook 2.5 tree to an editable xlsx
// workbook and back.
package workbook

import (
	"encoding/xml"
	"slices"
	"strings"
	"unicode"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

const (
	varSheet = "Variables"
	catSheet = "Categories"
	grpSheet = "Groups"
	docSheet = "Document"
)

// The reader finds columns by these header names.
var (
	varHeaders = []string{"ID", "name", "label", "question", "questionID", "questionRef", "responseDomainType", "intrvl",
		"format", "formatSchema", "formatCategory", "notes", "files", "locations", "attributes", "xml"}
	catHeaders = []string{"varID", "varName", "code", "label", "missing", "freq", "attributes", "xml"}
	grpHeaders = []string{"ID", "type", "name", "label", "vars", "attributes", "xml"}
	docHeaders = []string{"level", "element", "attributes", "text", "kind"}
	// keyHeaders are the columns a sheet's rows cannot be read without.
	keyHeaders = map[string][]string{varSheet: {"ID"}, catSheet: {"varID"}, grpSheet: {"ID"}, docSheet: {"level", "element"}}
)

// attrCol maps an attribute to the column that holds it. Each list is in the
// order the reader puts the attributes back; var's is Walla's, so a Walla
// file in this tool's layout converts back byte for byte.
type attrCol struct{ attr, col string }

var (
	varAttrs    = []attrCol{{"ID", "ID"}, {"name", "name"}, {"files", "files"}, {"intrvl", "intrvl"}, {"qstn", "questionRef"}}
	catAttrs    = []attrCol{{"missing", "missing"}}
	grpAttrs    = []attrCol{{"ID", "ID"}, {"type", "type"}, {"var", "vars"}, {"name", "name"}}
	qstnAttrs   = []attrCol{{"ID", "questionID"}, {"responseDomainType", "responseDomainType"}}
	formatAttrs = []attrCol{{"type", "format"}, {"schema", "formatSchema"}, {"category", "formatCategory"}}
)

// attrSources names, per sheet, the element whose attributes each list puts
// in columns, so a column can offer the values the XSD enumerates.
var attrSources = map[string][]struct {
	element string
	cols    []attrCol
}{
	varSheet: {{"var", varAttrs}, {"qstn", qstnAttrs}, {"varFormat", formatAttrs}},
	catSheet: {{"catgry", catAttrs}},
	grpSheet: {{"varGrp", grpAttrs}},
}

// tableName is the name of the Excel table that holds a sheet's rows.
func tableName(sheet string) string { return "ddi" + sheet }

var (
	wrapCols = map[string]bool{"question": true, "notes": true, "xml": true, "text": true}
	widths   = map[string]float64{"label": 40, "question": 50, "notes": 30, "attributes": 30, "xml": 60,
		"text": 60, "element": 24, "vars": 30}
)

const defaultWidth = 14

// cellValue reports whether v survives a column: the reader passes cells
// through ddi.NormalizeText, and an empty cell means the value is absent.
func cellValue(v string) bool { return v != "" && ddi.NormalizeText(v) == v }

// takeAttrs puts each attribute that has a column into r and returns the rest
// in input order. A repeated name keeps its first value in the column.
func takeAttrs(r map[string]string, attrs []ddi.Attr, cols []attrCol) []ddi.Attr {
	var rest []ddi.Attr
	for _, a := range attrs {
		i := slices.IndexFunc(cols, func(c attrCol) bool { return c.attr == a.Name })
		if i >= 0 && r[cols[i].col] == "" && cellValue(a.Value) {
			r[cols[i].col] = a.Value
		} else {
			rest = append(rest, a)
		}
	}
	return rest
}

// takeAll is takeAttrs for an element that columns hold only as a whole.
func takeAll(r map[string]string, attrs []ddi.Attr, cols []attrCol) bool {
	t := map[string]string{}
	if len(takeAttrs(t, attrs, cols)) > 0 {
		return false
	}
	for k, v := range t {
		r[k] = v
	}
	return true
}

// plainText returns n's text when a text column can hold n. Text implies no
// children and no Raw.
func plainText(n *ddi.Node) (string, bool) {
	if n == nil || len(n.Attrs) > 0 || !cellValue(n.Text) {
		return "", false
	}
	return n.Text, true
}

// only returns the child named name when there is exactly one.
func only(children []*ddi.Node, name string) *ddi.Node {
	var found *ddi.Node
	for _, c := range children {
		if c.Name == name {
			if found != nil {
				return nil
			}
			found = c
		}
	}
	return found
}

// split builds one row: the columns by header name, and the children the
// columns took, so that the rest goes to the xml cell.
type split struct {
	r    map[string]string
	used map[*ddi.Node]bool
}

func newSplit(n *ddi.Node, cols []attrCol) *split {
	s := &split{r: map[string]string{}, used: map[*ddi.Node]bool{}}
	s.r["attributes"] = ddi.FormatAttrs(takeAttrs(s.r, n.Attrs, cols))
	return s
}

func (s *split) text(n *ddi.Node, col string) {
	if t, ok := plainText(n); ok {
		s.r[col] = t
		s.used[n] = true
	}
}

// finish puts the rest of n's content in the xml cell. For n with text or kept
// as written that is what sits between n's tags, so the reader can tell it
// apart by parsing the cell as n's content.
func (s *split) finish(n *ddi.Node) map[string]string {
	switch {
	case n.Raw != "":
		s.r["xml"] = n.Raw
	case n.Text != "":
		var b strings.Builder
		xml.EscapeText(&b, []byte(n.Text))
		s.r["xml"] = b.String()
	default:
		var rest []*ddi.Node
		for _, c := range n.Children {
			if !s.used[c] {
				rest = append(rest, c)
			}
		}
		s.r["xml"] = ddi.WriteFragment(rest)
	}
	return s.r
}

// varRow splits v into its Variables row and the catgry that get Categories rows.
func varRow(v *ddi.Node) (map[string]string, []*ddi.Node) {
	s := newSplit(v, varAttrs)
	s.text(only(v.Children, "labl"), "label")
	if q := only(v.Children, "qstn"); q != nil && len(q.Children) == 1 && q.Children[0].Name == "qstnLit" {
		if t, ok := plainText(q.Children[0]); ok && takeAll(s.r, q.Attrs, qstnAttrs) {
			s.r["question"] = t
			s.used[q] = true
		}
	}
	// Without an attribute the columns would be empty, which reads as no varFormat.
	if f := only(v.Children, "varFormat"); f != nil && len(f.Attrs) > 0 && f.Text == "" && f.Raw == "" &&
		len(f.Children) == 0 && takeAll(s.r, f.Attrs, formatAttrs) {
		s.used[f] = true
	}
	// The reader puts column notes and Categories rows ahead of the notes and
	// catgry in the xml cell, so only a leading run of each may leave the cell.
	var notes []string
	var cats, locs []*ddi.Node
	notesRun, catsRun := true, true
	for _, c := range v.Children {
		switch c.Name {
		case "location":
			locs = append(locs, c)
		case "notes":
			t, ok := plainText(c)
			if notesRun = notesRun && ok && !strings.Contains(t, "\n"); notesRun {
				notes = append(notes, t)
				s.used[c] = true
			}
		case "catgry":
			if catsRun = catsRun && c.Raw == "" && c.Text == ""; catsRun {
				cats = append(cats, c)
				s.used[c] = true
			}
		}
	}
	s.r["notes"] = strings.Join(notes, "\n")
	// The reader puts column locations ahead of those in the xml cell, so a
	// var's locations leave the cell all together or not at all.
	if !slices.ContainsFunc(locs, func(c *ddi.Node) bool { return !fileOnly(c) }) {
		ids := make([]string, len(locs))
		for i, c := range locs {
			ids[i] = c.Attrs[0].Value
			s.used[c] = true
		}
		s.r["locations"] = strings.Join(ids, " ")
	}
	return s.finish(v), cats
}

// fileOnly reports whether the locations column can hold l: it is empty but
// for a fileid that survives as one token of the cell, which the reader
// splits on whitespace.
func fileOnly(l *ddi.Node) bool {
	return len(l.Attrs) == 1 && l.Attrs[0].Name == "fileid" && cellValue(l.Attrs[0].Value) &&
		!strings.ContainsFunc(l.Attrs[0].Value, unicode.IsSpace) && l.Text == "" && l.Raw == "" && len(l.Children) == 0
}

// catRow is the Categories row of c, without varID and varName.
func catRow(c *ddi.Node) map[string]string {
	s := newSplit(c, catAttrs)
	s.text(only(c.Children, "catValu"), "code")
	s.text(only(c.Children, "labl"), "label")
	isFreq := func(n *ddi.Node) bool {
		return n.Name == "catStat" && len(n.Attrs) == 1 && n.Attrs[0] == ddi.Attr{Name: "type", Value: "freq"}
	}
	// The freq column is read back ahead of the other catStat, so it may hold
	// only the first one.
	if i := slices.IndexFunc(c.Children, func(n *ddi.Node) bool { return n.Name == "catStat" }); i >= 0 {
		if st := c.Children[i]; isFreq(st) && cellValue(st.Text) && countFunc(c.Children, isFreq) == 1 {
			s.r["freq"] = st.Text
			s.used[st] = true
		}
	}
	return s.finish(c)
}

func grpRow(g *ddi.Node) map[string]string {
	s := newSplit(g, grpAttrs)
	s.text(only(g.Children, "labl"), "label")
	return s.finish(g)
}

func countFunc(nodes []*ddi.Node, f func(*ddi.Node) bool) int {
	n := 0
	for _, c := range nodes {
		if f(c) {
			n++
		}
	}
	return n
}

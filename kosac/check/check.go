// Package check runs the foundation's codebook rules C01–C18 on a KOSAC
// codebook model.
package check

import (
	"cmp"
	"slices"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/xuri/excelize/v2"
)

// Check runs every rule, also after a rejection, so one report lists all
// issues. The rule standards are the engine's own; nothing is read from the
// workbook's 구분 column or data validations, which a contractor can edit.
// A nil cb, which template.Read returns after an S05 rejection, gives nil.
func Check(cb *kosac.Codebook, ref kosac.RefData) []kosac.Issue {
	if cb == nil {
		return nil
	}
	c := newChecker(cb, ref)
	for _, rule := range []func(){
		c.c01, c.c02, c.c03, c.c04, c.c05, c.c06, c.c07, c.c08, c.c09,
		c.c10, c.c11, c.c12, c.c13, c.c14, c.c15, c.c16, c.c17, c.c18,
		c.unverifiedRows,
	} {
		rule()
	}
	sortIssues(c.issues)
	return c.issues
}

type key struct{ file, name string }

type checker struct {
	cb     *kosac.Codebook
	ref    kosac.RefData
	vars   map[key]*kosac.Var    // first row of each (FILE_NO, VAR_NAME)
	byName map[string]*kosac.Var // first row of each VAR_NAME in any file
	codes  map[key][]*kosac.Code
	issues []kosac.Issue
}

// newChecker indexes the rows once; the model's lookups scan, which is too
// slow across thousands of vars and codes.
func newChecker(cb *kosac.Codebook, ref kosac.RefData) *checker {
	c := &checker{cb: cb, ref: ref, vars: map[key]*kosac.Var{}, byName: map[string]*kosac.Var{}, codes: map[key][]*kosac.Code{}}
	for _, v := range cb.Vars {
		k := varKey(v.Rec)
		if _, ok := c.vars[k]; !ok {
			c.vars[k] = v
		}
		if _, ok := c.byName[k.name]; !ok {
			c.byName[k.name] = v
		}
	}
	for _, cd := range cb.Codes {
		k := varKey(cd.Rec)
		c.codes[k] = append(c.codes[k], cd)
	}
	return c
}

// varKey joins rows by the values as written, as the model's lookups do.
func varKey(r kosac.Record) key { return key{r.Get("FILE_NO"), r.Get("VAR_NAME")} }

// add reports an issue on the cell of key in r; the machine keys come from r.
func (c *checker) add(rule string, lv kosac.Level, r kosac.Record, k, actual, expected, msg string) {
	loc := r[k].Loc
	c.issues = append(c.issues, kosac.Issue{
		Rule: rule, Level: lv, Sheet: loc.Sheet, Cell: loc.Cell,
		FileNo: r.Get("FILE_NO"), VarName: r.Get("VAR_NAME"), Code: r.Get("CODE"),
		Actual: actual, Expected: expected, Message: msg,
	})
}

// level returns the kosac:measLevel of the var name in file fileNo, and
// whether it exists.
func (c *checker) level(fileNo, name string) (string, bool) {
	return levelOf(name, c.vars[key{fileNo, name}])
}

// studyLevel is level in any file, for the 01 sheet, which names no file.
func (c *checker) studyLevel(name string) (string, bool) {
	return levelOf(name, c.byName[name])
}

// levelOf takes a system var as present in every file.
func levelOf(name string, v *kosac.Var) (string, bool) {
	for _, s := range kosac.SystemVars {
		if s.Name == name {
			return s.MeasLevel, true
		}
	}
	if v == nil || blank(name) {
		return "", false
	}
	return kosac.MeasLevels[v.Rec.Get("MEAS_LEVEL")], true
}

// codesOf returns v's 04 rows, and those that are not missing codes.
func (c *checker) codesOf(v *kosac.Var) (all, valid []*kosac.Code) {
	all = c.codes[varKey(v.Rec)]
	for _, cd := range all {
		if !isMissingCode(cd.Rec.Get("CODE")) {
			valid = append(valid, cd)
		}
	}
	return all, valid
}

func isMissingCode(s string) bool {
	return slices.ContainsFunc(kosac.MissingCodes, func(m kosac.MissingCode) bool { return m.Code == s })
}

// A blank cell is one a person sees as empty, so spaces alone count.
func blank(s string) bool { return strings.TrimSpace(s) == "" }

func unverified(s string) bool {
	_, ok := kosac.Unverified(s)
	return ok
}

// checkable reports whether a value is there to judge: format and reference
// checks skip a blank cell (the required checks report it) and a "확인 불가"
// cell (its owning rule warns).
func checkable(s string) bool { return !blank(s) && !unverified(s) }

var sheetOrder = []string{kosac.SheetStudy, kosac.SheetFiles, kosac.SheetVars, kosac.SheetCodes}

// sortIssues orders by rule, sheet, row and column. The sort is stable, so
// issues without a cell keep the order the rules gave them.
func sortIssues(is []kosac.Issue) {
	pos := func(i kosac.Issue) (sheet, row, col int) {
		col, row, _ = excelize.CellNameToCoordinates(i.Cell)
		return slices.Index(sheetOrder, i.Sheet), row, col
	}
	slices.SortStableFunc(is, func(a, b kosac.Issue) int {
		as, ar, ac := pos(a)
		bs, br, bc := pos(b)
		return cmp.Or(strings.Compare(a.Rule, b.Rule), cmp.Compare(as, bs), cmp.Compare(ar, br), cmp.Compare(ac, bc))
	})
}

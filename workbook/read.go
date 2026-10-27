package workbook

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/schema"
	"github.com/xuri/excelize/v2"
)

// FromXLSX converts a workbook that ToXLSX wrote, edited or not, back to DDI
// Codebook 2.5. The XML and the JSON hold the same tree. Errors about a cell
// start with its address, such as "Categories!A12: ".
func FromXLSX(x []byte) (xmlOut, jsonOut []byte, warnings []string, err error) {
	f, err := excelize.OpenReader(bytes.NewReader(x), unzipLimits(len(x)))
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()
	r := &reader{f: f, heads: tableHeads(x)}
	root, err := r.read()
	if err != nil {
		return nil, nil, nil, err
	}
	// A cell's content counts depth from its own element, so the tree can
	// come out deeper than the reader of the output accepts.
	if p, deep := deepPath(root, 0); deep {
		p = append([]string{root.Name}, p...)
		if len(p) > 6 {
			p = append(p[:4:4], "…", p[len(p)-1])
		}
		return nil, nil, nil, fmt.Errorf("%s: elements nest more than %d deep", strings.Join(p, "/"), ddi.MaxDepth)
	}
	group(root)
	return ddi.WriteXML(root), ddi.WriteJSON(root), dedupe(r.warns), nil
}

// deepPath returns the path below n, which is at depth, to the first element
// nested deeper than ddi.MaxDepth, and whether there is one. Content kept as
// written counts too.
func deepPath(n *ddi.Node, depth int) ([]string, bool) {
	if depth > ddi.MaxDepth || n.Raw != "" && depth+rawDepth(n.Raw) > ddi.MaxDepth {
		return nil, true
	}
	for i, c := range n.Children {
		if p, deep := deepPath(c, depth+1); deep {
			k := 1 + countFunc(n.Children[:i], func(s *ddi.Node) bool { return s.Name == c.Name })
			return append([]string{fmt.Sprintf("%s[%d]", c.Name, k)}, p...), true
		}
	}
	return nil, false
}

// rawDepth is how many elements deep raw nests.
func rawDepth(raw string) int {
	d := xml.NewDecoder(strings.NewReader(raw))
	depth, deepest := 0, 0
	for {
		tok, err := d.RawToken()
		if err != nil {
			return deepest
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
			deepest = max(deepest, depth)
		case xml.EndElement:
			depth--
		}
	}
}

// unzipLimits bounds what excelize allocates while it unzips, which it sizes
// from what each zip entry declares: a small crafted file could ask for
// gigabytes and, in wasm, end the whole program. A genuine workbook unzips to
// far less than 100 times its size.
func unzipLimits(size int) excelize.Options {
	l := max(64<<20, 100*int64(size))
	return excelize.Options{UnzipSizeLimit: l, UnzipXMLSizeLimit: min(l, excelize.StreamChunkSize)}
}

type reader struct {
	f     *excelize.File
	heads map[string]tableRows // by table name
	warns []string
}

// tableRows are the Excel rows of a table's header and of its totals row, 0
// when it has none.
type tableRows struct{ head, totals int }

// tablePartLimit is far above any table part, which holds only the table's
// name, range and column names.
const tablePartLimit = 1 << 20

// tableHeads maps each table in the workbook x to the rows of its header and
// totals row. It reads only the table parts: excelize finds a sheet's tables
// by decoding the whole sheet, which doubles the time and memory of reading a
// large workbook. Excel keeps table names unique in a workbook, so a name
// identifies its sheet. A part that cannot be read is skipped, and its sheet
// falls back to row 1.
func tableHeads(x []byte) map[string]tableRows {
	z, err := zip.NewReader(bytes.NewReader(x), int64(len(x)))
	if err != nil {
		return nil
	}
	heads := map[string]tableRows{}
	for _, zf := range z.File {
		// A table part holds a few kilobytes. A larger one is skipped unread,
		// so a crafted part cannot make this pass slow; its sheet falls back
		// to row 1.
		if zf.UncompressedSize64 > tablePartLimit || !strings.HasPrefix(zf.Name, "xl/tables/") {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			continue
		}
		var t struct {
			Name   string `xml:"name,attr"`
			Ref    string `xml:"ref,attr"`
			Totals int    `xml:"totalsRowCount,attr"`
		}
		err = xml.NewDecoder(io.LimitReader(rc, tablePartLimit)).Decode(&t)
		rc.Close()
		first, last, _ := strings.Cut(t.Ref, ":")
		_, head, herr := excelize.CellNameToCoordinates(first)
		if err != nil || herr != nil {
			continue
		}
		rows := tableRows{head: head}
		// Excel's Total Row is the table's last row.
		if _, end, err := excelize.CellNameToCoordinates(last); err == nil && t.Totals > 0 && end > head {
			rows.totals = end
		}
		heads[t.Name] = rows
	}
	return heads
}

func (r *reader) warn(format string, a ...any) { r.warns = append(r.warns, fmt.Sprintf(format, a...)) }

// warnAt adds warnings about the cell at addr.
func (r *reader) warnAt(addr string, warns []string) {
	for _, w := range warns {
		r.warn("%s: %s", addr, w)
	}
}

func (r *reader) read() (*ddi.Node, error) {
	root, dd, err := r.document()
	if err != nil {
		return nil, err
	}
	// Prefixes in an xml cell resolve against the elements around it.
	ns := root.Attrs
	if dd != nil {
		ns = append(slices.Clip(ns), dd.Attrs...)
	}
	var tables [3]*table
	for i, s := range []struct {
		name    string
		headers []string
	}{{varSheet, varHeaders}, {catSheet, catHeaders}, {grpSheet, grpHeaders}} {
		if tables[i], err = r.table(s.name, s.headers); err != nil {
			return nil, err
		}
	}
	vars, cats, grps := tables[0], tables[1], tables[2]

	// var, varGrp and qstn IDs are all xs:ID, so they must differ from each
	// other too: an assigned ID skips the typed IDs of both sheets and the
	// typed questionIDs, and ids checks both sheets' IDs together.
	used := map[string]bool{}
	for _, t := range []*table{vars, grps} {
		for i := 1; i < len(t.rows); i++ {
			used[t.text(i, "ID")] = true
		}
	}
	for i := 1; i < len(vars.rows); i++ {
		used[vars.text(i, "questionID")] = true
	}
	if err := vars.fillIDs("V", used); err != nil {
		return nil, err
	}
	if err := grps.fillIDs("VG", used); err != nil {
		return nil, err
	}

	ids := map[string]string{}
	byID := map[string]*part{}
	byName := map[string][]string{} // var name -> the IDs of the vars with it
	var varRows []*part
	for i := 1; i < len(vars.rows); i++ {
		if vars.blank(i) {
			continue
		}
		id, err := vars.id(i, dd, ids)
		if err != nil {
			return nil, err
		}
		p, err := r.part(vars, i, "var", varAttrs, ns)
		if err != nil {
			return nil, err
		}
		if p.kids, err = varColumns(vars, i); err != nil {
			return nil, err
		}
		byID[id] = p
		if name, ok := p.n.Attr("name"); ok {
			byName[name] = append(byName[name], id)
		}
		varRows = append(varRows, p)
	}
	for i := 1; i < len(cats.rows); i++ {
		if cats.blank(i) {
			continue
		}
		id := cats.text(i, "varID")
		if id == "" {
			name := cats.text(i, "varName")
			switch n := len(byName[name]); {
			case name == "":
				return nil, fmt.Errorf("%s: a %s row needs a varID or a varName", cats.at(i, "varID"), catSheet)
			case n == 0:
				return nil, fmt.Errorf("%s: varName %q matches no name on the %s sheet", cats.at(i, "varName"), name, varSheet)
			case n > 1:
				return nil, fmt.Errorf("%s: varName %q is the name of %d variables on the %s sheet; write the varID instead",
					cats.at(i, "varName"), name, n, varSheet)
			}
			id = byName[name][0]
		}
		v := byID[id]
		if v == nil {
			return nil, fmt.Errorf("%s: varID %q matches no ID on the %s sheet", cats.at(i, "varID"), id, varSheet)
		}
		p, err := r.part(cats, i, "catgry", catAttrs, append(slices.Clip(ns), v.n.Attrs...))
		if err != nil {
			return nil, err
		}
		p.kids = slices.Concat(el("catValu", cats.text(i, "code")), el("labl", cats.text(i, "label")),
			el("catStat", cats.text(i, "freq"), ddi.Attr{Name: "type", Value: "freq"}))
		c, err := p.done()
		if err != nil {
			return nil, err
		}
		v.kids = append(v.kids, c)
	}
	var groups []*part
	for i := 1; i < len(grps.rows); i++ {
		if grps.blank(i) {
			continue
		}
		if _, err := grps.id(i, dd, ids); err != nil {
			return nil, err
		}
		vs := strings.Fields(grps.text(i, "vars"))
		renamed := false
		for k, v := range vs {
			switch match := byName[v]; {
			case byID[v] != nil:
			case len(match) == 1:
				vs[k], renamed = match[0], true
			default:
				r.warn("%s: %s is not an ID or the name of exactly one variable on the %s sheet", grps.at(i, "vars"), v, varSheet)
			}
		}
		if renamed {
			grps.set(i, "vars", strings.Join(vs, " "))
		}
		p, err := r.part(grps, i, "varGrp", grpAttrs, ns)
		if err != nil {
			return nil, err
		}
		p.kids = el("labl", grps.text(i, "label"))
		groups = append(groups, p)
	}
	if dd == nil {
		return root, nil
	}
	sortNested(dd)
	// A var or varGrp typed on the Document sheet gets a row on the next
	// conversion, and then dataDscr is sorted.
	if len(groups)+len(varRows) == 0 && !slices.ContainsFunc(dd.Children, func(c *ddi.Node) bool { return resorted(dd.Name, c.Name) }) {
		return root, nil
	}
	var kids []*ddi.Node
	for _, p := range append(groups, varRows...) {
		n, err := p.done()
		if err != nil {
			return nil, err
		}
		kids = append(kids, n)
	}
	dd.Children = sortChildren(dd.Name, kids, dd.Children)
	return root, nil
}

// document builds the tree the Document outline describes and returns its root
// and the dataDscr the other sheets fill, if there is one.
func (r *reader) document() (root, dataDscr *ddi.Node, err error) {
	t, err := r.table(docSheet, docHeaders)
	if err != nil {
		return nil, nil, err
	}
	var stack []*ddi.Node // stack[l] is the latest row at level l
	for i := 1; i < len(t.rows); i++ {
		if t.blank(i) {
			continue
		}
		level, err := strconv.Atoi(t.text(i, "level"))
		switch {
		case err != nil || level < 0:
			return nil, nil, fmt.Errorf("%s: level %q is not a whole number", t.at(i, "level"), t.text(i, "level"))
		case level > ddi.MaxDepth:
			return nil, nil, fmt.Errorf("%s: level %d is deeper than %d", t.at(i, "level"), level, ddi.MaxDepth)
		case root == nil && level != 0:
			return nil, nil, fmt.Errorf("%s: the first row must be codeBook at level 0", t.at(i, "level"))
		case root != nil && level == 0:
			return nil, nil, fmt.Errorf("%s: only the first row has level 0", t.at(i, "level"))
		case level > len(stack):
			return nil, nil, fmt.Errorf("%s: level %d is more than one below the level %d above it", t.at(i, "level"), level, len(stack)-1)
		}
		stack = stack[:level]
		var ns []ddi.Attr
		for _, a := range stack {
			ns = append(ns, a.Attrs...)
		}
		n, err := r.docElement(t, i, ns)
		if err != nil {
			return nil, nil, err
		}
		if level == 0 {
			// The XML and JSON readers take only DDI 2.5, so the output must declare it.
			if xmlns, _ := n.Attr("xmlns"); n.Name != "codeBook" || xmlns != ddi.Namespace {
				return nil, nil, fmt.Errorf("%s: the level-0 row must be codeBook with xmlns=%q", t.at(i, "element"), ddi.Namespace)
			}
			root = n
		} else {
			p := stack[level-1]
			if p.Text != "" || p.Raw != "" {
				return nil, nil, fmt.Errorf("%s: <%s> cannot go in <%s>, whose content is in its text cell", t.at(i, "element"), n.Name, p.Name)
			}
			if !schema.Allowed(p.Name, n.Name) {
				r.warn("%s: <%s> is not allowed in <%s>", t.at(i, "element"), n.Name, p.Name)
			}
			if level == 1 && n.Name == "dataDscr" {
				if dataDscr != nil {
					return nil, nil, fmt.Errorf("%s: only one dataDscr is supported", t.at(i, "element"))
				}
				// The writer splits the same elements, so the output converts back to itself.
				warns, err := splitMergePoints(n, ns, t.at(i, "text"))
				if err != nil {
					return nil, nil, err
				}
				r.warns = append(r.warns, warns...)
				dataDscr = n
			}
			p.Children = append(p.Children, n)
		}
		r.checkPlacement(t.at(i, "text"), n)
		stack = append(stack, n)
	}
	if root == nil {
		return nil, nil, fmt.Errorf("%s: there is no codeBook row at level 0", t.addr(1, 0))
	}
	return root, dataDscr, nil
}

func (r *reader) docElement(t *table, i int, ns []ddi.Attr) (*ddi.Node, error) {
	attrs, err := ddi.ParseAttrs(t.raw(i, "attributes"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", t.at(i, "attributes"), err)
	}
	name := t.text(i, "element")
	n, warns, err := parseElement(name, attrs, "", ns)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", t.at(i, "element"), err)
	}
	where := t.at(i, "element")
	switch kind := t.text(i, "kind"); kind {
	case "":
		n.Text = t.text(i, "text")
	case "xml":
		// This parse repeats the warnings about the name, so only its own are kept.
		if n, warns, err = parseElement(name, attrs, t.raw(i, "text"), ns); err != nil {
			return nil, fmt.Errorf("%s: %w", t.at(i, "text"), err)
		}
		where = t.at(i, "text")
	default:
		return nil, fmt.Errorf("%s: kind %q is neither empty nor xml", t.at(i, "kind"), kind)
	}
	r.warnAt(where, warns)
	return n, nil
}

// parseElement reads the element with this name and these attributes whose
// content is the XML between its tags. Going through the XML reader rejects a
// name XML cannot hold, keeps Raw checked, and classifies the content exactly
// as a document's.
func parseElement(name string, attrs []ddi.Attr, content string, ns []ddi.Attr) (*ddi.Node, []string, error) {
	nodes, warns, err := ddi.ParseFragmentWithWarnings("<"+name+" "+ddi.FormatAttrs(attrs)+">"+content+"</"+name+">", ns)
	if err != nil {
		return nil, nil, err
	}
	// Markup in the name breaks the end tag; content that closes the element
	// and opens another shows up as a second node.
	if len(nodes) != 1 {
		return nil, nil, fmt.Errorf("<%s> with this content is not one element", name)
	}
	return nodes[0], warns, nil
}

// checkPlacement warns about each element below n that n's content model does
// not allow.
func (r *reader) checkPlacement(addr string, n *ddi.Node) {
	for _, c := range n.Children {
		if !schema.Allowed(n.Name, c.Name) {
			r.warn("%s: <%s> is not allowed in <%s>", addr, c.Name, n.Name)
		}
		r.checkPlacement(addr, c)
	}
}

// part is an element being rebuilt from a row: its attributes and xml cell,
// and the children its columns and other sheets give it.
type part struct {
	n    *ddi.Node // name, attributes, and the xml cell read as its content
	kids []*ddi.Node
	xml  string // address of the xml cell
}

func (r *reader) part(t *table, i int, name string, cols []attrCol, ns []ddi.Attr) (*part, error) {
	attrs := t.colAttrs(i, cols)
	rest, err := ddi.ParseAttrs(t.raw(i, "attributes"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", t.at(i, "attributes"), err)
	}
	for _, a := range rest {
		// Writing both would repeat the attribute, which XML forbids.
		if slices.ContainsFunc(attrs, func(b ddi.Attr) bool { return b.Name == a.Name }) {
			return nil, fmt.Errorf("%s: %s is also in its own column", t.at(i, "attributes"), a.Name)
		}
	}
	attrs = append(attrs, rest...)
	p := &part{n: &ddi.Node{Name: name, Attrs: attrs}, xml: t.at(i, "xml")}
	if cell := t.raw(i, "xml"); strings.Trim(cell, " \t\n\r") != "" {
		n, warns, err := parseElement(name, attrs, cell, ns)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.xml, err)
		}
		p.n = n
		r.warnAt(p.xml, warns)
		// The writer splits the same elements, so the output converts back to itself.
		split, err := splitMergePoints(n, ns, p.xml)
		if err != nil {
			return nil, err
		}
		r.warns = append(r.warns, split...)
		sortNested(n)
		r.checkPlacement(p.xml, p.n)
	}
	return p, nil
}

// done puts the children together: those from columns and rows first, then
// the xml cell's, in the order of the element's content model.
func (p *part) done() (*ddi.Node, error) {
	n := p.n
	if n.Text != "" || n.Raw != "" {
		switch {
		case len(p.kids) == 0:
			return n, nil
		// Raw is left only when unraw found text between the children.
		case n.Raw != "":
			return nil, fmt.Errorf("%s: <%s> holds text between its child elements, so this cell holds all its content, "+
				"and its other columns and rows must be empty", p.xml, n.Name)
		}
		return nil, fmt.Errorf("%s: this cell holds all the content of <%s>, so its other columns and rows must be empty", p.xml, n.Name)
	}
	n.Children = sortChildren(n.Name, p.kids, n.Children)
	return n, nil
}

// varColumns rebuilds the children of a var that its columns hold.
func varColumns(t *table, i int) ([]*ddi.Node, error) {
	kids := el("labl", t.text(i, "label"))
	if q := t.text(i, "question"); q != "" {
		kids = append(kids, &ddi.Node{Name: "qstn", Attrs: t.colAttrs(i, qstnAttrs), Children: el("qstnLit", q)})
	} else {
		for _, c := range qstnAttrs {
			if t.text(i, c.col) != "" {
				return nil, fmt.Errorf("%s: %s needs a question", t.at(i, c.col), c.col)
			}
		}
	}
	if attrs := t.colAttrs(i, formatAttrs); len(attrs) > 0 {
		kids = append(kids, &ddi.Node{Name: "varFormat", Attrs: attrs})
	}
	for _, line := range strings.Split(t.text(i, "notes"), "\n") {
		kids = append(kids, el("notes", ddi.NormalizeText(line))...)
	}
	return kids, nil
}

// el is the element a text column holds, or nothing for an empty cell.
func el(name, text string, attrs ...ddi.Attr) []*ddi.Node {
	if text == "" {
		return nil
	}
	return []*ddi.Node{{Name: name, Attrs: attrs, Text: text}}
}

// table is a sheet's rows from the header row down, with cells found by
// header name.
type table struct {
	name  string
	cols  map[string]int
	rows  [][]string // rows[0] is the header row
	first int        // the Excel row number of rows[0]
}

func (r *reader) table(name string, headers []string) (*table, error) {
	rows, err := r.f.GetRows(name)
	if err != nil {
		return nil, err
	}
	// The header is the first row of the sheet's table, so a title row may
	// sit above it; without the table it is row 1. Rows below the table are
	// read too, since Excel does not always grow a table over typed rows.
	pos, ok := r.heads[tableName(name)]
	first := pos.head
	if !ok {
		first = 1
	}
	t := &table{name: name, cols: map[string]int{}, first: first}
	if len(rows) == 0 {
		return t, nil
	}
	// A header row past the last filled row has no columns, which the key
	// column check below reports instead of reading the sheet as empty.
	t.rows = [][]string{nil}
	if first <= len(rows) {
		t.rows = rows[first-1:]
	}
	// Excel's one-click Total Row holds a label and SUBTOTALs, not content, so
	// it reads as blank. Rows typed below it are still read.
	if k := pos.totals - first; pos.totals > 0 && k < len(t.rows) {
		t.rows[k] = nil
	}
	for c, h := range t.rows[0] {
		h = strings.TrimSpace(h)
		_, dup := t.cols[h]
		switch {
		case h == "":
		case !slices.Contains(headers, h):
			r.warn("%s: column %q is not one this tool reads, so it is ignored", t.addr(0, c), h)
		case dup:
			return nil, fmt.Errorf("%s: column %s appears twice", t.addr(0, c), h)
		default:
			t.cols[h] = c
		}
	}
	// Cells are found only through the header row, so without its key column
	// every row would read as blank and the sheet would drop out of the output.
	for _, h := range headers {
		switch _, ok := t.cols[h]; {
		case ok:
		case slices.Contains(keyHeaders[name], h):
			return nil, fmt.Errorf("%s: header row %d has no %s column; the header row is the first row of the table %s, or row 1 without it",
				t.addr(0, 0), first, h, tableName(name))
		default:
			r.warn("%s: header row %d has no %s column, so content that was in it is not read", t.addr(0, 0), first, h)
		}
	}
	// A pasted value skips the dropdown's check.
	for _, src := range attrSources[name] {
		for _, c := range src.cols {
			vals := schema.AttrEnum(src.element, c.attr)
			for i := 1; vals != nil && i < len(t.rows); i++ {
				if v := t.text(i, c.col); v != "" && !slices.Contains(vals, v) {
					r.warn("%s: %q is not one of the values DDI 2.5 lists for %s; it is kept as typed", t.at(i, c.col), v, c.col)
				}
			}
		}
	}
	return t, nil
}

func (t *table) addr(row, col int) string {
	cell, _ := excelize.CoordinatesToCellName(col+1, row+t.first)
	return t.name + "!" + cell
}

// at is the address of column h on row i, or of column A when the sheet has no h.
func (t *table) at(i int, h string) string { return t.addr(i, t.cols[h]) }

// raw returns the cell as written, for cells that hold XML or attributes,
// whose whitespace is content. CRLF needs no handling here: the XML reader and
// NormalizeText both read it as LF.
func (t *table) raw(i int, h string) string {
	c, ok := t.cols[h]
	if !ok || c >= len(t.rows[i]) {
		return ""
	}
	return t.rows[i][c]
}

// text returns a text cell as the XML reader would give its value back.
func (t *table) text(i int, h string) string { return ddi.NormalizeText(t.raw(i, h)) }

// set writes v to the cell in column h of row i, which then reads as if typed.
func (t *table) set(i int, h, v string) {
	c := t.cols[h]
	for len(t.rows[i]) <= c {
		t.rows[i] = append(t.rows[i], "")
	}
	t.rows[i][c] = v
}

// fillIDs gives each filled row of t without an ID the ID prefix<k>, k
// counting up from 1 in row order past the IDs in used. Written into the cell,
// the ID reaches the output like a typed one, so the output converts back to
// itself.
func (t *table) fillIDs(prefix string, used map[string]bool) error {
	k := 0
	for i := 1; i < len(t.rows); i++ {
		if t.blank(i) || t.text(i, "ID") != "" {
			continue
		}
		// The other sheets find a row by its ID column, so an ID only in the
		// attributes cell would end up beside an assigned one. A cell that
		// does not parse is reported where the row is read.
		if attrs, err := ddi.ParseAttrs(t.raw(i, "attributes")); err == nil &&
			slices.ContainsFunc(attrs, func(a ddi.Attr) bool { return a.Name == "ID" }) {
			return fmt.Errorf("%s: ID is set here while the ID column is blank; move it to the ID column", t.at(i, "attributes"))
		}
		id := ""
		for id == "" || used[id] {
			k++
			id = prefix + strconv.Itoa(k)
		}
		t.set(i, "ID", id)
	}
	return nil
}

func (t *table) blank(i int) bool {
	for _, c := range t.cols {
		if c < len(t.rows[i]) && strings.Trim(t.rows[i][c], " \t\n\r") != "" {
			return false
		}
	}
	return true
}

// colAttrs returns the attributes whose columns are filled, in cols order.
func (t *table) colAttrs(i int, cols []attrCol) []ddi.Attr {
	var out []ddi.Attr
	for _, c := range cols {
		if v := t.text(i, c.col); v != "" {
			out = append(out, ddi.Attr{Name: c.attr, Value: v})
		}
	}
	return out
}

// id checks the ID of row i, which the other sheets refer to. seen maps each
// ID read so far to its cell.
func (t *table) id(i int, dataDscr *ddi.Node, seen map[string]string) (string, error) {
	id := t.text(i, "ID")
	switch {
	case dataDscr == nil || dataDscr.Text != "" || dataDscr.Raw != "":
		return "", fmt.Errorf("%s: a %s row needs a dataDscr row without text on the %s sheet", t.at(i, "ID"), t.name, docSheet)
	case seen[id] != "":
		return "", fmt.Errorf("%s: ID %s is also the ID at %s", t.at(i, "ID"), id, seen[id])
	}
	seen[id] = t.at(i, "ID")
	return id, nil
}

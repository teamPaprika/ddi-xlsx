package kosac

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// physicalMax is the foundation's limit on a column name.
const physicalMax = 64

var (
	yearRe    = regexp.MustCompile(`^(\d{4})-`)
	nonIdent  = regexp.MustCompile(`[^a-z0-9_]`)
	decimalRe = regexp.MustCompile(`^-?\d+(\.\d+)?$`) // kosac:scale@min/max are xs:decimal
)

// DeriveDefaults drafts what the operator confirms: the analysis meta of every
// survey var, the missing codes, the operator defaults (category, round, file
// sensitivity), and STD_ID and FK candidates, which it returns and does not
// apply. It fills only empty fields, so a value an operator or FromWallaDDI
// set stays and a second call changes nothing. A field derived from another
// is not recomputed when that one changes; empty it and call again.
//
// One rule rewrites: a missing-code row already in 04 whose label is an
// accepted one gets the prescribed label. One whose label is not keeps it and,
// when untyped, is typed MISSING rather than its code's type.
func DeriveDefaults(cb *Codebook, ref RefData) []Suggestion {
	d := &deriver{cb: cb, ref: ref, files: map[string]*File{}, codes: map[[2]string][]*Code{}}
	for _, f := range cb.Files {
		if no := f.Rec.Get("FILE_NO"); d.files[no] == nil {
			d.files[no] = f
		}
	}
	d.admin()
	d.missingCodes()
	weighted := map[string]bool{}
	for _, v := range cb.Vars {
		fill(&v.Role, d.role(v))
		if v.Role == "WEIGHT" {
			weighted[v.Rec.Get("FILE_NO")] = true
		}
	}
	for _, v := range cb.Vars {
		g := v.Rec.Get
		fill(&v.DimMeasure, dimMeasure(v))
		fill(&v.AggRules, aggRules(v, weighted[g("FILE_NO")]))
		include := "Y"
		// Y exactly, as the check reads it: "Y " is no value the XSD takes.
		if v.Role == "TEXT" || g("PII") == "Y" {
			include = "N"
		}
		fill(&v.IncludeYn, include)
		fill(&v.LogicalName, ddi.NormalizeText(g("VAR_LABEL")))
		d.scale(v)
	}
	d.physicalNames()
	d.codeSets()
	d.mrGroups()
	d.grids()
	return d.suggest()
}

type deriver struct {
	cb    *Codebook
	ref   RefData
	files map[string]*File      // by FILE_NO
	codes map[[2]string][]*Code // by (FILE_NO, VAR_NAME) in 04 order, the added rows last
}

// fill sets an empty field only: a value already there is someone's decision.
func fill(dst *string, v string) {
	if strings.TrimSpace(*dst) == "" {
		*dst = v
	}
}

func codeOf(c *Code) string { return strings.TrimSpace(c.Rec.Get("CODE")) }

func (d *deriver) admin() {
	a, g := &d.cb.Admin, d.cb.Study.Get
	if cat, _, ok := strings.Cut(typed(g("DATA_NO")), "-"); ok {
		fill(&a.CategoryCode, cat)
	}
	if m := yearRe.FindStringSubmatch(typed(g("FIELD_START"))); m != nil {
		fill(&a.Round.Year, m[1])
	}
	fill(&a.Round.WaveType, "main")
	// Every 02 row is response data, the most sensitive file of a package.
	for _, f := range d.cb.Files {
		fill(&f.SensLevel, "3")
	}
}

// missingLabels are a missing code's accepted labels, the prescribed one
// first. A code RefData leaves out keeps the default labels.
func (d *deriver) missingLabels(code string) []string {
	if l := d.ref.MissingLabels[code]; len(l) > 0 {
		return l
	}
	return MissingLabelsFor(d.cb.Template.Version.Value)[code]
}

// missingCodes declares -8, -7 and -9 on every 명목·서열·연속 survey var that
// lacks them, appending rows with no cell, then types every code.
func (d *deriver) missingCodes() {
	for _, c := range d.cb.Codes {
		k := [2]string{c.Rec.Get("FILE_NO"), c.Rec.Get("VAR_NAME")}
		d.codes[k] = append(d.codes[k], c)
	}
	for _, v := range d.cb.Vars {
		switch typed(v.Rec.Get("MEAS_LEVEL")) {
		case "명목", "서열", "연속":
		default:
			continue
		}
		if IsSystemVar(v.Rec.Get("VAR_NAME")) {
			continue
		}
		if d.cb.Template.Version.Value == "4.0" {
			continue
		} // v4 declares missingness explicitly; do not invent categories.
		k := varKey(v)
		for _, mc := range MissingCodes {
			m := mc.Code
			if !mc.Declared || slices.ContainsFunc(d.codes[k], func(c *Code) bool { return codeOf(c) == m }) {
				continue
			}
			c := &Code{Rec: Record{}}
			c.Rec.Set("FILE_NO", k[0])
			c.Rec.Set("VAR_NAME", k[1])
			c.Rec.Set("CODE", m)
			c.Rec.Set("CODE_LABEL", d.missingLabels(m)[0])
			d.cb.Codes = append(d.cb.Codes, c)
			d.codes[k] = append(d.codes[k], c)
		}
	}
	for _, c := range d.cb.Codes {
		d.typeCode(c)
	}
}

func (d *deriver) typeCode(c *Code) {
	code := codeOf(c)
	if d.cb.Template.Version.Value == "4.0" {
		if DeclaredMissing(c.Rec.Get("IS_MISSING")) {
			kind := "MISSING"
			if code == "-7" {
				kind = "DK"
			}
			fill(&c.Type, kind)
		} else {
			fill(&c.Type, "VALID")
		}
		return
	}
	m, missing := missingCode(code)
	if !missing {
		fill(&c.Type, "VALID")
		return
	}
	allowed := d.missingLabels(code)
	switch {
	case allowed == nil: // -6 has no prescribed label
		fill(&c.Type, m.Type)
	case slices.Contains(allowed, strings.TrimSpace(c.Rec.Get("CODE_LABEL"))):
		c.Rec.Set("CODE_LABEL", allowed[0])
		fill(&c.Type, m.Type)
	default:
		// Not the foundation's wording, so not its meaning: walla's old export
		// writes -7 for a value the published form no longer has, which DK
		// would silently turn into "don't know".
		fill(&c.Type, "MISSING")
	}
}

func (d *deriver) valid(v *Var) []*Code {
	var out []*Code
	for _, c := range d.codes[varKey(v)] {
		if c.Type == "VALID" {
			out = append(out, c)
		}
	}
	return out
}

// role is the first rule that fits, in the order the foundation lists them.
func (d *deriver) role(v *Var) string {
	g, name := v.Rec.Get, v.Rec.Get("VAR_NAME")
	f := d.files[g("FILE_NO")]
	is := func(ref string) bool { return name != "" && typed(ref) == name }
	file := func(key string) string {
		if f == nil {
			return ""
		}
		return f.Rec.Get(key)
	}
	switch {
	case is(file("ID_VAR")):
		return "ID"
	case is(d.cb.Study.Get("WEIGHT_VAR")):
		return "WEIGHT"
	case is(d.cb.Study.Get("SPLIT_VAR")):
		return "FILTER"
	case strings.TrimSpace(g("DERIVATION")) != "":
		return "DERIVED"
	case is(file("LINK_VAR")):
		return "LINK"
	case typed(g("MEAS_LEVEL")) == "문자":
		return "TEXT"
	}
	return "ITEM"
}

var levelDim = map[string]string{"명목": "DIM", "서열": "BOTH", "연속": "MEASURE", "문자": "ATTR", "날짜": "ATTR"}

func dimMeasure(v *Var) string {
	switch {
	case v.Role == "WEIGHT":
		return "WEIGHT"
	case v.Role == "ID" || v.Role == "LINK":
		return "KEY"
	case typed(v.Rec.Get("MR_TYPE")) == "보기별 열":
		return "MEASURE"
	}
	if MeasurementLevel(v) == "scale" {
		return "MEASURE"
	}
	return levelDim[typed(v.Rec.Get("MEAS_LEVEL"))]
}

// aggRules looks at MR_TYPE first: a rank column is nominal, but counting it
// as one would lose the rank.
func aggRules(v *Var, weighted bool) string {
	switch typed(v.Rec.Get("MR_TYPE")) {
	case "보기별 열":
		return "SUM;RATIO_RESP"
	case "순위별 열":
		return "COUNT;RATIO_RANK"
	}
	switch v.DimMeasure {
	case "DIM":
		return "COUNT;RATIO"
	case "BOTH":
		if weighted {
			return "AVG;COUNT;RATIO;WAVG"
		}
		return "AVG;COUNT;RATIO"
	case "MEASURE":
		if MeasurementLevel(v) == "scale" {
			return "AVG;MIN;MAX;SUM"
		}
	}
	return ""
}

// scale is an ordinal var's valid code range, compared as numbers so 10 is
// above 9.
func (d *deriver) scale(v *Var) {
	if typed(v.Rec.Get("MEAS_LEVEL")) != "서열" {
		return
	}
	var lo, hi string
	var lov, hiv float64
	for _, c := range d.valid(v) {
		s := codeOf(c)
		if !decimalRe.MatchString(s) {
			continue
		}
		x, _ := strconv.ParseFloat(s, 64)
		if lo == "" || x < lov {
			lo, lov = s, x
		}
		if hi == "" || x > hiv {
			hi, hiv = s, x
		}
	}
	if lo == "" {
		return
	}
	dir := "POS"
	if v.Rec.Get("REVERSE") == "Y" { // exactly, as the check reads it
		dir = "NEG"
	}
	fill(&v.Scale.Min, lo)
	fill(&v.Scale.Max, hi)
	fill(&v.Scale.Dir, dir)
}

// physicalNames gives each var a column name unique in its file. Names
// already set are taken first, so a derived name never collides with one an
// operator chose.
func (d *deriver) physicalNames() {
	reserved := d.ref.ReservedWords
	if reserved == nil {
		reserved = defaultReservedWords
	}
	taken := map[string]map[string]bool{}
	for _, v := range d.cb.Vars {
		no := v.Rec.Get("FILE_NO")
		if taken[no] == nil {
			taken[no] = map[string]bool{}
		}
		if strings.TrimSpace(v.PhysicalName) != "" {
			taken[no][v.PhysicalName] = true
		}
	}
	for _, v := range d.cb.Vars {
		if strings.TrimSpace(v.PhysicalName) != "" {
			continue
		}
		base := physicalName(v.Rec.Get("VAR_NAME"), reserved)
		if it, ok := d.ref.StdItems[typed(v.Rec.Get("STD_ID"))]; ok && it.PhysicalName != "" {
			base = it.PhysicalName
		}
		v.PhysicalName = unique(base, taken[v.Rec.Get("FILE_NO")])
	}
}

// physicalName makes a var name a plain SQL column name. A reserved word or a
// walla_ name, which would clash with the system columns, gets q_ in front.
func physicalName(name string, reserved []string) string {
	p := nonIdent.ReplaceAllString(strings.ToLower(name), "_")
	if strings.HasPrefix(p, "walla_") || slices.ContainsFunc(reserved, func(w string) bool { return strings.EqualFold(w, p) }) {
		p = "q_" + p
	}
	return p
}

// unique cuts name to physicalMax and, when taken already has it, appends
// _2, _3, … cutting further so the suffix still fits.
func unique(name string, taken map[string]bool) string {
	out := cut(name, physicalMax)
	for n := 2; taken[out]; n++ {
		sfx := "_" + strconv.Itoa(n)
		out = cut(name, physicalMax-len(sfx)) + sfx
	}
	taken[out] = true
	return out
}

func cut(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// codeSets gives every var with valid codes the ID of its (code, label) set.
// Labels compare without spaces or case, so "보통 이다" and "보통이다" share
// one. IDs are numbered over all vars in 03 order, set or not, so emptying
// one ID and deriving again gives it back. "Representative label
// abbreviations" cannot be made deterministically, hence numbers.
func (d *deriver) codeSets() {
	fromRef := map[string]string{}
	for _, id := range slices.Sorted(maps.Keys(d.ref.CodeSets)) {
		var pairs []string
		for _, c := range d.ref.CodeSets[id] {
			pairs = append(pairs, pair(c.Code, c.Label))
		}
		if k, _ := setKey(pairs); fromRef[k] == "" {
			fromRef[k] = id
		}
	}
	ids, seen := map[string]string{}, map[string]int{}
	for _, v := range d.cb.Vars {
		var pairs []string
		for _, c := range d.valid(v) {
			pairs = append(pairs, pair(codeOf(c), c.Rec.Get("CODE_LABEL")))
		}
		if len(pairs) == 0 {
			continue
		}
		k, n := setKey(pairs)
		id, ok := ids[k]
		if !ok {
			if id = fromRef[k]; id == "" {
				p := "CODES"
				if typed(v.Rec.Get("MEAS_LEVEL")) == "서열" {
					p = "LIKERT"
				}
				p += strconv.Itoa(n) + "_"
				seen[p]++
				id = p + strconv.Itoa(seen[p])
			}
			ids[k] = id
		}
		fill(&v.CodeSetID, id)
	}
}

func pair(code, label string) string { return strings.TrimSpace(code) + "\x00" + normLabel(label) }

func normLabel(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), "")) }

// setKey is the set of pairs as one string, and its size.
func setKey(pairs []string) (string, int) {
	slices.Sort(pairs)
	pairs = slices.Compact(pairs)
	return strings.Join(pairs, "\x01"), len(pairs)
}

// mrGroups keys an MR var by its name prefix, or its whole name when it has
// none. NoneOption and RankNo are judged over the groups assembly writes.
func (d *deriver) mrGroups() {
	for _, v := range d.cb.Vars {
		if typed(v.Rec.Get("MR_TYPE")) == "" {
			continue
		}
		name := v.Rec.Get("VAR_NAME")
		p, ok := Prefix(name)
		if !ok {
			p = name
		}
		fill(&v.MRGroup, p)
	}
	for _, g := range groupBy(d.cb.Vars, mrKey("보기별 열")) {
		none := "N"
		if slices.ContainsFunc(g.vars, func(v *Var) bool { return strings.HasSuffix(ddi.NormalizeText(v.Rec.Get("VAR_LABEL")), "없음") }) {
			none = "Y"
		}
		for _, v := range g.vars {
			fill(&v.NoneOption, none)
		}
	}
	for _, g := range groupBy(d.cb.Vars, mrKey("순위별 열")) {
		for i, v := range g.vars {
			fill(&v.RankNo, strconv.Itoa(i+1))
		}
	}
}

// gridKey keys a grid row by (FILE_NO, VAR_GROUP, prefix, Q_TEXT stem,
// MEAS_LEVEL). The prefix keeps apart questions that share only their
// wording, as two-choice items do; a var without one joins no grid. Grouping
// wrongly is worse than not grouping, so an empty stem joins none either.
func gridKey(v *Var) string {
	g := v.Rec.Get
	level := typed(g("MEAS_LEVEL"))
	p, ok := Prefix(g("VAR_NAME"))
	stem, _, _ := GridStem(ddi.NormalizeText(g("Q_TEXT")))
	if (level != "명목" && level != "서열") || !ok || stem == "" || typed(g("MR_TYPE")) != "" || strings.TrimSpace(g("DERIVATION")) != "" {
		return ""
	}
	return strings.Join([]string{g("FILE_NO"), ddi.NormalizeText(g("VAR_GROUP")), p, stem, level}, "\x00")
}

// grids groups rows of two or more under the first member's name, then gives
// every member of a grid, preset ones too, its row label. A group with any
// member already keyed (walla's varGrp, an operator) is left as it is: filling
// only its empty members would split it into groups of one. A native codebook
// gets no grid by name: every walla field name has the prefix "f", so
// same-worded questions on one page would pass for one.
func (d *deriver) grids() {
	for _, g := range groupBy(d.cb.Vars, gridKey) {
		if d.cb.Native || len(g.vars) < 2 || slices.ContainsFunc(g.vars, func(v *Var) bool { return strings.TrimSpace(v.GridGroup) != "" }) {
			continue
		}
		for _, v := range g.vars {
			v.GridGroup = g.vars[0].Rec.Get("VAR_NAME")
		}
	}
	for _, g := range groupBy(d.cb.Vars, func(v *Var) string {
		if v.GridGroup == "" {
			return ""
		}
		return v.Rec.Get("FILE_NO") + "\x00" + v.GridGroup
	}) {
		labels := make([]string, len(g.vars))
		for i, v := range g.vars {
			labels[i] = ddi.NormalizeText(v.Rec.Get("VAR_LABEL"))
		}
		shared := LabelPrefix(labels)
		for i, v := range g.vars {
			_, row, ok := GridStem(ddi.NormalizeText(v.Rec.Get("Q_TEXT")))
			if !ok || row == "" {
				row = rowOf(labels[i], shared)
			}
			fill(&v.RowLabel, row)
		}
	}
}

// rowOf is label without the shared prefix and the separator after it, or the
// whole label when nothing is left.
func rowOf(label, shared string) string {
	if shared == "" {
		return label
	}
	r := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(label, shared)), ":-·"))
	if r == "" {
		return label
	}
	return r
}

// ApplyFK puts a decided FK into v's analysis meta: the code set's physical
// name, with _2, _3 when another var of the file has it (two vars both mapped
// to CC_SIDO), DIM, and the FK's dimension table and key column where empty.
func ApplyFK(cb *Codebook, v *Var, def CodeSetDef) error {
	switch {
	case v.FK == nil:
		return fmt.Errorf("%s: FK가 없습니다", varLoc(v))
	case v.FK.CodeSet != def.CodeSetID:
		return fmt.Errorf("%s: FK 코드셋 %s와 정의의 코드셋 %s가 다릅니다", varLoc(v), v.FK.CodeSet, def.CodeSetID)
	}
	if def.PhysicalName != "" {
		taken := map[string]bool{}
		for _, o := range cb.Vars {
			if o != v && o.Rec.Get("FILE_NO") == v.Rec.Get("FILE_NO") {
				taken[o.PhysicalName] = true
			}
		}
		v.PhysicalName = unique(def.PhysicalName, taken)
	}
	v.DimMeasure = "DIM"
	fill(&v.FK.DimTable, def.DimTable)
	fill(&v.FK.KeyColumn, def.KeyColumn)
	return nil
}

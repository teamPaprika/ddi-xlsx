package kosac

import (
	"slices"
	"strconv"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// dataDscr writes the groups (section, grid, multipleResp, rank), then the
// system vars of every file, then the survey vars in 03 order. kosac:seq is
// the column in the file's CSV, where the system vars come first.
func (b *builder) dataDscr() *ddi.Node {
	d := add(el("dataDscr"), b.varGrps()...)
	for i, f := range b.cb.Files {
		no := f.Rec.Get("FILE_NO")
		for j, s := range SystemVars {
			id := sysID(i, s.Name)
			add(d, add(b.varNode(id, s.Name, no, s.MeasLevel, map[string]string{
				"seq": strconv.Itoa(j + 1), "role": s.Role, "measLevel": s.MeasLevel, "pii": s.PII, "cleansed": yes(b.cleansed[id]),
			}),
				el("location", "fileid", "F"+no), text("labl", s.Label),
				el("varFormat", "type", s.VarFormat, "category", s.Category),
				el("kosac:analysis", "physicalName", s.Name, "dimMeasure", "KEY", "includeYn", "N")))
		}
	}
	seq := map[string]int{}
	for _, v := range b.cb.Vars {
		no := v.Rec.Get("FILE_NO")
		seq[no]++
		add(d, b.surveyVar(v, len(SystemVars)+seq[no]))
	}
	return d
}

type group struct {
	key  string
	vars []*Var
}

// groupBy collects vars by key in the 03 order of each group's first member.
// A var whose key is "" joins no group.
func groupBy(vs []*Var, key func(*Var) string) []*group {
	idx := map[string]*group{}
	var out []*group
	for _, v := range vs {
		k := key(v)
		if k == "" {
			continue
		}
		if idx[k] == nil {
			idx[k] = &group{key: k}
			out = append(out, idx[k])
		}
		idx[k].vars = append(idx[k].vars, v)
	}
	return out
}

// mrKey keys MR groups of one MR_TYPE by (FILE_NO, MRGroup), so groups of
// different files with the same name stay apart.
func mrKey(mrType string) func(*Var) string {
	return func(v *Var) string {
		if v.MRGroup == "" || typed(v.Rec.Get("MR_TYPE")) != mrType {
			return ""
		}
		return v.Rec.Get("FILE_NO") + "\x00" + v.MRGroup
	}
}

func (b *builder) varGrps() []*ddi.Node {
	var out []*ddi.Node
	grp := func(typ, otherType string, g *group, labl, mrGroup, none string) {
		ids := make([]string, len(g.vars))
		for i, v := range g.vars {
			ids[i] = v.ID
		}
		n := el("varGrp", "ID", "G"+strconv.Itoa(len(out)+1), "type", typ, "otherType", otherType,
			"var", strings.Join(ids, " "), "kosac:mrGroup", mrGroup)
		out = append(out, add(n, text("labl", labl), el("kosac:noneOption", "yn", none)))
	}
	vs := b.cb.Vars
	// Keyed by the label as written to XML, so values that read back the same
	// make one section.
	for _, g := range groupBy(vs, func(v *Var) string { return ddi.NormalizeText(v.Rec.Get("VAR_GROUP")) }) {
		grp("section", "", g, g.key, "", "")
	}
	for _, g := range groupBy(vs, func(v *Var) string {
		if v.GridGroup == "" {
			return ""
		}
		return v.Rec.Get("FILE_NO") + "\x00" + v.GridGroup
	}) {
		stem, _, _ := GridStem(ddi.NormalizeText(g.vars[0].Rec.Get("Q_TEXT")))
		grp("grid", "", g, stem, "", "")
	}
	for _, g := range groupBy(vs, mrKey("보기별 열")) {
		grp("multipleResp", "", g, mrLabel(g), g.vars[0].MRGroup, g.vars[0].NoneOption)
	}
	// The XSD has no rank type.
	for _, g := range groupBy(vs, mrKey("순위별 열")) {
		grp("other", "rank", g, mrLabel(g), g.vars[0].MRGroup, "")
	}
	return out
}

// mrLabel is the members' label prefix, or the group key when they share none.
// It works on the labels as written to XML, the only form a reader gets back.
func mrLabel(g *group) string {
	labels := make([]string, len(g.vars))
	for i, v := range g.vars {
		labels[i] = ddi.NormalizeText(v.Rec.Get("VAR_LABEL"))
	}
	if p := LabelPrefix(labels); p != "" {
		return p
	}
	return g.vars[0].MRGroup
}

// varNode makes a var with its attributes; k holds the kosac ones by local
// name and is written in KosacAttrs order.
func (b *builder) varNode(id, name, fileNo, measLevel string, k map[string]string) *ddi.Node {
	wgt, intrvl := "", "discrete"
	if id != "" && id == b.weight {
		wgt = "wgt"
	}
	if measLevel == "scale" {
		intrvl = "contin"
	}
	n := el("var", "ID", id, "name", name, "wgt", wgt, "files", "F"+fileNo, "intrvl", intrvl)
	for _, a := range KosacAttrs["var"] {
		if strings.TrimSpace(k[a]) != "" {
			n.Attrs = append(n.Attrs, ddi.Attr{Name: "kosac:" + a, Value: k[a]})
		}
	}
	return n
}

func (b *builder) surveyVar(v *Var, seq int) *ddi.Node {
	g, no := v.Rec.Get, v.Rec.Get("FILE_NO")
	level := MeasLevels[typed(g("MEAS_LEVEL"))]
	n := b.varNode(v.ID, g("VAR_NAME"), no, level, map[string]string{
		"seq": strconv.Itoa(seq), "role": v.Role, "measLevel": level, "pii": cell(g("PII")),
		"sensitive": cell(g("SENSITIVE")), "reverse": cell(g("REVERSE")), "mrType": cell(g("MR_TYPE")),
		"mrGroup": v.MRGroup, "rowLabel": v.RowLabel, "rankNo": v.RankNo, "cleansed": yes(b.cleansed[v.ID]),
		"textFormat": v.TextFormat, "parentVar": b.fileVar(no, varLoc(v)+" ParentVar", v.ParentVar),
	})
	codes := b.codes[varKey(v)]
	add(n, el("location", "fileid", "F"+no), text("labl", g("VAR_LABEL")),
		add(el("qstn", "seqNo", g("Q_NO")), text("qstnLit", g("Q_TEXT"))),
		text("universe", g("RESP_UNIVERSE"), "clusion", "I"))
	for _, c := range codes {
		add(n, add(el("catgry", "missing", yes(isMissing(c))),
			text("catValu", c.Rec.Get("CODE")), text("labl", c.Rec.Get("CODE_LABEL")), text("txt", c.Rec.Get("NOTE")),
			text("kosac:codeType", c.Type), text("kosac:stdCode", c.StdCode, "set", c.StdSet)))
	}
	format := el("varFormat", "type", "numeric")
	switch level {
	case "text":
		format = el("varFormat", "type", "character")
	case "date":
		format = el("varFormat", "type", "character", "category", "date")
	}
	add(n, text("concept", typed(g("STD_ID")), "vocab", "KOSAC-STD"),
		add(el("derivation"), text("drvdesc", g("DERIVATION"))),
		format, text("notes", g("NOTE")))

	fkRef := ""
	if fk := v.FK; fk != nil {
		fkRef = fk.CodeSet
		add(n, el("kosac:fk", "codeSet", fk.CodeSet, "dimTable", fk.DimTable, "keyColumn", fk.KeyColumn,
			"mapping", fk.Mapping, "codeSystem", fk.CodeSystem, "refYear", fk.RefYear, "linkKey", fk.LinkKey,
			"codeSetVersion", fk.CodeSetVersion, "validatedAt", fk.ValidatedAt,
			"matchedRows", integer(fk.MatchedRows), "unmatchedRows", integer(fk.UnmatchedRows)))
	}
	a := v.Analysis
	analysis := el("kosac:analysis", "physicalName", a.PhysicalName, "dimMeasure", a.DimMeasure, "aggRules", a.AggRules,
		"includeYn", a.IncludeYn, "logicalName", a.LogicalName, "fkRef", fkRef)
	for _, s := range a.Synonyms {
		add(analysis, text("kosac:synonym", s))
	}
	return add(n, el("kosac:scale", "min", a.Scale.Min, "max", a.Scale.Max, "dir", a.Scale.Dir),
		text("kosac:missingCodes", missingCodes(codes)), text("kosac:codeSetId", a.CodeSetID), analysis)
}

// isMissing reads an empty Type as VALID: a model not yet through
// DeriveDefaults has no types.
func isMissing(c *Code) bool { return c.Type != "" && c.Type != "VALID" }

// missingCodes lists the var's missing codes: the declared ones first, in the
// foundation's order, then the others (such as walla's -6) in catgry order,
// so a value check sees every code a missing value may take.
func missingCodes(codes []*Code) string {
	var have, rest []string
	for _, c := range codes {
		if !isMissing(c) {
			continue
		}
		if v := ddi.NormalizeText(c.Rec.Get("CODE")); declared(v) { // as catValu holds it
			have = append(have, v)
		} else {
			rest = append(rest, v)
		}
	}
	var out []string
	for _, m := range MissingCodes {
		if m.Declared && slices.Contains(have, m.Code) {
			out = append(out, m.Code)
		}
	}
	return strings.Join(append(out, rest...), ";")
}

func declared(code string) bool {
	m, ok := missingCode(code)
	return ok && m.Declared
}

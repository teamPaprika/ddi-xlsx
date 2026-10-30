package kosac

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// WallaOptions is what the caller knows of a walla export that it does not say.
type WallaOptions struct {
	// MeasHints gives a var's MEAS_LEVEL (명목·서열·연속·문자·날짜) by name.
	// Which walla field type is which level is the caller's to say, so the
	// engine needs no walla field type names.
	MeasHints map[string]string
}

// wallaRules are walla's naming and missing-code rules for one export.
type wallaRules struct {
	system []string       // walla's own columns, which assembly makes again
	rank   *regexp.Regexp // a rank column: its group, then k
	suffix *regexp.Regexp // a column that belongs to var $1, then its suffix
	typed  []string       // codes typed here; DeriveDefaults types the rest
}

var (
	// An ID piece never holds "_" and a fixed suffix follows "__", so the last
	// "__" is the one boundary; "_other" after one "_" is an option ID.
	wallaRank   = regexp.MustCompile(`^(.+)__rank([1-9][0-9]*)$`)
	wallaSuffix = regexp.MustCompile(`^(.+)__(other|custom|lat|lon|amount|orderid|status)$`)

	legacyRank   = regexp.MustCompile(`^(.+)_rank([1-9][0-9]*)$`)
	legacySuffix = regexp.MustCompile(`^(.+)_(other)$`)
)

// wallaKind gives the rules of the export that has the vars in names. Every
// walla rule lives here and in the patterns above, so a change on walla's
// side changes only this.
func wallaKind(names map[string]bool) wallaRules {
	if names["response_id"] && !names["walla_response_id"] {
		// The old export, which the KOFAC surveys still are. This branch goes
		// when they are re-exported with the current rules; its patterns must
		// not join those, where "rank1" after one "_" can be an option ID. Its
		// -7 "현재 게시본에 없는 값" stays untyped, so DeriveDefaults keeps the
		// label and types it MISSING instead of DK.
		return wallaRules{system: []string{"response_id", "submitted_at", "customer_key"},
			rank: legacyRank, suffix: legacySuffix}
	}
	var system []string
	for _, s := range SystemVars {
		system = append(system, s.Name)
	}
	// -6 is a value of an earlier published form.
	return wallaRules{system: system, rank: wallaRank, suffix: wallaSuffix, typed: []string{"-6"}}
}

// FromWallaDDI drafts a codebook from a walla DDI 2.5 export, XML or JSON. It
// fills TITLE, the walla ID, one CSV file, a 03 row per survey var with its
// MR, grid and rank group and its parent, and a 04 row per category, all of
// which DeriveDefaults keeps. VAR_GROUP, Q_NO and the 01 sheet but TITLE are
// left to the caller. The Parquet file holds the CSV's data again and walla's
// system vars are made again at assembly, so neither is read.
func FromWallaDDI(b []byte, opt WallaOptions) (*Codebook, []Issue, error) {
	for _, name := range slices.Sorted(maps.Keys(opt.MeasHints)) {
		if lv := opt.MeasHints[name]; MeasLevels[lv] == "" {
			return nil, nil, fmt.Errorf("MeasHints[%s] %q는 측정수준(명목·서열·연속·문자·날짜)이 아닙니다", name, lv)
		}
	}
	root, _, err := ddi.Parse(b)
	if err != nil {
		return nil, nil, err
	}
	cb := &Codebook{Study: record(StudyKeys), Native: true}
	ts := child(child(child(root, "stdyDscr"), "citation"), "titlStmt")
	cb.Study.Set("TITLE", textOf(ts, "titl"))
	for _, id := range kids(ts, "IDNo") {
		if a, _ := id.Attr("agency"); a == "walla" {
			cb.Admin.WallaID = id.Text
		}
	}
	fds := kids(root, "fileDscr")
	i := slices.IndexFunc(fds, func(n *ddi.Node) bool { return attr(n, "ID") == "F1" })
	if i < 0 {
		return nil, nil, errors.New("walla export에 파일 F1이 없습니다")
	}
	ft := child(fds[i], "fileTxt")
	f := &File{Rec: record(FileKeys)}
	f.Rec.Set("FILE_NO", "1")
	f.Rec.Set("FILE_NAME", textOf(ft, "fileName"))
	f.Rec.Set("FILE_FORMAT", "CSV")
	f.Rec.Set("ID_VAR", "walla_response_id")
	f.Rec.Set("VALID_N", textOf(child(ft, "dimensns"), "caseQnty"))
	cb.Files = []*File{f}

	dd := child(root, "dataDscr")
	vars := kids(dd, "var")
	names := map[string]bool{}
	qstns := map[string]*ddi.Node{} // a var after the first of its question only references it
	for _, n := range vars {
		names[attr(n, "name")] = true
		if q := child(n, "qstn"); q != nil {
			qstns[attr(q, "ID")] = q
		}
	}
	rules := wallaKind(names)
	group := map[[2]string]string{} // (var ID, varGrp type) → varGrp@name
	multiple := map[string]bool{}   // multipleResp varGrp@name
	for _, g := range kids(dd, "varGrp") {
		for _, id := range strings.Fields(attr(g, "var")) {
			group[[2]string{id, attr(g, "type")}] = attr(g, "name")
		}
		if attr(g, "type") == "multipleResp" {
			multiple[attr(g, "name")] = true
		}
	}

	for _, n := range vars {
		name := attr(n, "name")
		if slices.Contains(rules.system, name) {
			continue
		}
		v := &Var{Rec: record(VarKeys)}
		set := v.Rec.Set
		set("FILE_NO", "1")
		set("VAR_NAME", name)
		label := textOf(n, "labl")
		set("VAR_LABEL", label)
		q := child(n, "qstn")
		if q == nil {
			q = qstns[attr(n, "qstn")]
		}
		qtext := textOf(q, "qstnLit")
		if qtext == "" {
			qtext = label
		}
		set("Q_TEXT", qtext)
		var notes []string
		for _, nt := range kids(n, "notes") {
			notes = append(notes, nt.Text)
		}
		set("NOTE", strings.Join(notes, "\n"))
		set("PII", "N")
		set("SENSITIVE", "N")

		cats := kids(n, "catgry")
		level, hinted := opt.MeasHints[name]
		switch {
		case hinted:
		case len(cats) > 0:
			level = "명목"
		case attr(child(n, "varFormat"), "category") == "date":
			level = "날짜"
		case attr(child(n, "varFormat"), "type") == "character":
			level = "문자"
		default:
			level = "연속"
		}
		set("MEAS_LEVEL", level)
		for _, c := range cats {
			code := &Code{Rec: record(CodeKeys)}
			value := textOf(c, "catValu")
			code.Rec.Set("FILE_NO", "1")
			code.Rec.Set("VAR_NAME", name)
			code.Rec.Set("CODE", value)
			code.Rec.Set("CODE_LABEL", textOf(c, "labl"))
			if m, ok := missingCode(value); ok && slices.Contains(rules.typed, value) {
				code.Type = m.Type
			}
			cb.Codes = append(cb.Codes, code)
		}

		// A rank column is judged by its name whatever group walla puts it in,
		// and is never a grid row.
		id := attr(n, "ID")
		if m := rules.rank.FindStringSubmatch(name); m != nil {
			set("MR_TYPE", "순위별 열")
			v.MRGroup, v.RankNo = m[1], m[2]
		} else {
			if k := group[[2]string{id, "multipleResp"}]; k != "" {
				set("MR_TYPE", "보기별 열")
				v.MRGroup = k
			}
			v.GridGroup = group[[2]string{id, "grid"}]
		}
		if m := rules.suffix.FindStringSubmatch(name); m != nil {
			if names[m[1]] {
				v.ParentVar = m[1]
			}
			// An answer outside the codes is held as walla writes it: plain text
			// beside its single-choice var, an array of the choices for a
			// multiple choice, an object keyed by row or key otherwise.
			if m[2] == "other" {
				switch {
				case names[m[1]]:
					v.TextFormat = "TEXT"
				case multiple[m[1]]:
					v.TextFormat = "JSON_ARR"
				default:
					v.TextFormat = "JSON_OBJ"
				}
			}
		}
		cb.Vars = append(cb.Vars, v)
	}
	return cb, nil, nil
}

func child(n *ddi.Node, name string) *ddi.Node {
	if k := kids(n, name); len(k) > 0 {
		return k[0]
	}
	return nil
}

func kids(n *ddi.Node, name string) []*ddi.Node {
	var out []*ddi.Node
	if n != nil {
		for _, c := range n.Children {
			if c.Name == name {
				out = append(out, c)
			}
		}
	}
	return out
}

func textOf(n *ddi.Node, name string) string {
	if c := child(n, name); c != nil {
		return c.Text
	}
	return ""
}

func attr(n *ddi.Node, name string) string {
	if n == nil {
		return ""
	}
	v, _ := n.Attr(name)
	return v
}

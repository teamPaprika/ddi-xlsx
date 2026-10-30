package kosac

import (
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Jaccard thresholds of the foundation's suggestion rules.
const (
	stdHigh  = 0.8 // STD_ID: suggested with confidence
	stdCheck = 0.6 // STD_ID: suggested for a check
	fkMin    = 0.8 // FK code set by labels
)

// particles come off the end of a token; the longer ones are tried first so
// "에서" is not read as "서" after "에".
var particles = []string{"에서", "으로", "은", "는", "이", "가", "을", "를", "의", "에", "로", "와", "과", "도"}

// tokens splits s into lowercase runs of letters and digits and takes one
// trailing particle off a run of two or more runes, so "성별은" matches
// "성별". The rule is blind: "나이" loses "이" too, on both sides alike.
func tokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, t := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		if utf8.RuneCountInString(t) >= 2 {
			for _, p := range particles {
				if len(t) > len(p) && strings.HasSuffix(t, p) {
					t = strings.TrimSuffix(t, p)
					break
				}
			}
		}
		out[t] = true
	}
	return out
}

func jaccard(a, b map[string]bool) float64 {
	n := 0
	for t := range a {
		if b[t] {
			n++
		}
	}
	if u := len(a) + len(b) - n; u > 0 {
		return float64(n) / float64(u)
	}
	return 0
}

func confidenceLevel(confidence float64) string {
	if confidence >= stdHigh {
		return "high"
	}
	return "check"
}

// suggest offers a STD_ID for a var without one and a code set for a var
// without an FK. A var's label and question are each compared with a
// dictionary item's name and each synonym, and the best pair counts: a long
// question would otherwise drown a one-word name.
func (d *deriver) suggest() []Suggestion {
	type item struct {
		id    string
		names []map[string]bool
	}
	var items []item
	for _, id := range slices.Sorted(maps.Keys(d.ref.StdItems)) {
		it := d.ref.StdItems[id]
		if it.Active == "N" {
			continue
		}
		e := item{id: id}
		for _, name := range append([]string{it.NameKo}, it.Synonyms...) {
			e.names = append(e.names, tokens(name))
		}
		items = append(items, e)
	}
	sets := slices.Sorted(maps.Keys(d.ref.CodeSets))

	var out []Suggestion
	for _, v := range d.cb.Vars {
		g := v.Rec.Get
		add := func(kind, value string, confidence float64, codeMap map[string]string) {
			out = append(out, Suggestion{Kind: kind, FileNo: g("FILE_NO"), VarName: g("VAR_NAME"), Value: value,
				Confidence: confidence, Level: confidenceLevel(confidence), CodeMap: codeMap})
		}
		std := typed(g("STD_ID"))
		if std == "" && len(items) > 0 {
			sides := []map[string]bool{tokens(g("VAR_LABEL")), tokens(g("Q_TEXT"))}
			best, id := 0.0, ""
			for _, it := range items {
				for _, name := range it.names {
					for _, side := range sides {
						if j := jaccard(name, side); j > best {
							best, id = j, it.id
						}
					}
				}
			}
			if best >= stdCheck {
				add("STD_ID", id, best, nil)
			}
		}
		if v.FK != nil {
			continue
		}
		valid := d.valid(v)
		if it, ok := d.ref.StdItems[std]; ok && it.CodeSetID != "" {
			_, m := matchCodes(valid, d.ref.CodeSets[it.CodeSetID])
			add("FK", it.CodeSetID, 1, m)
			continue
		}
		for _, id := range sets {
			if j, m := matchCodes(valid, d.ref.CodeSets[id]); j >= fkMin {
				add("FK", id, j, m)
			}
		}
	}
	return out
}

// matchCodes maps each source code to the standard code whose label or an
// alternate label is the same, and scores the overlap of the two sets, each
// standard code counting once however many labels it has.
func matchCodes(codes []*Code, set []StdCode) (float64, map[string]string) {
	m, hit := map[string]string{}, map[string]bool{}
	for _, c := range codes {
		l := normLabel(c.Rec.Get("CODE_LABEL"))
		for _, s := range set {
			if normLabel(s.Label) == l || slices.ContainsFunc(s.AltLabels, func(a string) bool { return normLabel(a) == l }) {
				m[codeOf(c)], hit[s.Code] = s.Code, true
				break
			}
		}
	}
	if u := len(codes) + len(set) - len(hit); u > 0 {
		return float64(len(hit)) / float64(u), m
	}
	return 0, m
}

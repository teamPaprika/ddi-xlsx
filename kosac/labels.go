package kosac

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// gridRowMarkers find the row part a contractor appends to a grid row's
// Q_TEXT; the capture is the row. Only the 2024 sample's convention is known
// yet; other contractors' forms are added once seen, since grouping wrongly is
// worse than not grouping.
var gridRowMarkers = []*regexp.Regexp{
	regexp.MustCompile(`\s*▶\s*\[([^\]]*)\]\s*$`),
}

// GridStem splits a grid row's Q_TEXT into the question every row shares and
// the row. Without a row marker the stem is the whole Q_TEXT and ok is false.
func GridStem(qtext string) (stem, row string, ok bool) {
	for _, re := range gridRowMarkers {
		if m := re.FindStringSubmatchIndex(qtext); m != nil {
			return strings.TrimSpace(qtext[:m[0]]), strings.TrimSpace(qtext[m[2]:m[3]]), true
		}
	}
	return strings.TrimSpace(qtext), "", false
}

var labelSeparators = []string{": ", " - ", "·", " (", " ["}

// LabelPrefix is the part the labels share up to the last separator inside it,
// or "" when the shared part holds none. Stopping at a separator keeps a
// shared syllable from passing as a prefix: "과학기술분야의 …" and
// "과학기술과 …" share "과학기술", which names neither.
func LabelPrefix(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	p := labels[0]
	for _, l := range labels[1:] {
		n := 0
		for n < len(p) {
			_, size := utf8.DecodeRuneInString(p[n:])
			if !strings.HasPrefix(l[n:], p[n:n+size]) {
				break
			}
			n += size
		}
		p = p[:n]
	}
	cut := -1
	for _, s := range labelSeparators {
		cut = max(cut, strings.LastIndex(p, s))
	}
	if cut < 0 {
		return ""
	}
	return strings.TrimRight(p[:cut], " :-([")
}

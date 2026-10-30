package kosac

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	reportPagesRe = regexp.MustCompile(`^REPORT:p(\d+)(?:(-)(\d+)?)?$`)
	// "최종보고서 부록 수록 (PDF 336쪽~)": a contractor writes the pages in
	// prose at the end of a sentence about the report. Korean word processors
	// often turn "~" into a full-width tilde; missing one would read only the
	// last page.
	proseEndRe = regexp.MustCompile(`(\d+)\s*쪽?\s*(?:([~\-–～∼〜])\s*(?:(\d+)\s*쪽?)?)?\s*\)?\s*$`)
	fileNameRe = regexp.MustCompile(`\.[A-Za-z0-9]{2,5}$`)
)

// MaterialItem reads one QUESTIONNAIRE, REPORT_FILE or EXTRA_FILES item as a
// page range of the report (pages != ""), else as a file name (an extension at
// the end). A value can be both, as "…수록 p.336" is; the check and assembly
// both read items here so they cannot take it differently.
func MaterialItem(value string) (pages string, file bool) {
	if p, ok := ParsePages(value); ok {
		return p, false
	}
	return "", fileNameRe.MatchString(strings.TrimSpace(value))
}

// ParsePages reads a page range of the report: "REPORT:p{a}[-[{b}]]", or a
// value with "수록" that ends in "{a}[쪽][~[{b}[쪽]]][)]". It returns the range
// as "{a}", "{a}-{b}" or "{a}-" (to the end).
func ParsePages(value string) (pages string, ok bool) {
	value = strings.TrimSpace(value)
	m := reportPagesRe.FindStringSubmatch(value)
	if m == nil && strings.Contains(value, "수록") {
		m = proseEndRe.FindStringSubmatch(value)
	}
	if m == nil {
		return "", false
	}
	a, _ := strconv.Atoi(m[1])
	if a < 1 {
		return "", false
	}
	switch {
	case m[3] != "":
		if b, _ := strconv.Atoi(m[3]); b < a {
			return "", false
		}
		return m[1] + "-" + m[3], true
	case m[2] != "":
		return m[1] + "-", true
	}
	return m[1], true
}

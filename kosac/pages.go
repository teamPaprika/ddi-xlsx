package kosac

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	reportPagesRe = regexp.MustCompile(`^REPORT:p(\d+)(?:(-)(\d+)?)?$`)
	prosePagesRe  = regexp.MustCompile(`(?:(?:PDF|[pP]\.)\s*)?(\d+)\s*(?:쪽\s*)?(?:([~\-–～∼〜]|이후)\s*(?:(\d+)\s*(?:쪽)?)?)?`)
	barePagesRe   = regexp.MustCompile(`수록\s*\(?\s*(\d+)\s*(?:([~\-–～∼〜])\s*(\d+)?)?\s*\)?\s*$`)
	fileNameRe    = regexp.MustCompile(`\.[A-Za-z0-9]{2,5}$`)
	textFileRe    = regexp.MustCompile(`\.[A-Za-z][A-Za-z0-9]{1,4}$`)
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

// ParsePages accepts one unambiguous report page reference. Prose needs a page
// marker, so a guide's revision date cannot silently become a page number.
func ParsePages(value string) (pages string, ok bool) {
	value = strings.TrimSpace(value)
	m := reportPagesRe.FindStringSubmatch(value)
	if m == nil && strings.Contains(value, "수록") && !textFileRe.MatchString(value) {
		for _, loc := range prosePagesRe.FindAllStringSubmatchIndex(value, -1) {
			candidate := value[loc[0]:loc[1]]
			if !strings.Contains(candidate, "쪽") && !strings.Contains(candidate, "PDF") && !strings.Contains(strings.ToLower(candidate), "p.") {
				continue
			}
			if loc[0] > 0 && strings.ContainsRune(".0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_", rune(value[loc[0]-1])) {
				return "", false
			}
			if m != nil {
				return "", false
			}
			m = []string{candidate, "", "", ""}
			for i := 1; i < len(m); i++ {
				if loc[2*i] >= 0 {
					m[i] = value[loc[2*i]:loc[2*i+1]]
				}
			}
		}
		if m == nil {
			m = barePagesRe.FindStringSubmatch(value)
		}
	}
	if m == nil {
		return "", false
	}
	a, err := strconv.Atoi(m[1])
	if err != nil || a < 1 {
		return "", false
	}
	switch {
	case m[3] != "":
		if b, err := strconv.Atoi(m[3]); err != nil || b < a {
			return "", false
		}
		return m[1] + "-" + m[3], true
	case m[2] != "":
		return m[1] + "-", true
	}
	return m[1], true
}

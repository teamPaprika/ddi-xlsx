package check

import (
	"regexp"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
)

// ncNameRe is the XML NCName production (XML 1.0 fifth edition Name without
// ":"). ID forms are checked here even though the DDI XSD types them xs:ID,
// so the result does not hang on which validator the caller injects.
var ncNameRe = regexp.MustCompile(`^[A-Z_a-z\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2FF}\x{370}-\x{37D}\x{37F}-\x{1FFF}` +
	`\x{200C}-\x{200D}\x{2070}-\x{218F}\x{2C00}-\x{2FEF}\x{3001}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFFD}\x{10000}-\x{EFFFF}]` +
	`[A-Z_a-z\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2FF}\x{370}-\x{37D}\x{37F}-\x{1FFF}` +
	`\x{200C}-\x{200D}\x{2070}-\x{218F}\x{2C00}-\x{2FEF}\x{3001}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFFD}\x{10000}-\x{EFFFF}` +
	`\-.0-9\x{B7}\x{300}-\x{36F}\x{203F}-\x{2040}]*$`)

// varRefs and fileRefs list, per element, the attributes whose values are
// space-separated var IDs or fileDscr IDs.
var (
	varRefs = map[string][]string{
		"varGrp": {"var"}, "var": {"wgt-var", "kosac:parentVar"},
		"derivation": {"var"}, "qstn": {"var"}, "recGrp": {"keyvar"},
		"kosac:idVar": {"ref"}, "kosac:systemIdVar": {"ref"}, "kosac:link": {"var"},
		"kosac:sampling": {"strataVars", "clusterVar"}, "kosac:splitBallot": {"var"}, "kosac:weight": {"var"},
		"kosac:rule": {"var"}, "kosac:unverified": {"var"},
	}
	fileRefs = map[string][]string{
		"var": {"files"}, "location": {"fileid"}, "kosac:link": {"fileRef"}, "kosac:unverified": {"file"},
	}
)

// ref checks that every ID, the root's included, is an NCName used once, and
// that every reference names a var or fileDscr in the document: libxml2 does
// not resolve IDREF. It reads the tree as written, so it also judges hand-edited XML,
// where a dangling reference is most likely.
func ref(root *ddi.Node) []kosac.Issue {
	var out []kosac.Issue
	seen, vars, files := map[string]bool{}, map[string]bool{}, map[string]bool{}
	kosac.Walk(root, func(n, _ *ddi.Node, path string) bool {
		id, ok := n.Attr("ID")
		if !ok {
			return true
		}
		switch {
		case !ncNameRe.MatchString(id):
			out = append(out, refIssue(path+"/@ID", id, "NCName", "ID가 XML 이름(NCName) 형식이 아닙니다."))
		case seen[id]:
			out = append(out, refIssue(path+"/@ID", id, "문서 안에서 유일한 ID", "같은 ID가 문서에 두 번 이상 있습니다."))
		}
		seen[id] = true
		switch n.Name {
		case "var":
			vars[id] = true
		case "fileDscr":
			files[id] = true
		}
		return true
	})
	kosac.Walk(root, func(n, _ *ddi.Node, path string) bool {
		attrs := map[string][]string{"codingInstructions": {"relatedProcesses"}, "Link": {"refs"}, "var": {"weight", "qstn"}, "recGrp": {"recGrp"}}[n.Name]
		for _, name := range attrs {
			value, _ := n.Attr(name)
			for _, id := range strings.Fields(value) {
				if !seen[id] {
					out = append(out, refIssue(path+"/@"+name, id, "문서에 정의된 ID", "참조한 ID가 문서에 없습니다."))
				}
			}
		}
		for _, c := range []struct {
			attrs   []string
			ids     map[string]bool
			want    string
			message string
		}{
			{varRefs[n.Name], vars, "var의 ID", "참조한 변수 ID가 문서에 없습니다."},
			{fileRefs[n.Name], files, "fileDscr의 ID", "참조한 파일 ID가 문서에 없습니다."},
		} {
			for _, a := range c.attrs {
				v, _ := n.Attr(a)
				for _, id := range strings.Fields(v) {
					if !c.ids[id] {
						out = append(out, refIssue(path+"/@"+a, id, c.want, c.message))
					}
				}
			}
		}
		return true
	})
	return out
}

func refIssue(path, actual, expected, msg string) kosac.Issue {
	return kosac.Issue{Rule: "REF", Level: kosac.R, Path: path, Actual: actual, Expected: expected, Message: msg}
}

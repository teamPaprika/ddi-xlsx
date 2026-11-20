package check

import (
	"slices"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
)

// Only the foundation profile declares this marker. Other DDI documents may use
// notes and missing-value vocabularies defined by their own producers.
func profileXML423(root *ddi.Node) []kosac.Issue {
	if !kosac.IsProfile423DDI(root) {
		return nil
	}
	notesByParent := map[string][]string{
		"docDscr/citation/verStmt":  {"TEMPLATE"},
		"stdyDscr/citation/verStmt": {"RULE_VERSION"},
		"stdyDscr":                  {"SURVEY_TYPE", "RELATED_PROGRAM", "PROGRAM_SUB", "BIZ_NAME", "BIZ_YEAR", "ORG_CD", "AREA_CD", "TARGET_CD", "SCHOOL_CODE_YEAR", "SCALE_EXCEPTION"},
		"fileDscr":                  {"FILE_LINK", "SURVEY_TIME", "TIME_VAR", "NOTE"},
		"dataDscr/var":              {"KEY_ROLE", "UNIVERSE_EXPR", "REVERSE", "PII", "SENSITIVE", "MISSING_STATUS_VAR", "SOURCE_VAR", "NOTE", "CATGRY_NOTE"},
		"dataDscr/varGrp":           {"MR_TYPE", "SPLIT_VAR"},
	}
	var issues []kosac.Issue
	var visit func(*ddi.Node, string)
	visit = func(n *ddi.Node, parent string) {
		path := parent + "/" + n.Name
		for _, a := range n.Attrs {
			if a.Name == "xmlns:kosac" || (strings.HasPrefix(a.Name, "xmlns:") && a.Value == kosac.Namespace) {
				issues = append(issues, kosac.Issue{Rule: "S19", Level: kosac.R, Path: path, Actual: a.Value, Message: "재단 정보는 별도 네임스페이스 대신 허용된 notes에 기록하세요."})
			}
		}
		switch n.Name {
		case "notes":
			typ, _ := n.Attr("type")
			location := strings.TrimPrefix(parent, "/codeBook/")
			if typ == "" || strings.HasPrefix(typ, "KOSAC:") && !slices.Contains(notesByParent[location], strings.TrimPrefix(typ, "KOSAC:")) {
				issues = append(issues, kosac.Issue{Rule: "S19", Level: kosac.R, Path: path, Actual: typ, Message: "notes의 유형과 위치를 재단 규칙에서 확인하세요."})
			}
		case "catgry":
			missing, _ := n.Attr("missing")
			mt, _ := n.Attr("missType")
			if !(missing == "N" && mt == "" || missing == "Y" && slices.Contains([]string{"NO_ANSWER", "NOT_APPLICABLE", "DONT_KNOW_REFUSED"}, mt)) {
				issues = append(issues, kosac.Issue{Rule: "S20", Level: kosac.R, Path: path, Actual: missing + " " + mt, Message: "범주의 결측 여부와 missType을 확인하세요."})
			}
		}
		for _, c := range n.Children {
			visit(c, path)
		}
	}
	visit(root, "")
	return issues
}

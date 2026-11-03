package check

import (
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
)

// Schema names the XSD a Validator uses.
type Schema int

const (
	SchemaDDI   Schema = iota // the official DDI Codebook 2.5 XSD
	SchemaKosac               // Deprecated: ValidateXML checks only the complete DDI document.
)

// Validator validates doc against schema s and returns one message per
// problem. err means the validation could not run at all. The engine does no
// process I/O, so the caller supplies it (xmllint, a libxml2 binding). A nil
// Validator stands for none available: ValidateXML skips the XSD step.
type Validator func(s Schema, doc []byte) (problems []string, err error)

// ValidateXML checks references and validates the complete supplied document.
// It never removes extensions before the official DDI 2.5 XSD check.
func ValidateXML(xml []byte, v Validator) []kosac.Issue {
	root, _, err := ddi.Parse(xml)
	if err != nil {
		return []kosac.Issue{{Rule: "XSD-DDI", Level: kosac.R, Actual: err.Error(), Message: "XML을 읽을 수 없어 검증하지 못했습니다."}}
	}
	issues := ref(root)
	found := false
	kosac.Walk(root, func(n, _ *ddi.Node, path string) bool {
		if found {
			return false
		}
		extended := strings.HasPrefix(n.Name, "kosac:")
		for _, a := range n.Attrs {
			extended = extended || strings.HasPrefix(a.Name, "kosac:")
		}
		if extended {
			found = true
			issues = append(issues, kosac.Issue{Rule: "DDI-EXTENSION", Level: kosac.R, Path: path, Message: "kosac 확장이 포함되어 있습니다. DDI 2.5 표준 출력으로 변환해야 합니다."})
			return false
		}
		return true
	})
	if v == nil {
		return issues
	}
	problems, err := v(SchemaDDI, xml)
	if err != nil {
		return append(issues, notRun("XSD-DDI", err))
	}
	for _, p := range problems {
		issues = append(issues, kosac.Issue{Rule: "XSD-DDI", Level: kosac.R, Actual: p, Message: "DDI 문서가 DDI 2.5 XSD를 통과하지 못했습니다."})
	}
	return issues
}

// notRun reports a validator that could not run.
func notRun(rule string, err error) kosac.Issue {
	return kosac.Issue{Rule: rule, Level: kosac.R, Actual: err.Error(), Message: "XSD 검증을 실행할 수 없습니다."}
}

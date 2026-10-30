package check

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
)

// Schema names the XSD a Validator checks a projection against.
type Schema int

const (
	SchemaDDI   Schema = iota // the official DDI Codebook 2.5 XSD
	SchemaKosac               // the kosac 3.3 XSD
)

// Validator validates doc against schema s and returns one message per
// problem. err means the validation could not run at all. The engine does no
// process I/O, so the caller supplies it (xmllint, a libxml2 binding). A nil
// Validator stands for none available: ValidateXML skips both XSD steps.
type Validator func(s Schema, doc []byte) (problems []string, err error)

// ValidateXML checks a codebook in four steps and reports every finding as an
// R issue: PLACE (kosac nodes only where the placement table allows them) and
// REF (IDs and references) in Go, then the DDI projection against SchemaDDI
// (XSD-DDI) and the kosac projection against SchemaKosac (XSD-KOSAC). The
// official XSD accepts kosac nodes in no position, hence two projections.
// With a nil v only PLACE and REF run.
func ValidateXML(xml []byte, v Validator) []kosac.Issue {
	root, _, err := ddi.Parse(xml)
	if err != nil {
		return []kosac.Issue{{Rule: "XSD-DDI", Level: kosac.R, Actual: err.Error(), Message: "XML을 읽을 수 없어 검증하지 못했습니다."}}
	}
	issues := append(place(root), ref(root)...)
	if v == nil {
		return issues
	}

	ddiDoc := ddi.WriteXML(kosac.DDIProjection(root))
	problems, err := v(SchemaDDI, ddiDoc)
	if err != nil {
		return append(issues, notRun("XSD-DDI", err))
	}
	for _, p := range problems {
		issues = append(issues, kosac.Issue{Rule: "XSD-DDI", Level: kosac.R, Actual: p, Message: "DDI 투영이 DDI 2.5 XSD를 통과하지 못했습니다."})
	}

	proj := kosac.KosacProjection(root)
	kosacDoc := ddi.WriteXML(proj)
	problems, err = v(SchemaKosac, kosacDoc)
	if err != nil {
		return append(issues, notRun("XSD-KOSAC", err))
	}
	pathAt := placePaths(kosacDoc, proj)
	for _, p := range problems {
		issues = append(issues, kosac.Issue{Rule: "XSD-KOSAC", Level: kosac.R, Path: pathAt(p), Actual: p,
			Message: "kosac 투영이 kosac 3.3 XSD를 통과하지 못했습니다."})
	}
	return issues
}

// notRun reports a validator that could not run. The first failure ends the
// validation: the other schema would most likely fail the same way.
func notRun(rule string, err error) kosac.Issue {
	return kosac.Issue{Rule: rule, Level: kosac.R, Actual: err.Error(), Message: "XSD 검증을 실행할 수 없습니다."}
}

// libxml2, which xmllint and its bindings share, starts a message with
// "{file}:{line}:".
var lineRe = regexp.MustCompile(`:(\d+): `)

// placePaths maps a validator message to the @path of the place whose lines
// hold it, or "" when the message names no line. WriteXML starts each place on
// its own line indented by two spaces; when kept-as-written content repeats
// that start, the count no longer matches and every path is left out rather
// than guessed.
func placePaths(doc []byte, proj *ddi.Node) func(problem string) string {
	var starts []int
	for i, line := range strings.Split(string(doc), "\n") {
		if strings.HasPrefix(line, "  <kosac:place ") {
			starts = append(starts, i+1)
		}
	}
	if len(starts) != len(proj.Children) {
		starts = nil
	}
	return func(problem string) string {
		m := lineRe.FindStringSubmatch(problem)
		if m == nil {
			return ""
		}
		line, _ := strconv.Atoi(m[1])
		i, found := slices.BinarySearch(starts, line)
		if !found {
			i--
		}
		if i < 0 {
			return ""
		}
		v, _ := proj.Children[i].Attr("path")
		return v
	}
}

// allowedUnder inverts a placement table: kosac local name → DDI elements
// that may hold it.
func allowedUnder(table map[string][]string) map[string][]string {
	out := map[string][]string{}
	for parent, names := range table {
		for _, n := range names {
			out[n] = append(out[n], parent)
		}
	}
	for _, ps := range out {
		slices.Sort(ps)
	}
	return out
}

var (
	childParents = allowedUnder(kosac.KosacChildren)
	attrParents  = allowedUnder(kosac.KosacAttrs)
)

// place reports a kosac element under a DDI element the placement table does
// not give it, and a kosac attribute on such an element. What lies inside a
// kosac element is the kosac XSD's to judge.
func place(root *ddi.Node) []kosac.Issue {
	var out []kosac.Issue
	kosac.Walk(root, func(n, parent *ddi.Node, path string) bool {
		if local, ok := strings.CutPrefix(n.Name, "kosac:"); ok {
			// The root is always codeBook, so a kosac element has a parent.
			if !slices.Contains(kosac.KosacChildren[parent.Name], local) {
				out = append(out, misplaced(path, parent.Name, childParents[local], "kosac 요소가 정해진 부모 아래에 있지 않습니다."))
			}
			return false
		}
		for _, a := range n.Attrs {
			if local, ok := strings.CutPrefix(a.Name, "kosac:"); ok && !slices.Contains(kosac.KosacAttrs[n.Name], local) {
				out = append(out, misplaced(path+"/@"+a.Name, n.Name, attrParents[local], "kosac 속성이 정해진 요소에 있지 않습니다."))
			}
		}
		return true
	})
	return out
}

// misplaced names the element holding the node (Actual) and those that may
// hold it (Expected).
func misplaced(path, holder string, allowed []string, msg string) kosac.Issue {
	return kosac.Issue{Rule: "PLACE", Level: kosac.R, Path: path, Actual: holder, Expected: strings.Join(allowed, ", "), Message: msg}
}

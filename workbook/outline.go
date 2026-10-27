package workbook

import (
	"strconv"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// outline adds a Document row for n and each element below it, in document
// order. The var and varGrp children of dataDscr are left out: their content
// lives on the Variables, Categories and Groups sheets.
func outline(doc *sheet, n *ddi.Node, level int) {
	text, kind := n.Text, ""
	if n.Raw != "" {
		text, kind = n.Raw, "xml"
	}
	doc.add(map[string]string{
		"level": strconv.Itoa(level), "element": n.Name, "attributes": ddi.FormatAttrs(n.Attrs), "text": text, "kind": kind,
	})
	for _, c := range n.Children {
		if level == 1 && n.Name == "dataDscr" && (c.Name == "var" || c.Name == "varGrp") {
			continue
		}
		outline(doc, c, level+1)
	}
}

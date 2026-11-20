package check

import (
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"strings"
	"testing"
)

func TestProfile423XMLRulesUseDeclaredProfile(t *testing.T) {
	const marker = "<docDscr><citation><verStmt><notes type=\"KOSAC:TEMPLATE\">KOSAC-SURVEY-META 4.2.3</notes></verStmt></citation></docDscr>"
	xml := "<codeBook xmlns=\"ddi:codebook:2_5\" version=\"2.5\">" + marker + "<dataDscr><var><catgry/><notes type=\"KOSAC:FILE_LINK\">bad</notes></var></dataDscr></codeBook>"
	for _, tc := range []struct {
		doc   string
		count int
	}{{xml, 2}, {strings.Replace(xml, marker, "", 1), 0}} {
		issues := ValidateXML([]byte(tc.doc), nil)
		if len(issues) != tc.count {
			t.Fatal(issues)
		}
		for _, i := range issues {
			if i.Level != kosac.R || (i.Rule != "S19" && i.Rule != "S20") {
				t.Fatal(i)
			}
		}
	}
}

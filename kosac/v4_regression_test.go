package kosac

import (
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

func TestSuppliedPageReferences(t *testing.T) {
	for _, tc := range []struct{ value, pages string }{
		{"최종보고서 부록 수록(부록 설문지, 보고서 331쪽 이후)", "331-"},
		{"최종보고서 부록 수록(설문지 97~104쪽). 변수가이드(학생 251128·교사 251117)", "97-104"},
		{"최종보고서 부록 수록(초등 97쪽, 교사 119쪽~). 변수가이드(251117)", ""},
		{"최종보고서 부록 수록. 변수가이드(학생 251128·교사 251117)", ""},
		{"최종보고서 부록 수록. 변수가이드(251117)", ""},
		{"보고서 수록 1.5쪽", ""},
		{"보고서 수록 0쪽", ""},
		{"보고서 수록 104~97쪽", ""},
		{"부록수록_331쪽.pdf", ""},
	} {
		t.Run(tc.value, func(t *testing.T) {
			pages, ok := ParsePages(tc.value)
			if pages != tc.pages || ok != (tc.pages != "") {
				t.Fatalf("got %q %v, want %q", pages, ok, tc.pages)
			}
		})
	}
}

func TestV4CountNoteExportAndPreservation(t *testing.T) {
	cb := smallModel()
	cb.Template.Version.Value = "4.0"
	for _, c := range cb.Codes {
		value := "아니오"
		if c.Type != "" && c.Type != "VALID" {
			value = "예"
		}
		c.Rec.Set("IS_MISSING", value)
	}
	raw := "30530 (학생 30,004 + 교사 526, 중복 없음)"
	cb.Study.Set("VALID_N", raw)
	xml, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	root, _, err := ddi.Parse(xml)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(xml), "kosac:") {
		t.Fatal("private extension exported")
	}
	if !strings.Contains(string(xml), raw) || cb.Study.Get("VALID_N") != raw {
		t.Fatal("source count note lost")
	}
	if len(kids(child(root, "dataDscr"), "var")) != len(cb.Vars) {
		t.Fatal("invented variables")
	}
}

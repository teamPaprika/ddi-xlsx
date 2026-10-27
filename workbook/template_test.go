package workbook

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
	"github.com/xuri/excelize/v2"
)

func openTemplate(t *testing.T) *excelize.File {
	t.Helper()
	x, err := Template()
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(x))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestTemplateIsBlankAndValid(t *testing.T) {
	f := openTemplate(t)
	if got, want := f.GetSheetList(), []string{varSheet, catSheet, grpSheet, docSheet}; !slices.Equal(got, want) {
		t.Fatalf("sheets = %q, want %q", got, want)
	}
	for sheet, headers := range map[string][]string{varSheet: varHeaders, catSheet: catHeaders, grpSheet: grpHeaders} {
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || !slices.Equal(rows[0], headers) {
			t.Errorf("%s rows = %q, want only the header %q", sheet, rows, headers)
		}
	}
	rows, err := f.GetRows(docSheet)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		docHeaders,
		{"0", "codeBook", `xmlns="ddi:codebook:2_5" version="2.5"`},
		{"1", "stdyDscr"},
		{"2", "citation"},
		{"3", "titlStmt"},
		{"4", "titl"},
		{"1", "dataDscr"},
	}
	if fmt.Sprintf("%q", rows) != fmt.Sprintf("%q", want) {
		t.Errorf("%s rows = %q, want %q", docSheet, rows, want)
	}

	xo, w, err := fromFile(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) > 0 {
		t.Errorf("warnings = %q", w)
	}
	xsdtest.Validate(t, xo)
}

// filledTemplate is the template as a researcher fills it in: no IDs typed,
// categories and the group naming their variables by name.
func filledTemplate(t *testing.T) *excelize.File {
	t.Helper()
	f := openTemplate(t)
	for i, v := range [][2]string{{"age", "나이"}, {"sex", "성별"}, {"q1", "문항 1"}} {
		set(t, f, varSheet, "name", i+2, v[0])
		set(t, f, varSheet, "label", i+2, v[1])
	}
	for i, c := range [][2]string{{"1", "남"}, {"2", "여"}} {
		set(t, f, catSheet, "varName", i+2, "sex")
		set(t, f, catSheet, "code", i+2, c[0])
		set(t, f, catSheet, "label", i+2, c[1])
	}
	set(t, f, grpSheet, "type", 2, "multipleResp")
	set(t, f, grpSheet, "vars", 2, "sex q1")
	return f
}

// varIDs maps each var name in the DDI output xo to the var's ID.
func varIDs(t *testing.T, xo []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, dd := range parse(t, xo).Children {
		for _, v := range dd.Children {
			if dd.Name == "dataDscr" && v.Name == "var" {
				name, _ := v.Attr("name")
				out[name], _ = v.Attr("ID")
			}
		}
	}
	return out
}

func TestFilledTemplate(t *testing.T) {
	xo, w, err := fromFile(t, filledTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(w) > 0 {
		t.Errorf("warnings = %q", w)
	}
	if got, want := varIDs(t, xo), map[string]string{"age": "V1", "sex": "V2", "q1": "V3"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("var IDs = %v, want %v", got, want)
	}
	if got, want := names(findVar(t, parse(t, xo), "V2").Children), []string{"labl", "catgry", "catgry"}; !slices.Equal(got, want) {
		t.Errorf("V2 children = %q, want %q", got, want)
	}
	if want := `<varGrp ID="VG1" type="multipleResp" var="V2 V3"`; !strings.Contains(string(xo), want) {
		t.Errorf("output has no %s:\n%s", want, xo)
	}
	xsdtest.Validate(t, xo)
}

func TestAutoIDFixedPoint(t *testing.T) {
	xo, _, err := fromFile(t, filledTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	if xo2, _, _ := roundTrip(t, xo); !bytes.Equal(xo2, xo) {
		t.Errorf("converting the output again changes it:\n%s\nthen\n%s", xo, xo2)
	}
}

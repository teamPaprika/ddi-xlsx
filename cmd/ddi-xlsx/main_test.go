package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/check"
	ktemplate "github.com/teamPaprika/ddi-xlsx/kosac/template"
	"github.com/xuri/excelize/v2"
)

func runCLI(args ...string) (code int, stderr string) {
	code, _, stderr = runOut(args...)
	return code, stderr
}

func runOut(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestToAndFrom(t *testing.T) {
	golden := filepath.Join(xsdtest.Root(), "testdata", "kofac_2024_public")
	src, err := os.ReadFile(filepath.Join(golden, "survey.xml"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "survey.xml")
	if err := os.WriteFile(in, src, 0o644); err != nil {
		t.Fatal(err)
	}

	if code, stderr := runCLI("to-xlsx", in); code != 0 {
		t.Fatalf("to-xlsx exit %d, stderr %q", code, stderr)
	}
	xlsx := filepath.Join(dir, "survey.xlsx")
	if _, err := os.Stat(xlsx); err != nil {
		t.Fatal(err)
	}

	oXML, oJSON := filepath.Join(dir, "o.xml"), filepath.Join(dir, "o.json")
	if code, stderr := runCLI("from-xlsx", xlsx, "--xml", oXML, "--json", oJSON); code != 0 {
		t.Fatalf("from-xlsx exit %d, stderr %q", code, stderr)
	}
	for got, want := range map[string]string{oXML: "canonical.xml", oJSON: "canonical.json"} {
		g, err := os.ReadFile(got)
		if err != nil {
			t.Fatal(err)
		}
		w, err := os.ReadFile(filepath.Join(golden, want))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(g, w) {
			t.Errorf("%s differs from %s", filepath.Base(got), want)
		}
	}
}

func TestDefaultPaths(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(xsdtest.Root(), "testdata", "kofac_2024_public", "survey.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "survey")
	if err := os.WriteFile(in, src, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, stderr := runCLI("to-xlsx", in); code != 0 {
		t.Fatalf("to-xlsx exit %d, stderr %q", code, stderr)
	}
	if code, stderr := runCLI("from-xlsx", in+".xlsx"); code != 0 {
		t.Fatalf("from-xlsx exit %d, stderr %q", code, stderr)
	}
	if _, err := os.Stat(in + ".xml"); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(in + ".json"); err == nil {
		t.Error("from-xlsx without flags wrote JSON")
	}

	jsonOnly := filepath.Join(dir, "only.json")
	if code, stderr := runCLI("from-xlsx", "--json", jsonOnly, in+".xlsx"); code != 0 {
		t.Fatalf("from-xlsx --json exit %d, stderr %q", code, stderr)
	}
	if _, err := os.Stat(jsonOnly); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "only.xml")); err == nil {
		t.Error("from-xlsx --json wrote XML")
	}

	book, err := os.ReadFile(in + ".xlsx")
	if err != nil {
		t.Fatal(err)
	}
	misnamed := filepath.Join(dir, "book.xml")
	if err := os.WriteFile(misnamed, book, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, stderr := runCLI("from-xlsx", misnamed, "--xml", misnamed); code != 1 || !strings.Contains(stderr, "overwrite") {
		t.Errorf("output equal to input: exit %d, stderr %q", code, stderr)
	}
	if got, _ := os.ReadFile(misnamed); !bytes.Equal(got, book) {
		t.Error("from-xlsx overwrote its input")
	}
}

func TestDefaultTargetExists(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(xsdtest.Root(), "testdata", "kofac_2024_public", "survey.xml"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "survey.xml")
	if err := os.WriteFile(in, src, 0o644); err != nil {
		t.Fatal(err)
	}
	keep := []byte("keep")
	xlsx := filepath.Join(dir, "survey.xlsx")
	if err := os.WriteFile(xlsx, keep, 0o644); err != nil {
		t.Fatal(err)
	}
	unchanged := func(path string) {
		t.Helper()
		if got, _ := os.ReadFile(path); !bytes.Equal(got, keep) {
			t.Errorf("%s was overwritten", filepath.Base(path))
		}
	}

	code, stderr := runCLI("to-xlsx", in)
	if want := "error: " + xlsx + " already exists; pass -o to choose the output file\n"; code != 1 || stderr != want {
		t.Errorf("to-xlsx: exit %d, stderr %q, want 1 and %q", code, stderr, want)
	}
	unchanged(xlsx)

	// The check runs before reading the input, so a missing input still reports the existing target.
	if err := os.WriteFile(filepath.Join(dir, "missing.xlsx"), keep, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, stderr := runCLI("to-xlsx", filepath.Join(dir, "missing.xml")); !strings.Contains(stderr, "already exists") {
		t.Errorf("to-xlsx did not fail fast: stderr %q", stderr)
	}

	if code, stderr := runCLI("to-xlsx", in, "-o", xlsx); code != 0 {
		t.Fatalf("explicit -o onto an existing file: exit %d, stderr %q", code, stderr)
	}
	if got, _ := os.ReadFile(xlsx); bytes.Equal(got, keep) {
		t.Error("explicit -o did not overwrite")
	}

	if err := os.WriteFile(in, keep, 0o644); err != nil {
		t.Fatal(err)
	}
	code, stderr = runCLI("from-xlsx", xlsx)
	if want := "error: " + in + " already exists; pass --xml or --json to choose the output file\n"; code != 1 || stderr != want {
		t.Errorf("from-xlsx: exit %d, stderr %q, want 1 and %q", code, stderr, want)
	}
	unchanged(in)
}

func TestUsage(t *testing.T) {
	const usage = "usage: ddi-xlsx to-xlsx <in.xml|in.json> [-o out.xlsx]\n" +
		"       ddi-xlsx from-xlsx <in.xlsx> [--xml out.xml] [--json out.json]\n" +
		"       ddi-xlsx template [-o template.xlsx]\n" +
		"       ddi-xlsx kosac-check <양식.xlsx>\n" +
		"       ddi-xlsx kosac-xml <양식.xlsx> [-o out.xml]\n" +
		"       ddi-xlsx kosac-template [-o kosac-template.xlsx]\n"
	for _, args := range [][]string{
		nil,
		{"bogus"},
		{"to-xlsx"},
		{"to-xlsx", "a.xml", "b.xml"},
		{"from-xlsx", "a.xlsx", "--nope"},
		{"template", "a.xlsx"},
		{"template", "--nope"},
		{"kosac-check"},
		{"kosac-check", "a.xlsx", "-o", "b"},
		{"kosac-xml", "a.xlsx", "b.xlsx"},
		{"kosac-template", "a.xlsx"},
	} {
		code, stderr := runCLI(args...)
		if code != 2 {
			t.Errorf("%q: exit %d, want 2", args, code)
		}
		if stderr != usage {
			t.Errorf("%q: stderr %q, want the usage", args, stderr)
		}
	}
}

func TestErrorExit(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "old.xml")
	if err := os.WriteFile(in, []byte(`<codeBook xmlns="http://www.icpsr.umich.edu/DDI"/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stderr := runCLI("to-xlsx", in, "-o", filepath.Join(dir, "old.xlsx"))
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.HasPrefix(stderr, "error: ") || !strings.Contains(stderr, "DDI 1.2.2 is not supported") {
		t.Errorf("stderr %q", stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "old.xlsx")); err == nil {
		t.Error("wrote an output file after a failed conversion")
	}

	if code, stderr := runCLI("from-xlsx", filepath.Join(dir, "missing.xlsx")); code != 1 || !strings.HasPrefix(stderr, "error: ") {
		t.Errorf("missing input: exit %d, stderr %q", code, stderr)
	}
}

func TestTemplate(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "form.xlsx")
	if code, stderr := runCLI("template", "-o", p); code != 0 {
		t.Fatalf("template -o: exit %d, stderr %q", code, stderr)
	}
	if code, stderr := runCLI("from-xlsx", p); code != 0 {
		t.Fatalf("from-xlsx of the template: exit %d, stderr %q", code, stderr)
	}

	t.Chdir(t.TempDir())
	if code, stderr := runCLI("template"); code != 0 {
		t.Fatalf("template: exit %d, stderr %q", code, stderr)
	}
	keep := []byte("keep")
	if err := os.WriteFile("template.xlsx", keep, 0o644); err != nil {
		t.Fatal(err)
	}
	code, stderr := runCLI("template")
	if want := "error: template.xlsx already exists; pass -o to choose the output file\n"; code != 1 || stderr != want {
		t.Errorf("template onto an existing template.xlsx: exit %d, stderr %q, want 1 and %q", code, stderr, want)
	}
	if got, _ := os.ReadFile("template.xlsx"); !bytes.Equal(got, keep) {
		t.Error("template overwrote template.xlsx")
	}
}

var kosacSample = filepath.Join(xsdtest.Root(), "testdata", "kosac", "v32_sample_2024.xlsx")

func TestKosacCheck(t *testing.T) {
	code, stdout, stderr := runOut("kosac-check", kosacSample)
	if code != 1 || !strings.Contains(stdout, "R C07 03_변수정의!") {
		t.Errorf("sample: exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}

	// SQ1 has no codes in the sample, its only R; give it two in a copy.
	f, err := excelize.OpenFile(kosacSample)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	row := 4
	for v, _ := f.GetCellValue(kosac.SheetCodes, "B4"); v != ""; v, _ = f.GetCellValue(kosac.SheetCodes, "B"+strconv.Itoa(row)) {
		row++
	}
	for i, label := range []string{"남자", "여자"} {
		for col, v := range []string{"1", "SQ1", strconv.Itoa(i + 1), label} {
			if err := f.SetCellStr(kosac.SheetCodes, string(rune('A'+col))+strconv.Itoa(row+i), v); err != nil {
				t.Fatal(err)
			}
		}
	}
	fixed := filepath.Join(t.TempDir(), "fixed.xlsx")
	if err := f.SaveAs(fixed); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := runOut("kosac-check", fixed); code != 0 {
		t.Errorf("fixed sample: exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestKosacXML(t *testing.T) {
	src, err := os.ReadFile(kosacSample)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "sample.xlsx")
	if err := os.WriteFile(in, src, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := runOut("kosac-xml", in); code != 0 || stdout != "" {
		t.Fatalf("kosac-xml: exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	b, err := os.ReadFile(filepath.Join(dir, "sample.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kosac.Parse(b); err != nil {
		t.Errorf("Parse: %v", err)
	}
	if code, stderr := runCLI("kosac-xml", in); code != 1 || !strings.Contains(stderr, "already exists") {
		t.Errorf("default path taken: exit %d, stderr %q", code, stderr)
	}

	if _, err := exec.LookPath("xmllint"); err == nil {
		bad := bytes.Replace(b, []byte("</stdyDscr>"), []byte("<bogus/></stdyDscr>"), 1)
		is := check.ValidateXML(bad, xmllint)
		if len(is) != 1 || is[0].Rule != "XSD-DDI" || !strings.Contains(is[0].Actual, "bogus") {
			t.Fatalf("xmllint on a broken document: %+v", is)
		}
		var out bytes.Buffer
		printIssues(&out, is)
		if want := "R XSD-DDI " + is[0].Message + ": " + is[0].Actual + "\n"; out.String() != want {
			t.Errorf("XSD line %q, want %q", out.String(), want)
		}
	}

	t.Setenv("PATH", t.TempDir())
	code, stdout, stderr := runOut("kosac-xml", in, "-o", filepath.Join(dir, "o.xml"))
	if code != 0 || stdout != "" || stderr != "XSD 검증 건너뜀: xmllint 없음\n" {
		t.Errorf("without xmllint: exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestKosacTemplate(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k.xlsx")
	if code, stderr := runCLI("kosac-template", "-o", p); code != 0 {
		t.Fatalf("kosac-template -o: exit %d, stderr %q", code, stderr)
	}
	if got, _ := os.ReadFile(p); !bytes.Equal(got, ktemplate.Blank()) {
		t.Error("kosac-template -o did not write the blank")
	}

	t.Chdir(t.TempDir())
	if code, stderr := runCLI("kosac-template"); code != 0 {
		t.Fatalf("kosac-template: exit %d, stderr %q", code, stderr)
	}
	keep := []byte("keep")
	if err := os.WriteFile("kosac-template.xlsx", keep, 0o644); err != nil {
		t.Fatal(err)
	}
	code, stderr := runCLI("kosac-template")
	if want := "error: kosac-template.xlsx already exists; pass -o to choose the output file\n"; code != 1 || stderr != want {
		t.Errorf("kosac-template onto an existing file: exit %d, stderr %q, want 1 and %q", code, stderr, want)
	}
	if got, _ := os.ReadFile("kosac-template.xlsx"); !bytes.Equal(got, keep) {
		t.Error("kosac-template overwrote kosac-template.xlsx")
	}
}

func TestPrintIssues(t *testing.T) {
	var out bytes.Buffer
	err := printIssues(&out, []kosac.Issue{
		{Rule: "C07", Level: kosac.R, Sheet: "03_변수정의", Cell: "B5", Actual: "0", Message: "m1."},
		{Rule: "XSD-KOSAC", Level: kosac.R, Path: "dataDscr/var[@ID='V1']", Actual: "-:3: bad", Message: "m2."},
		{Rule: "XSD-DDI", Level: kosac.R, Actual: "xmllint: exit status 5", Message: "m3."},
		{Rule: "C08", Level: kosac.W, Message: "m4."},
	})
	want := "R C07 03_변수정의!B5 m1.\n" +
		"R XSD-KOSAC dataDscr/var[@ID='V1'] m2.: -:3: bad\n" +
		"R XSD-DDI m3.: xmllint: exit status 5\n" +
		"W C08 m4.\n"
	if out.String() != want || err != errRejected {
		t.Errorf("got %q, %v\nwant %q", out.String(), err, want)
	}
}

// An exit 3 whose every line is filtered out must still fail the document.
func TestXmllintExit3WithoutProblems(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake xmllint is a shell script")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "xmllint"), []byte("#!/bin/sh\necho '- fails to validate' >&2\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	problems, err := xmllint(check.SchemaDDI, []byte("<x/>"))
	if err != nil || len(problems) != 1 || problems[0] != "- fails to validate" {
		t.Errorf("problems %q, err %v", problems, err)
	}
}

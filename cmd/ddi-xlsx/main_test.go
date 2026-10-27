package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
)

func runCLI(args ...string) (code int, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, errb.String()
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
		"       ddi-xlsx template [-o template.xlsx]\n"
	for _, args := range [][]string{
		nil,
		{"bogus"},
		{"to-xlsx"},
		{"to-xlsx", "a.xml", "b.xml"},
		{"from-xlsx", "a.xlsx", "--nope"},
		{"template", "a.xlsx"},
		{"template", "--nope"},
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

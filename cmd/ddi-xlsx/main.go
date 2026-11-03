// Command ddi-xlsx converts a DDI Codebook 2.5 document to an editable xlsx
// and back, and checks and assembles the foundation's KOSAC codebook.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/check"
	ktemplate "github.com/teamPaprika/ddi-xlsx/kosac/template"
	"github.com/teamPaprika/ddi-xlsx/workbook"
)

const usage = "usage: ddi-xlsx to-xlsx <in.xml|in.json> [-o out.xlsx]\n" +
	"       ddi-xlsx from-xlsx <in.xlsx> [--xml out.xml] [--json out.json]\n" +
	"       ddi-xlsx template [-o template.xlsx]\n" +
	"       ddi-xlsx kosac-check <양식.xlsx>\n" +
	"       ddi-xlsx kosac-xml <양식.xlsx> [-o out.xml] [--json out.json]\n" +
	"       ddi-xlsx kosac-template [-o kosac-template.xlsx]\n"

var (
	errUsage = errors.New("usage")
	// errRejected exits 1 after the issues that say why are already printed.
	errRejected = errors.New("rejected")
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	err := errUsage
	if len(args) > 0 {
		switch args[0] {
		case "to-xlsx":
			err = toXLSX(args[1:], stderr)
		case "from-xlsx":
			err = fromXLSX(args[1:], stderr)
		case "template":
			err = template(args[1:])
		case "kosac-check":
			err = kosacCheck(args[1:], stdout)
		case "kosac-xml":
			err = kosacXML(args[1:], stdout, stderr)
		case "kosac-template":
			err = kosacTemplate(args[1:])
		}
	}
	switch {
	case errors.Is(err, errUsage):
		fmt.Fprint(stderr, usage)
		return 2
	case errors.Is(err, errRejected):
		return 1
	case err != nil:
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func toXLSX(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("to-xlsx", flag.ContinueOnError)
	out := fs.String("o", "", "")
	in, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *out == "" {
		*out = withExt(in, ".xlsx")
		if err := mustBeNew(*out, "-o"); err != nil {
			return err
		}
	}
	src, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	x, warns, err := workbook.ToXLSX(src)
	if err != nil {
		return err
	}
	warn(stderr, warns)
	return write(in, *out, x)
}

func fromXLSX(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("from-xlsx", flag.ContinueOnError)
	outXML := fs.String("xml", "", "")
	outJSON := fs.String("json", "", "")
	in, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *outXML == "" && *outJSON == "" {
		*outXML = withExt(in, ".xml")
		if err := mustBeNew(*outXML, "--xml or --json"); err != nil {
			return err
		}
	}
	src, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	xmlOut, jsonOut, warns, err := workbook.FromXLSX(src)
	if err != nil {
		return err
	}
	warn(stderr, warns)
	if *outXML != "" {
		if err := write(in, *outXML, xmlOut); err != nil {
			return err
		}
	}
	if *outJSON != "" {
		return write(in, *outJSON, jsonOut)
	}
	return nil
}

func template(args []string) error { return blank(args, "template.xlsx", workbook.Template) }

func kosacTemplate(args []string) error {
	return blank(args, "kosac-template.xlsx", func() ([]byte, error) { return ktemplate.Blank(), nil })
}

// blank writes an empty form to -o, or to name in the working directory.
func blank(args []string, name string, form func() ([]byte, error)) error {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	out := fs.String("o", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return errUsage
	}
	if *out == "" {
		*out = name
		if err := mustBeNew(*out, "-o"); err != nil {
			return err
		}
	}
	x, err := form()
	if err != nil {
		return err
	}
	return os.WriteFile(*out, x, 0o644)
}

// kosacCheck runs the structure check (S05) and the rules (C01-C18) the way
// the foundation will, so a contractor can fix the workbook before submitting.
func kosacCheck(args []string, stdout io.Writer) error {
	in, err := parse(flag.NewFlagSet("kosac-check", flag.ContinueOnError), args)
	if err != nil {
		return err
	}
	cb, issues, err := readKosac(in)
	if err != nil {
		return err
	}
	// A nil model means S05 rejected the layout; rules on it would misread.
	if cb != nil {
		issues = append(issues, check.Check(cb, kosac.RefData{})...)
	}
	return printIssues(stdout, issues)
}

// kosacXML assembles a preview codebook without a package. The file is
// written before validation so a failing document can be inspected.
func kosacXML(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("kosac-xml", flag.ContinueOnError)
	out := fs.String("o", "", "")
	outJSON := fs.String("json", "", "")
	in, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *out == "" {
		*out = withExt(in, ".xml")
		if err := mustBeNew(*out, "-o"); err != nil {
			return err
		}
	}
	if *outJSON != "" {
		for _, pair := range [][2]string{{in, *outJSON}, {*out, *outJSON}} {
			if sameFile(pair[0], pair[1]) {
				return fmt.Errorf("XML, JSON and input paths must differ")
			}
		}
	}
	cb, issues, err := readKosac(in)
	if err != nil {
		return err
	}
	if cb == nil {
		return printIssues(stdout, issues)
	}
	kosac.DeriveDefaults(cb, kosac.RefData{})
	if err := kosac.AssignIDs(cb, nil); err != nil {
		return err
	}
	root, err := cb.Tree()
	if err != nil {
		return err
	}
	x := ddi.WriteXML(root)
	if err := write(in, *out, x); err != nil {
		return err
	}
	if *outJSON != "" {
		if err := write(in, *outJSON, ddi.WriteJSON(root)); err != nil {
			return err
		}
	}
	var v check.Validator
	if _, err := exec.LookPath("xmllint"); err == nil {
		v = xmllint
	} else {
		fmt.Fprintln(stderr, "XSD 검증 건너뜀: xmllint 없음")
	}
	return printIssues(stdout, append(issues, check.ValidateXML(x, v)...))
}

func readKosac(path string) (*kosac.Codebook, []kosac.Issue, error) {
	x, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return ktemplate.Read(x, ktemplate.ReadOptions{})
}

// printIssues writes one issue per line, located by its cell or else its XML
// path, and returns errRejected when any issue is R. An XSD issue's message is
// the same for every finding; only the validator's text in Actual says what is
// wrong, so it follows the message.
func printIssues(w io.Writer, issues []kosac.Issue) error {
	var err error
	for _, i := range issues {
		f := []string{string(i.Level), i.Rule}
		switch {
		case i.Cell != "":
			f = append(f, i.Sheet+"!"+i.Cell)
		case i.Path != "":
			f = append(f, i.Path)
		}
		msg := i.Message
		if (i.Rule == "XSD-DDI" || i.Rule == "XSD-KOSAC") && i.Actual != "" {
			msg += ": " + i.Actual
		}
		fmt.Fprintln(w, strings.Join(append(f, msg), " "))
		if i.Level == kosac.R {
			err = errRejected
		}
	}
	return err
}

// parse accepts flags before and after the one positional argument, since
// the documented command lines put them after it and flag stops at the first
// non-flag.
func parse(fs *flag.FlagSet, args []string) (string, error) {
	fs.SetOutput(io.Discard)
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return "", errUsage
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
	if len(pos) != 1 {
		return "", errUsage
	}
	return pos[0], nil
}

// mustBeNew guards a derived output path: the user never named it, so
// replacing a file there (often the original source document) would be silent
// data loss. Lstat so a dangling symlink also counts as taken.
func mustBeNew(path, flags string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%s already exists; pass %s to choose the output file", path, flags)
	}
	return nil
}

func withExt(path, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}

// write refuses to replace the input: a workbook saved as book.xml would
// otherwise be lost to the default from-xlsx output path.
func write(in, out string, data []byte) error {
	if sameFile(in, out) {
		return fmt.Errorf("%s: refusing to overwrite the input file", out)
	}
	return os.WriteFile(out, data, 0o644)
}

func warn(stderr io.Writer, warns []string) {
	for _, w := range warns {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
}

// sameFile also catches absolute-path, symlink and hard-link aliases.
func sameFile(a, b string) bool {
	aa, ea := filepath.Abs(a)
	bb, eb := filepath.Abs(b)
	if ea == nil && eb == nil && aa == bb {
		return true
	}
	ai, ea := os.Stat(a)
	bi, eb := os.Stat(b)
	return ea == nil && eb == nil && os.SameFile(ai, bi)
}

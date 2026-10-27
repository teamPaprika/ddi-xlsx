// Command ddi-xlsx converts a DDI Codebook 2.5 document to an editable xlsx and back.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/workbook"
)

const usage = "usage: ddi-xlsx to-xlsx <in.xml|in.json> [-o out.xlsx]\n" +
	"       ddi-xlsx from-xlsx <in.xlsx> [--xml out.xml] [--json out.json]\n" +
	"       ddi-xlsx template [-o template.xlsx]\n"

var errUsage = errors.New("usage")

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
		}
	}
	switch {
	case errors.Is(err, errUsage):
		fmt.Fprint(stderr, usage)
		return 2
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

func template(args []string) error {
	fs := flag.NewFlagSet("template", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	out := fs.String("o", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return errUsage
	}
	if *out == "" {
		*out = "template.xlsx"
		if err := mustBeNew(*out, "-o"); err != nil {
			return err
		}
	}
	x, err := workbook.Template()
	if err != nil {
		return err
	}
	return os.WriteFile(*out, x, 0o644)
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
	if filepath.Clean(in) == filepath.Clean(out) {
		return fmt.Errorf("%s: refusing to overwrite the input file", out)
	}
	return os.WriteFile(out, data, 0o644)
}

func warn(stderr io.Writer, warns []string) {
	for _, w := range warns {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
}

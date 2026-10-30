package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/kosac/check"
	kosacxsd "github.com/teamPaprika/ddi-xlsx/kosac/xsd"
	ddixsd "github.com/teamPaprika/ddi-xlsx/xsd"
)

// xmllint is a check.Validator. codebook.xsd imports its siblings by relative
// path, so both embedded trees are written out whole to one directory.
// ponytail: the trees are rewritten on every call; keep one directory per run
// if validating many documents makes that show.
func xmllint(s check.Schema, doc []byte) ([]string, error) {
	dir, err := os.MkdirTemp("", "ddi-xlsx-xsd-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := os.CopyFS(dir, ddixsd.FS); err != nil {
		return nil, err
	}
	if err := os.CopyFS(dir, kosacxsd.FS); err != nil {
		return nil, err
	}
	schema := filepath.Join(dir, "ddi-codebook-2.5", "codebook.xsd")
	if s == check.SchemaKosac {
		schema = filepath.Join(dir, "kosac-3.3.xsd")
	}

	cmd := exec.Command("xmllint", "--nonet", "--noout", "--schema", schema, "-")
	cmd.Stdin = bytes.NewReader(doc)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err == nil {
		return nil, nil
	}
	// Exit 3 is a document that fails the schema. Any other exit (a schema
	// that does not compile, a document that does not parse) means the
	// validation did not happen, which must not read as a list of findings.
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 3 {
		return nil, fmt.Errorf("xmllint: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var problems []string
	for _, line := range strings.Split(stderr.String(), "\n") {
		if line = strings.TrimSpace(line); line != "" && line != "- fails to validate" {
			problems = append(problems, line)
		}
	}
	// No problems would read as a pass, but xmllint said the document fails.
	if len(problems) == 0 {
		raw := strings.TrimSpace(stderr.String())
		if raw == "" {
			raw = "xmllint: " + err.Error()
		}
		problems = []string{raw}
	}
	return problems, nil
}

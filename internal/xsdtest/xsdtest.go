// Package xsdtest validates DDI Codebook 2.5 XML against the vendored XSD in tests.
package xsdtest

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Root returns the repository root: the nearest ancestor of the working directory that holds go.mod.
func Root() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("xsdtest: go.mod not found above working directory")
		}
		dir = parent
	}
}

// Validate fails t unless xml validates against codebook.xsd. It skips when xmllint is unavailable.
func Validate(t testing.TB, xml []byte) {
	t.Helper()
	if p := ValidateSchema(t, filepath.Join(Root(), "xsd", "ddi-codebook-2.5", "codebook.xsd"), xml); p != nil {
		t.Fatalf("xmllint:\n%s", strings.Join(p, "\n"))
	}
}

// ValidateSchema returns xmllint's messages when xml does not validate against
// the schema at schemaPath, nil when it does. It skips when xmllint is unavailable.
func ValidateSchema(t testing.TB, schemaPath string, xml []byte) []string {
	t.Helper()
	if err := exec.Command("xmllint", "--version").Run(); err != nil {
		t.Skipf("xmllint unavailable: %v", err)
	}
	cmd := exec.Command("xmllint", "--nonet", "--noout", "--schema", schemaPath, "-")
	cmd.Stdin = bytes.NewReader(xml)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	// The exit code is checked as well so a schema load failure cannot pass on a stray "validates".
	if err == nil && strings.Contains(stderr.String(), "validates") {
		return nil
	}
	var problems []string
	for _, line := range strings.Split(stderr.String(), "\n") {
		if line = strings.TrimSpace(line); line != "" && line != "- fails to validate" {
			problems = append(problems, line)
		}
	}
	if len(problems) == 0 {
		problems = []string{fmt.Sprintf("xmllint: %v", err)}
	}
	return problems
}

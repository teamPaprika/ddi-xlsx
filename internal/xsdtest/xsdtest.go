// Package xsdtest validates DDI Codebook 2.5 XML against the vendored XSD in tests.
package xsdtest

import (
	"bytes"
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
	if err := exec.Command("xmllint", "--version").Run(); err != nil {
		t.Skipf("xmllint unavailable: %v", err)
	}
	schema := filepath.Join(Root(), "xsd", "ddi-codebook-2.5", "codebook.xsd")
	cmd := exec.Command("xmllint", "--nonet", "--noout", "--schema", schema, "-")
	cmd.Stdin = bytes.NewReader(xml)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	// The exit code is checked as well so a schema load failure cannot pass on a stray "validates".
	if err != nil || !strings.Contains(stderr.String(), "validates") {
		t.Fatalf("xmllint: %v\n%s", err, stderr.String())
	}
}

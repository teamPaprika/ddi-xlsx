package xsdtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKofacFixturesValidate(t *testing.T) {
	for _, name := range []string{"kofac_2024_public", "kofac_2025_edu"} {
		t.Run(name, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join(Root(), "testdata", name, "survey.xml"))
			if err != nil {
				t.Fatal(err)
			}
			Validate(t, xml)
		})
	}
}

package template

import (
	"bytes"
	_ "embed"
)

// blank retains the legacy layout for v3.2 callers.
//
//go:embed blank_v3.2.xlsx
var blank []byte

//go:embed blank_v4.0.xlsx
var blankV4 []byte

// Blank returns the official empty v4.0 template.
func Blank() []byte { return bytes.Clone(blankV4) }

package template

import (
	"bytes"
	_ "embed"
)

// blank is a stand-in built from the foundation's filled-in example until the
// official blank v3.2 arrives; its _meta NOTE row says so. Replacing this
// file is the whole upgrade.
//
//go:embed blank_v3.2.xlsx
var blank []byte

// Blank returns the empty v3.2 template that Write fills.
func Blank() []byte { return bytes.Clone(blank) }

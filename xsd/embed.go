// Package xsd embeds the official DDI Codebook 2.5 XSD. codebook.xsd imports
// its siblings by relative path, so a validator needs the whole tree written
// out to one directory.
package xsd

import "embed"

//go:embed ddi-codebook-2.5
var FS embed.FS

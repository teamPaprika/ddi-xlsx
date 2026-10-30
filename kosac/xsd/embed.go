// Package xsd embeds the kosac 3.3 extension XSD, which validates the kosac
// projection of a codebook.
package xsd

import "embed"

//go:embed kosac-3.3.xsd
var FS embed.FS

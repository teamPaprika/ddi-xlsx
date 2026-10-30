package template

import "github.com/xuri/excelize/v2"

// unzipLimits bounds what excelize allocates while it unzips, which it sizes
// from what each zip entry declares: a small crafted file could ask for
// gigabytes. A genuine workbook unzips to far less than 100 times its size.
// It is the workbook package's limit, which keeps it unexported.
func unzipLimits(size int) excelize.Options {
	l := max(64<<20, 100*int64(size))
	return excelize.Options{UnzipSizeLimit: l, UnzipXMLSizeLimit: min(l, excelize.StreamChunkSize)}
}

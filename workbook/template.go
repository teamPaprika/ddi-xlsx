package workbook

// skeleton is the least codebook that validates: the XSD requires a study
// title, and the dataDscr row is where the other sheets' rows go.
const skeleton = `<codeBook xmlns="ddi:codebook:2_5" version="2.5">` +
	`<stdyDscr><citation><titlStmt><titl/></titlStmt></citation></stdyDscr><dataDscr/></codeBook>`

// Template returns a blank workbook to fill in and convert with FromXLSX: the
// Variables, Categories and Groups sheets hold only their headers, and the
// Document sheet the least codebook that validates.
func Template() ([]byte, error) {
	x, _, err := ToXLSX([]byte(skeleton))
	return x, err
}

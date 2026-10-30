package kosac

// RefData is the foundation's master data a caller passes in. A nil field
// skips the checks that need it, unless noted.
type RefData struct {
	StdItems      map[string]StdItem    // by STD_ID
	CodeSetDefs   map[string]CodeSetDef // by CODE_SET_ID
	CodeSets      map[string][]StdCode  // by CODE_SET_ID
	MissingLabels map[string][]string   // "-8" → accepted labels, the first being the prescribed one; nil uses the defaults
	UploadedFiles []string
	PDFPages      map[string]int // file name → page count
	ReservedWords []string       // nil uses the default list
}

// StdItem is a row of std_items.csv. List columns are split on ";".
type StdItem struct {
	StdID, NameKo, PhysicalName, MeasLevel, CodeSetID string
	Synonyms                                          []string
	Series, Active                                    string
}

// CodeSetDef is a row of code_set_defs.csv.
type CodeSetDef struct {
	CodeSetID, NameKo, PhysicalName, DimTable, KeyColumn, CodeSystem, ParentSet string
}

// StdCode is a row of code_sets.csv. List columns are split on ";".
type StdCode struct {
	CodeSetID, Code, Label string
	AltLabels              []string
	Sort, ParentCode       string
}

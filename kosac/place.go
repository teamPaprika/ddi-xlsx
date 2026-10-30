package kosac

// Namespace is the kosac extension namespace; its prefix is always "kosac".
const Namespace = "urn:kosac:survey-meta:3.3"

// KosacChildren lists, per DDI parent, the kosac elements it may hold, in the
// order they follow its DDI children. Assembly writes in this order and the
// placement check reads the same table.
var KosacChildren = map[string][]string{
	"docDscr":  {"package"},
	"stdyDscr": {"study"},
	"dataColl": {"sampling", "splitBallot", "weight", "excludeN", "excludeRule"},
	"anlyInfo": {"contactN", "rrFormula"},
	"fileDscr": {"file", "cleansing"},
	"varGrp":   {"noneOption"},
	"var":      {"fk", "scale", "missingCodes", "codeSetId", "analysis"},
	"catgry":   {"codeType", "stdCode"},
	"otherMat": {"pages", "sha256", "bytes"},
}

// KosacAttrs lists the kosac attributes a DDI element may carry, in the order
// they follow its DDI attributes.
var KosacAttrs = map[string][]string{
	"var": {"seq", "role", "measLevel", "pii", "sensitive", "reverse", "mrType", "mrGroup", "rowLabel",
		"rankNo", "cleansed", "textFormat", "parentVar"},
	"varGrp": {"mrGroup"},
}

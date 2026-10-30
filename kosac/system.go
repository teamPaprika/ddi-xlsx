package kosac

import "slices"

// SystemVar is a variable walla adds to every data file. Its values are the
// codebook's, not the contractor's, so no template row describes it.
type SystemVar struct {
	Name, Role, MeasLevel, PII, VarFormat string
	Category                              string // varFormat@category: "date" or ""
	Label                                 string
}

// SystemVars are written first in every file, in this order.
var SystemVars = []SystemVar{
	{"walla_response_id", "SYSTEM", "text", "N", "character", "", "왈라 응답 ID"},
	{"walla_row_no", "SYSTEM", "scale", "N", "numeric", "", "왈라 행 번호"},
	{"walla_customer_key", "SYSTEM", "text", "Y", "character", "", "왈라 고객 키"},
	{"walla_submitted_at", "SYSTEM", "date", "Y", "character", "date", "왈라 제출 시각"},
	{"walla_started_at", "SYSTEM", "date", "Y", "character", "date", "왈라 시작 시각"},
	{"walla_duration_sec", "SYSTEM", "scale", "N", "numeric", "", "왈라 응답 소요 시간(초)"},
}

// IsSystemVar reports whether name is a system var. Every file has them, so a
// reference to one resolves in any file.
func IsSystemVar(name string) bool {
	return slices.ContainsFunc(SystemVars, func(s SystemVar) bool { return s.Name == name })
}

package kosac

import (
	"regexp"
	"slices"
	"strings"
)

const SheetRecodes = "05_재코딩이력"

var StudyExtraV4 = []string{"SERIES_NAME", "STUDY_ROUND", "ACCESS_RESTRICTIONS", "CLEANING_LOG", "SCALE_EXCEPTION", "SOURCE_FILES"}
var FileExtraV4 = []string{"SURVEY_TIME", "TIME_VAR", "UNIQUE_KEY", "VARIABLE_N"}
var VarExtraV4 = []string{"DATA_TYPE", "DDI_INTERVAL", "MISSING_STATUS_VAR", "ACCESS_LEVEL", "CODE_SYSTEM", "SOURCE_VAR", "RECODE_STATUS"}
var CodeExtraV4 = []string{"IS_MISSING", "MISSING_REASON", "RECODE_STATUS"}
var RecodeKeys = []string{"FILE_NO", "SOURCE_FILE", "SOURCE_VAR", "SOURCE_CODE_OR_CONDITION", "TARGET_VAR", "TARGET_CODE_OR_RULE", "TRANSFORM_TYPE", "BASIS", "RECORD_COUNT", "STATUS", "NOTE"}

var annotatedCountRe = regexp.MustCompile(`^([0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)\s*\([^\r\n]*\)$`)

// StudyValidN separates the v4 count from its explanatory note without rewriting the source cell.
func (cb *Codebook) StudyValidN() string {
	value := cb.Study.Get("VALID_N")
	if cb.Template.Version.Value == "4.0" {
		if match := annotatedCountRe.FindStringSubmatch(strings.TrimSpace(value)); match != nil {
			return match[1]
		}
	}
	return value
}

// Layout is the set of keys and the first data row of an accepted template.
type Layout struct {
	Study, Files, Vars, Codes []string
	FirstDataRow              int
}

func LayoutFor(version string) Layout {
	l := Layout{StudyKeys, FileKeys, VarKeys, CodeKeys, 4}
	if strings.TrimSpace(version) == "4.0" {
		l = Layout{append(slices.Clone(StudyKeys), StudyExtraV4...), append(slices.Clone(FileKeys), FileExtraV4...), append(slices.Clone(VarKeys), VarExtraV4...), append(slices.Clone(CodeKeys), CodeExtraV4...), 3}
	}
	return l
}

func (cb *Codebook) HasV4Fields() bool {
	if len(cb.Recodes) > 0 {
		return true
	}
	present := func(r Record, keys []string) bool {
		return slices.ContainsFunc(keys, func(k string) bool { return r.Get(k) != "" })
	}
	if present(cb.Study, StudyExtraV4) {
		return true
	}
	for _, f := range cb.Files {
		if present(f.Rec, FileExtraV4) {
			return true
		}
	}
	for _, v := range cb.Vars {
		if present(v.Rec, VarExtraV4) {
			return true
		}
	}
	for _, c := range cb.Codes {
		if present(c.Rec, CodeExtraV4) {
			return true
		}
	}
	return false
}

func MissingLabelsFor(version string) map[string][]string {
	if version == "4.0" {
		return map[string][]string{"-9": {"무응답"}, "-7": {"모름·거절"}, "-8": {"해당 없음", "비해당"}}
	}
	return DefaultMissingLabels
}

func DeclaredMissing(value string) bool { return value == "예" || value == "Y" }

// V4Absent is an explicit non-applicable marker used in the official examples.
func V4Absent(value string) bool {
	value = strings.TrimSpace(value)
	return value == "해당 없음" || strings.HasPrefix(value, "해당 없음:") || strings.HasPrefix(value, "해당 없음：")
}

func MeasurementLevel(v *Var) string {
	if _, v4 := v.Rec["DATA_TYPE"]; v4 {
		switch v.Rec.Get("MEAS_LEVEL") {
		case "등간", "비율":
			return "scale"
		case "해당 없음":
			return ""
		}
	}
	return MeasLevels[v.Rec.Get("MEAS_LEVEL")]
}

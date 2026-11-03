// Package kosac models v3.2 and v4.0 codebooks: the template's
// sheets as records keyed by template KEY, plus the values the engine and the
// operator add. Every value is the string written, so checks can judge it as
// written.
package kosac

import (
	"regexp"
	"slices"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// Sheet names of the v3.2 template.
const (
	SheetStudy = "01_조사정보"
	SheetFiles = "02_데이터파일"
	SheetVars  = "03_변수정의"
	SheetCodes = "04_코드값"
	SheetMeta  = "_meta"
)

// Loc is where a value was read: a sheet name and an A1 cell. It is zero for
// a value that did not come from a workbook.
type Loc struct{ Sheet, Cell string }

type Field struct {
	Value string
	Loc   Loc
}

// Record maps a template KEY to its cell.
type Record map[string]Field

func (r Record) Get(key string) string { return r[key].Value }

// Set changes the value and keeps where it was read, so a check still points
// at the cell the operator changed.
func (r Record) Set(key, value string) {
	f := r[key]
	f.Value = value
	r[key] = f
}

type Codebook struct {
	Template TemplateMeta
	// SourceSystemVars contains only system columns declared in an imported DDI.
	// External Excel imports leave it empty; output never fabricates these columns.
	SourceSystemVars []*ddi.Node
	Study            Record // 01 sheet
	Admin            StudyAdmin
	Files            []*File  // 02 rows in order
	Vars             []*Var   // 03 rows in order; imported system columns stay separate
	Codes            []*Code  // 04 rows in order, joined to vars by (FILE_NO, VAR_NAME)
	Recodes          []Record // 05 contractor transformations before submission
	Package          *Package
	// Native marks a codebook made from a walla export. Its var names say
	// nothing about grids, so walla's own groups are the only ones. It is
	// model state, not an output protocol. Keep the model for later edits.
	Native bool
}

// TemplateMeta is the _meta sheet.
type TemplateMeta struct{ ID, Version, Created Field }

type StudyAdmin struct {
	CategoryCode string
	Round        Round
	WallaID      string // titlStmt/IDNo@agency="walla"
}

type Round struct{ Year, WaveType, WaveNo string }

// Package is system-given delivery metadata. Standard output describes it
// in docDscr; it does not define a machine-readable package manifest.
type Package struct {
	// No Cleansed: package@cleansed follows from the files' cleansing rules
	// at assembly, so it has no second source here.
	ExportID, CodebookVersion, PreviousVersion, ApprovedBy, ApprovedAt string
	ExportedAt, Software, SoftwareVersion                              string
	Source                                                             SourceSurvey
	Files                                                              []PackageFile // written in this order
	Validation                                                         *Validation
}

// SourceSurvey records the input source in the model.
type SourceSurvey struct{ Kind, PublishedVersion, TemplateVersion string }

type PackageFile struct{ Role, FileRef, Name, Bytes, SHA256, Pages string }

type Validation struct {
	PassedAt, Ruleset, WarningCount string
	Notes                           []string
}

// Suggestion is a judgement the engine offers but does not apply.
type Suggestion struct {
	Kind            string // "STD_ID" | "FK"
	FileNo, VarName string
	Value           string // the STD_ID or the code set ID
	Confidence      float64
	Level           string            // "high" | "check"
	CodeMap         map[string]string // FK only: source code → standard code
}

type File struct {
	Rec                                                      Record // 02 row
	LinkLevel, SensLevel                                     string
	PackageName, SHA256, Bytes, RawName, RawSHA256, RawBytes string
	Cleansing                                                *Cleansing // nil without rules
}

type Var struct {
	ID  string // V{n}
	Rec Record // 03 row
	Analysis
	FK *FK
}

// Analysis is the analysis meta of a var.
type Analysis struct {
	Role, DimMeasure, AggRules, PhysicalName, LogicalName string
	Synonyms                                              []string
	IncludeYn, CodeSetID, MRGroup                         string
	// NoneOption repeats on every member of an MR group rather than living in
	// a map by group key, so groups of different files cannot collide.
	NoneOption, GridGroup, RowLabel, RankNo string
	Scale                                   Scale
	TextFormat, ParentVar                   string
}

type Scale struct{ Min, Max, Dir string }

// FK is kosac:fk, a var's link to a common standard code set.
type FK struct {
	CodeSet, DimTable, KeyColumn, Mapping, CodeSystem, RefYear, LinkKey string
	CodeSetVersion, ValidatedAt, MatchedRows, UnmatchedRows             string
}

type Code struct {
	Rec             Record // 04 row
	Type            string // VALID | MISSING | DK | NA
	StdCode, StdSet string
}

// Cleansing is a file's kosac:cleansing.
type Cleansing struct {
	AppliedBy, AppliedAt, RawFileRef, CellsChanged, RowsAffected, LogFile string
	Rules                                                                 []CleansingRule
}

// CleansingRule names its var by VAR_NAME within the file; assembly writes
// the var's ID.
type CleansingRule struct {
	ID, VarName, Action, Condition, Rows, Source, Reason, Evidence string
	Map                                                            []MapEntry
}

type MapEntry struct{ From, To string }

// ponytail: the lookups scan; index (FILE_NO, VAR_NAME) if a caller loops over them on large codebooks.

// File returns the 02 row whose FILE_NO is fileNo, or nil.
func (cb *Codebook) File(fileNo string) *File {
	for _, f := range cb.Files {
		if f.Rec.Get("FILE_NO") == fileNo {
			return f
		}
	}
	return nil
}

// Var returns the 03 row of name in file fileNo, or nil.
func (cb *Codebook) Var(fileNo, name string) *Var {
	for _, v := range cb.Vars {
		if v.Rec.Get("FILE_NO") == fileNo && v.Rec.Get("VAR_NAME") == name {
			return v
		}
	}
	return nil
}

// CodesOf returns v's 04 rows in order.
func (cb *Codebook) CodesOf(v *Var) []*Code {
	var out []*Code
	for _, c := range cb.Codes {
		if c.Rec.Get("FILE_NO") == v.Rec.Get("FILE_NO") && c.Rec.Get("VAR_NAME") == v.Rec.Get("VAR_NAME") {
			out = append(out, c)
		}
	}
	return out
}

// DefaultMissingLabels are the accepted labels of the missing codes when
// RefData gives none; the first is the prescribed one.
var DefaultMissingLabels = map[string][]string{
	"-8": {"묻지 않음"},
	"-7": {"모르겠다", "무응답", "모르겠다/무응답"},
	"-9": {"유효하지 않음"},
}

// MissingCode is a code that marks a missing value rather than an answer.
type MissingCode struct {
	Code string
	// Type is the kosac:codeType given under an accepted label; -7 is DK only
	// then.
	Type string
	// Declared codes go on every 명목·서열·연속 survey var and lead
	// kosac:missingCodes in this order.
	Declared bool
}

// MissingCodes is every missing code the checks and assembly know, in the
// order the foundation declares them, walla's last. The types of -9 and -6
// are defaults the foundation has yet to confirm; they live only here.
var MissingCodes = []MissingCode{
	{Code: "-8", Type: "MISSING", Declared: true},
	{Code: "-7", Type: "DK", Declared: true},
	// MISSING or NA: not settled by the foundation.
	{Code: "-9", Type: "MISSING", Declared: true},
	// walla's value of an earlier published form; its meaning is not settled.
	{Code: "-6", Type: "MISSING"},
}

// missingCode returns the entry of code, if it is a missing code.
func missingCode(code string) (MissingCode, bool) {
	i := slices.IndexFunc(MissingCodes, func(m MissingCode) bool { return m.Code == code })
	if i < 0 {
		return MissingCode{}, false
	}
	return MissingCodes[i], true
}

// MeasLevels maps the template's MEAS_LEVEL values to kosac:measLevel. Its keys
// are the only levels the template allows.
var MeasLevels = map[string]string{"명목": "nominal", "서열": "ordinal", "연속": "scale", "문자": "text", "날짜": "date"}

// Prefix is the part of a var name before its last "_", which groups the
// columns of one question (QA19_1, QA19_2 → QA19). A name without one has no
// prefix and joins no such group.
func Prefix(name string) (string, bool) {
	i := strings.LastIndex(name, "_")
	if i <= 0 {
		return "", false
	}
	return name[:i], true
}

// (?s) lets a reason typed over several lines in one cell still count.
var unverifiedRe = regexp.MustCompile(`(?s)^확인\s*불가\s*[:：]\s*(.*)$`)

// Unverified reports whether value says the contractor could not confirm it
// ("확인 불가: {reason}"), and the reason.
func Unverified(value string) (reason string, ok bool) {
	m := unverifiedRe.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// The template's KEYs in sheet order. Checks take them from here, never from
// the uploaded file.
var (
	StudyKeys = []string{
		"RULE_VERSION", "TITLE", "PROJECT_TITLE", "PROJECT_NO", "PURPOSE", "SUMMARY", "FUNDER", "PI_ORG",
		"FIELD_ORG", "PI_NAME", "FIELD_START", "FIELD_END", "KEYWORDS", "QUESTIONNAIRE", "REPORT_FILE",
		"EXTRA_FILES", "UNIVERSE", "POP_SOURCE", "ANALYSIS_UNIT", "SAMPLING_TYPE", "SAMPLING_DESC",
		"SAMPLE_FRAME", "STRATA_VARS", "CLUSTER_VAR", "TARGET_N", "VALID_N", "SPLIT_BALLOT", "SPLIT_VAR",
		"MODE", "TOOL", "CONTACT_N", "RESPONSE_RATE", "RR_FORMULA", "EXCLUDE_N", "EXCLUDE_RULE",
		"EDIT_METHOD", "WEIGHT_YN", "WEIGHT_VAR", "WEIGHT_METHOD", "SCHOOL_CODE_YEAR", "SURVEY_TYPE",
		"DATA_NO", "BIZ_NAME", "BIZ_YEAR",
	}
	FileKeys = []string{"FILE_NO", "FILE_NAME", "FILE_FORMAT", "CONTENT", "UNIT", "ID_VAR", "LINK_VAR", "LINK_FILE_NO", "NOTE", "VALID_N"}
	VarKeys  = []string{
		"FILE_NO", "VAR_NAME", "VAR_LABEL", "Q_NO", "Q_TEXT", "VAR_GROUP", "MEAS_LEVEL", "RESP_UNIVERSE",
		"MR_TYPE", "REVERSE", "DERIVATION", "STD_ID", "PII", "SENSITIVE", "NOTE",
	}
	CodeKeys = []string{"FILE_NO", "VAR_NAME", "CODE", "CODE_LABEL", "NOTE"}
)

// StudyRequired is the 01 sheet's 구분 per KEY: 필수, 선택, 조건부 or 재단.
var StudyRequired = map[string]string{
	"RULE_VERSION": "필수", "TITLE": "필수", "PROJECT_TITLE": "선택", "PROJECT_NO": "선택",
	"PURPOSE": "필수", "SUMMARY": "필수", "FUNDER": "재단", "PI_ORG": "필수", "FIELD_ORG": "필수",
	"PI_NAME": "선택", "FIELD_START": "필수", "FIELD_END": "필수", "KEYWORDS": "필수",
	"QUESTIONNAIRE": "필수", "REPORT_FILE": "필수", "EXTRA_FILES": "선택", "UNIVERSE": "필수",
	"POP_SOURCE": "필수", "ANALYSIS_UNIT": "필수", "SAMPLING_TYPE": "필수", "SAMPLING_DESC": "필수",
	"SAMPLE_FRAME": "선택", "STRATA_VARS": "조건부", "CLUSTER_VAR": "조건부", "TARGET_N": "필수",
	"VALID_N": "필수", "SPLIT_BALLOT": "필수", "SPLIT_VAR": "조건부", "MODE": "필수", "TOOL": "선택",
	"CONTACT_N": "조건부", "RESPONSE_RATE": "조건부", "RR_FORMULA": "조건부", "EXCLUDE_N": "필수",
	"EXCLUDE_RULE": "조건부", "EDIT_METHOD": "필수", "WEIGHT_YN": "필수", "WEIGHT_VAR": "조건부",
	"WEIGHT_METHOD": "조건부", "SCHOOL_CODE_YEAR": "조건부", "SURVEY_TYPE": "필수", "DATA_NO": "재단",
	"BIZ_NAME": "조건부", "BIZ_YEAR": "조건부",
}

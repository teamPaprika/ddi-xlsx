package kosac

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// profileTree assembles the legacy internal profile for validation and grouping. DDI
// elements are built in XSD order and the kosac ones follow the DDI children
// of their parent, so no sorting is needed. It fails, listing every item, when
// TITLE is blank, a var has no ID, a name or FILE_NO reference resolves to
// nothing, or a native codebook's Package names another source kind.
func (cb *Codebook) profileTree() (*ddi.Node, error) {
	if cb.HasV4Fields() && cb.Template.Version.Value != "4.0" {
		return nil, fmt.Errorf("v4.0 항목에는 양식 버전 4.0이 필요합니다")
	}
	b := newBuilder(cb)
	root := add(el("codeBook", "xmlns", ddi.Namespace, "xmlns:kosac", Namespace, "version", "2.5", "ID", b.codebookID()),
		b.docDscr(), b.stdyDscr())
	if cb.Template.Version.Value == "4.0" {
		for _, s := range root.Children {
			if s.Name == "stdyDscr" {
				cb.v4Native(s)
				at := len(s.Children)
				for i, n := range s.Children {
					if isKosac(n.Name) {
						at = i
						break
					}
				}
				s.Children = slices.Insert(s.Children, at, cb.v4Notes()...)
			}
		}
	}
	for i, f := range cb.Files {
		add(root, b.fileDscr(i, f))
	}
	add(root, b.dataDscr())
	add(root, b.otherMats()...)
	if len(b.problems) > 0 {
		return nil, fmt.Errorf("코드북을 조립할 수 없습니다: %s", strings.Join(b.problems, "; "))
	}
	return root, nil
}

// XML writes Tree as canonical XML.
func (cb *Codebook) XML() ([]byte, error) {
	root, err := cb.Tree()
	if err != nil {
		return nil, err
	}
	return ddi.WriteXML(root), nil
}

// JSON writes Tree as canonical JSON.
func (cb *Codebook) JSON() ([]byte, error) {
	root, err := cb.Tree()
	if err != nil {
		return nil, err
	}
	return ddi.WriteJSON(root), nil
}

type builder struct {
	cb       *Codebook
	vars     map[[2]string]*Var // first row of each (FILE_NO, VAR_NAME)
	codes    map[[2]string][]*Code
	fileIdx  map[string]int // FILE_NO → position in 02 order, which numbers the system vars
	byNo     []*File        // by FILE_NO, for 01 references
	varCount map[string]int // survey vars per FILE_NO
	cleansed map[string]bool
	weight   string // ID of the WEIGHT_VAR var
	problems []string
}

var fileNoRe = regexp.MustCompile(`^[1-9][0-9]*$`)

// ValidFileNo reports whether no is a FILE_NO assembly takes: a positive
// integer in its one plain spelling, since F{n} must be an xs:ID and rows are
// joined by FILE_NO as written ("01" and "1" would be two files). The check
// uses it too, so whatever passes C05 assembles.
func ValidFileNo(no string) bool { return fileNoRe.MatchString(no) }

// newBuilder indexes the rows once, since the model's lookups scan, and
// checks what every later step relies on.
func newBuilder(cb *Codebook) *builder {
	b := &builder{cb: cb, vars: map[[2]string]*Var{}, codes: map[[2]string][]*Code{}, fileIdx: map[string]int{},
		varCount: map[string]int{}, cleansed: map[string]bool{}}
	if ddi.NormalizeText(cb.Study.Get("TITLE")) == "" {
		b.fail("TITLE 비어 있음") // titl is required by the XSD
	}
	for i, f := range cb.Files {
		no := f.Rec.Get("FILE_NO")
		_, dup := b.fileIdx[no]
		switch {
		case !ValidFileNo(no):
			b.fail("FILE_NO %q: 1 이상의 정수가 아님", no)
		case dup:
			b.fail("FILE_NO %s 겹침", no)
		default:
			b.fileIdx[no] = i
		}
	}
	b.byNo = slices.Clone(cb.Files)
	slices.SortStableFunc(b.byNo, func(x, y *File) int {
		nx, _ := strconv.Atoi(x.Rec.Get("FILE_NO"))
		ny, _ := strconv.Atoi(y.Rec.Get("FILE_NO"))
		return cmp.Compare(nx, ny)
	})
	var orphans []string // FILE_NOs naming no file, in first-use order
	for _, v := range cb.Vars {
		k := varKey(v)
		if v.ID == "" {
			b.fail("%s: ID 없음", varLoc(v))
		}
		if _, ok := b.fileIdx[k[0]]; !ok && !slices.Contains(orphans, k[0]) {
			orphans = append(orphans, k[0])
		}
		if _, ok := b.vars[k]; !ok {
			b.vars[k] = v
		}
		b.varCount[k[0]]++
	}
	// Once per FILE_NO: a bad FILE_NO on the 02 sheet is on every var of the
	// file, which would bury the other problems.
	for _, no := range orphans {
		b.fail("FILE_NO %s인 파일 없음: 변수 %d개", no, b.varCount[no])
	}
	for _, c := range cb.Codes {
		k := [2]string{c.Rec.Get("FILE_NO"), c.Rec.Get("VAR_NAME")}
		b.codes[k] = append(b.codes[k], c)
	}
	for _, f := range cb.Files {
		if f.Cleansing == nil {
			continue
		}
		// An unknown var is reported where the rule is written.
		for _, r := range f.Cleansing.Rules {
			if id, ok := b.ref(f.Rec.Get("FILE_NO"), r.VarName); ok {
				b.cleansed[id] = true
			}
		}
	}
	b.weight = b.studyVar("WEIGHT_VAR", typed(cb.Study.Get("WEIGHT_VAR")))
	return b
}

func (b *builder) fail(format string, args ...any) {
	b.problems = append(b.problems, fmt.Sprintf(format, args...))
}

// sysID is the ID of system var name in the i-th file of the 02 sheet: every
// file holds the whole SystemVars list, numbered on across files.
func sysID(i int, name string) string {
	j := slices.IndexFunc(SystemVars, func(s SystemVar) bool { return s.Name == name })
	return "S" + strconv.Itoa(i*len(SystemVars)+j+1)
}

// ref is the ID of name in file fileNo: a survey var first, then a system
// var, which every file has.
func (b *builder) ref(fileNo, name string) (string, bool) {
	if v := b.vars[[2]string{fileNo, name}]; v != nil {
		return v.ID, true
	}
	for i, n := range b.cb.SourceSystemVars {
		if attr(n, "name") == name && slices.Contains(strings.Fields(attr(n, "files")), "F"+fileNo) {
			return "S" + strconv.Itoa(i+1), true
		}
	}
	if i, ok := b.fileIdx[fileNo]; ok && IsSystemVar(name) {
		return sysID(i, name), true
	}
	return "", false
}

// fileVar resolves a reference made within file fileNo; what names the
// referring item in the error.
func (b *builder) fileVar(fileNo, what, name string) string {
	name = b.absentRef(name)
	if name == "" {
		return ""
	}
	id, ok := b.ref(fileNo, name)
	if !ok {
		b.fail("%s: F%s에 변수 %s 없음", what, fileNo, name)
	}
	return id
}

// studyVar resolves a reference of 01 KEY key. The 01 sheet names no file, so
// the lowest FILE_NO that has the var wins.
func (b *builder) studyVar(key, name string) string {
	name = b.absentRef(name)
	if name == "" {
		return ""
	}
	for _, f := range b.byNo {
		if id, ok := b.ref(f.Rec.Get("FILE_NO"), name); ok {
			return id
		}
	}
	b.fail("%s: 변수 %s 없음", key, name)
	return ""
}

// studyVars resolves a list reference into space-separated IDs. The template
// lets contractors separate names with ";", "," or spaces.
func (b *builder) studyVars(key string) string {
	var ids []string
	for _, name := range strings.FieldsFunc(b.absentRef(typed(b.cb.Study.Get(key))), func(r rune) bool {
		return r == ';' || r == ',' || unicode.IsSpace(r)
	}) {
		ids = append(ids, b.studyVar(key, name))
	}
	return strings.Join(ids, " ")
}

// el makes an element from name, value pairs, leaving out a pair whose value
// is blank.
func el(name string, attrs ...string) *ddi.Node {
	n := &ddi.Node{Name: name}
	for i := 0; i < len(attrs); i += 2 {
		if strings.TrimSpace(attrs[i+1]) != "" {
			n.Attrs = append(n.Attrs, ddi.Attr{Name: attrs[i], Value: attrs[i+1]})
		}
	}
	return n
}

// text makes an element holding value, normalized as reading would leave it,
// or nil when the value is blank: its attributes only qualify the text.
func text(name, value string, attrs ...string) *ddi.Node {
	v := ddi.NormalizeText(value)
	if v == "" {
		return nil
	}
	n := el(name, attrs...)
	n.Text = v
	return n
}

// add appends the children that hold something, so an element left with no
// value, such as rspStmt without PI, disappears with its parent's add.
func add(parent *ddi.Node, children ...*ddi.Node) *ddi.Node {
	for _, c := range children {
		if c != nil && (len(c.Attrs) > 0 || c.Text != "" || len(c.Children) > 0) {
			parent.Children = append(parent.Children, c)
		}
	}
	return parent
}

// typed is the value for a slot with a format (date, integer, list value,
// reference).
func typed(v string) string { return strings.TrimSpace(cell(v)) }

// cell is a list value as written, so a value outside the template's list
// (REVERSE=N) reads back unchanged. A "확인 불가" value has no place in a slot
// with a format; kosac:unverified keeps it.
func cell(v string) string {
	if _, ok := Unverified(v); ok {
		return ""
	}
	return v
}

var dotZeroRe = regexp.MustCompile(`^(\d+)\.0+$`)

// integer is the value for an integer slot. The template accepts "10,000" and
// Excel can store "10000.0", neither of which an integer type accepts.
func integer(v string) string {
	v = strings.ReplaceAll(typed(v), ",", "")
	if m := dotZeroRe.FindStringSubmatch(v); m != nil {
		return m[1]
	}
	return v
}

func yes(b bool) string {
	if b {
		return "Y"
	}
	return ""
}

// codebookID is CB-{DATA_NO}-{year}[-{waveType}{waveNo}]-v{version}, the wave
// left out for the main wave; without every part there is no ID.
func (b *builder) codebookID() string {
	p, r := b.cb.Package, b.cb.Admin.Round
	dataNo := typed(b.cb.Study.Get("DATA_NO"))
	if p == nil || p.CodebookVersion == "" || dataNo == "" || r.Year == "" || r.WaveType == "" {
		return ""
	}
	wave := ""
	if r.WaveType != "main" {
		if r.WaveNo == "" {
			return ""
		}
		wave = "-" + r.WaveType + r.WaveNo
	}
	return "CB-" + dataNo + "-" + r.Year + wave + "-v" + p.CodebookVersion
}

func (b *builder) docDscr() *ddi.Node {
	p := b.cb.Package
	kind := ""
	if p != nil {
		kind = p.Source.Kind
	}
	if b.cb.Native {
		if kind != "" && kind != "native" {
			b.fail("Package.Source.Kind %q: 네이티브 코드북인데 native가 아님", kind)
		}
		kind = "native"
	}
	if p == nil {
		if kind == "" {
			return nil
		}
		// Native is written before datajam fills the Package, or reading the
		// draft back would lose it. No @cleansed: Parse tells this marker
		// from a Package by that.
		return add(el("docDscr"), add(el("kosac:package"), el("kosac:sourceSurvey", "wallaId", b.cb.Admin.WallaID, "kind", kind)))
	}
	d := el("docDscr")
	// citation requires a titl, which needs the version.
	if p.CodebookVersion != "" {
		add(d, add(el("citation"),
			add(el("titlStmt"), text("titl", ddi.NormalizeText(b.cb.Study.Get("TITLE"))+" 코드북 v"+p.CodebookVersion)),
			add(el("prodStmt"), text("prodDate", p.ExportedAt, "date", p.ExportedAt),
				text("software", p.Software, "version", p.SoftwareVersion))))
	}
	// ETL branches on @cleansed to keep the raw files, so N is written too.
	cleansed := "N"
	if slices.ContainsFunc(b.cb.Files, func(f *File) bool { return f.Cleansing != nil && len(f.Cleansing.Rules) > 0 }) {
		cleansed = "Y"
	}
	pkg := el("kosac:package", "exportId", p.ExportID, "codebookVersion", p.CodebookVersion, "previousVersion", p.PreviousVersion,
		"approvedBy", p.ApprovedBy, "approvedAt", p.ApprovedAt, "cleansed", cleansed)
	add(pkg, el("kosac:sourceSurvey", "wallaId", b.cb.Admin.WallaID, "kind", kind,
		"publishedVersion", p.Source.PublishedVersion, "templateVersion", p.Source.TemplateVersion))
	for _, f := range p.Files {
		if f.Role == "codebook" {
			add(pkg, el("kosac:file", "role", f.Role, "name", f.Name)) // a file cannot hold its own hash
			continue
		}
		add(pkg, el("kosac:file", "role", f.Role, "fileRef", f.FileRef, "name", f.Name, "bytes", integer(f.Bytes),
			"sha256", f.SHA256, "pages", f.Pages))
	}
	if v := p.Validation; v != nil {
		vn := el("kosac:validation", "passedAt", v.PassedAt, "ruleset", v.Ruleset, "warningCount", integer(v.WarningCount))
		for _, note := range v.Notes {
			add(vn, text("kosac:validationNote", note))
		}
		add(pkg, vn)
	}
	return add(d, pkg)
}

func (b *builder) stdyDscr() *ddi.Node {
	g, r := b.cb.Study.Get, b.cb.Admin.Round
	citation := add(el("citation"),
		add(el("titlStmt"), text("titl", g("TITLE")), text("altTitl", g("PROJECT_TITLE")),
			text("IDNo", g("PROJECT_NO"), "agency", "kosac"), text("IDNo", b.cb.Admin.WallaID, "agency", "walla")),
		add(el("rspStmt"), authEnty(g("PI_ORG"), g("PI_NAME"))),
		add(el("prodStmt"), text("producer", g("FIELD_ORG")), text("fundAg", g("FUNDER"))))
	stdyInfo := add(el("stdyInfo"),
		add(el("subject"), keywords(g("KEYWORDS"))...),
		text("abstract", g("PURPOSE"), "contentType", "purpose"),
		text("abstract", g("SUMMARY"), "contentType", "abstract"),
		add(el("sumDscr"), collDate("start", g("FIELD_START")), collDate("end", g("FIELD_END")),
			text("anlyUnit", g("ANALYSIS_UNIT")), text("universe", g("UNIVERSE"), "clusion", "I")))
	dataColl := add(el("dataColl"),
		text("sampProc", g("SAMPLING_DESC")),
		add(el("targetSampleSize"), text("sampleSize", integer(g("TARGET_N")))),
		text("collMode", g("MODE")), text("resInstru", g("TOOL")),
		text("weight", g("WEIGHT_METHOD")), text("cleanOps", g("EDIT_METHOD")),
		el("kosac:sampling", "type", typed(g("SAMPLING_TYPE")), "frame", g("SAMPLE_FRAME"),
			"strataVars", b.studyVars("STRATA_VARS"), "clusterVar", b.studyVar("CLUSTER_VAR", typed(g("CLUSTER_VAR")))),
		el("kosac:splitBallot", "yn", typed(g("SPLIT_BALLOT")), "var", b.studyVar("SPLIT_VAR", typed(g("SPLIT_VAR")))),
		el("kosac:weight", "yn", typed(g("WEIGHT_YN")), "var", b.weight),
		text("kosac:excludeN", integer(g("EXCLUDE_N"))), text("kosac:excludeRule", g("EXCLUDE_RULE")))
	anlyInfo := add(el("anlyInfo"), text("respRate", g("RESPONSE_RATE")),
		text("kosac:contactN", integer(g("CONTACT_N"))), text("kosac:rrFormula", g("RR_FORMULA")))
	study := add(el("kosac:study", "ruleVersion", typed(g("RULE_VERSION")), "surveyType", typed(g("SURVEY_TYPE")),
		"dataNo", typed(g("DATA_NO")), "categoryCode", b.cb.Admin.CategoryCode, "validN", integer(b.cb.StudyValidN()),
		"schoolCodeYear", integer(g("SCHOOL_CODE_YEAR")), "bizYear", integer(g("BIZ_YEAR"))),
		el("kosac:round", "year", r.Year, "waveType", r.WaveType, "waveNo", r.WaveNo),
		text("kosac:bizName", g("BIZ_NAME")), text("kosac:popSource", g("POP_SOURCE")))
	add(study, b.unverified()...)
	return add(el("stdyDscr"), citation, stdyInfo, add(el("method"), dataColl, anlyInfo), study)
}

// authEnty puts PI_ORG in @affiliation whenever PI_NAME is there, even empty,
// so reading back tells a name with no organization from an organization.
func authEnty(org, name string) *ddi.Node {
	if n := text("AuthEnty", name); n != nil {
		n.Attrs = []ddi.Attr{{Name: "affiliation", Value: org}}
		return n
	}
	return text("AuthEnty", org)
}

// keywords splits KEYWORDS on ";". A "확인 불가" value stays one keyword, so a
// ";" in the reason does not split it.
func keywords(v string) []*ddi.Node {
	items := []string{v}
	if _, ok := Unverified(v); !ok {
		items = strings.Split(v, ";")
	}
	var out []*ddi.Node
	for _, it := range items {
		out = append(out, text("keyword", it))
	}
	return out
}

// collDate carries the date only in @date, as the request's sample does.
func collDate(event, date string) *ddi.Node {
	if date = typed(date); date == "" {
		return nil
	}
	return el("collDate", "event", event, "date", date)
}

// unverified lists every "확인 불가" cell: 01 in KEY order, then 02 by file,
// then 03 by var. Text slots keep the value as well; this list is what
// restores the typed ones.
func (b *builder) unverified() []*ddi.Node {
	var out []*ddi.Node
	entry := func(value, key string, attrs ...string) {
		if reason, ok := Unverified(value); ok {
			n := el("kosac:unverified", append([]string{"key", key}, attrs...)...)
			n.Text = ddi.NormalizeText(reason)
			out = append(out, n)
		}
	}
	for _, k := range StudyKeys {
		entry(b.cb.Study.Get(k), k)
	}
	for _, f := range b.cb.Files {
		for _, k := range FileKeys {
			entry(f.Rec.Get(k), k, "file", "F"+f.Rec.Get("FILE_NO"))
		}
	}
	for _, v := range b.cb.Vars {
		for _, k := range VarKeys {
			entry(v.Rec.Get(k), k, "var", v.ID)
		}
	}
	return out
}

func (b *builder) fileDscr(i int, f *File) *ddi.Node {
	g, no := f.Rec.Get, f.Rec.Get("FILE_NO")
	name, source := g("FILE_NAME"), ""
	if f.PackageName != "" {
		name, source = f.PackageName, g("FILE_NAME")
	}
	fileTxt := add(el("fileTxt"), text("fileName", name), text("fileCont", g("CONTENT")),
		add(el("dimensns"), text("caseQnty", integer(g("VALID_N"))),
			text("varQnty", strconv.Itoa(len(SystemVars)+b.varCount[no]))),
		text("fileType", g("FILE_FORMAT"), "charset", "UTF-8"))
	link := b.absentRef(typed(g("LINK_FILE_NO")))
	if _, ok := b.fileIdx[link]; link != "" && !ok {
		b.fail("F%s LINK_FILE_NO: 파일 %s 없음", no, link)
		link = ""
	}
	kfile := add(el("kosac:file", "fileNo", no, "unit", g("UNIT"), "sensLevel", f.SensLevel, "sha256", f.SHA256,
		"bytes", integer(f.Bytes), "rawSha256", f.RawSHA256, "rawBytes", integer(f.RawBytes), "rawName", f.RawName,
		"sourceName", source),
		el("kosac:idVar", "ref", b.fileVar(no, "F"+no+" ID_VAR", typed(g("ID_VAR")))),
		el("kosac:systemIdVar", "ref", sysID(i, "walla_response_id")),
		el("kosac:link", "var", b.fileVar(no, "F"+no+" LINK_VAR", typed(g("LINK_VAR"))), "fileRef", prefixed("F", link),
			"level", f.LinkLevel))
	return add(el("fileDscr", "ID", "F"+no), fileTxt, text("notes", g("NOTE")), kfile, b.cleansing(no, f.Cleansing))
}

func prefixed(p, v string) string {
	if v == "" {
		return ""
	}
	return p + v
}

func (b *builder) cleansing(no string, c *Cleansing) *ddi.Node {
	if c == nil {
		return nil
	}
	n := el("kosac:cleansing", "appliedBy", c.AppliedBy, "appliedAt", c.AppliedAt, "rawFileRef", c.RawFileRef,
		"cellsChanged", integer(c.CellsChanged), "rowsAffected", integer(c.RowsAffected), "logFile", c.LogFile)
	for _, r := range c.Rules {
		m := el("kosac:map")
		for _, e := range r.Map {
			add(m, el("kosac:entry", "from", e.From, "to", e.To))
		}
		add(n, add(el("kosac:rule", "id", r.ID, "var", b.fileVar(no, "F"+no+" 클린징 규칙 "+r.ID, r.VarName),
			"action", r.Action, "condition", r.Condition, "rows", integer(r.Rows), "source", r.Source,
			"reason", r.Reason, "evidence", r.Evidence), m))
	}
	return n
}

// otherMats writes report, questionnaire, then EXTRA_FILES in order. Extra
// items pair with the package's extra files by position, since packaging
// renames them to …_extra_{k}.
func (b *builder) otherMats() []*ddi.Node {
	g := b.cb.Study.Get
	report := material("report", g("REPORT_FILE"), b.pkgFile("report", 0), "")
	reportURI := ""
	if report != nil {
		reportURI, _ = report.Attr("URI")
	}
	out := []*ddi.Node{report, material("questionnaire", g("QUESTIONNAIRE"), b.pkgFile("questionnaire", 0), reportURI)}
	extra := g("EXTRA_FILES")
	items := []string{extra}
	if _, ok := Unverified(extra); !ok {
		items = strings.Split(extra, ";")
	}
	k := 0
	for _, it := range items {
		if strings.TrimSpace(it) == "" {
			continue
		}
		out = append(out, material("extra", it, b.pkgFile("extra", k), ""))
		k++
	}
	return out
}

// material is one otherMat whose labl is the cell as written. A questionnaire
// given as pages of the report points at the report; otherwise the package
// file names it, else the cell when it is a file name (the report: always).
func material(typ, value string, pf *PackageFile, reportURI string) *ddi.Node {
	labl := text("labl", value)
	if labl == nil {
		return nil
	}
	var uri, pages, sha, size string
	// Derived from the text labl holds, which is all a reader gets back.
	v := typed(ddi.NormalizeText(value))
	switch p, isFile := MaterialItem(v); {
	case v == "":
	case p != "" && typ == "questionnaire":
		uri, pages = reportURI, p
	case pf != nil:
		uri, sha, size = pf.Name, pf.SHA256, integer(pf.Bytes)
	case typ == "report" || isFile:
		uri = v
	}
	return add(el("otherMat", "type", typ, "level", "study", "URI", uri), labl,
		text("kosac:pages", pages), text("kosac:sha256", sha), text("kosac:bytes", size))
}

// pkgFile is the k-th package file of role, or nil.
func (b *builder) pkgFile(role string, k int) *PackageFile {
	if b.cb.Package == nil {
		return nil
	}
	for i := range b.cb.Package.Files {
		if f := &b.cb.Package.Files[i]; f.Role == role {
			if k == 0 {
				return f
			}
			k--
		}
	}
	return nil
}

package kosac

import (
	"fmt"
	"slices"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// Parse reads codebook XML or JSON that this engine wrote back into the model,
// the inverse of Tree: writing the result gives the same bytes again. It takes
// only the engine's shape. An element or attribute the inverse does not read,
// mixed content, an ID that names no var, or the kosac namespace under another
// prefix is an error naming its path, since keeping unknown nodes would need a
// store nothing uses. System vars are not returned: assembly makes them.
func Parse(b []byte) (*Codebook, error) {
	root, _, err := ddi.Parse(b)
	if err == nil {
		err = kosacPrefix(root)
	}
	if err != nil {
		return nil, fmt.Errorf("코드북을 읽을 수 없습니다: %w", err)
	}
	r := &reader{seen: map[mark]bool{}, path: map[*ddi.Node]string{},
		names: map[string]string{}, vars: map[string]*Var{}, files: map[string]*File{}}
	paths(root, r.path)
	cb := r.codebook(root)
	r.unread(root)
	if len(r.problems) > 0 {
		return nil, fmt.Errorf("코드북을 읽을 수 없습니다: %s", strings.Join(r.problems, "; "))
	}
	return cb, nil
}

func kosacPrefix(root *ddi.Node) error {
	if v, _ := root.Attr("xmlns:kosac"); v == Namespace {
		return nil
	}
	for _, a := range root.Attrs {
		if p, ok := strings.CutPrefix(a.Name, "xmlns:"); ok && a.Value == Namespace {
			return fmt.Errorf("codeBook/@%s: kosac 네임스페이스의 접두가 kosac이 아니라 %s임", a.Name, p)
		}
	}
	return fmt.Errorf("codeBook: kosac 네임스페이스(%s)를 kosac 접두로 선언하지 않음", Namespace)
}

// paths maps every element to its Issue.Path.
func paths(root *ddi.Node, out map[*ddi.Node]string) {
	Walk(root, func(n, _ *ddi.Node, p string) bool {
		out[n] = p
		return true
	})
}

// mark is an element (attr "") or one of its attributes that the inverse read.
type mark struct {
	n    *ddi.Node
	attr string
}

type reader struct {
	seen     map[mark]bool
	path     map[*ddi.Node]string
	names    map[string]string // var ID → name, system vars included
	vars     map[string]*Var   // survey vars by ID
	files    map[string]*File  // by fileDscr@ID
	problems []string
}

// fail records a problem at n, or at its attribute attr when attr is set.
func (r *reader) fail(n *ddi.Node, attr, format string, args ...any) {
	p := r.path[n]
	if attr != "" {
		p += "/@" + attr
	}
	r.problems = append(r.problems, p+": "+fmt.Sprintf(format, args...))
}

// use marks n and its attributes attrs read and returns n. The helpers take a
// nil node, an element the engine left out for being empty.
func (r *reader) use(n *ddi.Node, attrs ...string) *ddi.Node {
	if n != nil {
		r.seen[mark{n, ""}] = true
		for _, a := range attrs {
			r.seen[mark{n, a}] = true
		}
	}
	return n
}

func (r *reader) attr(n *ddi.Node, name string) string {
	if n == nil {
		return ""
	}
	v, _ := r.use(n, name).Attr(name)
	return v
}

// child is n's first child named name. The engine writes these once, so a
// second one stays unread and is reported.
func (r *reader) child(n *ddi.Node, name string) *ddi.Node {
	if n != nil {
		for _, c := range n.Children {
			if c.Name == name {
				return r.use(c)
			}
		}
	}
	return nil
}

func (r *reader) kids(n *ddi.Node, name string) []*ddi.Node {
	var out []*ddi.Node
	if n != nil {
		for _, c := range n.Children {
			if c.Name == name {
				out = append(out, r.use(c))
			}
		}
	}
	return out
}

// text is the text of n's child name, whose attributes attrs are read too.
func (r *reader) text(n *ddi.Node, name string, attrs ...string) string {
	return r.str(r.use(r.child(n, name), attrs...))
}

// str is n's text, marked read so that text where the engine writes none is
// reported.
func (r *reader) str(n *ddi.Node) string {
	if n == nil {
		return ""
	}
	r.seen[mark{n, "#text"}] = true
	return n.Text
}

// pick reads attribute attr, which decides where n goes in the model; a value
// the engine does not write is reported rather than dropped.
func (r *reader) pick(n *ddi.Node, attr string, want ...string) string {
	v := r.attr(n, attr)
	if !slices.Contains(want, v) {
		r.fail(n, attr, "엔진이 쓰지 않는 값 %q", v)
		return ""
	}
	return v
}

// once passes v, read from attribute attr of n, unless an earlier sibling gave
// it: the engine writes each such value once, and a second would silently
// replace the first.
func (r *reader) once(seen map[string]bool, n *ddi.Node, attr, v string) string {
	if v != "" && seen[v] {
		r.fail(n, attr, "%q 값이 두 번 있음", v)
		return ""
	}
	seen[v] = true
	return v
}

// name maps the var IDs in attribute attr back to names, "; " between several
// as the template lists them.
func (r *reader) name(n *ddi.Node, attr string) string {
	var out []string
	for _, id := range strings.Fields(r.attr(n, attr)) {
		name, ok := r.names[id]
		if !ok {
			r.fail(n, attr, "변수 %s 없음", id)
		}
		out = append(out, name)
	}
	return strings.Join(out, "; ")
}

// unread reports what the inverse did not read. The engine did not write it,
// and passing over it would lose it silently.
func (r *reader) unread(n *ddi.Node) {
	if !r.seen[mark{n, ""}] {
		r.fail(n, "", "엔진이 쓰지 않는 요소")
		return
	}
	if n.Raw != "" {
		r.fail(n, "", "엔진이 쓰지 않는 혼합 내용")
	}
	if n.Text != "" && !r.seen[mark{n, "#text"}] {
		r.fail(n, "", "엔진이 쓰지 않는 텍스트")
	}
	for _, a := range n.Attrs {
		if !r.seen[mark{n, a.Name}] {
			r.fail(n, a.Name, "엔진이 쓰지 않는 속성")
		}
	}
	for _, c := range n.Children {
		r.unread(c)
	}
}

// record holds every KEY of a sheet, as a record read from the template does.
func record(keys []string) Record {
	r := Record{}
	for _, k := range keys {
		r[k] = Field{}
	}
	return r
}

// levels maps kosac:measLevel back to MEAS_LEVEL.
var levels = func() map[string]string {
	m := map[string]string{}
	for k, v := range MeasLevels {
		m[v] = k
	}
	return m
}()

func (r *reader) codebook(root *ddi.Node) *Codebook {
	// The ID follows from DATA_NO, the round and the version.
	r.use(root, "xmlns", "xmlns:kosac", "version", "ID")
	cb := &Codebook{Study: record(StudyKeys)}
	dd := r.child(root, "dataDscr")
	// Every reference is by ID, so the names come first.
	ids := map[string]bool{}
	for _, n := range r.kids(dd, "var") {
		r.names[r.once(ids, n, "ID", r.attr(n, "ID"))] = r.attr(n, "name")
	}
	fileIDs := map[string]bool{}
	for _, fd := range r.kids(root, "fileDscr") {
		cb.Files = append(cb.Files, r.file(fd, fileIDs))
	}
	for _, n := range r.kids(dd, "var") {
		v, codes := r.variable(n)
		no, name := v.Rec.Get("FILE_NO"), v.Rec.Get("VAR_NAME")
		i := slices.IndexFunc(cb.Files, func(f *File) bool { return f.Rec.Get("FILE_NO") == no })
		if i >= 0 && IsSystemVar(name) && v.ID == sysID(i, name) {
			continue // assembly makes it again from the file
		}
		r.vars[v.ID] = v
		cb.Vars = append(cb.Vars, v)
		cb.Codes = append(cb.Codes, codes...)
	}
	r.groups(r.kids(dd, "varGrp"))
	unverified := r.study(cb, r.child(root, "stdyDscr"))
	cb.Package = r.pkg(cb, r.child(root, "docDscr"))
	r.materials(cb, r.kids(root, "otherMat"))
	r.restore(cb, unverified)
	return cb
}

func (r *reader) study(cb *Codebook, s *ddi.Node) []*ddi.Node {
	g := cb.Study.Set
	cit := r.child(s, "citation")
	ts := r.child(cit, "titlStmt")
	g("TITLE", r.text(ts, "titl"))
	g("PROJECT_TITLE", r.text(ts, "altTitl"))
	agency := map[string]bool{}
	for _, id := range r.kids(ts, "IDNo") {
		switch r.once(agency, id, "agency", r.pick(id, "agency", "kosac", "walla")) {
		case "kosac":
			g("PROJECT_NO", r.str(id))
		case "walla":
			cb.Admin.WallaID = r.str(id)
		}
	}
	// affiliation is there exactly when PI_NAME is, even with no PI_ORG.
	if a := r.child(r.child(cit, "rspStmt"), "AuthEnty"); a != nil {
		if org, ok := a.Attr("affiliation"); ok {
			r.use(a, "affiliation")
			g("PI_ORG", org)
			g("PI_NAME", r.str(a))
		} else {
			g("PI_ORG", r.str(a))
		}
	}
	ps := r.child(cit, "prodStmt")
	g("FIELD_ORG", r.text(ps, "producer"))
	g("FUNDER", r.text(ps, "fundAg"))

	info := r.child(s, "stdyInfo")
	var kw []string
	for _, k := range r.kids(r.child(info, "subject"), "keyword") {
		kw = append(kw, r.str(k))
	}
	g("KEYWORDS", strings.Join(kw, "; "))
	content := map[string]bool{}
	for _, a := range r.kids(info, "abstract") {
		switch r.once(content, a, "contentType", r.pick(a, "contentType", "purpose", "abstract")) {
		case "purpose":
			g("PURPOSE", r.str(a))
		case "abstract":
			g("SUMMARY", r.str(a))
		}
	}
	sum := r.child(info, "sumDscr")
	event := map[string]bool{}
	for _, d := range r.kids(sum, "collDate") {
		switch r.once(event, d, "event", r.pick(d, "event", "start", "end")) {
		case "start":
			g("FIELD_START", r.attr(d, "date"))
		case "end":
			g("FIELD_END", r.attr(d, "date"))
		}
	}
	g("ANALYSIS_UNIT", r.text(sum, "anlyUnit"))
	g("UNIVERSE", r.text(sum, "universe", "clusion"))

	m := r.child(s, "method")
	dc := r.child(m, "dataColl")
	g("SAMPLING_DESC", r.text(dc, "sampProc"))
	g("TARGET_N", r.text(r.child(dc, "targetSampleSize"), "sampleSize"))
	g("MODE", r.text(dc, "collMode"))
	g("TOOL", r.text(dc, "resInstru"))
	g("WEIGHT_METHOD", r.text(dc, "weight"))
	g("EDIT_METHOD", r.text(dc, "cleanOps"))
	sp := r.child(dc, "kosac:sampling")
	g("SAMPLING_TYPE", r.attr(sp, "type"))
	g("SAMPLE_FRAME", r.attr(sp, "frame"))
	g("STRATA_VARS", r.name(sp, "strataVars"))
	g("CLUSTER_VAR", r.name(sp, "clusterVar"))
	sb := r.child(dc, "kosac:splitBallot")
	g("SPLIT_BALLOT", r.attr(sb, "yn"))
	g("SPLIT_VAR", r.name(sb, "var"))
	w := r.child(dc, "kosac:weight")
	g("WEIGHT_YN", r.attr(w, "yn"))
	g("WEIGHT_VAR", r.name(w, "var"))
	g("EXCLUDE_N", r.text(dc, "kosac:excludeN"))
	g("EXCLUDE_RULE", r.text(dc, "kosac:excludeRule"))
	ai := r.child(m, "anlyInfo")
	g("RESPONSE_RATE", r.text(ai, "respRate"))
	g("CONTACT_N", r.text(ai, "kosac:contactN"))
	g("RR_FORMULA", r.text(ai, "kosac:rrFormula"))

	st := r.child(s, "kosac:study")
	g("RULE_VERSION", r.attr(st, "ruleVersion"))
	g("SURVEY_TYPE", r.attr(st, "surveyType"))
	g("DATA_NO", r.attr(st, "dataNo"))
	g("VALID_N", r.attr(st, "validN"))
	g("SCHOOL_CODE_YEAR", r.attr(st, "schoolCodeYear"))
	g("BIZ_YEAR", r.attr(st, "bizYear"))
	cb.Admin.CategoryCode = r.attr(st, "categoryCode")
	rd := r.child(st, "kosac:round")
	cb.Admin.Round = Round{Year: r.attr(rd, "year"), WaveType: r.attr(rd, "waveType"), WaveNo: r.attr(rd, "waveNo")}
	g("BIZ_NAME", r.text(st, "kosac:bizName"))
	g("POP_SOURCE", r.text(st, "kosac:popSource"))
	return r.kids(st, "kosac:unverified")
}

func (r *reader) file(fd *ddi.Node, ids map[string]bool) *File {
	f := &File{Rec: record(FileKeys)}
	g := f.Rec.Set
	id := r.attr(fd, "ID")
	r.files[r.once(ids, fd, "ID", id)] = f
	g("FILE_NO", strings.TrimPrefix(id, "F"))
	ft, kf := r.child(fd, "fileTxt"), r.use(r.child(fd, "kosac:file"), "fileNo")
	name := r.text(ft, "fileName")
	// sourceName is there exactly when the package renamed the file.
	if src := r.attr(kf, "sourceName"); src != "" {
		f.PackageName, name = name, src
	}
	g("FILE_NAME", name)
	g("CONTENT", r.text(ft, "fileCont"))
	dim := r.child(ft, "dimensns")
	g("VALID_N", r.text(dim, "caseQnty"))
	r.text(dim, "varQnty") // counts the vars
	g("FILE_FORMAT", r.text(ft, "fileType", "charset"))
	g("NOTE", r.text(fd, "notes"))
	g("UNIT", r.attr(kf, "unit"))
	f.SensLevel, f.SHA256, f.Bytes = r.attr(kf, "sensLevel"), r.attr(kf, "sha256"), r.attr(kf, "bytes")
	f.RawSHA256, f.RawBytes, f.RawName = r.attr(kf, "rawSha256"), r.attr(kf, "rawBytes"), r.attr(kf, "rawName")
	g("ID_VAR", r.name(r.child(kf, "kosac:idVar"), "ref"))
	r.use(r.child(kf, "kosac:systemIdVar"), "ref") // always the file's walla_response_id
	link := r.child(kf, "kosac:link")
	g("LINK_VAR", r.name(link, "var"))
	g("LINK_FILE_NO", strings.TrimPrefix(r.attr(link, "fileRef"), "F"))
	f.LinkLevel = r.attr(link, "level")

	c := r.child(fd, "kosac:cleansing")
	if c == nil {
		return f
	}
	f.Cleansing = &Cleansing{AppliedBy: r.attr(c, "appliedBy"), AppliedAt: r.attr(c, "appliedAt"),
		RawFileRef: r.attr(c, "rawFileRef"), CellsChanged: r.attr(c, "cellsChanged"),
		RowsAffected: r.attr(c, "rowsAffected"), LogFile: r.attr(c, "logFile")}
	for _, n := range r.kids(c, "kosac:rule") {
		rule := CleansingRule{ID: r.attr(n, "id"), VarName: r.name(n, "var"), Action: r.attr(n, "action"),
			Condition: r.attr(n, "condition"), Rows: r.attr(n, "rows"), Source: r.attr(n, "source"),
			Reason: r.attr(n, "reason"), Evidence: r.attr(n, "evidence")}
		for _, e := range r.kids(r.child(n, "kosac:map"), "kosac:entry") {
			rule.Map = append(rule.Map, MapEntry{From: r.attr(e, "from"), To: r.attr(e, "to")})
		}
		f.Cleansing.Rules = append(f.Cleansing.Rules, rule)
	}
	return f
}

// variable reads a var element, a system var as well: reading it the same way
// marks all it holds as read.
func (r *reader) variable(n *ddi.Node) (*Var, []*Code) {
	// wgt follows from WEIGHT_VAR; intrvl, varFormat and seq from the level and
	// the order; cleansed, missing and missingCodes from the rules and codes.
	r.use(n, "wgt", "intrvl")
	v := &Var{ID: r.attr(n, "ID"), Rec: record(VarKeys)}
	g := v.Rec.Set
	no, name := strings.TrimPrefix(r.attr(n, "files"), "F"), r.attr(n, "name")
	g("FILE_NO", no)
	g("VAR_NAME", name)
	k := map[string]string{}
	for _, a := range KosacAttrs["var"] {
		k[a] = r.attr(n, "kosac:"+a)
	}
	if lv := k["measLevel"]; lv != "" {
		if levels[lv] == "" {
			r.fail(n, "kosac:measLevel", "엔진이 쓰지 않는 값 %q", lv)
		}
		g("MEAS_LEVEL", levels[lv])
	}
	g("PII", k["pii"])
	g("SENSITIVE", k["sensitive"])
	g("REVERSE", k["reverse"])
	g("MR_TYPE", k["mrType"])
	v.Role, v.MRGroup, v.RowLabel, v.RankNo, v.TextFormat = k["role"], k["mrGroup"], k["rowLabel"], k["rankNo"], k["textFormat"]
	v.ParentVar = r.name(n, "kosac:parentVar")

	r.use(r.child(n, "location"), "fileid")
	g("VAR_LABEL", r.text(n, "labl"))
	q := r.child(n, "qstn")
	g("Q_NO", r.attr(q, "seqNo"))
	g("Q_TEXT", r.text(q, "qstnLit"))
	g("RESP_UNIVERSE", r.text(n, "universe", "clusion"))
	var codes []*Code
	for _, c := range r.kids(n, "catgry") {
		code := &Code{Rec: record(CodeKeys), Type: r.text(c, "kosac:codeType")}
		r.use(c, "missing")
		code.Rec.Set("FILE_NO", no)
		code.Rec.Set("VAR_NAME", name)
		code.Rec.Set("CODE", r.text(c, "catValu"))
		code.Rec.Set("CODE_LABEL", r.text(c, "labl"))
		code.Rec.Set("NOTE", r.text(c, "txt"))
		sc := r.child(c, "kosac:stdCode")
		code.StdCode, code.StdSet = r.str(sc), r.attr(sc, "set")
		codes = append(codes, code)
	}
	g("STD_ID", r.text(n, "concept", "vocab"))
	g("DERIVATION", r.text(r.child(n, "derivation"), "drvdesc"))
	r.use(r.child(n, "varFormat"), "type", "category")
	g("NOTE", r.text(n, "notes"))

	if fk := r.child(n, "kosac:fk"); fk != nil {
		a := func(name string) string { return r.attr(fk, name) }
		v.FK = &FK{CodeSet: a("codeSet"), DimTable: a("dimTable"), KeyColumn: a("keyColumn"), Mapping: a("mapping"),
			CodeSystem: a("codeSystem"), RefYear: a("refYear"), LinkKey: a("linkKey"), CodeSetVersion: a("codeSetVersion"),
			ValidatedAt: a("validatedAt"), MatchedRows: a("matchedRows"), UnmatchedRows: a("unmatchedRows")}
	}
	sc := r.child(n, "kosac:scale")
	v.Scale = Scale{Min: r.attr(sc, "min"), Max: r.attr(sc, "max"), Dir: r.attr(sc, "dir")}
	r.text(n, "kosac:missingCodes")
	v.CodeSetID = r.text(n, "kosac:codeSetId")
	an := r.use(r.child(n, "kosac:analysis"), "fkRef") // fkRef follows from the FK
	v.PhysicalName, v.DimMeasure, v.AggRules = r.attr(an, "physicalName"), r.attr(an, "dimMeasure"), r.attr(an, "aggRules")
	v.IncludeYn, v.LogicalName = r.attr(an, "includeYn"), r.attr(an, "logicalName")
	for _, s := range r.kids(an, "kosac:synonym") {
		v.Synonyms = append(v.Synonyms, r.str(s))
	}
	return v, codes
}

// groups puts back what the groups hold of their members: the section name,
// the grid key and noneOption. Labels and IDs follow from the members.
func (r *reader) groups(gs []*ddi.Node) {
	// A var is in at most one group of each type; a second would replace what
	// the first gave it.
	grouped := map[string]bool{}
	for _, n := range gs {
		r.use(n, "ID", "kosac:mrGroup")
		typ := r.pick(n, "type", "section", "grid", "multipleResp", "other")
		if typ == "other" {
			r.pick(n, "otherType", "rank")
		}
		labl, none := r.text(n, "labl"), r.attr(r.child(n, "kosac:noneOption"), "yn")
		var members []*Var
		for _, id := range strings.Fields(r.attr(n, "var")) {
			switch v := r.vars[id]; {
			case v == nil:
				r.fail(n, "var", "조사 변수 %s 없음", id)
			case grouped[typ+" "+id]:
				r.fail(n, "var", "변수 %s: 같은 유형(%s)의 묶음에 이미 있음", id, typ)
			default:
				grouped[typ+" "+id] = true
				members = append(members, v)
			}
		}
		for _, v := range members {
			switch typ {
			case "section":
				v.Rec.Set("VAR_GROUP", labl)
			case "grid":
				// The key itself is not written. Any value the members share groups
				// them the same, and the first member's name is the key the grouping
				// rule gives.
				v.GridGroup = members[0].Rec.Get("VAR_NAME")
			case "multipleResp":
				v.NoneOption = none
			}
		}
	}
}

// pkg reads docDscr into the Package and Native. Assembly writes @cleansed on
// every Package, so a native kind on a package without it is the marker of a
// native codebook whose Package was nil, and Package stays nil.
func (r *reader) pkg(cb *Codebook, d *ddi.Node) *Package {
	if d == nil {
		return nil
	}
	k := r.child(d, "kosac:package")
	src := r.child(k, "kosac:sourceSurvey")
	// The attribute holds the ID exactly; IDNo holds its normalized text.
	if id := r.attr(src, "wallaId"); id != "" {
		cb.Admin.WallaID = id
	}
	kind := r.attr(src, "kind")
	cb.Native = kind == "native"
	if k != nil && cb.Native {
		if _, ok := k.Attr("cleansed"); !ok {
			return nil
		}
	}
	p := &Package{}
	cit := r.child(d, "citation")
	r.text(r.child(cit, "titlStmt"), "titl") // TITLE and the version
	ps := r.child(cit, "prodStmt")
	pd := r.child(ps, "prodDate")
	r.str(pd) // repeats @date
	p.ExportedAt = r.attr(pd, "date")
	sw := r.child(ps, "software")
	p.Software, p.SoftwareVersion = r.str(sw), r.attr(sw, "version")
	r.use(k, "cleansed") // follows from the cleansing rules
	p.ExportID, p.CodebookVersion, p.PreviousVersion = r.attr(k, "exportId"), r.attr(k, "codebookVersion"), r.attr(k, "previousVersion")
	p.ApprovedBy, p.ApprovedAt = r.attr(k, "approvedBy"), r.attr(k, "approvedAt")
	p.Source = SourceSurvey{Kind: kind, PublishedVersion: r.attr(src, "publishedVersion"),
		TemplateVersion: r.attr(src, "templateVersion")}
	for _, f := range r.kids(k, "kosac:file") {
		p.Files = append(p.Files, PackageFile{Role: r.attr(f, "role"), FileRef: r.attr(f, "fileRef"), Name: r.attr(f, "name"),
			Bytes: r.attr(f, "bytes"), SHA256: r.attr(f, "sha256"), Pages: r.attr(f, "pages")})
	}
	if v := r.child(k, "kosac:validation"); v != nil {
		p.Validation = &Validation{PassedAt: r.attr(v, "passedAt"), Ruleset: r.attr(v, "ruleset"), WarningCount: r.attr(v, "warningCount")}
		for _, n := range r.kids(v, "kosac:validationNote") {
			p.Validation.Notes = append(p.Validation.Notes, r.str(n))
		}
	}
	return p
}

// materials reads the cells back from the labl; URI and the kosac children
// follow from the package files and the cell.
func (r *reader) materials(cb *Codebook, ms []*ddi.Node) {
	var extra []string
	types := map[string]bool{}
	for _, m := range ms {
		r.use(m, "level", "URI")
		for _, k := range KosacChildren["otherMat"] {
			r.text(m, "kosac:"+k)
		}
		labl := r.text(m, "labl")
		typ := r.pick(m, "type", "report", "questionnaire", "extra")
		if typ != "extra" {
			typ = r.once(types, m, "type", typ)
		}
		switch typ {
		case "report":
			cb.Study.Set("REPORT_FILE", labl)
		case "questionnaire":
			cb.Study.Set("QUESTIONNAIRE", labl)
		case "extra":
			extra = append(extra, labl)
		}
	}
	cb.Study.Set("EXTRA_FILES", strings.Join(extra, "; "))
}

// restore puts "확인 불가" back into the typed slots, which could not hold it.
// A text slot kept the value as written, which is closer to the cell.
func (r *reader) restore(cb *Codebook, us []*ddi.Node) {
	seen := map[string]bool{}
	for _, u := range us {
		key, file, id, reason := r.attr(u, "key"), r.attr(u, "file"), r.attr(u, "var"), r.str(u)
		keys, rec := StudyKeys, cb.Study
		switch {
		case file != "" && id != "":
			r.fail(u, "", "@file과 @var가 함께 있음")
			continue
		case file != "":
			f := r.files[file]
			if f == nil {
				r.fail(u, "file", "파일 %s 없음", file)
				continue
			}
			keys, rec = FileKeys, f.Rec
		case id != "":
			v := r.vars[id]
			if v == nil {
				r.fail(u, "var", "조사 변수 %s 없음", id)
				continue
			}
			keys, rec = VarKeys, v.Rec
		}
		if !slices.Contains(keys, key) {
			r.fail(u, "key", "엔진이 쓰지 않는 값 %q", key)
			continue
		}
		if r.once(seen, u, "key", strings.TrimSpace(file+id+" "+key)) != "" && rec.Get(key) == "" {
			rec.Set(key, strings.TrimSpace("확인 불가: "+reason))
		}
	}
}

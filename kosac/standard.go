package kosac

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// Tree writes standard DDI Codebook 2.5. Template/workflow fields without a
// dedicated standard slot are readable descriptions, not a private codec.
// Keep the original workbook/model when those fields must remain editable.
func (cb *Codebook) Tree() (*ddi.Node, error) {
	profile, err := cb.profileTree()
	if err != nil {
		return nil, err
	}
	b := newBuilder(cb)
	root := el("codeBook", "xmlns", ddi.Namespace, "version", "2.5", "ID", b.codebookID())
	s := &standardBuilder{builder: b, used: map[string]bool{}}
	Walk(profile, func(n, _ *ddi.Node, _ string) bool {
		if id, ok := n.Attr("ID"); ok {
			s.used[id] = true
		}
		return true
	})
	add(root, s.document(), s.study())
	for _, f := range cb.Files {
		add(root, s.file(f))
	}
	add(root, s.data(profile))
	for _, n := range b.otherMats() {
		if n != nil {
			add(root, standardDescription(n))
		}
	}
	if value := cb.Study.Get("SOURCE_FILES"); strings.TrimSpace(value) != "" {
		add(root, add(el("otherMat", "level", "study"), text("labl", "수집 원본 파일 목록·버전"), text("txt", value)))
	}
	if len(b.problems) > 0 {
		return nil, fmt.Errorf("코드북을 조립할 수 없습니다: %s", strings.Join(b.problems, "; "))
	}
	return root, nil
}

type standardBuilder struct {
	*builder
	used map[string]bool
}

func (s *standardBuilder) id(prefix string) string {
	for n := 1; ; n++ {
		id := prefix + strconv.Itoa(n)
		if !s.used[id] {
			s.used[id] = true
			return id
		}
	}
}

// standardDescription preserves legacy administrative information as prose.
// It does not create a machine-readable replacement for the old extension.
func standardDescription(n *ddi.Node) *ddi.Node {
	out := DDIProjection(n)
	var lines []string
	for _, a := range n.Attrs {
		if isKosac(a.Name) {
			lines = append(lines, strings.TrimPrefix(a.Name, "kosac:")+": "+a.Value)
		}
	}
	for _, c := range n.Children {
		if isKosac(c.Name) {
			lines = append(lines, profileDescription(c))
		}
	}
	if len(lines) > 0 {
		add(out, text("notes", strings.Join(lines, "\n")))
	}
	return out
}
func profileDescription(n *ddi.Node) string {
	var parts []string
	for _, a := range n.Attrs {
		parts = append(parts, strings.TrimPrefix(a.Name, "kosac:")+"="+a.Value)
	}
	if n.Text != "" {
		parts = append(parts, n.Text)
	}
	for _, c := range n.Children {
		parts = append(parts, profileDescription(c))
	}
	return strings.TrimPrefix(n.Name, "kosac:") + ": " + strings.Join(parts, "; ")
}

func (s *standardBuilder) document() *ddi.Node {
	cb := s.cb
	d := add(el("docDscr"), add(el("citation"), add(el("titlStmt"), text("titl", cb.Study.Get("TITLE")+" 코드북"))))
	if p := cb.Package; p != nil {
		cit := child(d, "citation")
		add(cit, add(el("prodStmt"), text("prodDate", p.ExportedAt, "date", p.ExportedAt), text("software", p.Software, "version", p.SoftwareVersion)),
			add(el("verStmt"), text("version", p.CodebookVersion, "date", p.ExportedAt)))
		// Package approval and delivery state remain in the original model.
		add(d, prose("notes", fmt.Sprintf("내보내기 식별자: %s\n이전 버전: %s\n승인자: %s\n승인 시각: %s\n입수 경로: %s\n원본 게시 버전: %s\n입력 양식 버전: %s", p.ExportID, p.PreviousVersion, p.ApprovedBy, p.ApprovedAt, p.Source.Kind, p.Source.PublishedVersion, p.Source.TemplateVersion)))
		if p.Validation != nil {
			v := p.Validation
			add(d, prose("notes", fmt.Sprintf("검사 기록: %s\n규칙 버전: %s\n경고 수: %s\n%s", v.PassedAt, v.Ruleset, v.WarningCount, strings.Join(v.Notes, "\n"))))
		}
		for _, f := range p.Files {
			// A codebook cannot contain its own final byte count or hash.
			if f.Role == "codebook" {
				f.Bytes, f.SHA256 = "", ""
			}
			add(d, prose("notes", fmt.Sprintf("패키지 파일: %s\n역할: %s\n파일 참조: %s\n크기: %s\nSHA-256: %s\n쪽 범위: %s", f.Name, f.Role, f.FileRef, f.Bytes, f.SHA256, f.Pages)))
		}
	}
	if cb.Template.ID.Value != "" {
		add(d, text("notes", "입력 양식: "+cb.Template.ID.Value+"\n양식 버전: "+cb.Template.Version.Value+"\n양식 작성일: "+cb.Template.Created.Value))
	}
	if v := cb.Study.Get("RULE_VERSION"); v != "" {
		add(d, text("notes", "적용 규칙 버전: "+v))
	}
	if cb.Admin.CategoryCode != "" {
		add(d, text("notes", "재단 분류: "+cb.Admin.CategoryCode))
	}
	if r := cb.Admin.Round; r.Year != "" || r.WaveType != "" || r.WaveNo != "" {
		add(d, prose("notes", fmt.Sprintf("관리 차수: 연도 %s / 구분 %s / 회차 %s", r.Year, r.WaveType, r.WaveNo)))
	}
	return d
}

func sourceKnown(v string) bool {
	return strings.TrimSpace(v) != "" && !V4Absent(v) && !strings.HasPrefix(strings.TrimSpace(v), "확인 불가") && !strings.HasPrefix(strings.TrimSpace(v), "확인불가")
}

var sourceCountRe = regexp.MustCompile(`^(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)$`)

func sourceCount(v string) string {
	v = strings.TrimSpace(v)
	if !sourceCountRe.MatchString(v) {
		return ""
	}
	return strings.ReplaceAll(v, ",", "")
}

// recordText marks only the fields represented here. Remainders are written
// at their own study/file/variable/category scope with their source labels.
type recordText struct {
	rec  Record
	used map[string]bool
}

func fromRecord(r Record) *recordText        { return &recordText{r, map[string]bool{}} }
func (r *recordText) take(key string) string { r.used[key] = true; return r.rec.Get(key) }
func (r *recordText) field(tag, key string, attrs ...string) *ddi.Node {
	return text(tag, r.take(key), attrs...)
}
func (r *recordText) description(tag string, keys ...string) *ddi.Node {
	var lines []string
	for _, key := range keys {
		if v := r.take(key); strings.TrimSpace(v) != "" {
			label := fieldLabels[key]
			if label == "" {
				label = key
			}
			lines = append(lines, label+": "+v)
		}
	}
	return text(tag, strings.Join(lines, "\n"))
}
func (r *recordText) remaining(tag string) []*ddi.Node {
	var out []*ddi.Node
	for _, key := range slices.Sorted(maps.Keys(r.rec)) {
		if !r.used[key] {
			if n := r.description(tag, key); n != nil {
				out = append(out, n)
			}
		}
	}
	return out
}

func (s *standardBuilder) study() *ddi.Node {
	r := fromRecord(s.cb.Study)
	r.take("RULE_VERSION")
	r.take("SOURCE_FILES")
	title := add(el("titlStmt"), r.field("titl", "TITLE"), r.field("IDNo", "DATA_NO", "agency", "한국과학창의재단"), text("IDNo", s.cb.Admin.WallaID, "agency", "walla"))
	cit := add(el("citation"), title)
	org, name := r.take("PI_ORG"), r.take("PI_NAME")
	if sourceKnown(name) {
		add(cit, add(el("rspStmt"), text("AuthEnty", name, "affiliation", org)))
	} else {
		add(cit, add(el("rspStmt"), text("AuthEnty", org)))
	}
	add(cit, add(el("prodStmt"), r.field("fundAg", "FUNDER")), add(el("serStmt"), r.field("serName", "SERIES_NAME")), r.description("notes", "PROJECT_TITLE", "PROJECT_NO"))
	if name != "" && !sourceKnown(name) {
		add(cit, text("notes", "연구책임자: "+name))
	}
	info := add(el("stdyInfo"), add(el("subject"), keywords(r.take("KEYWORDS"))...), r.field("abstract", "PURPOSE", "contentType", "purpose"), r.field("abstract", "SUMMARY", "contentType", "abstract"))
	summary := el("sumDscr")
	round := r.rec.Get("STUDY_ROUND")
	hasDate := false
	for _, pair := range [][2]string{{"FIELD_START", "start"}, {"FIELD_END", "end"}} {
		if v := r.rec.Get(pair[0]); sourceKnown(v) {
			r.take(pair[0])
			add(summary, el("collDate", "event", pair[1], "date", v, "cycle", round))
			hasDate = true
		}
	}
	if hasDate {
		r.take("STUDY_ROUND")
	}
	add(info, add(summary, r.field("anlyUnit", "ANALYSIS_UNIT"), r.field("universe", "UNIVERSE", "clusion", "I")))
	coll := add(el("dataColl"), r.field("dataCollector", "FIELD_ORG"), r.description("sampProc", "SAMPLING_TYPE", "SAMPLING_DESC", "STRATA_VARS", "CLUSTER_VAR"),
		add(el("sampleFrame"), r.field("sampleFrameName", "SAMPLE_FRAME"), r.description("txt", "POP_SOURCE")))
	if v := sourceCount(r.rec.Get("TARGET_N")); v != "" {
		r.take("TARGET_N")
		add(coll, add(el("targetSampleSize"), text("sampleSize", v)))
	}
	add(coll, r.field("collMode", "MODE"), r.description("resInstru", "SPLIT_BALLOT", "SPLIT_VAR"), r.description("weight", "WEIGHT_YN", "WEIGHT_VAR", "WEIGHT_METHOD"), r.field("cleanOps", "EDIT_METHOD"), r.field("cleanOps", "CLEANING_LOG"))
	method := add(el("method"), coll, r.description("notes", "TOOL", "CONTACT_N", "RR_FORMULA"), add(el("anlyInfo"), r.field("respRate", "RESPONSE_RATE")), r.description("dataProcessing", "EXCLUDE_N", "EXCLUDE_RULE"))
	var instructions []*ddi.Node
	for _, rec := range s.cb.Recodes {
		rr := fromRecord(rec)
		typ := rr.take("TRANSFORM_TYPE")
		id := s.id("P")
		add(method, text("dataProcessing", "05_재코딩이력의 제출자 기록입니다. 원자료를 이용한 실행 검증은 수행하지 않았습니다.", "ID", id, "type", typ))
		instructions = append(instructions, add(el("codingInstructions", "ID", s.id("CI"), "type", typ, "relatedProcesses", id), rr.description("txt", RecodeKeys...)))
	}
	// Legacy system-created cleansing is also documented without inventing SQL.
	for _, f := range s.cb.Files {
		if c := f.Cleansing; c != nil {
			id := s.id("CL")
			add(coll, prose("cleanOps", fmt.Sprintf("정제 기록: 파일 %s\n작업자: %s\n작업 시각: %s\n원본 파일 참조: %s\n수정 셀: %s\n영향 행: %s\n이력 파일: %s", f.Rec.Get("FILE_NO"), c.AppliedBy, c.AppliedAt, c.RawFileRef, c.CellsChanged, c.RowsAffected, c.LogFile), "ID", id))
			for _, rule := range c.Rules {
				var pairs []string
				for _, m := range rule.Map {
					pairs = append(pairs, m.From+" → "+m.To)
				}
				instructions = append(instructions, add(el("codingInstructions", "ID", s.id("CI"), "type", rule.Action, "relatedProcesses", id), prose("txt", fmt.Sprintf("규칙: %s\n변수: %s\n조건: %s\n대응: %s\n영향 행: %s\n출처: %s\n이유: %s\n근거: %s", rule.ID, rule.VarName, rule.Condition, strings.Join(pairs, ", "), rule.Rows, rule.Source, rule.Reason, rule.Evidence))))
			}
		}
	}
	add(method, instructions...)
	study := add(el("stdyDscr"), cit, info, method, add(el("dataAccs"), add(el("useStmt"), r.field("restrctn", "ACCESS_RESTRICTIONS"))))
	add(study, r.remaining("notes")...)
	return study
}

func (s *standardBuilder) file(f *File) *ddi.Node {
	r := fromRecord(f.Rec)
	no := r.take("FILE_NO")
	name := r.take("FILE_NAME")
	if f.PackageName != "" {
		name = f.PackageName
	}
	ft := add(el("fileTxt"), text("fileName", name), r.field("fileCont", "CONTENT"))
	dims := el("dimensns")
	if v := sourceCount(r.rec.Get("VALID_N")); v != "" {
		r.take("VALID_N")
		add(dims, text("caseQnty", v))
	}
	quantity := sourceCount(r.rec.Get("VARIABLE_N"))
	if quantity != "" {
		r.take("VARIABLE_N")
	} else {
		quantity = strconv.Itoa(s.varCount[no] + s.sourceSystemCount(no))
	}
	add(ft, add(dims, text("varQnty", quantity)), r.field("fileType", "FILE_FORMAT"))
	out := add(el("fileDscr", "ID", "F"+no), ft)
	if f.PackageName != "" {
		add(out, text("notes", "입력 파일명: "+f.Rec.Get("FILE_NAME")))
	}
	add(out, r.remaining("notes")...)
	if f.LinkLevel != "" || f.SensLevel != "" {
		add(out, prose("notes", fmt.Sprintf("연결 단위: %s\n보안 등급: %s", f.LinkLevel, f.SensLevel)))
	}
	if f.SHA256 != "" || f.Bytes != "" {
		add(out, prose("notes", fmt.Sprintf("SHA-256: %s\n파일 크기: %s", f.SHA256, f.Bytes)))
	}
	if f.RawName != "" || f.RawSHA256 != "" || f.RawBytes != "" {
		add(out, prose("notes", fmt.Sprintf("원본 파일: %s\n원본 SHA-256: %s\n원본 크기: %s", f.RawName, f.RawSHA256, f.RawBytes)))
	}
	return out
}
func (s *standardBuilder) sourceSystemCount(fileNo string) int {
	count := 0
	for _, n := range s.cb.SourceSystemVars {
		if attr(n, "files") == "F"+fileNo {
			count++
		}
	}
	return count
}
func copyNode(n *ddi.Node) *ddi.Node {
	out := &ddi.Node{Name: n.Name, Text: n.Text, Raw: n.Raw, Attrs: slices.Clone(n.Attrs)}
	for _, c := range n.Children {
		out.Children = append(out.Children, copyNode(c))
	}
	return out
}
func (s *standardBuilder) data(profile *ddi.Node) *ddi.Node {
	out := el("dataDscr")
	for _, g := range kids(child(profile, "dataDscr"), "varGrp") {
		add(out, standardDescription(g))
	}
	add(out, s.systemVariables()...)
	for _, v := range s.cb.Vars {
		add(out, s.variable(v))
	}
	return out
}

func (s *standardBuilder) variable(v *Var) *ddi.Node {
	r := fromRecord(v.Rec)
	no, name := r.take("FILE_NO"), r.take("VAR_NAME")
	n := el("var", "ID", v.ID, "name", name, "files", "F"+no)
	if v.ID == s.weight {
		n.Attrs = append(n.Attrs, ddi.Attr{Name: "wgt", Value: "wgt"})
	}
	nature := map[string]string{"명목": "nominal", "서열": "ordinal", "등간": "interval", "비율": "ratio"}[r.rec.Get("MEAS_LEVEL")]
	if nature != "" {
		r.take("MEAS_LEVEL")
		n.Attrs = append(n.Attrs, ddi.Attr{Name: "nature", Value: nature})
	}
	interval := r.rec.Get("DDI_INTERVAL")
	if interval == "discrete" || interval == "contin" {
		r.take("DDI_INTERVAL")
		n.Attrs = append(n.Attrs, ddi.Attr{Name: "intrvl", Value: interval})
	}
	if s.cb.Template.Version.Value != "4.0" && r.rec.Get("MEAS_LEVEL") == "연속" {
		n.Attrs = append(n.Attrs, ddi.Attr{Name: "intrvl", Value: "contin"})
	}
	add(n, el("location", "fileid", "F"+no), r.field("labl", "VAR_LABEL"), r.description("security", "PII", "SENSITIVE", "ACCESS_LEVEL"), add(el("qstn", "seqNo", r.take("Q_NO")), r.field("qstnLit", "Q_TEXT")), r.field("universe", "RESP_UNIVERSE", "clusion", "I"), r.field("stdCatgry", "CODE_SYSTEM"))
	for _, c := range s.codes[varKey(v)] {
		cr := fromRecord(c.Rec)
		cr.take("FILE_NO")
		cr.take("VAR_NAME")
		cat := el("catgry")
		if s.cb.Template.Version.Value == "4.0" {
			switch cr.rec.Get("IS_MISSING") {
			case "예", "Y":
				cr.take("IS_MISSING")
				cat.Attrs = append(cat.Attrs, ddi.Attr{Name: "missing", Value: "Y"})
			case "아니오", "N":
				cr.take("IS_MISSING")
				cat.Attrs = append(cat.Attrs, ddi.Attr{Name: "missing", Value: "N"})
			default:
				s.fail("F%s/%s/%s IS_MISSING: 예·아니오로 확정되지 않음", no, name, cr.rec.Get("CODE"))
			}
			if isMissing(c) && strings.TrimSpace(cr.rec.Get("MISSING_REASON")) != "" && !strings.HasPrefix(strings.TrimSpace(cr.rec.Get("MISSING_REASON")), "확인 불가") {
				cat.Attrs = append(cat.Attrs, ddi.Attr{Name: "missType", Value: cr.take("MISSING_REASON")})
			}
		} else {
			missing := "N"
			if isMissing(c) {
				missing = "Y"
			}
			cat.Attrs = append(cat.Attrs, ddi.Attr{Name: "missing", Value: missing})
			if isMissing(c) && c.Type != "" {
				cat.Attrs = append(cat.Attrs, ddi.Attr{Name: "missType", Value: c.Type})
			}
		}
		add(cat, cr.field("catValu", "CODE"), cr.field("labl", "CODE_LABEL"))
		add(cat, cr.remaining("txt")...)
		if c.StdCode != "" || c.StdSet != "" {
			add(cat, text("txt", "표준 코드 매핑: "+c.StdSet+" / "+c.StdCode))
		}
		add(n, cat)
	}
	add(n, r.description("codInstr", "REVERSE"), r.field("concept", "STD_ID"))
	if value := r.rec.Get("DERIVATION"); value != "" {
		add(n, add(el("derivation"), r.field("drvdesc", "DERIVATION"), r.description("drvdesc", "SOURCE_VAR")))
	}
	typ, category := v4Format(r.rec.Get("DATA_TYPE"))
	if typ != "" {
		dtype := r.take("DATA_TYPE")
		representation := map[string]string{"숫자": "numeric", "정수": "numeric", "실수": "numeric", "numeric": "numeric", "문자": "text", "문자열": "text", "character": "text", "날짜": "datetime", "date": "datetime", "일시": "datetime"}[dtype]
		if representation != "" {
			n.Attrs = append(n.Attrs, ddi.Attr{Name: "representationType", Value: representation})
		}
		if category == "other" {
			category = ""
		}
		add(n, el("varFormat", "type", typ, "category", category))
	} else if s.cb.Template.Version.Value != "4.0" {
		format, category := "numeric", ""
		switch MeasurementLevel(v) {
		case "text":
			format = "character"
		case "date":
			format, category = "character", "date"
		}
		add(n, el("varFormat", "type", format, "category", category))
	}
	add(n, r.remaining("notes")...)
	// Analysis settings and FK outcomes are retained as descriptions; they are
	// not standardized ETL configuration or a guarantee of code-table validity.
	if fk := v.FK; fk != nil {
		add(n, prose("notes", fmt.Sprintf("표준 코드 연결: %s\n테이블: %s\n키 열: %s\n매핑 방법: %s\n코드 체계: %s\n기준연도: %s\n연결 키: %s\n코드 버전: %s\n검사 시각: %s\n일치 행: %s\n불일치 행: %s", fk.CodeSet, fk.DimTable, fk.KeyColumn, fk.Mapping, fk.CodeSystem, fk.RefYear, fk.LinkKey, fk.CodeSetVersion, fk.ValidatedAt, fk.MatchedRows, fk.UnmatchedRows)))
	}
	for _, pair := range [][2]string{{"역할", v.Role}, {"분석 차원", v.DimMeasure}, {"집계 규칙", v.AggRules}, {"분석용 물리명", v.PhysicalName}, {"분석용 논리명", v.LogicalName}, {"분석 포함", v.IncludeYn}, {"코드셋", v.CodeSetID}, {"별칭", strings.Join(v.Synonyms, ", ")}, {"복수응답 그룹", v.MRGroup}, {"배타 보기", v.NoneOption}, {"행렬 그룹", v.GridGroup}, {"행 라벨", v.RowLabel}, {"순위", v.RankNo}, {"텍스트 저장 형식", v.TextFormat}, {"상위 변수", v.ParentVar}, {"척도 최솟값", v.Scale.Min}, {"척도 최댓값", v.Scale.Max}, {"척도 방향", v.Scale.Dir}} {
		if pair[1] != "" {
			add(n, text("notes", pair[0]+": "+pair[1]))
		}
	}
	return n
}

// Renumber imported IDs and their links together; source IDs can clash with
// the new survey IDs. Historical source extensions become descriptions.
func (s *standardBuilder) systemVariables() []*ddi.Node {
	var nodes []*ddi.Node
	ids := map[string]string{}
	for _, source := range s.cb.SourceSystemVars {
		n := standardDescription(source)
		Walk(n, func(c, _ *ddi.Node, _ string) bool {
			for i, a := range c.Attrs {
				if a.Name == "ID" {
					if _, duplicate := ids[a.Value]; duplicate {
						s.fail("입력 시스템 변수 ID %s 겹침", a.Value)
					}
					id := s.id("SOURCE_SYSTEM_")
					ids[a.Value], c.Attrs[i].Value = id, id
				}
			}
			return true
		})
		if attr(n, "ID") == "" {
			n.Attrs = append([]ddi.Attr{{Name: "ID", Value: s.id("SOURCE_SYSTEM_")}}, n.Attrs...)
		}
		nodes = append(nodes, n)
	}
	for _, n := range nodes {
		Walk(n, func(c, _ *ddi.Node, _ string) bool {
			for i, a := range c.Attrs {
				switch a.Name {
				case "wgt-var", "weight", "qstn", "var", "refs", "sdatrefs", "methrefs", "pubrefs", "access", "recRef":
					refs := strings.Fields(a.Value)
					for j, ref := range refs {
						if id, ok := ids[ref]; ok {
							refs[j] = id
						}
					}
					c.Attrs[i].Value = strings.Join(refs, " ")
				}
			}
			return true
		})
	}
	return nodes
}

// Avoid trailing spaces on empty lines in generated descriptions.
func prose(tag, value string, attrs ...string) *ddi.Node {
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return text(tag, strings.Join(lines, "\n"), attrs...)
}

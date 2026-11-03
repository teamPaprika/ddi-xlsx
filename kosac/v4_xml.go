package kosac

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

const v4NoteType = "KOFAC-SURVEY-META-v4.0"

type v4Record struct {
	Template *TemplateMeta     `json:"template,omitempty"`
	FileNo   string            `json:"fileNo,omitempty"`
	VarName  string            `json:"variable,omitempty"`
	Code     string            `json:"code,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
}

// v4Notes preserves template-only metadata without adding another XML namespace.
// CSV inspection and execution of the contractor's recoding remain outside the converter.
func (cb *Codebook) v4Notes() []*ddi.Node {
	var out []*ddi.Node
	emit := func(sheet string, rec v4Record) {
		b, _ := json.Marshal(rec)
		out = append(out, text("notes", string(b), "type", v4NoteType, "subject", sheet))
	}
	fields := func(r Record, keys []string) map[string]string {
		m := map[string]string{}
		for _, k := range keys {
			m[k] = r.Get(k)
		}
		return m
	}
	emit(SheetMeta, v4Record{Template: &cb.Template})
	studyFields := fields(cb.Study, StudyExtraV4)
	if cb.StudyValidN() != cb.Study.Get("VALID_N") {
		studyFields["VALID_N"] = cb.Study.Get("VALID_N")
	}
	delete(studyFields, "SERIES_NAME")
	delete(studyFields, "ACCESS_RESTRICTIONS")
	if typed(cb.Study.Get("FIELD_START")) != "" || typed(cb.Study.Get("FIELD_END")) != "" {
		delete(studyFields, "STUDY_ROUND")
	}
	emit(SheetStudy, v4Record{Fields: studyFields})
	for _, f := range cb.Files {
		emit(SheetFiles, v4Record{FileNo: f.Rec.Get("FILE_NO"), Fields: fields(f.Rec, FileExtraV4)})
	}
	for _, v := range cb.Vars {
		emit(SheetVars, v4Record{FileNo: v.Rec.Get("FILE_NO"), VarName: v.Rec.Get("VAR_NAME"), Fields: fields(v.Rec, append(slices.Clone(VarExtraV4), "MEAS_LEVEL"))})
	}
	for _, c := range cb.Codes {
		emit(SheetCodes, v4Record{FileNo: c.Rec.Get("FILE_NO"), VarName: c.Rec.Get("VAR_NAME"), Code: c.Rec.Get("CODE"), Fields: fields(c.Rec, CodeExtraV4)})
	}
	for _, rec := range cb.Recodes {
		emit(SheetRecodes, v4Record{Fields: fields(rec, RecodeKeys)})
	}
	return out
}

func (r *reader) readV4(cb *Codebook, root *ddi.Node) {
	study := r.child(root, "stdyDscr")
	if study == nil {
		return
	}
	files := map[string]*File{}
	vars := map[[2]string]*Var{}
	codes := map[[3]string]*Code{}
	for _, f := range cb.Files {
		files[f.Rec.Get("FILE_NO")] = f
	}
	for _, v := range cb.Vars {
		vars[varKey(v)] = v
	}
	for _, c := range cb.Codes {
		codes[[3]string{c.Rec.Get("FILE_NO"), c.Rec.Get("VAR_NAME"), c.Rec.Get("CODE")}] = c
	}
	seen := map[string]bool{}
	for _, n := range study.Children {
		typ, _ := n.Attr("type")
		if n.Name != "notes" || typ != v4NoteType {
			continue
		}
		r.use(n, "type", "subject")
		subject := r.attr(n, "subject")
		var entry v4Record
		d := json.NewDecoder(bytes.NewBufferString(r.str(n)))
		d.DisallowUnknownFields()
		if err := d.Decode(&entry); err != nil {
			r.fail(n, "", "v4.0 메타데이터를 읽을 수 없음: %v", err)
			continue
		}
		var trailing any
		if err := d.Decode(&trailing); err != io.EOF {
			r.fail(n, "", "v4.0 JSON 뒤에 데이터가 남아 있음")
			continue
		}
		var rec Record
		var keys []string
		switch subject {
		case SheetMeta:
			if entry.Template == nil || entry.Template.Version.Value != "4.0" {
				r.fail(n, "", "v4.0 양식 버전이 없음")
				continue
			}
			cb.Template = *entry.Template
		case SheetStudy:
			rec, keys = cb.Study, append(slices.Clone(StudyExtraV4), "VALID_N")
		case SheetFiles:
			if f := files[entry.FileNo]; f != nil {
				rec = f.Rec
			}
			keys = FileExtraV4
		case SheetVars:
			if v := vars[[2]string{entry.FileNo, entry.VarName}]; v != nil {
				rec = v.Rec
			}
			keys = append(slices.Clone(VarExtraV4), "MEAS_LEVEL")
		case SheetCodes:
			if c := codes[[3]string{entry.FileNo, entry.VarName, entry.Code}]; c != nil {
				rec = c.Rec
			}
			keys = CodeExtraV4
		case SheetRecodes:
			rec, keys = Record{}, RecodeKeys
			cb.Recodes = append(cb.Recodes, rec)
		default:
			r.fail(n, "", "알 수 없는 v4.0 시트 %s", subject)
			continue
		}
		if subject == SheetMeta {
			if seen[SheetMeta] {
				r.fail(n, "", "v4.0 양식 메타데이터 중복")
			}
			seen[SheetMeta] = true
			continue
		}
		if rec == nil {
			r.fail(n, "", "v4.0 메타데이터가 없는 파일·변수·코드를 가리킴")
			continue
		}
		id := fmt.Sprintf("%s/%s/%s/%s", subject, entry.FileNo, entry.VarName, entry.Code)
		if subject != SheetRecodes && seen[id] {
			r.fail(n, "", "v4.0 메타데이터 중복: %s", id)
			continue
		}
		seen[id] = true
		for key, value := range entry.Fields {
			if !slices.Contains(keys, key) {
				r.fail(n, "", "%s의 알 수 없는 항목 %s", subject, key)
				continue
			}
			rec.Set(key, value)
		}
	}
	if len(seen) > 0 && !seen[SheetMeta] {
		r.fail(study, "", "v4.0 양식 버전 메타데이터가 없음")
	}
}

func (b *builder) absentRef(value string) string {
	if b.cb.Template.Version.Value == "4.0" && V4Absent(value) {
		return ""
	}
	return value
}

func v4Format(value string) (string, string) {
	switch strings.TrimSpace(value) {
	case "숫자", "정수", "실수", "numeric":
		return "numeric", ""
	case "문자", "문자열", "character":
		return "character", ""
	case "일시":
		return "character", "other"
	case "날짜", "date":
		return "character", "date"
	}
	return "", ""
}

func (cb *Codebook) v4Native(study *ddi.Node) {
	for _, n := range study.Children {
		if n.Name == "citation" {
			add(n, add(el("serStmt"), text("serName", cb.Study.Get("SERIES_NAME"))))
		}
	}
	Walk(study, func(n, parent *ddi.Node, path string) bool {
		if n.Name == "collDate" && strings.TrimSpace(cb.Study.Get("STUDY_ROUND")) != "" {
			n.Attrs = append(n.Attrs, ddi.Attr{Name: "cycle", Value: cb.Study.Get("STUDY_ROUND")})
		}
		return true
	})
	access := add(el("dataAccs"), add(el("useStmt"), text("restrctn", cb.Study.Get("ACCESS_RESTRICTIONS"))))
	if len(access.Children) > 0 {
		at := len(study.Children)
		for i, n := range study.Children {
			if isKosac(n.Name) {
				at = i
				break
			}
		}
		study.Children = slices.Insert(study.Children, at, access)
	}
}

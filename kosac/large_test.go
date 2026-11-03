// The sample and large-codebook tests run Read, Check and ValidateXML, whose
// packages import kosac, so they live in the external test package.
package kosac_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/check"
	"github.com/teamPaprika/ddi-xlsx/kosac/template"
)

var update = flag.Bool("update", false, "rewrite testdata/kosac/v32_sample_2024.{xml,json}")

var sampleDir = filepath.Join("..", "testdata", "kosac")

func readSample(t *testing.T) *kosac.Codebook {
	t.Helper()
	x, err := os.ReadFile(filepath.Join(sampleDir, "v32_sample_2024.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	cb, is, err := template.Read(x, template.ReadOptions{})
	if err != nil || len(is) != 0 {
		t.Fatalf("Read: %v %v", is, err)
	}
	return cb
}

// appendixC is the request's example dictionary and code sets.
func appendixC() kosac.RefData {
	item := func(id, name, phys, level, set string, syn ...string) kosac.StdItem {
		return kosac.StdItem{StdID: id, NameKo: name, PhysicalName: phys, MeasLevel: level, CodeSetID: set, Synonyms: syn,
			Series: "공통", Active: "Y"}
	}
	return kosac.RefData{
		StdItems: map[string]kosac.StdItem{
			"PERSON.GENDER": item("PERSON.GENDER", "성별", "gender_cd", "nominal", "CC_GENDER", "남녀", "성"),
			"PERSON.AGE":    item("PERSON.AGE", "연령", "age", "scale", "", "나이", "만 나이"),
			"REGION.SIDO":   item("REGION.SIDO", "시도", "sido_cd", "nominal", "CC_SIDO", "지역", "거주 지역"),
			"SCI.TRUST":     item("SCI.TRUST", "과학기술 신뢰도", "sci_trust", "ordinal", "LIKERT5_TRUST", "과학 신뢰", "과학기술에 대한 신뢰"),
		},
		CodeSetDefs: map[string]kosac.CodeSetDef{
			"CC_GENDER": {CodeSetID: "CC_GENDER", NameKo: "성별", PhysicalName: "gender_cd", DimTable: "dw.t_common_code",
				KeyColumn: "standard_code", CodeSystem: "KOSAC"},
			"CC_SIDO": {CodeSetID: "CC_SIDO", NameKo: "시도", PhysicalName: "sido_cd", DimTable: "dw.t_common_code",
				KeyColumn: "standard_code", CodeSystem: "행정표준코드", ParentSet: "CC_REGION"},
		},
		CodeSets: map[string][]kosac.StdCode{
			"CC_GENDER": {
				{CodeSetID: "CC_GENDER", Code: "M", Label: "남", AltLabels: []string{"남성", "남자", "남학생"}, Sort: "1"},
				{CodeSetID: "CC_GENDER", Code: "F", Label: "여", AltLabels: []string{"여성", "여자", "여학생"}, Sort: "2"},
			},
			"CC_SIDO": {
				{CodeSetID: "CC_SIDO", Code: "11", Label: "서울특별시", AltLabels: []string{"서울"}, Sort: "1", ParentCode: "R1"},
				{CodeSetID: "CC_SIDO", Code: "26", Label: "부산광역시", AltLabels: []string{"부산"}, Sort: "2", ParentCode: "R4"},
			},
		},
	}
}

// groupKeys lists the distinct non-empty keys in 03 order with their members.
func groupKeys(cb *kosac.Codebook, key func(*kosac.Var) string) ([]string, map[string][]*kosac.Var) {
	var keys []string
	members := map[string][]*kosac.Var{}
	for _, v := range cb.Vars {
		k := key(v)
		if k == "" {
			continue
		}
		if members[k] == nil {
			keys = append(keys, k)
		}
		members[k] = append(members[k], v)
	}
	return keys, members
}

func mrOf(typ string) func(*kosac.Var) string {
	return func(v *kosac.Var) string {
		if v.Rec.Get("MR_TYPE") != typ {
			return ""
		}
		return v.MRGroup
	}
}

func TestSampleDerived(t *testing.T) {
	cb := readSample(t)
	kosac.DeriveDefaults(cb, kosac.RefData{})

	mr, mrMembers := groupKeys(cb, mrOf("보기별 열"))
	if want := []string{"QA19", "QA21", "QD1_1"}; !slices.Equal(mr, want) {
		t.Errorf("MR groups %q, want %q", mr, want)
	}
	for k, want := range map[string]string{"QA19": "Y", "QA21": "Y", "QD1_1": "N"} {
		for _, v := range mrMembers[k] {
			if v.NoneOption != want {
				t.Errorf("%s NoneOption = %q, want %q", v.Rec.Get("VAR_NAME"), v.NoneOption, want)
			}
		}
	}
	rank, rankMembers := groupKeys(cb, mrOf("순위별 열"))
	if want := []string{"QA22", "QA30", "QA31", "QA37"}; !slices.Equal(rank, want) {
		t.Errorf("rank groups %q, want %q", rank, want)
	}
	n := 0
	for _, k := range rank {
		for i, v := range rankMembers[k] {
			if v.RankNo != strconv.Itoa(i+1) {
				t.Errorf("%s RankNo = %q", v.Rec.Get("VAR_NAME"), v.RankNo)
			}
		}
		n += len(rankMembers[k])
	}
	for _, k := range mr {
		n += len(mrMembers[k])
	}
	if n != 27 {
		t.Errorf("MR and rank members = %d, want 27", n)
	}

	grids, gridMembers := groupKeys(cb, func(v *kosac.Var) string { return v.GridGroup })
	var prefixes []string
	for _, k := range grids {
		if first := gridMembers[k][0].Rec.Get("VAR_NAME"); k != first {
			t.Errorf("grid key %s is not its first member %s", k, first)
		}
		p, _ := kosac.Prefix(k)
		prefixes = append(prefixes, p)
	}
	want := []string{"QA1", "QA3", "QA5_1", "QA5_2", "QA11", "QA12", "QA13", "QA15", "QA23", "QA26", "QA35", "QA36",
		"QB2", "QB5", "QB9", "QC6", "QC7", "QC8", "A1", "C7", "C8", "C9"}
	if !slices.Equal(prefixes, want) {
		t.Errorf("grid groups %q\nwant %q", prefixes, want)
	}

	// A row label is a marked row or the whole label, never a label cut mid-word.
	labelled := 0
	for _, v := range cb.Vars {
		if v.RowLabel == "" {
			continue
		}
		labelled++
		_, row, ok := kosac.GridStem(v.Rec.Get("Q_TEXT"))
		if !(ok && v.RowLabel == row) && v.RowLabel != v.Rec.Get("VAR_LABEL") {
			t.Errorf("%s RowLabel %q is neither the marked row nor the label", v.Rec.Get("VAR_NAME"), v.RowLabel)
		}
	}
	if labelled == 0 {
		t.Error("no row labels")
	}
}

// xmllint validates against the vendored DDI XSD and the kosac draft.
func xmllint(t *testing.T) check.Validator {
	return func(s check.Schema, doc []byte) ([]string, error) {
		p := filepath.Join(xsdtest.Root(), "xsd", "ddi-codebook-2.5", "codebook.xsd")
		if s == check.SchemaKosac {
			p = filepath.Join(xsdtest.Root(), "kosac", "xsd", "kosac-3.3.xsd")
		}
		return xsdtest.ValidateSchema(t, p, doc), nil
	}
}

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join(sampleDir, name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs; rerun with -update and review the diff", name)
	}
}

func TestSampleGolden(t *testing.T) {
	cb := readSample(t)
	ref := appendixC()
	kosac.DeriveDefaults(cb, ref)
	if err := kosac.AssignIDs(cb, nil); err != nil {
		t.Fatal(err)
	}
	admin := kosac.StudyAdmin{CategoryCode: "문화", Round: kosac.Round{Year: "2024", WaveType: "main"}}
	if cb.Admin != admin {
		t.Errorf("derived Admin = %+v, want %+v", cb.Admin, admin)
	}
	cb.Admin.WallaID = "KosacSample000000001"

	hex := func(c byte) string { return string(bytes.Repeat([]byte{c}, 64)) }
	f := cb.Files[0]
	f.PackageName, f.SHA256, f.Bytes = "문화-01-01_2024_v1_data_1.csv", hex('c'), "4096"
	f.RawName, f.RawSHA256, f.RawBytes = "2024_과학기술국민인식도_응답데이터.csv", hex('b'), "4100"
	f.Cleansing = &kosac.Cleansing{AppliedBy: `kosac\hsham`, AppliedAt: "2026-11-02T10:50:00+09:00", RawFileRef: "F1_raw",
		CellsChanged: "4", RowsAffected: "3", LogFile: "cleansing.json",
		Rules: []kosac.CleansingRule{
			{ID: "CL-01", VarName: "QA2", Action: "REPLACE", Condition: "QA2=9", Rows: "2", Source: "MANUAL",
				Reason: "범위 밖 코드", Evidence: "용역사 회신", Map: []kosac.MapEntry{{From: "9", To: "5"}}},
			{ID: "CL-02", VarName: "DQ4a", Action: "MISSING_9", Condition: "DQ4a>120", Rows: "1", Source: "MANUAL",
				Reason: "논리 오류"},
		}}
	sq1 := cb.Var("1", "SQ1")
	sq1.FK = &kosac.FK{CodeSet: "CC_GENDER", Mapping: "CODE", CodeSystem: "KOSAC", RefYear: "2026", LinkKey: "N",
		CodeSetVersion: "2026.2", ValidatedAt: "2026-11-03T10:05:00+09:00", MatchedRows: "10,000", UnmatchedRows: "0"}
	if err := kosac.ApplyFK(cb, sq1, ref.CodeSetDefs["CC_GENDER"]); err != nil {
		t.Fatal(err)
	}
	cb.Package = &kosac.Package{ExportID: "exp_20261103_0001", CodebookVersion: "1", ApprovedBy: `kosac\hsham`,
		ApprovedAt: "2026-11-03T10:12:00+09:00", ExportedAt: "2026-11-03", Software: "Walla", SoftwareVersion: "2.4",
		Source: kosac.SourceSurvey{Kind: "upload", TemplateVersion: "3.2"},
		Files: []kosac.PackageFile{
			{Role: "codebook", Name: "문화-01-01_2024_v1_codebook.xml"},
			{Role: "data", FileRef: "F1", Name: f.PackageName, Bytes: f.Bytes, SHA256: f.SHA256},
			{Role: "data_raw", FileRef: "F1_raw", Name: f.RawName, Bytes: f.RawBytes, SHA256: f.RawSHA256},
			{Role: "cleansing", Name: "cleansing.json", Bytes: "812", SHA256: hex('e')},
			{Role: "report", Name: "문화-01-01_2024_v1_report.pdf", Bytes: "52428800", SHA256: hex('d')},
		},
		Validation: &kosac.Validation{PassedAt: "2026-11-01T15:40:00+09:00", Ruleset: "2026.10", WarningCount: "154",
			Notes: []string{"C08 145건 확인", "C01 확인 불가 4건 승인"}},
	}

	x, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	j, err := cb.JSON()
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "v32_sample_2024.xml", x)
	golden(t, "v32_sample_2024.json", j)

	for _, doc := range []struct {
		name string
		b    []byte
	}{{"XML", x}, {"JSON", j}} {
		parsed, _, err := ddi.Parse(doc.b)
		if err != nil {
			t.Fatalf("Parse %s: %v", doc.name, err)
		}
		if !bytes.Equal(ddi.WriteXML(parsed), x) || !bytes.Equal(ddi.WriteJSON(parsed), j) {
			t.Errorf("%s standard tree roundtrip differs", doc.name)
		}
	}
	if is := check.ValidateXML(x, xmllint(t)); len(is) != 0 {
		for _, i := range is {
			t.Errorf("%s %s %s", i.Rule, i.Path, i.Actual)
		}
	}
}

// TestCheckPassBuilds holds the promise that a model Check passes without an
// R assembles, and its XML validates: on the sample, made clean, and on edits
// that once slipped through Check and broke XML() or the XSD.
func TestCheckPassBuilds(t *testing.T) {
	study := func(kv ...string) func(*kosac.Codebook) {
		return func(cb *kosac.Codebook) {
			for i := 0; i < len(kv); i += 2 {
				cb.Study.Set(kv[i], kv[i+1])
			}
		}
	}
	fileNo := func(no string) func(*kosac.Codebook) {
		return func(cb *kosac.Codebook) {
			cb.Files[0].Rec.Set("FILE_NO", no)
			for _, v := range cb.Vars {
				v.Rec.Set("FILE_NO", no)
			}
			for _, c := range cb.Codes {
				c.Rec.Set("FILE_NO", no)
			}
		}
	}
	row := func(get func(*kosac.Codebook) kosac.Record, k, v string) func(*kosac.Codebook) {
		return func(cb *kosac.Codebook) { get(cb).Set(k, v) }
	}
	file1 := func(cb *kosac.Codebook) kosac.Record { return cb.Files[0].Rec }
	qa2 := func(cb *kosac.Codebook) kosac.Record { return cb.Var("1", "QA2").Rec }
	// No codes, universe or reference, so only its own FILE_NO check sees it.
	qc4 := func(cb *kosac.Codebook) kosac.Record { return cb.Var("1", "QC4").Rec }
	for _, c := range []struct {
		name string
		edit func(*kosac.Codebook)
	}{
		{"clean", func(*kosac.Codebook) {}},
		{"FILE_NO 01", fileNo("01")},
		{"FILE_NO +1", fileNo("+1")},
		{"FILE_NO unverified", fileNo("확인 불가: 미정")},
		{"var FILE_NO unverified", row(qc4, "FILE_NO", "확인 불가: 미정")},
		{"stale SPLIT_VAR", study("SPLIT_BALLOT", "N", "SPLIT_VAR", "NOPE")},
		{"WEIGHT_VAR under an unverified WEIGHT_YN", study("WEIGHT_YN", "확인 불가: 미정", "WEIGHT_VAR", "NOPE")},
		{"two CLUSTER_VAR names", study("CLUSTER_VAR", "SQ1 SQ2")},
		{"LINK_FILE_NO without LINK_VAR", row(file1, "LINK_FILE_NO", "9")},
		{"PII 예", row(qa2, "PII", "예")},
		{"PII with a space", row(qa2, "PII", "Y ")},
		{"SENSITIVE y", row(qa2, "SENSITIVE", "y")},
		{"REVERSE with a space", row(qa2, "REVERSE", "N ")},
		{"UNIT off the list", row(file1, "UNIT", "가구")},
		{"02 VALID_N not an integer", row(file1, "VALID_N", "약 10000")},
	} {
		t.Run(c.name, func(t *testing.T) {
			cb := readSample(t)
			// SQ1 has no codes, the sample's only R (C07); give it two.
			for _, s := range []string{"1", "2"} {
				r := kosac.Record{}
				r.Set("FILE_NO", "1")
				r.Set("VAR_NAME", "SQ1")
				r.Set("CODE", s)
				r.Set("CODE_LABEL", "보기 "+s)
				cb.Codes = append(cb.Codes, &kosac.Code{Rec: r})
			}
			c.edit(cb)
			var rs []string
			for _, i := range check.Check(cb, kosac.RefData{}) {
				if i.Level == kosac.R {
					rs = append(rs, i.Rule+" "+i.Cell+" "+i.Actual)
				}
			}
			if len(rs) > 0 {
				if c.name == "clean" {
					t.Fatalf("R on the clean sample: %q", rs)
				}
				t.Logf("Check rejects: %q", rs)
				return
			}
			kosac.DeriveDefaults(cb, kosac.RefData{})
			if err := kosac.AssignIDs(cb, nil); err != nil {
				t.Fatal(err)
			}
			x, err := cb.XML()
			if err != nil {
				t.Fatalf("Check passed, XML failed: %v", err)
			}
			for _, i := range check.ValidateXML(x, xmllint(t)) {
				t.Errorf("Check passed, ValidateXML: %s %s %s", i.Rule, i.Path, i.Actual)
			}
		})
	}
}

// largeModel has 3,500 vars and 30,000 codes over two files: per file an ID,
// 1,500 ordinal grid rows of ten codes each, and 249 continuous vars.
func largeModel() *kosac.Codebook {
	rec := func(kv ...string) kosac.Record {
		r := kosac.Record{}
		for i := 0; i < len(kv); i += 2 {
			r.Set(kv[i], kv[i+1])
		}
		return r
	}
	cb := &kosac.Codebook{Template: kosac.TemplateMeta{Version: kosac.Field{Value: "3.2"}}, Study: rec("TITLE", "대용량 코드북", "DATA_NO", "문화-02-01", "FIELD_START", "2025-05-01")}
	for _, no := range []string{"1", "2"} {
		cb.Files = append(cb.Files, &kosac.File{Rec: rec("FILE_NO", no, "FILE_NAME", "data"+no+".csv", "FILE_FORMAT", "CSV", "ID_VAR", "ID")})
		cb.Vars = append(cb.Vars, &kosac.Var{Rec: rec("FILE_NO", no, "VAR_NAME", "ID", "VAR_LABEL", "응답자 ID", "MEAS_LEVEL", "연속")})
		for i := range 1500 {
			q, r := i/10+1, i%10+1
			name := fmt.Sprintf("Q%d_%d", q, r)
			cb.Vars = append(cb.Vars, &kosac.Var{Rec: rec("FILE_NO", no, "VAR_NAME", name, "VAR_LABEL", fmt.Sprintf("행 %d", r),
				"Q_TEXT", fmt.Sprintf("문항 %d에 동의하십니까? ▶ [행 %d]", q, r), "VAR_GROUP", fmt.Sprintf("영역 %d", q/20+1),
				"MEAS_LEVEL", "서열")})
			for c := 1; c <= 10; c++ {
				cb.Codes = append(cb.Codes, &kosac.Code{Rec: rec("FILE_NO", no, "VAR_NAME", name, "CODE", strconv.Itoa(c),
					"CODE_LABEL", fmt.Sprintf("척도 %d", c))})
			}
		}
		for i := range 249 {
			cb.Vars = append(cb.Vars, &kosac.Var{Rec: rec("FILE_NO", no, "VAR_NAME", fmt.Sprintf("N%d", i+1),
				"VAR_LABEL", fmt.Sprintf("수치 %d", i+1), "VAR_GROUP", "수치", "MEAS_LEVEL", "연속")})
		}
	}
	return cb
}

func TestLargeCodebook(t *testing.T) {
	if testing.Short() {
		t.Skip("large codebook")
	}
	cb := largeModel()
	if len(cb.Vars) != 3500 || len(cb.Codes) != 30000 {
		t.Fatalf("model has %d vars and %d codes", len(cb.Vars), len(cb.Codes))
	}
	start, last := time.Now(), time.Now()
	step := func(name string) {
		t.Logf("%-14s %v", name, time.Since(last).Round(time.Millisecond))
		last = time.Now()
	}
	x, err := template.Write(cb)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	step("Write")
	got, _, err := template.Read(x, template.ReadOptions{})
	if err != nil || got == nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got.Vars) != 3500 || len(got.Codes) != 30000 {
		t.Fatalf("Read gave %d vars and %d codes", len(got.Vars), len(got.Codes))
	}
	step("Read")
	check.Check(got, kosac.RefData{})
	step("Check")
	kosac.DeriveDefaults(got, kosac.RefData{})
	step("DeriveDefaults")
	if err := kosac.AssignIDs(got, nil); err != nil {
		t.Fatal(err)
	}
	step("AssignIDs")
	b, err := got.XML()
	if err != nil {
		t.Fatal(err)
	}
	step("XML")
	p, _, err := ddi.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	step("Parse")
	count := 0
	kosac.Walk(p, func(n, _ *ddi.Node, _ string) bool {
		if n.Name == "var" {
			count++
		}
		return true
	})
	if count != 3500 {
		t.Errorf("Parse gave %d vars", count)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("took %v, want under 10s", d)
	}
}

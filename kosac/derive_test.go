package kosac

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
)

// deriveModel holds one var per derivation rule TestDeriveRules checks. File 1
// has the weight; file 2 has none.
func deriveModel() *Codebook {
	ord := func(file, name string, kv ...string) *Var {
		return sv(file, name, append([]string{"MEAS_LEVEL", "서열", "VAR_LABEL", name}, kv...)...)
	}
	nom := func(file, name string, kv ...string) *Var {
		return sv(file, name, append([]string{"MEAS_LEVEL", "명목", "VAR_LABEL", name}, kv...)...)
	}
	mr := func(name, typ, label string) *Var {
		return nom("1", name, "MR_TYPE", typ, "VAR_LABEL", label, "VAR_GROUP", "인식")
	}
	grid := func(name, qtext, label string) *Var {
		return ord("1", name, "Q_TEXT", qtext, "VAR_LABEL", label, "VAR_GROUP", "인식")
	}
	preset := mr("P_1", "보기별 열", "미리 정한 묶음")
	preset.MRGroup, preset.DimMeasure = "X", "ATTR"
	cb := &Codebook{
		Study: rec("WEIGHT_VAR", "WT", "SPLIT_VAR", "SPL", "DATA_NO", "문화-01-01", "FIELD_START", "2024-10-04"),
		Files: []*File{
			{Rec: rec("FILE_NO", "1", "ID_VAR", "IDX", "LINK_VAR", "LNK")},
			{Rec: rec("FILE_NO", "2", "ID_VAR", "ID2")},
		},
		Vars: []*Var{
			sv("1", "IDX", "MEAS_LEVEL", "문자", "DERIVATION", "행 번호"),
			sv("1", "WT", "MEAS_LEVEL", "연속"),
			nom("1", "SPL"),
			sv("1", "DRV", "MEAS_LEVEL", "연속", "DERIVATION", "QA1+QA2"),
			sv("1", "LNK", "MEAS_LEVEL", "문자"),
			sv("1", "TXT", "MEAS_LEVEL", "문자"),
			sv("1", "BORN", "MEAS_LEVEL", "날짜", "PII", "Y"),
			ord("1", "ORD"),
			sv("1", "AGE", "MEAS_LEVEL", "연속"),
			nom("1", "Q-1"), nom("1", "Q_1"), nom("1", "select"), nom("1", strings.Repeat("A", 70)), nom("1", "walla_extra"),
			mr("QA19_1", "보기별 열", "STEM 학위·자격: 대학원"), mr("QA19_2", "보기별 열", "STEM 학위·자격: 학사"),
			mr("QA19_3", "보기별 열", "STEM 학위·자격: 없음"),
			mr("MRX", "보기별 열", "단독"),
			mr("QA22_1", "순위별 열", "분야 (1순위)"), mr("QA22_2", "순위별 열", "분야 (2순위)"),
			preset,
			grid("G1_1", "다음에 관심이 있으십니까? ▶ [새로운 발견]", "새로운 발견"),
			grid("G1_2", "다음에 관심이 있으십니까?  ▶  [발명과 기술]", "발명과 기술"),
			grid("H1_1", "다음에 동의하십니까?", "주제: 하나"),
			grid("H1_2", "다음에 동의하십니까?", "주제: 둘"),
			nom("1", "QA6", "Q_TEXT", "더 가까운 진술은?"), nom("1", "QA7", "Q_TEXT", "더 가까운 진술은?"),
			nom("1", "QA8", "Q_TEXT", "더 가까운 진술은?"), nom("1", "QA9", "Q_TEXT", "더 가까운 진술은?"),
			ord("1", "SC", "REVERSE", "Y"),
			// Not exactly Y, so neither flag holds (C06 and C12 reject both).
			sv("1", "BORN2", "MEAS_LEVEL", "날짜", "PII", "Y "), ord("1", "SC2", "REVERSE", "Y "),
			ord("1", "L1"), ord("1", "L2"), ord("1", "L3"),
			nom("1", "S1"), nom("1", "YN"),
			ord("2", "ORD2"), nom("2", "Q-1"),
		},
	}
	add := func(name string, pairs ...string) {
		for i := 0; i < len(pairs); i += 2 {
			cb.Codes = append(cb.Codes, code("1", name, pairs[i], pairs[i+1], ""))
		}
	}
	likert := []string{"1", "전혀 아니다", "2", "아니다", "3", "보통 이다", "4", "그렇다", "5", "매우 그렇다"}
	add("SC", "1", "낮음", "9", "중간", "10", "높음")
	add("SC2", "1", "낮음", "9", "중간", "10", "높음")
	add("L1", likert...)
	likert[5] = "보통이다"
	add("L2", likert...)
	add("L3", "1", "가", "2", "나", "3", "다", "4", "라", "5", "마")
	add("S1", "7", "유일한 값")
	add("YN", "1", "예", "2", "아니오")
	return cb
}

func TestDeriveRules(t *testing.T) {
	cb := deriveModel()
	ref := RefData{CodeSets: map[string][]StdCode{"CC_YN": {{Code: "1", Label: "예"}, {Code: "2", Label: "아니 오"}}}}
	DeriveDefaults(cb, ref)
	get := map[string]func(*Var) string{
		"Role": func(v *Var) string { return v.Role }, "DimMeasure": func(v *Var) string { return v.DimMeasure },
		"AggRules": func(v *Var) string { return v.AggRules }, "PhysicalName": func(v *Var) string { return v.PhysicalName },
		"IncludeYn": func(v *Var) string { return v.IncludeYn }, "LogicalName": func(v *Var) string { return v.LogicalName },
		"CodeSetID": func(v *Var) string { return v.CodeSetID }, "MRGroup": func(v *Var) string { return v.MRGroup },
		"NoneOption": func(v *Var) string { return v.NoneOption }, "RankNo": func(v *Var) string { return v.RankNo },
		"GridGroup": func(v *Var) string { return v.GridGroup }, "RowLabel": func(v *Var) string { return v.RowLabel },
		"Scale": func(v *Var) string { return v.Scale.Min + ".." + v.Scale.Max + " " + v.Scale.Dir },
	}
	long := strings.Repeat("a", 64)
	for _, c := range []struct{ file, name, field, want string }{
		// Role takes the first rule that fits: an ID var with a derivation is ID.
		{"1", "IDX", "Role", "ID"}, {"1", "WT", "Role", "WEIGHT"}, {"1", "SPL", "Role", "FILTER"},
		{"1", "DRV", "Role", "DERIVED"}, {"1", "LNK", "Role", "LINK"}, {"1", "TXT", "Role", "TEXT"},
		{"1", "BORN", "Role", "ITEM"}, {"1", "ORD", "Role", "ITEM"},

		{"1", "IDX", "DimMeasure", "KEY"}, {"1", "WT", "DimMeasure", "WEIGHT"}, {"1", "LNK", "DimMeasure", "KEY"},
		{"1", "SPL", "DimMeasure", "DIM"}, {"1", "ORD", "DimMeasure", "BOTH"}, {"1", "AGE", "DimMeasure", "MEASURE"},
		{"1", "TXT", "DimMeasure", "ATTR"}, {"1", "BORN", "DimMeasure", "ATTR"}, {"1", "QA19_1", "DimMeasure", "MEASURE"},
		{"1", "QA22_1", "DimMeasure", "DIM"},

		// MR_TYPE decides before DimMeasure; WAVG only where the file has a weight.
		{"1", "QA19_1", "AggRules", "SUM;RATIO_RESP"}, {"1", "QA22_1", "AggRules", "COUNT;RATIO_RANK"},
		{"1", "SPL", "AggRules", "COUNT;RATIO"}, {"1", "ORD", "AggRules", "AVG;COUNT;RATIO;WAVG"},
		{"2", "ORD2", "AggRules", "AVG;COUNT;RATIO"}, {"1", "AGE", "AggRules", "AVG;MIN;MAX;SUM"},
		{"1", "WT", "AggRules", ""}, {"1", "TXT", "AggRules", ""}, {"1", "IDX", "AggRules", ""},

		{"1", "Q-1", "PhysicalName", "q_1"}, {"1", "Q_1", "PhysicalName", "q_1_2"}, {"2", "Q-1", "PhysicalName", "q_1"},
		{"1", "select", "PhysicalName", "q_select"}, {"1", strings.Repeat("A", 70), "PhysicalName", long},
		{"1", "walla_extra", "PhysicalName", "q_walla_extra"}, {"1", "ORD", "PhysicalName", "ord"},

		{"1", "TXT", "IncludeYn", "N"}, {"1", "BORN", "IncludeYn", "N"}, {"1", "ORD", "IncludeYn", "Y"},
		{"1", "QA19_1", "LogicalName", "STEM 학위·자격: 대학원"},

		{"1", "L1", "CodeSetID", "LIKERT5_1"}, {"1", "L2", "CodeSetID", "LIKERT5_1"}, {"1", "L3", "CodeSetID", "LIKERT5_2"},
		{"1", "SC", "CodeSetID", "LIKERT3_1"}, {"1", "S1", "CodeSetID", "CODES1_1"}, {"1", "YN", "CodeSetID", "CC_YN"},
		{"1", "AGE", "CodeSetID", ""},

		{"1", "QA19_1", "MRGroup", "QA19"}, {"1", "QA19_3", "MRGroup", "QA19"}, {"1", "MRX", "MRGroup", "MRX"},
		{"1", "QA22_2", "MRGroup", "QA22"}, {"1", "ORD", "MRGroup", ""},
		{"1", "QA19_1", "NoneOption", "Y"}, {"1", "QA19_2", "NoneOption", "Y"}, {"1", "QA19_3", "NoneOption", "Y"},
		{"1", "MRX", "NoneOption", "N"}, {"1", "QA22_1", "NoneOption", ""},
		{"1", "QA22_1", "RankNo", "1"}, {"1", "QA22_2", "RankNo", "2"}, {"1", "QA19_1", "RankNo", ""},

		// Fill only empty: the preset group and DimMeasure stay.
		{"1", "P_1", "MRGroup", "X"}, {"1", "P_1", "DimMeasure", "ATTR"},

		{"1", "G1_1", "GridGroup", "G1_1"}, {"1", "G1_2", "GridGroup", "G1_1"},
		{"1", "G1_1", "RowLabel", "새로운 발견"}, {"1", "G1_2", "RowLabel", "발명과 기술"},
		// No marker: the whole Q_TEXT is the stem and the label's shared prefix goes.
		{"1", "H1_1", "GridGroup", "H1_1"}, {"1", "H1_2", "GridGroup", "H1_1"},
		{"1", "H1_1", "RowLabel", "하나"}, {"1", "H1_2", "RowLabel", "둘"},
		// Same Q_TEXT, but no prefix: different questions.
		{"1", "QA6", "GridGroup", ""}, {"1", "QA9", "GridGroup", ""}, {"1", "QA6", "RowLabel", ""},
		{"1", "QA19_1", "GridGroup", ""}, {"1", "ORD", "GridGroup", ""},

		{"1", "SC", "Scale", "1..10 NEG"}, {"1", "L1", "Scale", "1..5 POS"}, {"1", "S1", "Scale", ".. "},
		{"1", "BORN2", "IncludeYn", "Y"}, {"1", "SC2", "Scale", "1..10 POS"},
	} {
		v := cb.Var(c.file, c.name)
		if v == nil {
			t.Fatalf("no var %s/%s", c.file, c.name)
		}
		if got := get[c.field](v); got != c.want {
			t.Errorf("F%s/%s %s = %q, want %q", c.file, c.name, c.field, got, c.want)
		}
	}
}

func TestDeriveAdmin(t *testing.T) {
	cb := deriveModel()
	DeriveDefaults(cb, RefData{})
	if got, want := cb.Admin, (StudyAdmin{CategoryCode: "문화", Round: Round{Year: "2024", WaveType: "main"}}); got != want {
		t.Errorf("Admin = %+v, want %+v", got, want)
	}
	if cb.Files[0].SensLevel != "3" || cb.Files[1].SensLevel != "3" {
		t.Errorf("SensLevel = %q %q", cb.Files[0].SensLevel, cb.Files[1].SensLevel)
	}

	cb = deriveModel()
	cb.Admin = StudyAdmin{CategoryCode: "교육", Round: Round{WaveType: "pre", WaveNo: "1"}, WallaID: "w"}
	cb.Files[0].SensLevel = "1"
	DeriveDefaults(cb, RefData{})
	if got, want := cb.Admin, (StudyAdmin{CategoryCode: "교육", Round: Round{Year: "2024", WaveType: "pre", WaveNo: "1"}, WallaID: "w"}); got != want {
		t.Errorf("preset Admin = %+v, want %+v", got, want)
	}
	if cb.Files[0].SensLevel != "1" || cb.Files[1].SensLevel != "3" {
		t.Errorf("preset SensLevel = %q %q", cb.Files[0].SensLevel, cb.Files[1].SensLevel)
	}
}

// snapshot prints everything DeriveDefaults may change; fmt sorts map keys.
func snapshot(cb *Codebook) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%+v\n", cb.Admin)
	for _, f := range cb.Files {
		fmt.Fprintf(&b, "%+v %s\n", f.Rec, f.SensLevel)
	}
	for _, v := range cb.Vars {
		fmt.Fprintf(&b, "%+v %+v\n", v.Rec, v.Analysis)
	}
	for _, c := range cb.Codes {
		fmt.Fprintf(&b, "%+v\n", *c)
	}
	return b.String()
}

func TestDeriveMissingCodes(t *testing.T) {
	n1 := sv("1", "N1", "MEAS_LEVEL", "명목", "VAR_LABEL", "명목")
	n2 := sv("1", "N2", "MEAS_LEVEL", "서열", "VAR_LABEL", "서열")
	cont := sv("1", "C", "MEAS_LEVEL", "연속", "VAR_LABEL", "연속")
	txt := sv("1", "T", "MEAS_LEVEL", "문자", "VAR_LABEL", "문자")
	day := sv("1", "D", "MEAS_LEVEL", "날짜", "VAR_LABEL", "날짜")
	sys := sv("1", "walla_row_no", "MEAS_LEVEL", "연속", "VAR_LABEL", "행")
	relabel := code("1", "N2", "-7", "무응답", "")
	relabel.Rec["CODE_LABEL"] = Field{Value: "무응답", Loc: Loc{SheetCodes, "D9"}}
	cb := &Codebook{
		Study: rec("TITLE", "t"),
		Files: []*File{{Rec: rec("FILE_NO", "1")}},
		Vars:  []*Var{n1, n2, cont, txt, day, sys},
		Codes: []*Code{
			code("1", "N1", "1", "예", ""), code("1", "N1", "2", "아니오", ""), code("1", "N1", "-8", "무응답(잘못된 라벨)", ""),
			code("1", "N2", "1", "낮음", ""), relabel, code("1", "N2", "-6", "이전 게시본 값", ""),
			// A reviewer's type on a row outside the accepted labels stays.
			code("1", "N2", "-8", "엉뚱한 라벨", "NA"),
		},
	}
	DeriveDefaults(cb, RefData{})

	rows := func(v *Var) string {
		var out []string
		for _, c := range cb.CodesOf(v) {
			out = append(out, c.Rec.Get("CODE")+"/"+c.Rec.Get("CODE_LABEL")+"/"+c.Type)
		}
		return strings.Join(out, " ")
	}
	for _, c := range []struct {
		v    *Var
		want string
	}{
		// A label outside the accepted ones stays, typed MISSING; -7 and -9 follow.
		{n1, "1/예/VALID 2/아니오/VALID -8/무응답(잘못된 라벨)/MISSING -7/모르겠다/DK -9/유효하지 않음/MISSING"},
		{n2, "1/낮음/VALID -7/모르겠다/DK -6/이전 게시본 값/MISSING -8/엉뚱한 라벨/NA -9/유효하지 않음/MISSING"},
		{cont, "-8/묻지 않음/MISSING -7/모르겠다/DK -9/유효하지 않음/MISSING"},
		{txt, ""}, {day, ""}, {sys, ""},
	} {
		if got := rows(c.v); got != c.want {
			t.Errorf("%s codes = %q\nwant %q", c.v.Rec.Get("VAR_NAME"), got, c.want)
		}
	}
	if relabel.Rec["CODE_LABEL"].Loc.Cell != "D9" {
		t.Errorf("relabelled row lost its cell: %+v", relabel.Rec["CODE_LABEL"])
	}
	for _, c := range cb.Codes[len(cb.Codes)-3:] {
		if c.Rec["CODE"].Loc != (Loc{}) || len(c.Rec) != 4 {
			t.Errorf("added row %+v: want 4 keys and no Loc", c.Rec)
		}
	}

	if err := AssignIDs(cb, nil); err != nil {
		t.Fatal(err)
	}
	root := tree(t, cb)
	for _, c := range []struct {
		v    *Var
		want string
	}{{n1, "-8;-7;-9"}, {n2, "-8;-7;-9;-6"}, {txt, ""}} {
		got := ""
		if mc := at(withAttr(all(at(root, "dataDscr"), "var"), "ID", c.v.ID), "kosac:missingCodes"); mc != nil {
			got = mc.Text
		}
		if got != c.want {
			t.Errorf("%s missingCodes = %q, want %q", c.v.Rec.Get("VAR_NAME"), got, c.want)
		}
	}

	before := snapshot(cb)
	DeriveDefaults(cb, RefData{})
	if after := snapshot(cb); after != before {
		t.Errorf("second DeriveDefaults changed the model:\n%s\nvs\n%s", before, after)
	}
}

func TestDeriveGridPreset(t *testing.T) {
	row := func(name, row string) *Var {
		return sv("1", name, "MEAS_LEVEL", "서열", "VAR_GROUP", "인식", "VAR_LABEL", row,
			"Q_TEXT", "다음에 동의하십니까? ▶ ["+row+"]")
	}
	for _, c := range []struct {
		preset, want1, want2 string
	}{
		// Filling Q5_2 with "Q5_1" would leave two grids of one member each.
		{"W", "W", ""},
		{"", "Q5_1", "Q5_1"},
	} {
		q1, q2 := row("Q5_1", "가"), row("Q5_2", "나")
		q1.GridGroup = c.preset
		cb := &Codebook{Files: []*File{{Rec: rec("FILE_NO", "1")}}, Vars: []*Var{q1, q2}}
		for range 2 {
			DeriveDefaults(cb, RefData{})
			if q1.GridGroup != c.want1 || q2.GridGroup != c.want2 {
				t.Errorf("preset %q: GridGroup = %q, %q, want %q, %q", c.preset, q1.GridGroup, q2.GridGroup, c.want1, c.want2)
			}
		}
	}
}

// Every walla name has the prefix "f", so in a native codebook two same-worded
// questions on one page would pass for a grid.
func TestDeriveNativeSkipsNameGrids(t *testing.T) {
	for _, native := range []bool{false, true} {
		q := func(name string) *Var {
			return sv("1", name, "MEAS_LEVEL", "명목", "VAR_GROUP", "1쪽", "VAR_LABEL", name, "Q_TEXT", "만족하십니까?")
		}
		a, b := q("f_a"), q("f_b")
		cb := &Codebook{Native: native, Files: []*File{{Rec: rec("FILE_NO", "1")}}, Vars: []*Var{a, b}}
		DeriveDefaults(cb, RefData{})
		want := "f_a"
		if native {
			want = ""
		}
		if a.GridGroup != want || b.GridGroup != want {
			t.Errorf("Native=%v: GridGroup = %q, %q, want %q", native, a.GridGroup, b.GridGroup, want)
		}
	}
}

func TestDeriveMissingLabelsFromRefData(t *testing.T) {
	v := sv("1", "N", "MEAS_LEVEL", "명목")
	cb := &Codebook{Files: []*File{{Rec: rec("FILE_NO", "1")}}, Vars: []*Var{v},
		Codes: []*Code{code("1", "N", "-7", "무응답", "")}}
	DeriveDefaults(cb, RefData{MissingLabels: map[string][]string{"-7": {"응답 거절", "무응답"}}})
	var got []string
	for _, c := range cb.CodesOf(v) {
		got = append(got, c.Rec.Get("CODE")+"/"+c.Rec.Get("CODE_LABEL")+"/"+c.Type)
	}
	// RefData gives -7 only; the others fall back to the defaults.
	want := []string{"-7/응답 거절/DK", "-8/묻지 않음/MISSING", "-9/유효하지 않음/MISSING"}
	if !slices.Equal(got, want) {
		t.Errorf("codes = %q, want %q", got, want)
	}
}

func suggestRef() RefData {
	return RefData{
		StdItems: map[string]StdItem{
			"PERSON.GENDER": {StdID: "PERSON.GENDER", NameKo: "성별", PhysicalName: "gender_cd", MeasLevel: "nominal",
				CodeSetID: "CC_GENDER", Synonyms: []string{"남녀", "성"}, Active: "Y"},
			"SCI.TRUST": {StdID: "SCI.TRUST", NameKo: "과학기술 신뢰도", PhysicalName: "sci_trust", MeasLevel: "ordinal", Active: "Y"},
		},
		CodeSetDefs: map[string]CodeSetDef{"CC_GENDER": {CodeSetID: "CC_GENDER", NameKo: "성별", PhysicalName: "gender_cd",
			DimTable: "dw.t_common_code", KeyColumn: "standard_code", CodeSystem: "KOSAC"}},
		CodeSets: map[string][]StdCode{"CC_GENDER": {
			{CodeSetID: "CC_GENDER", Code: "M", Label: "남", AltLabels: []string{"남성", "남자", "남학생"}, Sort: "1"},
			{CodeSetID: "CC_GENDER", Code: "F", Label: "여", AltLabels: []string{"여성", "여자", "여학생"}, Sort: "2"},
		}},
	}
}

func TestSuggest(t *testing.T) {
	cb := &Codebook{
		Files: []*File{{Rec: rec("FILE_NO", "1")}},
		Vars: []*Var{
			sv("1", "A", "VAR_LABEL", "성별", "Q_TEXT", "귀하의 성별은 어떻게 되십니까?", "MEAS_LEVEL", "명목"),
			sv("1", "B", "VAR_LABEL", "응답자 구분", "MEAS_LEVEL", "명목"),
			sv("1", "C", "VAR_LABEL", "과학기술의 신뢰도", "MEAS_LEVEL", "서열"),
			sv("1", "D", "VAR_LABEL", "과학기술 신뢰도 평가", "MEAS_LEVEL", "서열"),
			sv("1", "E", "VAR_LABEL", "학생 성별", "STD_ID", "PERSON.GENDER", "MEAS_LEVEL", "명목"),
		},
		Codes: []*Code{code("1", "B", "1", "남성", ""), code("1", "B", "2", "여성", ""),
			code("1", "E", "1", "남 자", ""), code("1", "E", "2", "기타", "")},
	}
	got := DeriveDefaults(cb, suggestRef())
	var lines []string
	for _, s := range got {
		lines = append(lines, fmt.Sprintf("%s %s/%s %s %.2f %s %v", s.Kind, s.FileNo, s.VarName, s.Value, s.Confidence, s.Level, s.CodeMap))
	}
	want := []string{
		"STD_ID 1/A PERSON.GENDER 1.00 high map[]",
		"FK 1/B CC_GENDER 1.00 high map[1:M 2:F]",
		// A particle comes off before the comparison.
		"STD_ID 1/C SCI.TRUST 1.00 high map[]",
		"STD_ID 1/D SCI.TRUST 0.67 check map[]",
		// A STD_ID gives its code set outright; codes map by label.
		"FK 1/E CC_GENDER 1.00 high map[1:M]",
	}
	if !slices.Equal(lines, want) {
		t.Errorf("suggestions:\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
	a := cb.Var("1", "A")
	if a.Rec.Get("STD_ID") != "" || a.FK != nil || cb.Var("1", "B").FK != nil || a.PhysicalName != "a" {
		t.Errorf("a suggestion was applied: STD_ID %q FK %v physicalName %q", a.Rec.Get("STD_ID"), a.FK, a.PhysicalName)
	}
	// The dictionary's physical name applies to a var that has the STD_ID.
	if e := cb.Var("1", "E"); e.PhysicalName != "gender_cd" {
		t.Errorf("E physicalName = %q", e.PhysicalName)
	}
}

func TestTokens(t *testing.T) {
	got := slices.Sorted(maps.Keys(tokens("귀하의 성별은? 지역에서 Q1·나이로 학교 도")))
	want := []string{"q1", "귀하", "나이", "도", "성별", "지역", "학교"}
	if !slices.Equal(got, want) {
		t.Errorf("tokens = %q, want %q", got, want)
	}
}

func TestApplyFK(t *testing.T) {
	gender := suggestRef().CodeSetDefs["CC_GENDER"]
	sido := CodeSetDef{CodeSetID: "CC_SIDO", PhysicalName: "sido_cd", DimTable: "dw.t_common_code", KeyColumn: "standard_code"}
	sq1 := sv("1", "SQ1", "MEAS_LEVEL", "명목")
	sq1.PhysicalName, sq1.DimMeasure = "sq1", "BOTH"
	sq1.FK = &FK{CodeSet: "CC_GENDER"}
	a, b, c := sv("1", "A", "MEAS_LEVEL", "명목"), sv("1", "B", "MEAS_LEVEL", "명목"), sv("2", "C", "MEAS_LEVEL", "명목")
	for _, v := range []*Var{a, b, c} {
		v.FK = &FK{CodeSet: "CC_SIDO", DimTable: "dw.custom"}
	}
	cb := &Codebook{Vars: []*Var{sq1, a, b, c}}
	for _, x := range []struct {
		v   *Var
		def CodeSetDef
	}{{sq1, gender}, {a, sido}, {b, sido}, {c, sido}, {a, sido}} {
		if err := ApplyFK(cb, x.v, x.def); err != nil {
			t.Fatal(err)
		}
	}
	for _, x := range []struct {
		v                             *Var
		physical, dimTable, keyColumn string
	}{
		{sq1, "gender_cd", "dw.t_common_code", "standard_code"},
		// Applying again keeps the name the var already holds.
		{a, "sido_cd", "dw.custom", "standard_code"},
		{b, "sido_cd_2", "dw.custom", "standard_code"},
		{c, "sido_cd", "dw.custom", "standard_code"},
	} {
		if x.v.PhysicalName != x.physical || x.v.DimMeasure != "DIM" || x.v.FK.DimTable != x.dimTable || x.v.FK.KeyColumn != x.keyColumn {
			t.Errorf("%s: physicalName %q dimMeasure %q FK %+v", x.v.Rec.Get("VAR_NAME"), x.v.PhysicalName, x.v.DimMeasure, *x.v.FK)
		}
	}
	if err := ApplyFK(cb, sv("1", "N", "MEAS_LEVEL", "명목"), gender); err == nil {
		t.Error("nil FK: no error")
	}
	if err := ApplyFK(cb, sq1, sido); err == nil {
		t.Error("FK of another code set: no error")
	}
}

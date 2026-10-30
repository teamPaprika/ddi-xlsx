package kosac_test

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/check"
)

// fromWalla converts testdata/{dir}/{file}; any issue fails the test.
func fromWalla(t *testing.T, dir, file string, opt kosac.WallaOptions) *kosac.Codebook {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "testdata", dir, file))
	if err != nil {
		t.Fatal(err)
	}
	cb, is, err := kosac.FromWallaDDI(b, opt)
	if err != nil || len(is) != 0 {
		t.Fatalf("FromWallaDDI(%s/%s): %v %v", dir, file, is, err)
	}
	return cb
}

// fromBoth converts dir's survey.xml and checks survey.json gives the same model.
func fromBoth(t *testing.T, dir string) *kosac.Codebook {
	t.Helper()
	cb := fromWalla(t, dir, "survey.xml", kosac.WallaOptions{})
	if j := fromWalla(t, dir, "survey.json", kosac.WallaOptions{}); !reflect.DeepEqual(cb, j) {
		t.Errorf("%s: survey.json gives another model than survey.xml", dir)
	}
	return cb
}

type wallaGroups struct{ mr, grid, rank []string }

func groupsOf(cb *kosac.Codebook) wallaGroups {
	mr, _ := groupKeys(cb, mrOf("보기별 열"))
	grid, _ := groupKeys(cb, func(v *kosac.Var) string { return v.GridGroup })
	rank, _ := groupKeys(cb, mrOf("순위별 열"))
	return wallaGroups{mr, grid, rank}
}

// textFormats counts the vars of each TextFormat.
func textFormats(cb *kosac.Codebook) map[string]int {
	out := map[string]int{}
	for _, v := range cb.Vars {
		if v.TextFormat != "" {
			out[v.TextFormat]++
		}
	}
	return out
}

// wallaKeys are the values FromWallaDDI fills that DeriveDefaults must keep.
func wallaKeys(cb *kosac.Codebook) []string {
	var out []string
	for _, v := range cb.Vars {
		out = append(out, fmt.Sprint(v.Rec.Get("VAR_NAME"), v.Rec.Get("MR_TYPE"), v.MRGroup, v.GridGroup, v.RankNo, v.ParentVar, v.TextFormat))
	}
	return out
}

// draft does with a native draft what datajam does before export: fills
// VAR_GROUP with the page label, derives and numbers. It returns the XML,
// which must pass ValidateXML.
func draft(t *testing.T, cb *kosac.Codebook) []byte {
	t.Helper()
	for _, v := range cb.Vars {
		v.Rec.Set("VAR_GROUP", "기본")
	}
	kosac.DeriveDefaults(cb, kosac.RefData{})
	if err := kosac.AssignIDs(cb, nil); err != nil {
		t.Fatal(err)
	}
	x, err := cb.XML()
	if err != nil {
		t.Fatal(err)
	}
	if is := check.ValidateXML(x, xmllint(t)); len(is) != 0 {
		t.Errorf("ValidateXML: %d issues, first %+v", len(is), is[0])
	}
	return x
}

// codesTyped counts the codes valued code with walla's label for a value the
// published form no longer has, failing on one whose Type is not typ. A
// relabelled code drops out of the count.
func codesTyped(t *testing.T, cb *kosac.Codebook, code, typ string) int {
	t.Helper()
	n := 0
	for _, c := range cb.Codes {
		if c.Rec.Get("CODE") != code || c.Rec.Get("CODE_LABEL") != "현재 게시본에 없는 값" {
			continue
		}
		if c.Type != typ {
			t.Errorf("%s %s Type %q, want %q", c.Rec.Get("VAR_NAME"), code, c.Type, typ)
		}
		n++
	}
	return n
}

func TestWallaSynthetic(t *testing.T) {
	const dir = "walla_synthetic"
	cb := fromBoth(t, dir)

	if len(cb.Vars) != 46 {
		t.Errorf("%d survey vars, want 46", len(cb.Vars))
	}
	for _, v := range cb.Vars {
		if kosac.IsSystemVar(v.Rec.Get("VAR_NAME")) {
			t.Errorf("system var %s imported", v.Rec.Get("VAR_NAME"))
		}
		if v.Rec.Get("FILE_NO") != "1" {
			t.Errorf("%s FILE_NO %q", v.Rec.Get("VAR_NAME"), v.Rec.Get("FILE_NO"))
		}
	}
	if len(cb.Files) != 1 {
		t.Fatalf("%d files, want 1", len(cb.Files))
	}
	f := cb.Files[0].Rec
	for k, want := range map[string]string{"FILE_NO": "1", "FILE_NAME": "survey.csv", "FILE_FORMAT": "CSV",
		"ID_VAR": "walla_response_id", "VALID_N": ""} {
		if got := f.Get(k); got != want {
			t.Errorf("02 %s = %q, want %q", k, got, want)
		}
	}
	if cb.Admin.WallaID != "SampleForm0000000001" || cb.Package != nil || !cb.Native {
		t.Errorf("WallaID %q, Package %+v, Native %v", cb.Admin.WallaID, cb.Package, cb.Native)
	}
	if got := cb.Study.Get("TITLE"); got != "합성 설문(모든 필드 타입)" {
		t.Errorf("TITLE %q", got)
	}

	_, mrMembers := groupKeys(cb, mrOf("보기별 열"))
	_, gridMembers := groupKeys(cb, func(v *kosac.Var) string { return v.GridGroup })
	_, rankMembers := groupKeys(cb, mrOf("순위별 열"))
	g := groupsOf(cb)
	want := wallaGroups{[]string{"f_Rk8Mn3Tq1b"}, []string{"f_Ux1Dz4Yf2g", "f_Vy6Ea7Zg3h", "f_Xa3Gc5Bi6j"}, []string{"f_Wz9Fb2Ah4i"}}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("groups %+v, want %+v", g, want)
	}
	names := func(vs []*kosac.Var) []string {
		var out []string
		for _, v := range vs {
			out = append(out, v.Rec.Get("VAR_NAME"))
		}
		return out
	}
	if got, want := names(mrMembers["f_Rk8Mn3Tq1b"]), []string{"f_Rk8Mn3Tq1b_cD7eF9gH2", "f_Rk8Mn3Tq1b_other",
		"f_Rk8Mn3Tq1b_custom", "f_Rk8Mn3Tq1b__custom"}; !slices.Equal(got, want) {
		t.Errorf("MR members %q, want %q", got, want)
	}
	for k, n := range map[string]int{"f_Ux1Dz4Yf2g": 2, "f_Vy6Ea7Zg3h": 6, "f_Xa3Gc5Bi6j": 4} {
		if len(gridMembers[k]) != n {
			t.Errorf("grid %s members %q, want %d", k, names(gridMembers[k]), n)
		}
	}
	for i, v := range rankMembers["f_Wz9Fb2Ah4i"] {
		if want := fmt.Sprintf("f_Wz9Fb2Ah4i__rank%d", i+1); v.Rec.Get("VAR_NAME") != want || v.RankNo != fmt.Sprint(i+1) || v.GridGroup != "" {
			t.Errorf("rank member %d: %s RankNo %q GridGroup %q", i, v.Rec.Get("VAR_NAME"), v.RankNo, v.GridGroup)
		}
	}

	for name, want := range map[string]string{
		"f_6558d0ad57__other":   "f_6558d0ad57",
		"f_Ux1Dz4Yf2g__other":   "", // no var f_Ux1Dz4Yf2g
		"f_Rk8Mn3Tq1b_other":    "", // "other" is an option ID
		"f_Rk8Mn3Tq1b__custom":  "",
		"f_Dg5Mi6Ho7p__lat":     "",
		"f_Gj6Pl7Kr8s_zip_code": "",
	} {
		if got := cb.Var("1", name).ParentVar; got != want {
			t.Errorf("%s ParentVar = %q, want %q", name, got, want)
		}
	}
	// The sample's own record of what each column holds is the oracle.
	tsv, err := os.ReadFile(filepath.Join("..", "testdata", dir, "variables.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	holds := map[string]string{`"kind":"otherText"`: "TEXT", `"kind":"otherList"`: "JSON_ARR",
		`"kind":"otherGrid"`: "JSON_OBJ", `"kind":"otherKeys"`: "JSON_OBJ"}
	for _, line := range strings.Split(strings.TrimSpace(string(tsv)), "\n")[1:] {
		col := strings.Split(line, "\t")
		v := cb.Var("1", col[0])
		if v == nil {
			continue // a system var
		}
		want := ""
		for kind, f := range holds {
			if strings.Contains(col[2], kind) {
				want = f
			}
		}
		if v.TextFormat != want {
			t.Errorf("%s TextFormat %q, want %q", col[0], v.TextFormat, want)
		}
	}
	if got, want := textFormats(cb), map[string]int{"TEXT": 4, "JSON_ARR": 1, "JSON_OBJ": 4}; !maps.Equal(got, want) {
		t.Errorf("text formats %v, want %v", got, want)
	}
	for name, want := range map[string][2]string{ // MEAS_LEVEL, Q_TEXT
		"f_6558d0ad57":           {"명목", "성별을(를) 응답해 주세요."},
		"f_Ux1Dz4Yf2g_Wn5tYb2Lc": {"명목", "항목별 만족도을(를) 응답해 주세요."}, // through var@qstn
		"f_Zc1Ie2Dk3l":           {"문자", "이름을(를) 응답해 주세요."},
		"f_Yb7Hd8Cj9k":           {"연속", "나이을(를) 응답해 주세요."},
		"f_Be8Kg9Fm1n":           {"날짜", "방문일을(를) 응답해 주세요."},
		"f_Cf2Lh3Gn4o":           {"문자", "방문 시각을(를) 응답해 주세요."}, // a time is no date
		"h_utmSrc001":            {"문자", "utm_source"},         // no question: the label
	} {
		v := cb.Var("1", name)
		if got := [2]string{v.Rec.Get("MEAS_LEVEL"), v.Rec.Get("Q_TEXT")}; got != want {
			t.Errorf("%s MEAS_LEVEL, Q_TEXT = %q, want %q", name, got, want)
		}
	}
	if got := cb.Var("1", "f_Rk8Mn3Tq1b__other").Rec.Get("NOTE"); got != "현재 보기에 없는 값을 JSON 배열로 담습니다. 예: [\"값\"]\n직접 입력을 받는 문항이라, 이전 게시본의 선택지 값도 '기타'로 분류됩니다." {
		t.Errorf("NOTE %q", got)
	}

	if n := codesTyped(t, cb, "-6", "MISSING"); n != 7 {
		t.Errorf("%d -6 codes, want 7", n)
	}
	var minus6 []string
	for _, c := range cb.Codes {
		if c.Rec.Get("CODE") == "-6" {
			minus6 = append(minus6, c.Rec.Get("VAR_NAME"))
		}
	}

	keys := wallaKeys(cb)
	x := draft(t, cb)
	if after := wallaKeys(cb); !slices.Equal(keys, after) {
		t.Errorf("DeriveDefaults changed walla's values:\n%q\nvs\n%q", keys, after)
	}
	root, _, err := ddi.Parse(x)
	if err != nil {
		t.Fatal(err)
	}
	missing := map[string]string{}
	kosac.Walk(root, func(n, _ *ddi.Node, _ string) bool {
		if n.Name != "var" {
			return true
		}
		name, _ := n.Attr("name")
		for _, c := range n.Children {
			if c.Name == "kosac:missingCodes" {
				missing[name] = c.Text
			}
		}
		return false
	})
	for _, name := range minus6 {
		if missing[name] != "-8;-7;-9;-6" {
			t.Errorf("%s missingCodes %q", name, missing[name])
		}
	}

	hinted := fromWalla(t, dir, "survey.xml", kosac.WallaOptions{MeasHints: map[string]string{"f_Yb7Hd8Cj9k": "서열"}})
	if got := hinted.Var("1", "f_Yb7Hd8Cj9k").Rec.Get("MEAS_LEVEL"); got != "서열" {
		t.Errorf("hinted MEAS_LEVEL %q", got)
	}
	b, _ := os.ReadFile(filepath.Join("..", "testdata", dir, "survey.xml"))
	if _, _, err := kosac.FromWallaDDI(b, kosac.WallaOptions{MeasHints: map[string]string{"f_Yb7Hd8Cj9k": "ordinal"}}); err == nil {
		t.Error("a hint outside MEAS_LEVEL's list was accepted")
	}
}

// The KOFAC surveys exported with the current naming rules. They count the
// same as their legacy exports, which checks one mode against the other.
var kofac = []struct {
	year, legacy   string
	vars           int
	missing        int // codes of values the published form no longer has
	mr, grid, rank int
}{
	{"2024", "kofac_2024_public", 286, 156, 3, 22, 4},
	{"2025", "kofac_2025_edu", 634, 351, 14, 67, 0},
}

func TestWallaKofacB01(t *testing.T) {
	for _, c := range kofac {
		t.Run(c.year, func(t *testing.T) {
			cb := fromBoth(t, filepath.Join("kofac_b01", c.year))
			if len(cb.Vars) != c.vars {
				t.Errorf("%d survey vars, want %d", len(cb.Vars), c.vars)
			}
			g := groupsOf(cb)
			if len(g.mr) != c.mr || len(g.grid) != c.grid || len(g.rank) != c.rank {
				t.Errorf("MR %d, grid %d, rank %d groups, want %d, %d, %d", len(g.mr), len(g.grid), len(g.rank), c.mr, c.grid, c.rank)
			}
			if n := codesTyped(t, cb, "-6", "MISSING"); n != c.missing {
				t.Errorf("%d -6 codes, want %d", n, c.missing)
			}
			draft(t, cb)
		})
	}
}

// A native draft saved without a Package stays native through Parse; else
// deriving again groups walla's same-worded questions into grids by name.
func TestWallaNativeSurvivesParse(t *testing.T) {
	for _, c := range kofac {
		t.Run(c.year, func(t *testing.T) {
			x := draft(t, fromWalla(t, filepath.Join("kofac_b01", c.year), "survey.xml", kosac.WallaOptions{}))
			p, err := kosac.Parse(x)
			if err != nil {
				t.Fatal(err)
			}
			if !p.Native || p.Package != nil {
				t.Errorf("Parse: Native %v, Package %+v", p.Native, p.Package)
			}
			if again, err := p.XML(); err != nil || !bytes.Equal(again, x) {
				t.Errorf("Parse then XML is not a fixed point: %v", err)
			}
			gridded := func() int {
				n := 0
				for _, v := range p.Vars {
					if v.GridGroup != "" {
						n++
					}
				}
				return n
			}
			before := gridded()
			kosac.DeriveDefaults(p, kosac.RefData{})
			if n := gridded() - before; n != 0 {
				t.Errorf("DeriveDefaults after Parse put %d more vars in grids", n)
			}
		})
	}
}

func TestWallaLegacy(t *testing.T) {
	for _, c := range kofac {
		t.Run(c.legacy, func(t *testing.T) {
			cb := fromWalla(t, c.legacy, "survey.xml", kosac.WallaOptions{})
			if len(cb.Vars) != c.vars {
				t.Errorf("%d survey vars, want %d", len(cb.Vars), c.vars)
			}
			for _, name := range []string{"response_id", "submitted_at", "customer_key"} {
				if cb.Var("1", name) != nil {
					t.Errorf("system var %s imported", name)
				}
			}
			g := groupsOf(cb)
			if len(g.mr) != c.mr || len(g.grid) != c.grid || len(g.rank) != c.rank {
				t.Errorf("MR %d, grid %d, rank %d groups, want %d, %d, %d", len(g.mr), len(g.grid), len(g.rank), c.mr, c.grid, c.rank)
			}
			if c.rank > 0 {
				if want := []string{"Q30", "Q38", "Q39", "Q46"}; !slices.Equal(g.rank, want) {
					t.Errorf("rank groups %q, want %q", g.rank, want)
				}
				for i, name := range []string{"Q30_rank1", "Q30_rank2"} {
					v := cb.Var("1", name)
					if v.Rec.Get("MR_TYPE") != "순위별 열" || v.RankNo != fmt.Sprint(i+1) || v.GridGroup != "" {
						t.Errorf("%s MR_TYPE %q RankNo %q GridGroup %q", name, v.Rec.Get("MR_TYPE"), v.RankNo, v.GridGroup)
					}
				}
			}
			if v := cb.Var("1", "Q1_other"); v.ParentVar != "Q1" || v.TextFormat != "TEXT" {
				t.Errorf("Q1_other ParentVar %q TextFormat %q", v.ParentVar, v.TextFormat)
			}
			current := fromWalla(t, filepath.Join("kofac_b01", c.year), "survey.xml", kosac.WallaOptions{})
			if got, want := textFormats(cb), textFormats(current); !maps.Equal(got, want) {
				t.Errorf("text formats %v, the current export's %v", got, want)
			}
			if f := cb.Files[0].Rec; f.Get("ID_VAR") != "walla_response_id" {
				t.Errorf("ID_VAR %q", f.Get("ID_VAR"))
			}

			// Left untyped, so DeriveDefaults keeps the label and types it MISSING
			// rather than DK.
			if n := codesTyped(t, cb, "-7", ""); n != c.missing {
				t.Errorf("%d -7 codes, want %d", n, c.missing)
			}
			draft(t, cb)
			if n := codesTyped(t, cb, "-7", "MISSING"); n != c.missing {
				t.Errorf("after DeriveDefaults %d -7 codes, want %d", n, c.missing)
			}
		})
	}
}

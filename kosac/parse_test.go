package kosac

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
)

// unverifiedModel is fullModel with every cell "확인 불가" except the keys that
// identify a row. The reason names the cell, so a value restored on the wrong
// file or var shows, and the spelling differs from the restored one.
func unverifiedModel() *Codebook {
	cb := fullModel()
	for _, k := range StudyKeys {
		cb.Study.Set(k, "확인불가："+k)
	}
	for _, f := range cb.Files {
		for _, k := range FileKeys[1:] { // not FILE_NO
			f.Rec.Set(k, "확인불가：F"+f.Rec.Get("FILE_NO")+"."+k)
		}
	}
	for _, v := range cb.Vars {
		for _, k := range VarKeys[2:] { // not FILE_NO, VAR_NAME
			v.Rec.Set(k, "확인불가："+v.ID+"."+k)
		}
	}
	return cb
}

func mustXML(t *testing.T, cb *Codebook) []byte {
	t.Helper()
	root, err := cb.profileTree()
	if err != nil {
		t.Fatal(err)
	}
	return ddi.WriteXML(root)
}

func mustJSON(t *testing.T, cb *Codebook) []byte {
	t.Helper()
	root, err := cb.profileTree()
	if err != nil {
		t.Fatal(err)
	}
	return ddi.WriteJSON(root)
}

func mustParse(t *testing.T, doc []byte) *Codebook {
	t.Helper()
	cb, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	return cb
}

// sameBytes reports the first line where got leaves want.
func sameBytes(t *testing.T, what string, want, got []byte) {
	t.Helper()
	if bytes.Equal(want, got) {
		return
	}
	a, b := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			t.Errorf("%s differs at line %d:\nwant %s\n got %s", what, i+1, a[i], b[i])
			return
		}
	}
	t.Errorf("%s: %d lines, want %d", what, len(b), len(a))
}

func TestLegacyProfileFixedPoint(t *testing.T) {
	noPkg, noVersion, nativeNoPkg, nativePkg := smallModel(), smallModel(), smallModel(), smallModel()
	noPkg.Package = nil
	noVersion.Package.CodebookVersion = ""
	nativeNoPkg.Native, nativeNoPkg.Package = true, nil
	nativePkg.Native, nativePkg.Package.Source.Kind = true, ""
	for _, m := range []struct {
		name string
		cb   *Codebook
	}{{"small", smallModel()}, {"full", fullModel()}, {"unverified", unverifiedModel()},
		{"no package", noPkg}, {"no version", noVersion}, {"native, no package", nativeNoPkg}, {"native package", nativePkg}} {
		x, j := mustXML(t, m.cb), mustJSON(t, m.cb)
		for _, in := range []struct {
			format string
			doc    []byte
		}{{"XML", x}, {"JSON", j}} {
			t.Run(m.name+"/"+in.format, func(t *testing.T) {
				cb := mustParse(t, in.doc)
				sameBytes(t, "XML", x, mustXML(t, cb))
				sameBytes(t, "JSON", j, mustJSON(t, cb))
				if cb.Native != m.cb.Native || (cb.Package == nil) != (m.cb.Package == nil) {
					t.Errorf("Native %v, Package %v; want %v, %v", cb.Native, cb.Package != nil, m.cb.Native, m.cb.Package != nil)
				}
			})
		}
	}

	// The fixed point covers assembly's tables only if the full model writes
	// every name in them.
	seen := map[string]bool{}
	walk(tree(t, fullModel()), func(n *ddi.Node) {
		seen[n.Name] = true
		for _, a := range n.Attrs {
			seen[n.Name+"@"+a.Name] = true
		}
	})
	for parent, kids := range KosacChildren {
		for _, k := range kids {
			if !seen["kosac:"+k] {
				t.Errorf("the full model writes no kosac:%s (under %s)", k, parent)
			}
		}
	}
	for parent, attrs := range KosacAttrs {
		for _, a := range attrs {
			if !seen[parent+"@kosac:"+a] {
				t.Errorf("the full model writes no %s@kosac:%s", parent, a)
			}
		}
	}
}

// typedKeys are the slots with a format, which hold no "확인 불가" value: it
// comes back from kosac:unverified in one spelling.
var typedKeys = []string{
	"FIELD_START", "FIELD_END", "TARGET_N", "VALID_N", "CONTACT_N", "EXCLUDE_N", "SCHOOL_CODE_YEAR", "BIZ_YEAR",
	"RULE_VERSION", "SURVEY_TYPE", "DATA_NO", "SAMPLING_TYPE", "SPLIT_BALLOT", "WEIGHT_YN", "STRATA_VARS",
	"CLUSTER_VAR", "SPLIT_VAR", "WEIGHT_VAR", "ID_VAR", "LINK_VAR", "LINK_FILE_NO",
	"MEAS_LEVEL", "MR_TYPE", "REVERSE", "PII", "SENSITIVE", "STD_ID",
}

// parsedValue is cell v of key as Parse gives it back: v, apart from the four
// changes reading allows (text normalization, list separators, integers and
// the spelling of a restored "확인 불가").
func parsedValue(key, v string) string {
	if reason, ok := Unverified(v); ok {
		if slices.Contains(typedKeys, key) {
			return strings.TrimSpace("확인 불가: " + ddi.NormalizeText(reason))
		}
		return ddi.NormalizeText(v)
	}
	switch {
	case slices.Contains([]string{"KEYWORDS", "EXTRA_FILES", "STRATA_VARS"}, key):
		var items []string
		for _, it := range strings.Split(v, ";") {
			if it = ddi.NormalizeText(it); it != "" {
				items = append(items, it)
			}
		}
		return strings.Join(items, "; ")
	case slices.Contains([]string{"TARGET_N", "VALID_N", "CONTACT_N", "EXCLUDE_N", "SCHOOL_CODE_YEAR", "BIZ_YEAR"}, key):
		return integer(v)
	}
	return ddi.NormalizeText(v)
}

// checkValues compares every Record of out with in, row by row; codes per var,
// since Parse returns them grouped by var.
func checkValues(t *testing.T, in, out *Codebook) {
	t.Helper()
	cmp := func(what string, a, b Record) {
		t.Helper()
		for _, r := range []Record{a, b} {
			for k := range r {
				if got, want := b.Get(k), parsedValue(k, a.Get(k)); got != want {
					t.Errorf("%s %s: got %q, want %q", what, k, got, want)
				}
			}
		}
	}
	cmp("01", in.Study, out.Study)
	if len(out.Files) != len(in.Files) || len(out.Vars) != len(in.Vars) {
		t.Fatalf("%d files, %d vars; want %d, %d", len(out.Files), len(out.Vars), len(in.Files), len(in.Vars))
	}
	for i, f := range in.Files {
		cmp("02 F"+f.Rec.Get("FILE_NO"), f.Rec, out.Files[i].Rec)
	}
	for i, v := range in.Vars {
		w := out.Vars[i]
		cmp("03 "+varLoc(v), v.Rec, w.Rec)
		a, b := in.CodesOf(v), out.CodesOf(w)
		if len(a) != len(b) {
			t.Errorf("04 %s: %d codes, want %d", varLoc(v), len(b), len(a))
			continue
		}
		for j := range a {
			cmp("04 "+varLoc(v), a[j].Rec, b[j].Rec)
			if a[j].Type != b[j].Type || a[j].StdCode != b[j].StdCode || a[j].StdSet != b[j].StdSet {
				t.Errorf("04 %s code %d: %+v, want %+v", varLoc(v), j, *b[j], *a[j])
			}
		}
	}
}

func TestParseValues(t *testing.T) {
	for name, cb := range map[string]*Codebook{"small": smallModel(), "full": fullModel(), "unverified": unverifiedModel()} {
		t.Run(name, func(t *testing.T) {
			checkValues(t, cb, mustParse(t, mustXML(t, cb)))
		})
	}

	t.Run("the four exceptions", func(t *testing.T) {
		out := mustParse(t, mustXML(t, unverifiedModel()))
		eq(t, "typed slot", out.Study.Get("FIELD_START"), "확인 불가: FIELD_START")
		eq(t, "text slot keeps the cell", out.Study.Get("TOOL"), "확인불가：TOOL")
		out = mustParse(t, mustXML(t, fullModel()))
		eq(t, "TARGET_N", out.Study.Get("TARGET_N"), "10000")
		eq(t, "KEYWORDS", out.Study.Get("KEYWORDS"), "과학기술 관심도; 과학자본")
		eq(t, "EXTRA_FILES", out.Study.Get("EXTRA_FILES"), "부록1.xlsx; 부록2.hwp")
		eq(t, "BIZ_NAME is one value", out.Study.Get("BIZ_NAME"), "과학문화 확산; 사업")
	})

	t.Run("the rest of the model", func(t *testing.T) {
		in := fullModel()
		in.Admin.WallaID = "ppb\n  CL08" // IDNo holds it normalized; sourceSurvey@wallaId exactly
		out := mustParse(t, mustXML(t, in))
		if out.Admin != in.Admin {
			t.Errorf("Admin %+v, want %+v", out.Admin, in.Admin)
		}
		for i, v := range in.Vars {
			if w := out.Vars[i]; w.ID != v.ID || !reflect.DeepEqual(w.Analysis, v.Analysis) {
				t.Errorf("%s: %s %+v, want %s %+v", varLoc(v), w.ID, w.Analysis, v.ID, v.Analysis)
			}
		}
		if !reflect.DeepEqual(out.Files[1].Cleansing, in.Files[1].Cleansing) {
			t.Errorf("cleansing %+v", out.Files[1].Cleansing)
		}
		if out.Files[0].PackageName != in.Files[0].PackageName {
			t.Errorf("PackageName %q", out.Files[0].PackageName)
		}
		// Like a record read from the template, every record holds every KEY.
		for _, c := range []struct {
			rec  Record
			keys []string
		}{{out.Study, StudyKeys}, {out.Files[0].Rec, FileKeys}, {out.Vars[0].Rec, VarKeys}, {out.Codes[0].Rec, CodeKeys}} {
			if len(c.rec) != len(c.keys) {
				t.Errorf("record of %d keys, want %v", len(c.rec), c.keys)
			}
		}
	})

	t.Run("AuthEnty", func(t *testing.T) {
		for _, c := range [][2]string{{"서울대학교", "임동균"}, {"", "임동균"}, {"서울대학교", ""}} {
			cb := smallModel()
			cb.Study.Set("PI_ORG", c[0])
			cb.Study.Set("PI_NAME", c[1])
			out := mustParse(t, mustXML(t, cb))
			if got := [2]string{out.Study.Get("PI_ORG"), out.Study.Get("PI_NAME")}; got != c {
				t.Errorf("PI_ORG, PI_NAME = %q, want %q", got, c)
			}
		}
	})

	t.Run("02 and 03 unverified land on their row", func(t *testing.T) {
		cb := smallModel()
		cb.Files[1].Rec.Set("VALID_N", "확인불가：c")
		cb.Vars[11].Rec.Set("MEAS_LEVEL", "확인 불가: d") // SQ1 of file 2; file 1 has an SQ1 too
		out := mustParse(t, mustXML(t, cb))
		eq(t, "F2 VALID_N", out.Files[1].Rec.Get("VALID_N"), "확인 불가: c")
		eq(t, "F1 VALID_N", out.Files[0].Rec.Get("VALID_N"), "10000")
		eq(t, "F2/SQ1 MEAS_LEVEL", out.Vars[11].Rec.Get("MEAS_LEVEL"), "확인 불가: d")
		eq(t, "F1/SQ1 MEAS_LEVEL", out.Vars[1].Rec.Get("MEAS_LEVEL"), "명목")
	})

	t.Run("groups", func(t *testing.T) {
		cb := smallModel()
		cb.Vars[3].GridGroup, cb.Vars[4].GridGroup = "격자", "격자"
		cb.Vars[6].NoneOption, cb.Vars[7].NoneOption = "", ""
		out := mustParse(t, mustXML(t, cb))
		eq(t, "grid key is the first member's name", out.Vars[4].GridGroup, "QA1_1")
		eq(t, "noneOption goes to every member", out.Vars[7].NoneOption, "Y")
	})

	t.Run("system vars and package", func(t *testing.T) {
		cb := smallModel()
		out := mustParse(t, mustXML(t, cb))
		if slices.ContainsFunc(out.Vars, func(v *Var) bool { return IsSystemVar(v.Rec.Get("VAR_NAME")) }) {
			t.Error("system vars came back as survey vars")
		}
		cb.Package = nil
		if out := mustParse(t, mustXML(t, cb)); out.Package != nil {
			t.Errorf("no docDscr gave a package %+v", out.Package)
		}
	})

	t.Run("native", func(t *testing.T) {
		for kind, want := range map[string]bool{"native": true, "upload": false} {
			cb := smallModel()
			cb.Package.Source.Kind = kind
			if got := mustParse(t, mustXML(t, cb)).Native; got != want {
				t.Errorf("kind %s: Native = %v, want %v", kind, got, want)
			}
		}
	})
}

func TestParseRejects(t *testing.T) {
	x := mustXML(t, smallModel())
	edit := func(f func(root *ddi.Node)) *ddi.Node {
		root, _, err := ddi.Parse(x)
		if err != nil {
			t.Fatal(err)
		}
		f(root)
		return root
	}
	appendTo := func(path string, ns ...*ddi.Node) *ddi.Node {
		return edit(func(root *ddi.Node) {
			p := at(root, path)
			p.Children = append(p.Children, ns...)
		})
	}
	// insertAfter puts n right after the first child named name, so that XML
	// reads the two as siblings rather than as mixed content.
	insertAfter := func(path, name string, n *ddi.Node) *ddi.Node {
		return edit(func(root *ddi.Node) {
			p := root
			if path != "" {
				p = at(root, path)
			}
			i := slices.IndexFunc(p.Children, func(c *ddi.Node) bool { return c.Name == name })
			p.Children = slices.Insert(p.Children, i+1, n)
		})
	}
	setVar := func(id, attr, value string) *ddi.Node {
		return edit(func(root *ddi.Node) {
			v := withAttr(all(at(root, "dataDscr"), "var"), "ID", id)
			if i := slices.IndexFunc(v.Attrs, func(a ddi.Attr) bool { return a.Name == attr }); i >= 0 {
				v.Attrs[i].Value = value
			} else {
				v.Attrs = append(v.Attrs, ddi.Attr{Name: attr, Value: value})
			}
		})
	}
	unverified := func(kv ...string) *ddi.Node {
		n := &ddi.Node{Name: "kosac:unverified", Text: "사유"}
		for i := 0; i < len(kv); i += 2 {
			n.Attrs = append(n.Attrs, ddi.Attr{Name: kv[i], Value: kv[i+1]})
		}
		return n
	}
	const u = "stdyDscr/kosac:study/kosac:unverified"
	// want is the path, followed by the message where the path alone is shared
	// with another rejection.
	trees := []struct {
		name string
		root *ddi.Node
		want string
	}{
		{"unknown DDI element", appendTo("stdyDscr/stdyInfo", &ddi.Node{Name: "notes", Text: "x"}), "stdyDscr/stdyInfo/notes"},
		{"unknown kosac element", appendTo("stdyDscr/kosac:study", &ddi.Node{Name: "kosac:foo"}), "stdyDscr/kosac:study/kosac:foo"},
		{"kosac element out of place", edit(func(root *ddi.Node) {
			d, s := at(root, "docDscr"), at(root, "stdyDscr")
			s.Children = append(s.Children, d.Children[1])
			d.Children = d.Children[:1]
		}), "stdyDscr/kosac:package"},
		{"unknown attribute", setVar("V1", "kosac:foo", "x"), "dataDscr/var[@ID='V1']/@kosac:foo"},
		{"mixed content", edit(func(root *ddi.Node) {
			n := at(root, "stdyDscr/citation/titlStmt/titl")
			n.Text, n.Raw = "", "제목 <emph>강조</emph>"
		}), "stdyDscr/citation/titlStmt/titl"},
		{"text where the engine writes none", edit(func(root *ddi.Node) {
			at(root, "stdyDscr/stdyInfo/sumDscr/collDate").Text = "2024년 10월"
		}), "stdyDscr/stdyInfo/sumDscr/collDate"},
		{"ID of no var", edit(func(root *ddi.Node) {
			n := at(root, "stdyDscr/method/dataColl/kosac:weight")
			n.Attrs[1].Value = "V99"
		}), "stdyDscr/method/dataColl/kosac:weight/@var"},
		{"value the engine does not write", edit(func(root *ddi.Node) {
			at(root, "stdyDscr/citation/titlStmt/IDNo").Attrs[0].Value = "x"
		}), "stdyDscr/citation/titlStmt/IDNo/@agency"},
		{"unknown measLevel", setVar("V2", "kosac:measLevel", "foo"), "dataDscr/var[@ID='V2']/@kosac:measLevel"},
		{"second abstract of a type", insertAfter("stdyDscr/stdyInfo", "abstract",
			&ddi.Node{Name: "abstract", Attrs: []ddi.Attr{{Name: "contentType", Value: "purpose"}}, Text: "다른 목적"}),
			"stdyDscr/stdyInfo/abstract/@contentType"},
		{"second report", insertAfter("", "otherMat", &ddi.Node{Name: "otherMat",
			Attrs:    []ddi.Attr{{Name: "type", Value: "report"}, {Name: "level", Value: "study"}},
			Children: []*ddi.Node{{Name: "labl", Text: "다른 보고서.pdf"}}}), "otherMat/@type"},
		{"var in two sections", edit(func(root *ddi.Node) {
			g := withAttr(all(at(root, "dataDscr"), "varGrp"), "ID", "G2")
			g.Attrs[2].Value += " V1"
		}), "dataDscr/varGrp[@ID='G2']/@var"},
		{"var ID twice", setVar("V2", "ID", "V1"), "dataDscr/var[@ID='V1']/@ID"},
		{"file ID twice", edit(func(root *ddi.Node) {
			all(root, "fileDscr")[1].Attrs[0].Value = "F1"
		}), "fileDscr[@ID='F1']/@ID"},
		{"unverified with @file and @var", appendTo("stdyDscr/kosac:study", unverified("key", "VALID_N", "file", "F1", "var", "V1")), u + ": "},
		{"unverified of no file", appendTo("stdyDscr/kosac:study", unverified("key", "VALID_N", "file", "F9")), u + "/@file"},
		{"unverified of no var", appendTo("stdyDscr/kosac:study", unverified("key", "PII", "var", "V99")), u + "/@var"},
		{"unverified of no KEY", appendTo("stdyDscr/kosac:study", unverified("key", "FOO")), u + "/@key: 엔진이 쓰지 않는"},
		{"unverified twice", appendTo("stdyDscr/kosac:study", unverified("key", "FIELD_START"), unverified("key", "FIELD_START")),
			u + `/@key: "FIELD_START"`},
		{"kosac namespace under prefix k", edit(func(root *ddi.Node) {
			k := func(s string) string {
				if s == "xmlns:kosac" {
					return "xmlns:k"
				}
				if rest, ok := strings.CutPrefix(s, "kosac:"); ok {
					return "k:" + rest
				}
				return s
			}
			walk(root, func(n *ddi.Node) {
				n.Name = k(n.Name)
				for i := range n.Attrs {
					n.Attrs[i].Name = k(n.Attrs[i].Name)
				}
			})
		}), "codeBook/@xmlns:k"},
	}
	type reject struct{ name, want string }
	docs := map[reject][]byte{}
	for _, c := range trees {
		docs[reject{c.name + "/XML", c.want}] = ddi.WriteXML(c.root)
		docs[reject{c.name + "/JSON", c.want}] = ddi.WriteJSON(c.root)
	}
	archive, _ := filepath.Glob("../testdata/archive/*.xml")
	if len(archive) == 0 {
		t.Fatal("no archive fixtures")
	}
	for _, p := range archive {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		docs[reject{"archive " + filepath.Base(p), "codeBook"}] = b
	}
	for c, doc := range docs {
		t.Run(c.name, func(t *testing.T) {
			cb, err := Parse(doc)
			if err == nil || cb != nil {
				t.Fatalf("no error (codebook %v)", cb != nil)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error does not name %s: %v", c.want, err)
			}
		})
	}
}

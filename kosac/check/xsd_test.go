package check

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/internal/xsdtest"
	"github.com/teamPaprika/ddi-xlsx/kosac"
)

// xmllint validates against the vendored DDI XSD and the kosac draft.
func xmllint(t *testing.T) Validator {
	return func(s Schema, doc []byte) ([]string, error) {
		p := filepath.Join(xsdtest.Root(), "xsd", "ddi-codebook-2.5", "codebook.xsd")
		if s == SchemaKosac {
			p = filepath.Join(xsdtest.Root(), "kosac", "xsd", "kosac-3.3.xsd")
		}
		return xsdtest.ValidateSchema(t, p, doc), nil
	}
}

// engineModel is base plus what puts every kind of reference in the XML: a
// package, a coded var, a cleansing rule and a "확인 불가" 02 value.
func engineModel() *kosac.Codebook {
	cb := base()
	cb.Study.Set("DATA_NO", "문화-01-01")
	cb.Admin.Round = kosac.Round{Year: "2024", WaveType: "main"}
	cb.Package = &kosac.Package{CodebookVersion: "1"}
	cb.Vars = append(cb.Vars, v("Q1", "MEAS_LEVEL", "명목"))
	cb.Codes = codes("Q1", "1", "2")
	for _, c := range cb.Codes {
		c.Type = "VALID"
	}
	f := cb.Files[0]
	f.Rec.Set("CONTENT", "확인 불가: 미기재")
	f.Cleansing = &kosac.Cleansing{Rules: []kosac.CleansingRule{{ID: "CL-01", VarName: "Q1", Action: "REPLACE", Reason: "코드 오류"}}}
	if err := kosac.AssignIDs(cb, nil); err != nil {
		panic(err)
	}
	return cb
}

// engineTree is the parsed XML of engineModel, for a test to edit.
func engineTree(t *testing.T) *ddi.Node {
	t.Helper()
	b, err := engineModel().XML()
	if err != nil {
		t.Fatal(err)
	}
	root, _, err := ddi.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// at follows path through the first child of each name.
func at(n *ddi.Node, path string) *ddi.Node {
	for _, name := range strings.Split(path, "/") {
		i := slices.IndexFunc(n.Children, func(c *ddi.Node) bool { return c.Name == name })
		if i < 0 {
			panic("no " + name + " in " + path)
		}
		n = n.Children[i]
	}
	return n
}

func varOf(root *ddi.Node, id string) *ddi.Node {
	for _, c := range at(root, "dataDscr").Children {
		if v, _ := c.Attr("ID"); c.Name == "var" && v == id {
			return c
		}
	}
	panic("no var " + id)
}

func setAttr(n *ddi.Node, name, value string) {
	for i := range n.Attrs {
		if n.Attrs[i].Name == name {
			n.Attrs[i].Value = value
			return
		}
	}
	n.Attrs = append(n.Attrs, ddi.Attr{Name: name, Value: value})
}

func issueSums(is []kosac.Issue) string {
	var out []string
	for _, i := range is {
		out = append(out, string(i.Level)+" "+i.Rule+" "+i.Path+" "+i.Actual)
	}
	return strings.Join(out, "\n")
}

func TestValidateXML(t *testing.T) {
	for _, c := range []struct {
		name, rule, path string // rule "" wants no issue
		edit             func(root *ddi.Node)
	}{
		{"engine output", "", "", func(*ddi.Node) {}},
		{"DDI break", "XSD-DDI", "", func(r *ddi.Node) {
			o := at(r, "otherMat")
			o.Attrs = slices.DeleteFunc(o.Attrs, func(a ddi.Attr) bool { return a.Name == "level" })
		}},
		{"kosac break", "XSD-KOSAC", "stdyDscr/method/dataColl", func(r *ddi.Node) {
			at(r, "stdyDscr/method/dataColl/kosac:excludeN").Text = "많음"
		}},
		{"package under stdyDscr", "PLACE", "stdyDscr/kosac:package", func(r *ddi.Node) {
			d, s := at(r, "docDscr"), at(r, "stdyDscr")
			pkg := at(d, "kosac:package")
			d.Children = slices.DeleteFunc(d.Children, func(c *ddi.Node) bool { return c == pkg })
			s.Children = append(s.Children, pkg)
		}},
		{"seq on catgry", "PLACE", "dataDscr/var[@ID='V2']/catgry/@kosac:seq", func(r *ddi.Node) {
			setAttr(at(varOf(r, "V2"), "catgry"), "kosac:seq", "1")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := engineTree(t)
			c.edit(root)
			got := ValidateXML(ddi.WriteXML(root), xmllint(t))
			if c.rule == "" {
				if len(got) > 0 {
					t.Errorf("issues:\n%s", issueSums(got))
				}
				return
			}
			if len(got) != 1 || got[0].Rule != c.rule || got[0].Level != kosac.R || got[0].Path != c.path || got[0].Message == "" {
				t.Errorf("want one R %s at %q, got:\n%s", c.rule, c.path, issueSums(got))
			}
		})
	}
}

// A nil Validator skips the XSD steps; PLACE and REF still run.
func TestValidateXMLNilValidator(t *testing.T) {
	root := engineTree(t)
	if got := ValidateXML(ddi.WriteXML(root), nil); len(got) > 0 {
		t.Errorf("engine output:\n%s", issueSums(got))
	}
	setAttr(at(varOf(root, "V2"), "catgry"), "kosac:seq", "1")
	got := ValidateXML(ddi.WriteXML(root), nil)
	if len(got) != 1 || got[0].Rule != "PLACE" || got[0].Level != kosac.R {
		t.Errorf("seq on catgry:\n%s", issueSums(got))
	}
}

func TestValidateXMLUnreadable(t *testing.T) {
	got := ValidateXML([]byte("<codeBook"), func(Schema, []byte) ([]string, error) {
		t.Fatal("validator called on an unreadable document")
		return nil, nil
	})
	if len(got) != 1 || got[0].Level != kosac.R || got[0].Actual == "" || got[0].Message == "" {
		t.Errorf("got:\n%s", issueSums(got))
	}
}

func TestValidatorError(t *testing.T) {
	b, err := engineModel().XML()
	if err != nil {
		t.Fatal(err)
	}
	got := ValidateXML(b, func(Schema, []byte) ([]string, error) { return nil, errors.New("xmllint: not found") })
	if len(got) != 1 || got[0].Level != kosac.R || !strings.Contains(got[0].Message, "실행할 수 없") ||
		!strings.Contains(got[0].Actual, "not found") {
		t.Errorf("got:\n%s", issueSums(got))
	}
}

// changesRows are the request-side texts of the rows of the foundation report
// table, which CHANGES.md must reproduce.
var changesRows = []string{
	`abstract@contentType="summary"`,
	`varGrp@type="rank"`,
	`otherMat에 level 없음`,
	`<targetSampleSize>10000</…>`,
	`dataColl/respRate`,
	`fileTxt/notes`,
	`그리드가 qstn@ID 공유, Q_NO→qstn@ID`,
	`부록 D의 sumDscr: universe 다음 anlyUnit`,
	`role=codebook 파일에 bytes·sha256`,
	`kosac 요소를 DDI 순서 중간에 둠`,
	`EXCLUDE_RULE·EDIT_METHOD를 cleanOps 하나에`,
	`확인 불가 값을 원래 자리에 둠(부록 D cleanOps)`,
	`시스템 변수 8개(device·channel 포함)`,
	`kosac:popSource 위치: sumDscr 옆(4-2 표)`,
	`kosac:validN 요소(4-2 표)`,
	`원래 업로드 파일명 자리 없음`,
	`unverified@key만`,
	`kosac:codeHistory`,
	`같은 이름 kosac:file 두 가지(fileDscr 자식, package 자식)`,
}

// xsdNames lists every name the kosac XSD declares as CHANGES.md writes it:
// `kosac:E` for an element, `kosac:E/@a` for an attribute of E and `@kosac:a`
// for a global attribute. types lists every @type the declarations use.
func xsdNames(t *testing.T, b []byte) (names, types []string) {
	type node struct {
		XMLName  xml.Name
		Name     string `xml:"name,attr"`
		Type     string `xml:"type,attr"`
		Children []node `xml:",any"`
	}
	var schema node
	if err := xml.Unmarshal(b, &schema); err != nil {
		t.Fatal(err)
	}
	var out []string
	var walk func(n node, elem string)
	walk = func(n node, elem string) {
		if n.Type != "" && !slices.Contains(types, n.Type) {
			types = append(types, n.Type)
		}
		switch n.XMLName.Local {
		case "element":
			elem = "kosac:" + n.Name
			out = append(out, elem)
		case "attribute":
			if elem == "" {
				out = append(out, "@kosac:"+n.Name)
			} else {
				out = append(out, elem+"/@"+n.Name)
			}
		}
		for _, c := range n.Children {
			walk(c, elem)
		}
	}
	walk(schema, "")
	return out, types
}

func TestChangesDocCoversTable(t *testing.T) {
	dir := filepath.Join(xsdtest.Root(), "kosac", "xsd")
	doc, err := os.ReadFile(filepath.Join(dir, "CHANGES.md"))
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range append(changesRows, "kosac:artifact") {
		if !strings.Contains(string(doc), row) {
			t.Errorf("row %d: %q not in CHANGES.md", i+1, row)
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, "kosac-3.3.xsd"))
	if err != nil {
		t.Fatal(err)
	}
	names, types := xsdNames(t, b)
	if len(names) < 100 {
		t.Fatalf("only %d names read from the XSD", len(names))
	}
	for _, n := range names {
		if !strings.Contains(string(doc), "`"+n+"`") {
			t.Errorf("`%s` not listed in CHANGES.md", n)
		}
	}
	// Every form beyond a plain string constrains what datajam may pass, so
	// each is put to the foundation as a draft choice.
	_, choices, _ := strings.Cut(string(doc), "### 요청서가 정하지 않아 초안에서 정한 것")
	choices, _, _ = strings.Cut(choices, "\n## ")
	for _, ty := range types {
		if ty != "xs:string" && !strings.Contains(choices, "`"+ty+"`") {
			t.Errorf("type `%s` not in the draft choices of CHANGES.md", ty)
		}
	}
}

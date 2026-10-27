package schema

import (
	"slices"
	"testing"
)

func TestRepeatable(t *testing.T) {
	for _, n := range []string{"labl", "IDNo", "software", "var", "varGrp", "catgry", "notes", "sumStat", "catStat", "xhtml:p"} {
		if !IsRepeatable(n) {
			t.Errorf("%s should be repeatable", n)
		}
	}
	// codeBook is never a particle; it must still be known, or the root would become a JSON array.
	for _, n := range []string{"varFormat", "derivation", "codeBook"} {
		if IsRepeatable(n) {
			t.Errorf("%s should not be repeatable", n)
		}
	}
}

func TestVarChildOrder(t *testing.T) {
	prev := -1
	for _, c := range []string{"location", "labl", "qstn", "valrng", "invalrng", "sumStat", "catgry", "varFormat", "notes"} {
		p, ok := Position("var", c)
		if !ok {
			t.Fatalf("Position(var, %s) not found", c)
		}
		if p <= prev {
			t.Errorf("Position(var, %s) = %d, want > %d", c, p, prev)
		}
		prev = p
	}
}

func TestChoiceMembersShareAPosition(t *testing.T) {
	item, ok1 := Position("valrng", "item")
	rng, ok2 := Position("valrng", "range")
	notes, ok3 := Position("valrng", "notes")
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("missing positions: item %v, range %v, notes %v", ok1, ok2, ok3)
	}
	if item != rng {
		t.Errorf("item %d != range %d", item, rng)
	}
	if item >= notes {
		t.Errorf("item %d should be < notes %d", item, notes)
	}
}

func TestLocalElementsGroupsAndDerivation(t *testing.T) {
	for _, c := range [][2]string{
		{"developmentActivity", "outcome"},
		{"developmentActivity", "description"},
		{"labl", "ExtLink"},
		{"labl", "Link"},
		{"labl", "xhtml:p"},
		{"nonexistent", "anything"},
	} {
		if !Allowed(c[0], c[1]) {
			t.Errorf("Allowed(%s, %s) = false, want true", c[0], c[1])
		}
	}
	for _, c := range [][2]string{
		{"labl", "catgry"},
		{"var", "titl"},
	} {
		if Allowed(c[0], c[1]) {
			t.Errorf("Allowed(%s, %s) = true, want false", c[0], c[1])
		}
	}
}

func TestAttrEnum(t *testing.T) {
	for _, tc := range []struct {
		el, attr string
		want     []string
	}{
		// XSD order: varType lists contin before discrete.
		{"var", "intrvl", []string{"contin", "discrete"}},
		{"catgry", "missing", []string{"Y", "N"}},
		{"qstn", "responseDomainType", []string{"text", "numeric", "code", "category", "datetime", "geographic", "multiple", "other"}},
		{"varFormat", "schema", []string{"SAS", "SPSS", "IBM", "ANSI", "ISO", "XML-Data", "other"}},
		// From the GLOBALS attribute group, through the base type.
		{"catgry", "source", []string{"archive", "producer"}},
		{"var", "name", nil},
		{"nonexistent", "x", nil},
	} {
		if got := AttrEnum(tc.el, tc.attr); !slices.Equal(got, tc.want) {
			t.Errorf("AttrEnum(%s, %s) = %q, want %q", tc.el, tc.attr, got, tc.want)
		}
	}
	types := AttrEnum("varGrp", "type")
	if len(types) != 14 || types[0] != "section" || types[13] != "other" {
		t.Errorf("AttrEnum(varGrp, type) = %q, want 14 values from section to other", types)
	}
}

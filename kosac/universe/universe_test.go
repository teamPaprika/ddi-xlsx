package universe

import (
	"slices"
	"strings"
	"testing"
)

type row map[string]string

func (r row) Value(name string) (string, bool) {
	v, ok := r[name]
	return v, ok
}

func TestParseSample(t *testing.T) {
	for _, s := range []string{
		"SQ2_1=2", "SQ3_1=2", "QA5=1,2", "QA5=4,5", "QA28=6", "SQ3_1=1,2",
		"QA37_1=1~10", "C1=2~4", "SQ2_1=3~7", "DQ4a=2~5", "DQ4a=3~5", "DQ4a=4~5",
		"SQ3!=1 AND SQ3_1!=-8", "SQ2_1_3=B100203,B100217,B100231", "Random_Ballot=1",
		"Q5 = 1, 2",
	} {
		if _, err := Parse(s); err != nil {
			t.Errorf("Parse(%q): %v", s, err)
		}
	}

	e, err := Parse("SQ3!=1 AND SQ3_1!=-8")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := e.Vars(), []string{"SQ3", "SQ3_1"}; !slices.Equal(got, want) {
		t.Errorf("Vars() = %q, want %q", got, want)
	}
}

func TestParseErrors(t *testing.T) {
	for _, s := range []string{
		"", "Q5=", "Q5=1,", "=1", "(Q5=1)", "Q5=a~b", "Q5=1 and Q6=2", "Q5=1 AND",
		"Q5=pattern:[", "1Q=1",
	} {
		_, err := Parse(s)
		if err == nil {
			t.Errorf("Parse(%q): want error", s)
			continue
		}
		if !strings.Contains(err.Error(), "번째 글자") {
			t.Errorf("Parse(%q) error %q has no position", s, err)
		}
	}

	// Positions count runes, not bytes: the missing value after "서울," is the 7th rune.
	if _, err := Parse("Q5=서울,"); err == nil || !strings.HasPrefix(err.Error(), "7번째 글자") {
		t.Errorf("Parse(%q) error = %v, want 7번째 글자", "Q5=서울,", err)
	}
}

func TestEval(t *testing.T) {
	for _, tc := range []struct {
		expr string
		r    row
		want bool
	}{
		{"Q5=1,2", row{"Q5": "2"}, true},
		{"Q5=1,2", row{"Q5": "3"}, false},
		{"SQ2=14~18", row{"SQ2": "14"}, true},
		{"SQ2=14~18", row{"SQ2": "18"}, true},
		{"SQ2=14~18", row{"SQ2": "13.5"}, false},
		{"Q5!=1", row{"Q5": ""}, true},
		{"Q5=1", row{"Q5": ""}, false},
		{"A=1 OR B=2 AND C=3", row{"A": "1", "B": "0", "C": "0"}, true},
		{"X=B100203", row{"X": "B100203"}, true},
		{"X=B100203", row{"X": "B100204"}, false},
		{"Q5=01", row{"Q5": "1"}, true},
	} {
		e, err := Parse(tc.expr)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.expr, err)
		}
		got, err := e.Eval(tc.r)
		if err != nil {
			t.Fatalf("Eval(%q, %v): %v", tc.expr, tc.r, err)
		}
		if got != tc.want {
			t.Errorf("Eval(%q, %v) = %v, want %v", tc.expr, tc.r, got, tc.want)
		}
	}

	e, err := Parse("Q5=1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Eval(row{"Q6": "1"}); err == nil {
		t.Error("Eval with Q5 missing from the row: want error")
	}
}

func TestParseConditionPattern(t *testing.T) {
	e, err := ParseCondition("pattern:^[0-9]{7}$", "SCH")
	if err != nil {
		t.Fatal(err)
	}
	for v, want := range map[string]bool{"7230115": true, "B100203": false} {
		if got, err := e.Eval(row{"SCH": v}); err != nil || got != want {
			t.Errorf("Eval(SCH=%q) = %v, %v; want %v", v, got, err, want)
		}
	}

	if _, err := Parse("pattern:x"); err == nil {
		t.Error(`Parse("pattern:x"): want error`)
	}

	e, err = ParseCondition("pattern:^a b,c$ AND X=1", "T")
	if err != nil {
		t.Fatal(err)
	}
	if got := e.or[0][0].re.String(); got != "^a b,c$" {
		t.Errorf("regex = %q, want %q", got, "^a b,c$")
	}
	if got, want := e.Vars(), []string{"T", "X"}; !slices.Equal(got, want) {
		t.Errorf("Vars() = %q, want %q", got, want)
	}

	// Whitespace around a regex is dropped, and any whitespace around AND/OR
	// ends it, as everywhere else in the grammar.
	for _, tc := range []struct {
		expr string
		vars []string
	}{
		{"SCH=pattern:^[0-9]{7}$ ", []string{"SCH"}},
		{"SCH=pattern: ^[0-9]{7}$", []string{"SCH"}},
		{"SCH=pattern:^[0-9]{7}$  AND X=1", []string{"SCH", "X"}},
		{"SCH=pattern:^[0-9]{7}$\tAND X=1", []string{"SCH", "X"}},
		{"SCH=pattern:^[0-9]{7}$\nAND\nX=1", []string{"SCH", "X"}},
	} {
		e, err := Parse(tc.expr)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.expr, err)
			continue
		}
		if got, err := e.Eval(row{"SCH": "7230115", "X": "1"}); err != nil || !got {
			t.Errorf("Eval(%q) = %v, %v; want true", tc.expr, got, err)
		}
		if got := e.Vars(); !slices.Equal(got, tc.vars) {
			t.Errorf("Vars(%q) = %q, want %q", tc.expr, got, tc.vars)
		}
	}

	_, err = ParseCondition("pattern:a AND b", "T")
	if err == nil || !strings.Contains(err.Error(), "pattern 안의 AND/OR는 지원하지 않습니다") {
		t.Errorf(`ParseCondition("pattern:a AND b") error = %v, want AND/OR-in-pattern error`, err)
	}
}

func TestValues(t *testing.T) {
	e, err := Parse("Q5=1,2,3~5 AND Q6!=-8 OR Q5=x OR Q7=pattern:^a$")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][]string{"Q5": {"1", "2", "x"}, "Q6": {"-8"}, "Q7": nil, "Z": nil} {
		if got := e.Values(name); !slices.Equal(got, want) {
			t.Errorf("Values(%s) = %q, want %q", name, got, want)
		}
	}
}

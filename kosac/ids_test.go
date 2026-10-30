package kosac

import (
	"slices"
	"strings"
	"testing"
)

func ids(cb *Codebook) []string {
	var out []string
	for _, v := range cb.Vars {
		out = append(out, v.ID)
	}
	return out
}

func vars(fileNames ...string) *Codebook {
	cb := &Codebook{}
	for _, fn := range fileNames {
		f, n, _ := strings.Cut(fn, "/")
		cb.Vars = append(cb.Vars, &Var{Rec: rec("FILE_NO", f, "VAR_NAME", n)})
	}
	return cb
}

func TestAssignIDs(t *testing.T) {
	assign := func(cb, prev *Codebook) []string {
		t.Helper()
		if err := AssignIDs(cb, prev); err != nil {
			t.Fatal(err)
		}
		return ids(cb)
	}

	t.Run("no prev", func(t *testing.T) {
		if got := assign(vars("1/A", "1/B", "1/C"), nil); !slices.Equal(got, []string{"V1", "V2", "V3"}) {
			t.Errorf("got %v", got)
		}
	})

	prev := vars("1/A", "1/B", "1/C", "1/D")
	assign(prev, nil) // A=V1 B=V2 C=V3 D=V4

	t.Run("kept, new, deleted, reordered", func(t *testing.T) {
		// D (V4) is deleted, C and A swap, E is new: max(prev)=4, so E=V5.
		got := assign(vars("1/C", "1/E", "1/A", "1/B"), prev)
		if !slices.Equal(got, []string{"V3", "V5", "V1", "V2"}) {
			t.Errorf("got %v", got)
		}
	})

	t.Run("same name in two files", func(t *testing.T) {
		p := vars("1/LINK", "2/LINK")
		if got := assign(p, nil); !slices.Equal(got, []string{"V1", "V2"}) {
			t.Fatalf("got %v", got)
		}
		if got := assign(vars("2/LINK", "2/X", "1/LINK"), p); !slices.Equal(got, []string{"V2", "V3", "V1"}) {
			t.Errorf("across versions: got %v", got)
		}
	})

	t.Run("preset ID kept and counted", func(t *testing.T) {
		cb := vars("1/A", "1/B")
		cb.Vars[1].ID = "V7"
		if got := assign(cb, nil); !slices.Equal(got, []string{"V8", "V7"}) {
			t.Errorf("got %v", got)
		}
	})

	t.Run("duplicate preset ID", func(t *testing.T) {
		cb := vars("1/A", "1/B")
		cb.Vars[0].ID, cb.Vars[1].ID = "V1", "V1"
		if err := AssignIDs(cb, nil); err == nil || !strings.Contains(err.Error(), "V1") {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("preset ID taken by an inherited one", func(t *testing.T) {
		cb := vars("1/A", "1/B")
		cb.Vars[1].ID = "V1" // A inherits V1 from prev
		if err := AssignIDs(cb, prev); err == nil {
			t.Error("no error")
		}
	})
}

package kosac

import (
	"fmt"
	"regexp"
	"strconv"
)

var varIDRe = regexp.MustCompile(`^V(\d+)$`)

// AssignIDs gives every var without an ID a V{n}. A var that prev has under
// the same (FILE_NO, VAR_NAME) keeps prev's ID; any other gets one past the
// highest number prev or cb uses, so the ID of a deleted var is never reused.
// A var that already has an ID keeps it. prev may be nil.
func AssignIDs(cb *Codebook, prev *Codebook) error {
	top := 0
	inherited := map[[2]string]string{}
	count := func(id string) {
		if m := varIDRe.FindStringSubmatch(id); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > top {
				top = n
			}
		}
	}
	if prev != nil {
		for _, v := range prev.Vars {
			count(v.ID)
			k := varKey(v)
			if _, ok := inherited[k]; !ok && v.ID != "" {
				inherited[k] = v.ID
			}
		}
	}
	for _, v := range cb.Vars {
		count(v.ID)
	}
	seen := map[string]*Var{}
	for _, v := range cb.Vars {
		if v.ID == "" {
			if id, ok := inherited[varKey(v)]; ok {
				v.ID = id
			} else {
				top++
				v.ID = "V" + strconv.Itoa(top)
			}
		}
		if o := seen[v.ID]; o != nil {
			return fmt.Errorf("변수 ID %s가 %s와 %s에 겹칩니다", v.ID, varLoc(o), varLoc(v))
		}
		seen[v.ID] = v
	}
	return nil
}

func varKey(v *Var) [2]string { return [2]string{v.Rec.Get("FILE_NO"), v.Rec.Get("VAR_NAME")} }

// varLoc names a var as F{FILE_NO}/{VAR_NAME} in messages.
func varLoc(v *Var) string { return "F" + v.Rec.Get("FILE_NO") + "/" + v.Rec.Get("VAR_NAME") }

//go:build js && wasm

// Command wasm registers conversion and checks on globalThis.__ddiXlsx for the npm
// package's JavaScript wrapper.
package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/check"
	ktemplate "github.com/teamPaprika/ddi-xlsx/kosac/template"
	"github.com/teamPaprika/ddi-xlsx/workbook"
)

func main() {
	js.Global().Set("__ddiXlsx", map[string]any{
		"toXlsx": handler(func(in []byte) (map[string]any, error) {
			x, warns, err := workbook.ToXLSX(in)
			if err != nil {
				return nil, err
			}
			return map[string]any{"xlsx": u8(x), "warnings": strs(warns)}, nil
		}),
		"fromXlsx": handler(func(in []byte) (map[string]any, error) {
			x, j, warns, err := workbook.FromXLSX(in)
			if err != nil {
				return nil, err
			}
			return map[string]any{"xml": u8(x), "json": u8(j), "warnings": strs(warns)}, nil
		}),
		"template": handler(func([]byte) (map[string]any, error) {
			x, err := workbook.Template()
			if err != nil {
				return nil, err
			}
			return map[string]any{"xlsx": u8(x)}, nil
		}),
		"checkDDI":      handler(checking("ddi")),
		"checkWorkbook": handler(checking("ddi-xlsx")),
		"checkKosac":    handler(checking("kosac-xlsx")),
		"kosacTemplate": handler(func([]byte) (map[string]any, error) {
			return map[string]any{"xlsx": u8(ktemplate.Blank())}, nil
		}),
		"kosacToDdi": handler(kosacToDDI),
	})
	select {}
}

// kosacToDDI follows `ddi-xlsx kosac-xml` without XSD: when S05 rejects the
// form, it returns the issues and no XML or JSON.
func kosacToDDI(in []byte) (map[string]any, error) {
	cb, issues, err := ktemplate.Read(in, ktemplate.ReadOptions{})
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if cb != nil {
		kosac.DeriveDefaults(cb, kosac.RefData{})
		if err := kosac.AssignIDs(cb, nil); err != nil {
			return nil, err
		}
		root, err := cb.Tree()
		if err != nil {
			return nil, err
		}
		x := ddi.WriteXML(root)
		issues = append(issues, check.ValidateXML(x, nil)...)
		out["xml"], out["json"] = u8(x), u8(ddi.WriteJSON(root))
	}
	if issues == nil {
		issues = []kosac.Issue{}
	}
	report, err := json.Marshal(issues)
	if err != nil {
		return nil, err
	}
	out["issues"] = u8(report)
	return out, nil
}

// handler copies the Uint8Array argument, if any, into Go and wraps the
// result as {ok: true, ...} or {ok: false, error}.
func handler(conv func([]byte) (map[string]any, error)) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) (res any) {
		// An unrecovered panic exits the Go program, and every later call on the
		// page would then fail; a bad input file must fail only its own call.
		defer func() {
			if r := recover(); r != nil {
				res = map[string]any{"ok": false, "error": fmt.Sprint(r)}
			}
		}()
		var in []byte
		if len(args) > 0 {
			in = make([]byte, args[0].Get("length").Int())
			js.CopyBytesToGo(in, args[0])
		}
		out, err := conv(in)
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}
		}
		out["ok"] = true
		return out
	})
}

func u8(b []byte) js.Value {
	a := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(a, b)
	return a
}

func strs(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

//go:build js && wasm

package main

import (
	"encoding/json"

	"github.com/teamPaprika/ddi-xlsx/ddi"
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/check"
	ktemplate "github.com/teamPaprika/ddi-xlsx/kosac/template"
	"github.com/teamPaprika/ddi-xlsx/workbook"
)

type checkStep struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type checkReport struct {
	Format          string        `json:"format"`
	TemplateVersion string        `json:"templateVersion,omitempty"`
	Rejections      int           `json:"rejections"`
	Warnings        int           `json:"warnings"`
	Steps           []checkStep   `json:"steps"`
	Issues          []kosac.Issue `json:"issues"`
}

func checking(format string) func([]byte) (map[string]any, error) {
	return func(input []byte) (map[string]any, error) {
		r := checkReport{Format: format, Issues: []kosac.Issue{}}
		if format == "kosac-xlsx" {
			cb, issues, err := ktemplate.Read(input, ktemplate.ReadOptions{})
			r.Issues = append(r.Issues, issues...)
			if err != nil {
				r.readError(err)
			}
			r.addStep("S05", "재단 양식 구조", r.Issues, "_meta 버전과 필수 시트·머신 키를 검사합니다.")
			if cb != nil && err == nil {
				r.TemplateVersion = cb.Template.Version.Value
				issues = check.Check(cb, kosac.RefData{})
				r.Issues = append(r.Issues, issues...)
				r.addStep("C", "코드북 규칙", issues, "현재 엔진의 공통 규칙과 버전별 규칙을 실행합니다.")
			} else {
				r.Steps = append(r.Steps, checkStep{"C", "코드북 규칙", "not_run", "양식 구조를 먼저 수정해야 합니다."})
			}
			r.Steps = append(r.Steps,
				checkStep{"RECODING", "재코딩 실행·재현", "not_run", "변환 실행과 영향 건수 재현은 검사하지 않습니다. 05 이력의 형식·참조 검사는 v4.0 양식의 C 단계가 실행된 경우에만 수행합니다."},
				checkStep{"REFERENCE", "공통코드·기관 기준정보 대조", "not_run", "이 데모는 외부 기준정보를 공급하지 않습니다."},
				checkStep{"RESPONSE", "응답 CSV 교차검사", "not_run", "코드북만 검사합니다. 응답 데이터는 검사하지 않습니다."})
		} else {
			var xml []byte
			var warnings []string
			var err error
			if format == "ddi-xlsx" {
				xml, _, warnings, err = workbook.FromXLSX(input)
			} else {
				var root *ddi.Node
				root, warnings, err = ddi.Parse(input)
				if err == nil {
					xml = ddi.WriteXML(root)
				}
			}
			for _, warning := range warnings {
				r.Issues = append(r.Issues, kosac.Issue{Rule: "READ", Level: kosac.W, Message: warning})
			}
			if err != nil {
				r.readError(err)
			}
			r.addStep("READ", "DDI 문서·편집 양식 읽기", r.Issues, "지원 버전·워크북 구조·변환 경고를 확인합니다.")
			if err == nil {
				issues := check.ValidateXML(xml, nil)
				r.Issues = append(r.Issues, issues...)
				r.addStep("REF", "ID·변수·파일 참조", issues, "엔진이 지원하는 ID·변수·파일 참조를 검사하고 kosac 확장을 반려합니다. 모든 DDI 참조를 검사하는 것은 아닙니다.")
			} else {
				r.Steps = append(r.Steps, checkStep{"REF", "ID·변수·파일 참조", "not_run", "문서를 먼저 읽을 수 있어야 합니다."})
			}
		}
		r.Steps = append(r.Steps, checkStep{"XSD", "DDI 2.5 XSD", "not_run", "이 WASM은 XSD 검사기를 포함하지 않습니다. 구조·참조 검사와 구분합니다."})
		for _, issue := range r.Issues {
			if issue.Level == kosac.R {
				r.Rejections++
			} else if issue.Level == kosac.W {
				r.Warnings++
			}
		}
		out, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		return map[string]any{"report": u8(out)}, nil
	}
}

func (r *checkReport) readError(err error) {
	r.Issues = append(r.Issues, kosac.Issue{Rule: "READ", Level: kosac.R, Actual: err.Error(), Message: "입력 파일을 읽을 수 없습니다. 형식과 오류 내용을 확인하세요."})
}

func (r *checkReport) addStep(id, name string, issues []kosac.Issue, detail string) {
	status := "pass"
	for _, issue := range issues {
		if issue.Level == kosac.R {
			status = "fail"
			break
		}
		if issue.Level == kosac.W {
			status = "warning"
		}
	}
	r.Steps = append(r.Steps, checkStep{id, name, status, detail})
}

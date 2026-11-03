---
title: 재단 Go API
weight: 2
---

```go
cb, issues, err := template.Read(input, template.ReadOptions{})
if err != nil { return err }
if cb == nil { return fmt.Errorf("양식 구조 반려: %v", issues) }
issues = append(issues, check.Check(cb, referenceData)...)
// 호출자가 반려·경고를 확인하고 초안과 담당자 입력을 확정합니다.
kosac.DeriveDefaults(cb, referenceData)
if err := kosac.AssignIDs(cb, previousModel); err != nil { return err }
root, err := cb.Tree()
if err != nil { return err }
xmlOut, jsonOut := ddi.WriteXML(root), ddi.WriteJSON(root)
issues = append(issues, check.ValidateXML(xmlOut, validator)...)
```

위 코드는 호출 순서 예시입니다. `issues`의 반려·승인 정책과 실제 파일 저장은 호출자가 구현합니다.
`previousModel`은 보관한 재단 모델입니다. 새 표준 XML을 `kosac.Parse`로 역변환하는 모델이 아닙니다.

- `template.Read` → `template.Write` → `template.Read`는 버전별 원본 레코드를 보존합니다. 모델의 분석·승인·연결 정보는 양식에 없는 열까지 저장하지 않습니다.
- `Tree/XML/JSON`은 입력 모델을 수정하지 않습니다. `DeriveDefaults`와 `AssignIDs`는 모델을 수정하므로 원본 모델을 별도로 보관하세요.
- 표준 출력의 구조화되지 않은 관리 항목은 해당 범위의 `notes` 또는 `txt`로 기록합니다. 설명문을 전용 역변환 규약으로 사용하지 않습니다.
- 새 출력은 `ddi.Parse`와 범용 `workbook.ToXLSX/FromXLSX`로 읽고 편집합니다. 이 워크북은 재단 제출 양식과 다릅니다.
- `kosac.Parse`는 과거 `urn:kosac:survey-meta:3.3` 확장 XML·JSON의 읽기 호환용입니다. 새 표준 출력의 입력 API가 아닙니다.
- 재단 양식의 관리 필드까지 다시 편집하려면 원본 XLSX와 모델을 보관해야 합니다.

### 자동 초안과 Walla 입력

`DeriveDefaults`는 빈 항목에 분석 역할·묶음·결측 분류 등의 초안을 채웁니다.
이미 채운 항목은 덮어쓰지 않습니다. 초안은 담당자가 확인해야 합니다.
v4.0에서는 입력에 없는 결측 코드를 추가하지 않습니다.

`FromWallaDDI`는 Walla XML·JSON에서 CSV 기준의 재단 초안을 만듭니다.
중복 Parquet 파일은 제외하고, 입력에 선언된 시스템 변수는 `SourceSystemVars`에 보관합니다.
외부 기관 Excel에는 Walla 시스템 변수를 추가하지 않습니다.

- Walla 묶음은 원본 `varGrp`에서 가져옵니다. 변수명 접두로 grid를 추정하지 않습니다.
- `Native`는 내부 모델 상태입니다. 새 XML의 제품 전용 표시로 저장하지 않습니다.
- 영역, 조사 정보, 추가 v4.0 항목은 호출자가 채워야 합니다. 이 어댑터는 Walla XML 전체의 무손실 변환기가 아닙니다.

---
title: 받는 입력
weight: 9
---

이 페이지의 조건은 범용 변환기(`to-xlsx`, `workbook.ToXLSX`, `ddiToXlsx`)에 적용합니다. 재단 Excel 입력은 [재단 코드북 엔진](/docs/kosac)에 있습니다.

## 입력 조건

| 항목 | 조건 | 조건에 맞지 않으면 |
|---|---|---|
| 문서 | DDI Codebook 2.5입니다. 루트 요소는 네임스페이스 `ddi:codebook:2_5`의 `codeBook`입니다. | 오류 |
| 접두사 루트 | 루트가 `<ddi:codeBook>`처럼 접두사를 쓰면, 기본 네임스페이스(`xmlns`)는 없거나 `ddi:codebook:2_5`입니다. 출력은 DDI 요소를 접두사 없이 쓰기 때문입니다. | 오류 |
| 인코딩 | UTF-8입니다. | `only UTF-8 input is supported` 오류 |
| 중첩 깊이 | 256단계까지입니다. | 위치를 표시한 오류 |
| XML 형식 | 잘 구성된(well-formed) XML입니다. XSD 유효성은 요구하지 않습니다. | 오류 |
| ID | 모든 `var`와 `varGrp`에 `ID`가 있습니다. 다른 시트가 행을 ID로 가리키기 때문입니다. | 위치를 표시한 오류 |
| ID 고유성 | `var`와 `varGrp`를 통틀어 같은 `ID`가 한 번만 나옵니다. | 오류 |
| `dataDscr` | 하나만 있습니다. | 오류 |

- XML과 JSON을 모두 받습니다. 도구는 형식을 내용으로 판별합니다. 공백이 아닌 첫 문자가 `<`이면 XML이고, `{`이면 JSON입니다.
- `ID`가 비어 있거나 앞뒤에 공백이 있으면, `ID`가 없는 것과 같은 오류가 나옵니다.
- 스키마상 자리가 맞지 않는 요소도 받습니다. 허용되지 않는 부모 아래의 요소는 워크북을 DDI로 되돌릴 때(`from-xlsx`) 경고가 나옵니다.

## 받지 않는 문서

- **DDI 1.2.2**(네임스페이스 `http://www.icpsr.umich.edu/DDI`): 이 도구의 규칙 표는 DDI 2.5 XSD에서 만들었습니다. 그래서 구조가 다른 1.2.2에는 맞지 않습니다. 도구는 `DDI 1.2.2 is not supported; convert the file to DDI 2.5 first` 오류를 냅니다. 먼저 DDI 2.5로 변환하세요.
- **그 밖의 네임스페이스**: 오류로 거부합니다.
- **UTF-8이 아닌 파일**: XML 선언에 다른 인코딩(예: `encoding="EUC-KR"`)이 있으면 거부합니다. 먼저 파일을 UTF-8로 변환하세요.

## 원문 보관

도구는 요소마다 내용을 보고, 해석할지 원문 그대로 보관할지 정합니다. 아래 조건 중 하나에 맞으면, 그 요소의 안쪽 XML을 원문 그대로 보관합니다.

- 공백이 아닌 텍스트와 요소가 섞여 있습니다. 예: `<p>a <emph>b</emph></p>`
- 요소 사이에 줄바꿈 없는 공백만 있습니다. 예: `<emph>a</emph> <emph>b</emph>`의 가운데 공백은 내용입니다.
- 같은 이름의 자식 요소가 떨어져서 다시 나옵니다. 예: `p, list, p`

원문 보관 요소는 워크북의 `xml` 셀이나, `Document` 시트에서 `kind`가 `xml`인 행에 들어갑니다.

### `var`, `catgry`, `varGrp`, `dataDscr`

이 네 요소는 요소만 담습니다. 워크북을 되돌릴 때 도구가 자식을 스키마 순서로 정렬합니다. 그래서 자식 사이에 텍스트가 없으면 원문 보관하지 않습니다.

- 자식 요소 사이의 공백은 줄바꿈이 없어도 버립니다.
- 같은 이름의 자식이 떨어져 있어도, 자식 요소로 나누어 열과 시트에 넣습니다.
- 자식 요소 사이에 공백이 아닌 텍스트가 있으면, 그 텍스트는 내용입니다. 그래서 요소를 원문 그대로 보관합니다.

원문 보관한 요소가 들어가는 곳은 아래와 같습니다.

| 요소 | 들어가는 곳 |
|---|---|
| `var`, `varGrp` | 그 행의 `xml` 셀 |
| `catgry` | 그 변수의 `xml` 셀 |
| `dataDscr` | `Document` 시트에서 `kind`가 `xml`인 행 |

원문 보관한 `dataDscr` 안의 변수는 `Variables` 시트에 나오지 않습니다.

원문이 셀 하나(32,767자)에 들어가지 않으면, 도구가 위치를 표시한 오류로 거부합니다.

```text
dataDscr/var[3]: <var> holds text between its child elements, so it is kept as written in one cell, and its 40000 characters are more than the 32767 a cell can hold
```

## JSON 형식

XML과 JSON은 같은 트리를 담습니다.

| 규칙 | 예 |
|---|---|
| 루트는 `{"codeBook": …}`입니다. | |
| 속성은 `@`로 시작하는 키입니다. | `"@ID"`, `"@xml:lang"` |
| 텍스트만 있고 속성이 없는 요소는 문자열입니다. | `"Walla"` |
| 텍스트와 속성이 함께 있으면, 텍스트는 `#text`에 들어갑니다. | `{"@agency": "walla", "#text": "…"}` |
| 원문 보관 요소의 내용은 `#xml`에 XML 원문으로 들어갑니다. | |
| XSD에서 반복할 수 있는 요소는 한 번만 나와도 배열입니다. | `var`, `catgry`, `labl`, `IDNo` |

읽을 때는 배열 자리에 값이 하나만 있어도 받습니다.

아래는 도구가 쓰는 JSON의 예입니다. 값은 합성한 값입니다.

```json
{
  "codeBook": {
    "@xmlns": "ddi:codebook:2_5",
    "@version": "2.5",
    "@ID": "CB1",
    "stdyDscr": [
      {
        "citation": [
          {
            "titlStmt": {
              "titl": ["합성 설문"],
              "IDNo": [{ "@agency": "walla", "#text": "SampleForm0000000001" }]
            }
          }
        ]
      }
    ]
  }
}
```

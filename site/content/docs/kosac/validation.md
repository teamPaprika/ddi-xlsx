---
title: XML 검사와 양식 파일
weight: 3
---

## XML 검사(`check.ValidateXML`)

공식 DDI 2.5 XSD에는 다른 네임스페이스의 요소를 받는 자리가 없습니다. 그래서 kosac 요소가 있는 코드북은 공식 XSD를 그대로 통과하지 않습니다. `ValidateXML`은 아래 네 단계로 검사하고, 찾은 문제를 모두 `R`로 냅니다.

| 단계 | 검사 | 검사하는 곳 |
|---|---|---|
| PLACE | kosac 요소와 속성이 정해진 DDI 요소 아래에만 있습니다. | Go 코드 |
| REF | ID 형식과 유일성, ID를 가리키는 참조가 끊기지 않았습니다. libxml2는 IDREF의 대상을 확인하지 않습니다. | Go 코드 |
| XSD-DDI | kosac 요소와 속성을 모두 걷어낸 문서(DDI 투영)가 공식 DDI 2.5 XSD를 통과합니다. | `check.Validator` |
| XSD-KOSAC | kosac 내용만 모은 문서(kosac 투영)가 kosac 3.3 XSD를 통과합니다. | `check.Validator` |

- XSD-DDI와 XSD-KOSAC는 호출자가 넘긴 `check.Validator`가 실행합니다. 엔진은 외부 프로세스를 실행하지 않기 때문입니다.
- `Validator`가 `nil`이면, XSD-DDI와 XSD-KOSAC를 건너뛰고 PLACE와 REF만 검사합니다.
- XSD 파일은 `xsd.FS`(DDI 2.5)와 `kosac/xsd.FS`(kosac 3.3)로 공개합니다. 두 패키지의 이름이 모두 `xsd`입니다. import할 때 별칭을 붙이세요.
- `codebook.xsd`는 다른 XSD 파일을 상대 경로로 가져옵니다. `Validator`를 구현하면, FS 전체를 한 디렉터리에 푸세요. 예는 CLI의 구현(`cmd/ddi-xlsx/xmllint.go`)에 있습니다.
- 두 투영의 구조와, 요청서와 다르게 정한 점은 `kosac/xsd/CHANGES.md`에 있습니다.

## 양식 파일

### 임시 내장본

`kosac-template`, `template.Blank`, `template.Write`가 쓰는 빈 양식은 임시본입니다. 재단의 작성 예시에서 값을 지워 만들었습니다. `_meta` 시트에 `NOTE | 임시 내장본, 원본 교체 예정`이 있습니다.

재단이 공식 빈 양식을 주면, `kosac/template/blank_v3.2.xlsx`만 바꿉니다.

### 목록이 걸린 행

목록(데이터 유효성)은 원본 양식의 범위까지만 걸려 있습니다.

| 시트 | 목록이 걸린 마지막 행 |
|---|---|
| `02_데이터파일` | 23행 |
| `03_변수정의` | 3003행 |
| `04_코드값` | 목록 없음 |

이 범위 아래의 행에서는 Excel이 목록 밖의 값을 막지 않습니다. 그 행은 `kosac-check`로 검사하세요.

### 검사용 예시

`testdata/kosac/`에는 재단 작성 예시, XML·JSON 변환 결과, 검사 결과가 있습니다. 응답자별 응답 원자료는 없습니다. 출처와 용도는 [`testdata/kosac/README.md`](https://github.com/teamPaprika/ddi-xlsx/blob/main/testdata/kosac/README.md)에 있습니다.

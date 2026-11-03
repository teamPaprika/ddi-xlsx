---
title: 빈 양식으로 시작하기
weight: 8
---

DDI 파일이 없으면, 빈 양식 워크북을 채워서 DDI를 만듭니다.

## 절차

{{% steps %}}

### 빈 양식 만들기

```bash
ddi-xlsx template -o my-study.xlsx
```

`-o`가 없으면, 도구는 현재 디렉터리에 `template.xlsx`를 만듭니다. 그 파일이 이미 있으면, 도구는 덮어쓰지 않고 오류로 끝납니다. `-o`로 지정한 경로는 덮어씁니다.

### 연구 제목 쓰기

`Document` 시트에서 `titl` 행의 `text` 셀에 연구 제목을 쓰세요.

```text
level  element    text
0      codeBook
1      stdyDscr
2      citation
3      titlStmt
4      titl       연구 제목        (여기에 씁니다)
1      dataDscr
```

### 변수, 값 라벨, 그룹 쓰기

| 시트 | 쓰는 내용 |
|---|---|
| `Variables` | 변수마다 한 행을 쓰세요. `name`, `label`, `question` 같은 열을 채우세요. `ID`는 비워 둘 수 있습니다. |
| `Categories` | 값 라벨마다 한 행을 쓰세요. `varName`에 변수 이름을, `code`와 `label`에 값과 라벨을 쓰세요. 결측값이면 `missing`에 `Y`를 쓰세요. |
| `Groups` | 변수 묶음이 있을 때만 쓰세요. `vars`에 변수 이름이나 ID를 공백으로 나누어 쓰세요. |

각 열의 뜻은 [워크북 구조](workbook.md)에 있습니다.

### DDI로 바꾸기

```bash
ddi-xlsx from-xlsx my-study.xlsx --xml my-study.xml --json my-study.json
```

이 명령은 `my-study.xml`과 `my-study.json`을 만듭니다. 지정한 경로에 파일이 있으면 덮어씁니다.

{{% /steps %}}

## 빈 양식의 내용

- 시트는 `Variables`, `Categories`, `Groups`, `Document` 네 개입니다.
- `Variables`, `Categories`, `Groups`에는 머리글 행만 있습니다.
- `Document` 시트에는 DDI 2.5 XSD를 통과하는 가장 작은 문서가 있습니다.
- 시트마다 데이터 영역이 Excel 표입니다. 표 바로 아래 행에 입력하면, Excel이 표를 넓힙니다. 표가 넓어지지 않아도 도구는 그 행을 읽습니다.
- 아무것도 채우지 않은 양식도 DDI로 바꿀 수 있습니다. 그 결과도 XSD를 통과합니다.

## ID 자동 부여

워크북을 DDI로 바꿀 때(`from-xlsx`), 도구가 빈 ID를 채웁니다.

| 시트 | 붙이는 ID |
|---|---|
| `Variables` | `V1`, `V2`, `V3` 순서입니다. 번호는 1부터 행 순서대로 붙입니다. |
| `Groups` | `VG1`, `VG2` 순서입니다. |

- 도구는 이미 쓰인 ID를 건너뜁니다. `Variables`의 `ID`와 `questionID`, `Groups`의 `ID`를 통틀어 봅니다. 예를 들어 `V1`을 직접 쓴 행이 있으면, `ID`가 빈 행 두 개는 `V2`와 `V3`이 됩니다.
- 붙인 ID는 출력 DDI에 들어갑니다. 그 출력을 다시 워크북으로 바꾸면 `ID` 열이 채워집니다. 다시 DDI로 바꾸면 같은 출력이 나옵니다.
- 빈 행에는 ID를 붙이지 않습니다.
- `ID` 열이 비어 있고 `attributes` 셀에 `ID`가 있으면, 셀 주소와 함께 오류가 나옵니다. ID는 `ID` 열에 쓰세요.
- DDI를 워크북으로 바꿀 때(`to-xlsx`)는 ID를 붙이지 않습니다. 입력 DDI의 모든 `var`와 `varGrp`에는 `ID`가 있어야 합니다.

{{< callout type="warning" >}}
도구는 `ID`가 빈 행에 변환할 때마다 번호를 다시 붙입니다. 그래서 행을 넣거나 지우면, 다른 변수의 ID가 바뀝니다. 다른 곳에서 변수를 ID로 가리키면, 그 ID를 `Variables`의 `ID` 열에 직접 쓰세요.
{{< /callout >}}

## 이름으로 연결

`Categories`와 `Groups`는 변수를 ID 대신 이름으로 가리킬 수 있습니다. 도구는 이름을 `Variables` 시트의 `name`과 비교합니다. 비교 전에 셀의 앞뒤 공백을 지웁니다. 대소문자는 구분합니다.

### `Categories`

| `varID` | `varName` | 결과 |
|---|---|---|
| 있음 | 무시 | ID로만 찾습니다. |
| 없음 | 있음 | 그 이름의 변수가 정확히 하나일 때만 연결합니다. |
| 없음 | 있음, 변수가 없거나 둘 이상 | `varName` 셀 주소와 함께 오류가 나옵니다. |
| 없음 | 없음, 다른 셀은 있음 | `varID` 셀 주소와 함께 오류가 나옵니다. |

여러 파일로 된 데이터셋처럼 변수 이름이 겹치면, `varID`를 쓰세요.

### `Groups`

도구는 `vars`의 항목마다 아래 순서로 찾습니다.

1. 항목이 `Variables` 시트의 ID이면, 그대로 둡니다.
2. 항목이 ID가 아니고 그 이름의 변수가 정확히 하나이면, 그 변수의 ID로 바꿉니다.
3. 둘 다 아니면, 경고를 내고 그대로 둡니다. 오류는 아닙니다.

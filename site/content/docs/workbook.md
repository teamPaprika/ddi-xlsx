---
title: 워크북 구조
weight: 11
---

워크북에는 시트가 네 개 있습니다.

| 시트 | 한 행의 내용 | Excel 표 이름 |
|---|---|---|
| `Variables` | `var` 하나 | `ddiVariables` |
| `Categories` | `catgry` 하나 | `ddiCategories` |
| `Groups` | `varGrp` 하나 | `ddiGroups` |
| `Document` | 나머지 문서의 요소 하나 | `ddiDocument` |

## 공통 규칙

### 셀과 서식

- 도구는 모든 셀을 문자열로 씁니다.
- 모든 열에 텍스트 서식(`@`)이 열 단위로 걸려 있습니다. 그래서 빈 행에 새로 입력한 `01`도 `01`로 남습니다.
- 머리글 행은 틀 고정되어 있습니다.
- 읽을 때는 셀에 표시된 문자열을 씁니다. 숫자로 저장된 셀도 읽습니다.
- 빈 행은 건너뜁니다.

### 열과 표

- 도구는 열을 머리글 이름으로 찾습니다. 그래서 열 순서를 바꿀 수 있습니다.
- 도구가 모르는 열은 경고와 함께 무시합니다. 같은 머리글이 두 번 나오면 오류입니다.
- 머리글 행은 Excel 표의 첫 행입니다. 그래서 표 위에 제목 행을 넣을 수 있습니다.
- 표 바로 아래에 이어서 입력한 행도 읽습니다.
- 표의 요약 행(Total Row)은 읽지 않습니다. 요약 행 아래에 입력한 행은 읽습니다.
- 표가 없으면(범위로 변환했거나 다른 도구로 만든 파일), 1행을 머리글로 봅니다.
- 시트의 기준 열이 없으면, `Variables!A1: …`처럼 머리글 행의 A열을 표시한 오류가 나옵니다. 기준 열은 `Variables`·`Groups`의 `ID`, `Categories`의 `varID`, `Document`의 `level`·`element`입니다.
- 다른 열이 없으면, 그 열의 내용을 읽지 못한다는 경고가 나옵니다.

### 드롭다운 목록

값이 정해진 열에는 드롭다운 목록이 있습니다. 목록의 값은 DDI 2.5 XSD가 정한 값입니다.

| 시트 | 열 |
|---|---|
| `Variables` | `intrvl`, `responseDomainType`, `format`, `formatSchema`, `formatCategory` |
| `Categories` | `missing` |
| `Groups` | `type` |

- 목록에 없는 값을 입력하면 Excel이 경고합니다. 하지만 입력을 막지는 않습니다. 이 동작은 입력 DDI에 있던 표준 밖의 값을 남깁니다.
- 붙여 넣은 값은 Excel이 검사하지 않습니다. 그래서 `from-xlsx`가 목록에 없는 값을 찾으면 셀 주소와 함께 경고합니다. 값은 그대로 출력에 들어갑니다.

## 공통 열: `attributes`, `xml`

`Variables`, `Categories`, `Groups`의 모든 행에는 아래 두 열이 있습니다.

| 열 | 내용 |
|---|---|
| `attributes` | 다른 열에 들어가지 않는 그 요소의 속성입니다. XML 속성 문법으로 씁니다. 예: `dcml="0" nature="nominal"` |
| `xml` | 다른 열에 들어가지 않는 자식 요소의 XML 원문입니다. 예: `<location fileid="F1" StartPos="1"/><sumStat type="vald">580</sumStat>` |

- 내용이 행에 붙어 있습니다. 그래서 행을 옮겨도 다른 시트와의 연결이 유지됩니다.
- 같은 속성을 그 속성의 열과 `attributes` 셀에 함께 쓰면 오류입니다. 예: `name` 열과 `attributes` 셀의 `name="…"`
- `xml` 셀에서는 루트 `codeBook`에 선언된 접두사(예: `xhtml:`)를 쓸 수 있습니다. 선언되지 않은 접두사를 쓰면 셀 주소와 함께 경고가 나옵니다. 네임스페이스를 따지는 프로그램은 그 출력을 거부하기 때문입니다. `Document` 시트의 `element` 셀도 같은 규칙입니다.
- 셀 내용이 잘 구성된 XML이 아니면, 셀 주소와 함께 오류가 나옵니다.
- 셀 안의 주석과 처리 명령(PI)은 셀 주소와 함께 경고를 내고 버립니다.
- 요소에 텍스트만 있으면(예: `<catgry>x</catgry>`), 그 텍스트가 `xml` 셀에 들어갑니다. 이때 그 행의 다른 내용 열과, 그 요소를 가리키는 다른 시트의 행은 비어 있어야 합니다.

{{< callout type="warning" >}}
변수의 `ID`를 고치면, 그 변수를 가리키는 `Categories`의 `varID`와 `Groups`의 `vars`도 고치세요. `varID`를 고치지 않으면 오류가 나옵니다. `vars`를 고치지 않으면 경고가 나옵니다.
{{< /callout >}}

## `Variables`

행 순서가 출력의 `var` 순서입니다.

| 열 | DDI | 열에 들어가는 조건 |
|---|---|---|
| `ID` | `var/@ID` | `var`와 `varGrp`를 통틀어 고유해야 합니다. 비워 두면 도구가 붙입니다. [ID 자동 부여](blank-template.md#id-자동-부여)를 읽으세요. |
| `name` | `var/@name` | |
| `label` | `labl` | `labl`이 하나이고, 속성이 없고, 텍스트만 있을 때 |
| `question` | `qstn/qstnLit` | `qstn`이 하나이고, 그 자식이 속성 없는 텍스트 `qstnLit` 하나뿐이고, `qstn`의 속성이 `ID`와 `responseDomainType`뿐일 때 |
| `questionID` | `qstn/@ID` | `question`과 같은 조건 |
| `questionRef` | `var/@qstn` | 값이 비어 있지 않고, 앞뒤 공백과 둘째 줄부터의 줄 앞 공백이 없을 때. 조건에 맞지 않으면 `attributes` 셀에 들어갑니다. |
| `responseDomainType` | `qstn/@responseDomainType` | `question`과 같은 조건 |
| `intrvl` | `var/@intrvl` | |
| `format` | `varFormat/@type` | `varFormat`이 하나이고, 내용이 없고, 속성이 있으며 그 속성이 `type`·`schema`·`category`뿐일 때 |
| `formatSchema` | `varFormat/@schema` | `format`과 같은 조건 |
| `formatCategory` | `varFormat/@category` | `format`과 같은 조건 |
| `notes` | `notes` | 속성이 없는 한 줄 텍스트 `notes`입니다. 한 줄에 하나씩 씁니다. 조건에 맞지 않는 `notes`가 나오면, 그 `notes`와 뒤의 `notes`는 `xml` 셀에 들어갑니다. |
| `files` | `var/@files` | `questionRef`와 같은 조건 |
| `locations` | `location/@fileid` | 아래 설명을 읽으세요. |
| `attributes`, `xml` | 공통 열 | |

- 조건에 맞지 않는 요소는 `xml` 셀에 들어갑니다. `catgry`는 `Categories` 시트에 들어갑니다.
- 출력의 `var` 속성 순서는 `ID`, `name`, `files`, `intrvl`, `qstn`이고, 그다음이 `attributes` 셀의 속성입니다. 이 순서는 Walla가 쓰는 순서와 같습니다.

### `locations`

- 그 변수의 모든 `location`이 아래 조건에 맞으면, `fileid` 값을 문서 순서대로 공백으로 나누어 씁니다. 예: `F1 F2`
  - 속성이 `fileid` 하나뿐입니다.
  - 값이 비어 있지 않고 공백이 없습니다.
  - 내용이 없습니다.
- `location` 하나라도 조건에 맞지 않으면, 그 변수의 `location`은 모두 `xml` 셀에 들어갑니다. 이 규칙은 `location` 사이의 순서를 지킵니다.
- 되돌릴 때 도구는 공백이나 줄바꿈으로 나눈 값마다 `<location fileid="…"/>`를 하나 만듭니다. 그리고 그 변수의 자식 중 맨 앞에 씁니다.

## `Categories`

| 열 | DDI | 열에 들어가는 조건 |
|---|---|---|
| `varID` | 소속 `var`의 ID | 비워 두면 `varName`으로 변수를 찾습니다. |
| `varName` | 소속 `var`의 `name` | 도구가 채웁니다. `varID`가 비어 있을 때만 읽습니다. |
| `code` | `catValu` | `catValu`가 하나이고, 속성이 없고, 텍스트만 있을 때 |
| `label` | `labl` | `Variables`의 `label`과 같은 조건 |
| `missing` | `catgry/@missing` | 값을 그대로 씁니다(`Y`, `N`). 빈칸은 속성이 없다는 뜻입니다. |
| `freq` | `catStat` | 첫 번째 `catStat`의 속성이 `type="freq"` 하나뿐이고, 그런 `catStat`이 하나뿐일 때 |
| `attributes`, `xml` | 공통 열 | `missType`, `excls`, 가중 `catStat` 같은 내용이 들어갑니다. |

- 한 변수 안에서 `catgry` 순서는 행 순서입니다.
- 도구가 만든 워크북은 변수를 이름이 아니라 ID로 가리킵니다. 여러 파일로 된 데이터셋에서는 변수 이름이 겹치기 때문입니다. 이름으로 가리키는 규칙은 [이름으로 연결](blank-template.md#이름으로-연결)에 있습니다.

## `Groups`

| 열 | DDI |
|---|---|
| `ID` | `varGrp/@ID`. 비워 두면 도구가 붙입니다. |
| `type` | `varGrp/@type` |
| `name` | `varGrp/@name` |
| `label` | `labl`. `Variables`의 `label`과 같은 조건입니다. |
| `vars` | `varGrp/@var`. 변수 ID를 공백으로 나누어 씁니다. 변수 이름을 쓰면 도구가 ID로 바꿉니다. |
| `attributes`, `xml` | 공통 열 |

`vars`의 항목이 `Variables` 시트의 ID가 아니고 정확히 한 변수의 이름도 아니면, 경고가 나옵니다. 변환은 계속합니다.

## `Document`

`Document` 시트는 다른 세 시트에 들어가지 않는 내용을 개요(outline) 형태로 담습니다.

- 루트 속성
- `docDscr`, `stdyDscr`, `fileDscr`, `otherMat`
- `dataDscr` 안에서 `var`·`varGrp`가 아닌 자식

| 열 | 뜻 |
|---|---|
| `level` | 깊이입니다. `codeBook`이 0입니다. |
| `element` | 요소 이름입니다. |
| `attributes` | XML 속성 문법으로 쓴 속성입니다. |
| `text` | 텍스트 값입니다. 원문 보관 요소이면 안쪽 XML 원문입니다. |
| `kind` | 빈칸이거나, 원문 보관이면 `xml`입니다. |

- 행 순서가 출력 순서입니다. 예외는 `dataDscr`의 자식과, 떨어져 있는 같은 이름의 형제입니다. [알려진 손실과 변경](roundtrip.md#알려진-손실과-변경)을 읽으세요.
- `element` 셀은 `level`만큼 들여쓰기되어 보입니다. 이 들여쓰기는 보기 위한 것입니다.
- 첫 행은 `level` 0의 `codeBook`이고, `xmlns="ddi:codebook:2_5"` 속성이 있어야 합니다.
- 어떤 행의 `level`은 바로 위 행보다 2 이상 깊을 수 없습니다.
- `text`가 있는 행 아래에는 자식 행을 둘 수 없습니다.
- 도구는 `dataDscr` 안에 다른 시트의 `var`와 `varGrp`를 스키마가 정한 자리에 끼워 넣습니다. 이때 `Document` 시트에 있는 `dataDscr`의 다른 자식도 스키마 순서로 정렬합니다.

{{< callout type="error" >}}
`Document` 표를 정렬하지 마세요. 행 순서가 개요의 구조입니다. 정렬하면 요소가 다른 부모 아래로 옮겨집니다.
{{< /callout >}}

## 자주 하는 편집

### 라벨 고치기

- 변수 라벨은 `Variables` 시트에서 그 변수 행의 `label` 셀을 고치세요.
- 값 라벨은 `Categories` 시트의 `label` 셀을 고치세요.

`label` 셀이 비어 있으면, 그 라벨은 열의 조건에 맞지 않아서 `xml` 셀에 있습니다. 예를 들어 `labl`이 둘 이상이거나 `xml:lang` 같은 속성이 있습니다. 이때는 `xml` 셀 안의 `<labl>` 원문을 고치세요.

### 범주 추가

1. `Categories` 시트에서 그 변수의 마지막 범주 행 아래에 행을 넣으세요.
2. `varID`에 변수의 ID를 쓰세요. `varID`를 쓰면 `varName`은 비워 둘 수 있습니다.
3. `code`와 `label`에 값과 라벨을 쓰세요.
4. 결측값이면, `missing`에 `Y`를 쓰세요.

### 키워드 추가

1. `Document` 시트에서 `stdyDscr` → `stdyInfo` → `subject` 행을 찾으세요.
2. 그 아래 마지막 `keyword` 행 다음에 행을 넣으세요.
3. `level`은 `subject`보다 1 크게, `element`는 `keyword`로, `text`는 키워드로 쓰세요.

```text
level  element   text
3      subject
4      keyword   households
4      keyword   새 키워드        (추가한 행)
```

`subject` 행이 없으면, 아래 두 경우 중 하나입니다.

- `stdyInfo` 행이 있으면, 그 아래에 `subject`(level 3)와 `keyword`(level 4) 행을 넣으세요. `subject`는 `stdyInfo`의 첫 자식 자리입니다. `studyBudget`이 있으면 그 다음 자리입니다.
- `stdyInfo` 행도 없으면, `stdyInfo`(level 2), `subject`(level 3), `keyword`(level 4) 행을 넣으세요. `stdyInfo`는 `stdyDscr` 안에서 `citation` 다음 자리입니다. `studyAuthorization`이 있으면 그 다음 자리입니다.

새 행은 앞 요소의 더 깊은 행이 모두 끝난 다음에 넣으세요. 예를 들어 `citation` 다음에 넣으면, `citation` 아래의 `titlStmt` 같은 행이 모두 끝난 다음 줄에 넣으세요.

`Document` 시트는 행 순서가 출력 순서입니다. 그래서 스키마가 정한 자리에 넣어야 출력이 XSD를 통과합니다. 허용되지 않는 부모 아래에 요소를 넣으면 경고가 나옵니다. 하지만 도구는 형제 사이의 순서를 검사하지 않습니다. 결과는 [출력 XML 검사](cli.md#출력-xml-검사)의 `xmllint` 명령으로 검사하세요.

### 다시 계산하지 않는 값

도구는 `fileDscr/fileTxt/dimensns/varQnty`를 그대로 둡니다. 변수를 추가하거나 지워도 이 값은 바뀌지 않습니다. 이 값을 바꾸려면, `Document` 시트에서 직접 고치세요.

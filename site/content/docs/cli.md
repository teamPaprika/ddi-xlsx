---
title: CLI 사용법
weight: 5
---

## 명령

```text
ddi-xlsx to-xlsx   <in.xml|in.json> [-o out.xlsx]
ddi-xlsx from-xlsx <in.xlsx> [--xml out.xml] [--json out.json]
ddi-xlsx template  [-o template.xlsx]
ddi-xlsx kosac-check    <양식.xlsx>
ddi-xlsx kosac-xml      <양식.xlsx> [-o out.xml] [--json out.json]
ddi-xlsx kosac-template [-o kosac-template.xlsx]
```

`kosac-`으로 시작하는 명령은 [KOSAC CLI](kosac/cli.md)에 있습니다. `template` 명령은 [빈 양식으로 시작하기](blank-template.md)에 있습니다.

## DDI를 워크북으로 바꾸기

`to-xlsx`는 DDI XML이나 JSON을 워크북으로 바꿉니다. `-o`가 없으면, 입력 파일의 확장자를 `.xlsx`로 바꾼 경로에 씁니다. 아래 명령은 `codebook.xlsx`를 만듭니다.

```bash
ddi-xlsx to-xlsx codebook.xml
```

출력 경로를 지정하려면 `-o`를 쓰세요. JSON 입력도 같은 방법으로 넣습니다.

```bash
ddi-xlsx to-xlsx codebook.json -o codebook-edit.xlsx
```

## 워크북을 DDI로 바꾸기

`from-xlsx`는 워크북을 DDI XML, JSON, 또는 둘 다로 바꿉니다.

```bash
ddi-xlsx from-xlsx codebook.xlsx --xml codebook-edited.xml
```

```bash
ddi-xlsx from-xlsx codebook.xlsx --xml codebook-edited.xml --json codebook-edited.json
```

`--xml`과 `--json`이 모두 없으면, 입력 파일의 확장자를 `.xml`로 바꾼 경로에 XML만 씁니다.

{{< callout type="info" >}}
원본 `codebook.xml`이 같은 디렉터리에 있으면, `ddi-xlsx from-xlsx codebook.xlsx`는 파일을 쓰지 않고 오류로 끝납니다. 기본 출력 경로에 파일이 이미 있기 때문입니다.
{{< /callout >}}

## 출력 경로 규칙

- 기본 출력 경로에 파일이 이미 있으면, 도구는 덮어쓰지 않고 오류로 끝납니다. 이 규칙은 원본 DDI와 고친 워크북을 보호합니다.
- 기본 출력 경로는 `-o`가 없는 `to-xlsx`·`template`, `--xml`·`--json`이 없는 `from-xlsx`의 경로입니다.
- `-o`, `--xml`, `--json`으로 지정한 경로는 덮어씁니다.
- 출력 경로가 입력 파일과 같으면, 도구가 명령을 거부합니다.
- 옵션은 입력 파일의 앞이나 뒤에 둘 수 있습니다.

## 경고, 오류, 종료 코드

| 결과 | 출력 | 종료 코드 |
|---|---|---|
| 경고 | `warning:`으로 시작하는 줄을 stderr에 씁니다. 변환은 끝까지 진행합니다. | 0 |
| 오류 | `error:`로 시작하는 줄을 stderr에 씁니다. | 1 |
| 명령 형식 오류 | 사용법을 출력합니다. | 2 |

워크북 오류는 셀 주소를 표시합니다. 아래 오류는 `Categories` 시트의 `varID`가 없는 변수를 가리킬 때 나옵니다.

```text
error: Categories!A12: varID "V999" matches no ID on the Variables sheet
```

## 출력 XML 검사

변환이 성공해도 출력이 XSD를 통과한다는 보장은 없습니다. 열 값, `Document` 시트, `xml` 셀을 고쳤으면, 출력 XML을 검사하세요. 아래 명령은 저장소 루트에서 실행합니다.

```bash
xmllint --nonet --noout --schema xsd/ddi-codebook-2.5/codebook.xsd codebook-edited.xml
```

`codebook-edited.xml` 자리에 검사할 파일의 경로를 쓰세요.

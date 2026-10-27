# ddi-xlsx

`ddi-xlsx`는 DDI Codebook 2.5 문서(XML 또는 JSON)를 편집할 수 있는 xlsx 워크북으로 바꾸고, 고친 워크북을 다시 DDI로 되돌리는 도구입니다.

연구자와 아카이브 담당자가 코드북을 스프레드시트에서 읽고 고칠 수 있게 하는 것이 목적입니다. 변수 라벨, 질문 문구, 값 라벨, 결측 표시, 노트, 그룹 구성, 연구 메타데이터는 셀만 고쳐서 바꿀 수 있습니다. 열에 들어가지 않는 내용은 같은 행의 `xml` 셀에 XML 원문으로 보관합니다. 그래서 DDI → xlsx → DDI 왕복에서 내용을 잃지 않습니다(예외는 "알려진 손실과 변경" 참고).

같은 변환기를 세 가지 형태로 제공합니다.

- 명령줄 도구(CLI): macOS, Linux, Windows용 실행 파일입니다.
- Go 라이브러리: `workbook.ToXLSX`, `workbook.FromXLSX`, `workbook.Template`입니다.
- npm 패키지: Node와 웹 브라우저에서 쓰는 WebAssembly 모듈입니다.

## 받는 입력

- DDI Codebook 2.5 문서만 받습니다. 루트 요소는 네임스페이스 `ddi:codebook:2_5`의 `codeBook`이어야 합니다.
- 루트에 접두사를 붙였다면(`<ddi:codeBook>`), 기본 네임스페이스(`xmlns`)는 선언하지 않거나 `ddi:codebook:2_5`로 선언해야 합니다. 출력은 DDI 요소를 접두사 없이 쓰기 때문입니다.
- UTF-8 입력만 받습니다. XML 선언에 다른 인코딩(예: `encoding="EUC-KR"`)이 적혀 있으면 `only UTF-8 input is supported` 오류로 거부합니다. 먼저 파일을 UTF-8로 변환해 주세요.
- 요소는 256단계까지 중첩할 수 있습니다. 더 깊으면 위치를 짚은 오류로 거부합니다.
- 잘 구성된(well-formed) XML이면 됩니다. XSD 유효성은 요구하지 않습니다. 스키마상 자리가 맞지 않는 요소도 거부하지 않습니다. 허용되지 않는 부모 아래에 있는 요소는 워크북을 DDI로 되돌릴 때(`from-xlsx`) 경고가 나옵니다.
- XML과 JSON을 모두 받습니다. 형식은 내용으로 판별합니다. 첫 번째 공백 아닌 문자가 `<`이면 XML이고, `{`이면 JSON입니다.
- DDI 1.2.2(네임스페이스 `http://www.icpsr.umich.edu/DDI`)는 받지 않습니다. 이 도구의 규칙 표는 DDI 2.5 XSD에서 만든 것이라서, 구조가 다른 1.2.2에는 맞지 않습니다. 1.2.2 파일을 넣으면 `DDI 1.2.2 is not supported; convert the file to DDI 2.5 first` 오류가 납니다. 먼저 DDI 2.5로 변환한 뒤 넣어 주세요.
- 그 밖의 네임스페이스도 오류로 거부합니다.
- 모든 `var`와 `varGrp`에 `ID`가 있어야 합니다. 다른 시트가 행을 ID로 가리키기 때문입니다. `ID`가 없거나, 비어 있거나, 앞뒤에 공백이 붙어 있으면 위치를 짚은 오류로 거부합니다.
- `var`와 `varGrp`를 통틀어 같은 `ID`가 두 번 나오면 거부합니다.
- `dataDscr`는 하나만 받습니다.

### 원문 보관

요소마다 내용을 보고, 해석할지 원문 그대로 보관할지 정합니다. 아래 중 하나에 해당하면 그 요소의 안쪽 XML을 원문 그대로 보관합니다.

- 공백이 아닌 텍스트와 요소가 섞여 있습니다. 예: `<p>a <emph>b</emph></p>`
- 요소 사이에 줄바꿈 없이 공백만 있습니다. 예: `<emph>a</emph> <emph>b</emph>`의 가운데 공백은 내용입니다.
- 같은 이름의 자식 요소가 연속하지 않고 다시 나옵니다. 예: `p, list, p`

원문 보관 요소는 워크북에서 `xml` 셀이나 `Document` 시트의 `kind`가 `xml`인 행에 들어갑니다.

`var`, `catgry`, `varGrp`, `dataDscr`는 자식 사이에 텍스트가 없으면 워크북에서 원문 보관하지 않습니다. 이 네 요소는 요소만 담고, 워크북을 되돌릴 때 자식을 스키마 순서로 정렬하기 때문입니다.

- 자식 요소 사이의 공백은 줄바꿈이 없어도 버립니다.
- 같은 이름의 자식이 떨어져 있어도 자식 요소로 나누어 열과 시트에 넣습니다.
- 자식 요소 사이에 공백이 아닌 텍스트가 있으면, 그 텍스트는 내용이므로 요소를 원문 그대로 보관합니다. `var`와 `varGrp`는 자기 행의 `xml` 셀에, `catgry`는 그 변수의 `xml` 셀에, `dataDscr`는 `Document` 시트에서 `kind`가 `xml`인 행에 들어갑니다. 원문 보관한 `dataDscr` 안의 변수는 `Variables` 시트에 나오지 않습니다.
- 원문이 셀 하나(32,767자)에 들어가지 않으면, 위치를 짚은 오류로 거부합니다. 예: `dataDscr/var[3]: <var> holds text between its child elements, so it is kept as written in one cell, and its 40000 characters are more than the 32767 a cell can hold`

### JSON 형식

XML과 JSON은 같은 트리를 담습니다. JSON 규칙은 아래와 같습니다.

- 루트는 `{"codeBook": …}`입니다.
- 속성은 `@`로 시작하는 키에 씁니다. 예: `"@ID"`, `"@xml:lang"`
- 텍스트만 있고 속성이 없는 요소는 문자열로 씁니다. 텍스트와 속성이 함께 있으면 텍스트를 `#text`에 넣습니다.
- 원문 보관 요소의 내용은 `#xml`에 XML 원문으로 넣습니다.
- XSD에서 반복할 수 있는 요소 이름은 한 번만 나와도 배열로 씁니다. 예를 들어 `var`, `catgry`, `labl`, `IDNo`는 언제나 배열입니다.
- 읽을 때는 배열 자리에 값이 하나만 와도 받습니다.

## 설치

### CLI 실행 파일

`scripts/build.sh`를 실행하면 `dist/`에 실행 파일 여섯 개가 생깁니다. 빌드에는 Go 1.25 이상이 필요합니다.

```bash
scripts/build.sh
```

| 파일 | 운영체제 |
|---|---|
| `ddi-xlsx-darwin-arm64` | macOS (Apple silicon) |
| `ddi-xlsx-darwin-amd64` | macOS (Intel) |
| `ddi-xlsx-linux-arm64`, `ddi-xlsx-linux-amd64` | Linux |
| `ddi-xlsx-windows-arm64.exe`, `ddi-xlsx-windows-amd64.exe` | Windows |

파일 하나의 크기는 6.4–6.9 MB입니다. 따로 설치할 런타임은 없습니다. 운영체제에 맞는 파일을 `PATH`에 있는 디렉터리로 복사하고, 이름을 `ddi-xlsx`(Windows는 `ddi-xlsx.exe`)로 바꿔 주세요. 이 문서의 명령 예시는 그 이름을 씁니다.

### Go

현재 플랫폼용 CLI만 빌드하려면 아래 명령을 씁니다.

```bash
go build -o dist/ddi-xlsx ./cmd/ddi-xlsx
```

라이브러리로 쓸 때는 `workbook` 패키지를 가져옵니다. 함수는 파일을 읽거나 쓰지 않고, 바이트를 받아 바이트를 돌려줍니다.

```go
import "github.com/teamPaprika/ddi-xlsx/workbook"

// src는 DDI 2.5 문서(XML 또는 JSON)의 바이트입니다.
xlsx, warnings, err := workbook.ToXLSX(src)

// 워크북을 XML과 JSON으로 되돌립니다. 둘은 같은 트리를 담습니다.
xmlOut, jsonOut, warnings, err := workbook.FromXLSX(xlsx)

// 빈 양식 워크북을 만듭니다("빈 양식으로 시작하기" 참고).
blank, err := workbook.Template()
```

- `warnings`는 경고 문구 목록입니다. 경고가 있어도 변환은 성공한 것입니다.
- 셀 내용 때문에 난 `FromXLSX` 오류는 셀 주소로 시작합니다. 예: `Categories!A12: …`
- 모듈 경로 `github.com/teamPaprika/ddi-xlsx`는 임시 경로입니다. 공개할 때 바뀔 수 있습니다.

### npm (Node와 브라우저)

`npm/` 디렉터리가 npm 패키지입니다. 아직 npm 레지스트리에는 올리지 않았습니다. `ddi-xlsx.wasm`과 `wasm_exec.js`는 빌드 산출물이라 저장소에 없습니다. 먼저 `scripts/build.sh`를 실행한 뒤, 쓰려는 프로젝트에서 로컬 경로로 설치합니다.

```bash
npm install /path/to/ddi-xlsx/npm
```

함수는 세 개입니다.

```ts
ddiToXlsx(input: string | Uint8Array): Promise<{ xlsx: Uint8Array; warnings: string[] }>
xlsxToDdi(xlsx: Uint8Array): Promise<{ xml: string; json: string; warnings: string[] }>
templateXlsx(): Promise<Uint8Array>
```

- `templateXlsx`는 빈 양식 워크북의 바이트를 돌려줍니다("빈 양식으로 시작하기" 참고).

- 변환할 수 없는 입력이면, 변환기의 오류 문구를 담은 `Error`로 거부됩니다. 이때 실패하는 것은 그 호출뿐입니다.
- 메모리가 부족해 변환기가 멈추면, 그 호출은 `ddi-xlsx converter stopped; try again` 오류로 거부됩니다. 다음 호출은 변환기를 새로 로드하므로, 다시 호출하면 됩니다.
- wasm은 처음 호출할 때 한 번만 로드합니다. Node에서는 파일에서 읽고, 브라우저에서는 `fetch`로 받습니다.

Node 예시입니다. `writeFile`은 같은 이름의 파일을 덮어쓰므로, 아래 "CLI 사용법"의 예시와 겹치지 않는 이름을 씁니다.

```js
import { readFile, writeFile } from 'node:fs/promises';
import { ddiToXlsx, xlsxToDdi } from 'ddi-xlsx';

const { xlsx, warnings } = await ddiToXlsx(await readFile('codebook.xml'));
await writeFile('codebook-npm.xlsx', xlsx);

const out = await xlsxToDdi(await readFile('codebook-npm.xlsx'));
await writeFile('codebook-npm-edited.xml', out.xml);
await writeFile('codebook-npm-edited.json', out.json);
```

#### 브라우저에서 쓸 때

- 패키지는 ES 모듈입니다. `index.js`는 `ddi-xlsx.wasm`을 자기와 같은 위치에서 찾습니다(`new URL('./ddi-xlsx.wasm', import.meta.url)`). 번들러를 쓴다면 이 파일이 함께 배포되는지 확인해 주세요.
- 서버는 `.wasm` 파일을 `Content-Type: application/wasm`으로 보내야 합니다. 다른 타입으로 보내면 브라우저가 모듈을 로드하지 않습니다.
- wasm 파일은 11.4 MB이고, gzip으로 압축하면 약 2.9 MB입니다.
- 변환은 호출한 스레드에서 끝날 때까지 실행됩니다. 메인 스레드에서 호출하면 그동안 화면이 멈춥니다.

#### 큰 코드북

브라우저에서 큰 코드북을 변환할 때는 Web Worker 안에서 실행하기를 권장합니다. 아래는 합성 코드북 두 개로 측정한 결과입니다.

- 파일 A: 변수 700개, 변수마다 범주 40개입니다. `Categories` 시트가 10.5 MB입니다.
- 파일 B: 변수 3,358개, 변수마다 범주 10개입니다.

| 런타임 | 파일 | 걸린 시간 | 메모리 |
|---|---|---|---|
| Node 22의 wasm | A | DDI → xlsx 약 1.3초, xlsx → DDI 약 2.6초 | 약 0.7 GB |
| CLI(네이티브) | A | DDI → xlsx 약 0.22초, xlsx → DDI 약 0.44초 (왕복 약 0.7초) | 측정하지 않음 |
| CLI(네이티브) | B | 왕복 약 0.95초 | 측정하지 않음 |

브라우저에서는 큰 코드북의 시간과 메모리를 측정하지 않았습니다.

wasm 안에서 메모리가 부족해 치명적 오류가 나면, 그 호출은 `ddi-xlsx converter stopped; try again` 오류로 거부됩니다. 다음 호출은 wasm을 새로 로드하므로, 페이지를 새로 고치거나 프로세스를 다시 시작하지 않아도 됩니다. 같은 입력은 다시 실패할 수 있으니, 그런 코드북은 CLI로 변환해 주세요.

## CLI 사용법

```text
ddi-xlsx to-xlsx   <in.xml|in.json> [-o out.xlsx]
ddi-xlsx from-xlsx <in.xlsx> [--xml out.xml] [--json out.json]
ddi-xlsx template  [-o template.xlsx]
```

DDI를 워크북으로 바꿉니다. `-o`를 생략하면 입력 파일의 확장자를 `.xlsx`로 바꾼 경로에 씁니다. 아래 명령은 `codebook.xlsx`를 만듭니다.

```bash
ddi-xlsx to-xlsx codebook.xml
```

출력 파일을 직접 정할 수도 있습니다. JSON 입력도 같은 방법으로 넣습니다.

```bash
ddi-xlsx to-xlsx codebook.json -o codebook-edit.xlsx
```

워크북을 DDI XML로 되돌립니다.

```bash
ddi-xlsx from-xlsx codebook.xlsx --xml codebook-edited.xml
```

XML과 JSON을 함께 쓸 수도 있습니다.

```bash
ddi-xlsx from-xlsx codebook.xlsx --xml codebook-edited.xml --json codebook-edited.json
```

`--xml`과 `--json`을 모두 생략하면, 입력 파일의 확장자를 `.xml`로 바꾼 경로에 XML만 씁니다. 그런데 위 예시처럼 원본 `codebook.xml`이 같은 디렉터리에 있으면, 아래 명령은 아무 파일도 쓰지 않고 오류로 끝납니다.

```bash
ddi-xlsx from-xlsx codebook.xlsx
```

출력 규칙은 아래와 같습니다.

- 도구가 정한 기본 출력 경로(`-o` 없는 `to-xlsx`, `--xml`·`--json` 없는 `from-xlsx`, `-o` 없는 `template`)에 파일이 이미 있으면, 덮어쓰지 않고 오류로 끝납니다. 원본 DDI나 고친 워크북을 실수로 지우지 않기 위해서입니다. 이때는 출력 경로를 직접 지정해 주세요.
- `-o`, `--xml`, `--json`으로 직접 지정한 경로는 덮어씁니다.
- 출력 경로가 입력 파일과 같으면 거부합니다.
- 옵션은 입력 파일 앞에 와도 되고, 뒤에 와도 됩니다.

경고와 오류는 아래와 같이 나옵니다.

- 경고는 `warning:`으로 시작하는 줄로 stderr에 씁니다. 경고가 있어도 변환은 끝까지 진행합니다.
- 오류는 `error:`로 시작하는 줄로 stderr에 쓰고, 종료 코드 1로 끝납니다.
- 명령 형식이 틀리면 사용법을 출력하고 종료 코드 2로 끝납니다.

워크북의 오류는 셀 주소를 짚습니다. 예를 들어 `Categories` 시트의 `varID`가 없는 변수를 가리키면 아래 오류가 납니다.

```text
error: Categories!A12: varID "V999" matches no ID on the Variables sheet
```

## 빈 양식으로 시작하기

DDI 파일이 없어도, 빈 양식 워크북을 채워서 DDI를 만들 수 있습니다.

먼저 빈 양식을 만듭니다. 아래 명령은 현재 디렉터리에 `template.xlsx`를 만듭니다. 그 파일이 이미 있으면 덮어쓰지 않고 오류로 끝납니다.

```bash
ddi-xlsx template
```

파일 이름을 직접 정할 수도 있습니다. 이때는 같은 이름의 파일이 있으면 덮어씁니다. 이 절의 나머지 예시는 아래 명령으로 만든 `my-study.xlsx`를 채운다고 보고 설명합니다. 앞 절에서 만든 `codebook` 파일들을 덮어쓰지 않도록 새 이름을 씁니다.

```bash
ddi-xlsx template -o my-study.xlsx
```

양식에는 시트 네 개가 있습니다. `Variables`, `Categories`, `Groups`에는 머리글 행만 있습니다. `Document` 시트에는 DDI 2.5 XSD를 통과하는 가장 작은 문서가 들어 있습니다. 연구 제목은 `titl` 행의 `text` 셀에 씁니다.

```text
level  element    text
0      codeBook
1      stdyDscr
2      citation
3      titlStmt
4      titl       연구 제목        (여기에 씁니다)
1      dataDscr
```

양식을 아래와 같이 채웁니다. 각 열의 뜻은 "워크북 구조"를 참고해 주세요.

- `Variables`: 변수마다 한 행을 씁니다. `name`, `label`, `question` 등을 채웁니다. `ID`는 비워 두어도 됩니다.
- `Categories`: 값 라벨마다 한 행을 씁니다. `varName`에 변수 이름을 쓰고, `code`와 `label`에 값과 라벨을 씁니다. 결측값이면 `missing`에 `Y`를 씁니다.
- `Groups`: 변수 묶음이 있을 때만 씁니다. `vars`에 변수 이름이나 ID를 공백으로 구분해 씁니다.

시트마다 데이터 영역이 Excel 표입니다. 표 바로 아래 행에 입력하면 Excel이 보통 표를 넓힙니다. 표가 넓어지지 않아도 도구는 그 행을 읽습니다.

다 채웠으면 DDI로 바꿉니다. 아래 명령은 `my-study.xml`과 `my-study.json`을 만듭니다. 직접 지정한 경로이므로, 같은 이름의 파일이 있으면 덮어씁니다.

```bash
ddi-xlsx from-xlsx my-study.xlsx --xml my-study.xml --json my-study.json
```

아무것도 채우지 않은 양식도 DDI로 바꿀 수 있고, 그 결과도 XSD를 통과합니다.

### ID 자동 부여

워크북을 DDI로 바꿀 때(`from-xlsx`), 비어 있는 ID를 도구가 붙입니다.

- `Variables`의 `ID`가 비어 있으면 `V1`, `V2`, `V3` 같은 ID를 붙입니다. 번호는 1부터 행 순서대로 매깁니다.
- `Groups`의 `ID`가 비어 있으면 같은 방식으로 `VG1`, `VG2` 같은 ID를 붙입니다.
- 이미 쓰인 ID는 건너뜁니다. `Variables`의 `ID`와 `questionID`, `Groups`의 `ID`를 통틀어 봅니다. 예를 들어 `V1`을 직접 쓴 행이 있으면, `ID`가 빈 행 두 개는 `V2`와 `V3`이 됩니다.
- 붙인 ID는 출력 DDI에 들어갑니다. 그 출력을 다시 워크북으로 바꾸면 `ID` 열이 채워져 있고, 다시 DDI로 바꾸면 같은 출력이 나옵니다.
- `ID`가 비어 있는 행은 변환할 때마다 행 순서대로 번호를 다시 매깁니다. 그래서 행을 넣거나 지우면 다른 변수의 ID가 바뀔 수 있습니다. 다른 곳에서 변수를 ID로 가리키려면, 그 ID를 `Variables`의 `ID` 열에 직접 써 주세요.
- 빈 행에는 ID를 붙이지 않습니다.
- `ID` 열이 비어 있는데 `attributes` 셀에 `ID`가 있으면, 그 셀 주소와 함께 오류가 납니다. ID는 `ID` 열에 써 주세요.

DDI를 워크북으로 바꿀 때(`to-xlsx`)는 ID를 붙이지 않습니다. 입력 DDI의 모든 `var`와 `varGrp`에는 `ID`가 있어야 합니다("받는 입력" 참고).

### 이름으로 연결

`Categories`와 `Groups`에서는 변수를 ID 대신 이름으로 가리킬 수 있습니다. 이름은 `Variables` 시트의 `name`과 비교합니다. 셀의 앞뒤 공백은 지운 뒤 비교하고, 대소문자는 구분합니다.

`Categories` 시트의 규칙은 아래와 같습니다.

- `varID`가 채워져 있으면 ID로만 찾습니다. `varName`은 읽지 않습니다.
- `varID`가 비어 있으면 `varName`으로 찾습니다. 그 이름의 변수가 정확히 하나일 때만 연결합니다.
- 그 이름의 변수가 없거나 둘 이상이면, `varName` 셀 주소와 함께 오류가 납니다. 파일이 여러 개인 데이터셋처럼 이름이 겹친다면 `varID`를 써 주세요.
- `varID`와 `varName`이 모두 비어 있는데 다른 셀이 채워져 있으면, `varID` 셀 주소와 함께 오류가 납니다.

`Groups` 시트의 `vars`는 항목마다 아래 순서로 찾습니다.

1. `Variables` 시트의 ID이면 그대로 둡니다.
2. ID가 아니고, 그 이름의 변수가 정확히 하나이면 그 변수의 ID로 바꿔 씁니다.
3. 둘 다 아니면 경고를 내고 그대로 둡니다. 오류는 아닙니다.

## 워크북 구조

시트는 네 개입니다: `Variables`, `Categories`, `Groups`, `Document`.

- 모든 셀은 문자열로 씁니다.
- 모든 열에 텍스트 서식(`@`)을 열 단위로 겁니다. 그래서 빈 행에 새로 입력한 `01`도 `01`로 남습니다.
- 머리글 행은 틀 고정되어 있습니다.
- 읽을 때는 셀에 표시된 문자열을 씁니다. 숫자로 저장된 셀도 읽습니다.
- 열은 머리글 이름으로 찾습니다. 열 순서를 바꾸어도 됩니다. 도구가 모르는 열은 경고와 함께 무시합니다. 같은 머리글이 두 번 나오면 오류입니다.
- 시트마다 데이터 영역이 Excel 표(`ddiVariables`, `ddiCategories`, `ddiGroups`, `ddiDocument`)입니다. 머리글 행은 그 표의 첫 행이므로, 표 위에 제목 행을 넣어도 됩니다. 표 바로 아래에 이어서 입력한 행도 읽습니다.
- Excel 표의 요약 행(Total Row)은 읽지 않습니다. 요약 행 아래에 입력한 행은 읽습니다.
- 표가 없으면(범위로 변환했거나 다른 도구로 만든 파일) 1행을 머리글로 봅니다.
- 값이 정해진 열에는 드롭다운 목록이 있습니다. 목록의 값은 DDI 2.5 XSD가 정한 값입니다. 대상 열은 `Variables`의 `intrvl`, `responseDomainType`, `format`, `formatSchema`, `formatCategory`, `Categories`의 `missing`, `Groups`의 `type`입니다.
- 목록에 없는 값을 넣으면 Excel이 경고하지만, 막지는 않습니다. 입력 DDI에 있던 표준 밖의 값도 그대로 남기기 위해서입니다.
- 붙여 넣은 값은 Excel이 검사하지 않습니다. 그래서 워크북을 DDI로 바꿀 때(`from-xlsx`), 이 열에 목록에 없는 값이 있으면 셀 주소와 함께 경고가 나옵니다. 값은 그대로 출력에 들어갑니다.
- 머리글 행에 시트의 기준 열(`Variables`와 `Groups`의 `ID`, `Categories`의 `varID`, `Document`의 `level`과 `element`)이 없으면 `Variables!A1: …`처럼 머리글 행의 A열을 짚은 오류가 납니다. 다른 열이 없으면 그 열의 내용을 읽지 못한다는 경고가 나옵니다.
- 빈 행은 건너뜁니다.

### 공통 열: `attributes`, `xml`

`Variables`, `Categories`, `Groups`의 모든 행에는 아래 두 열이 있습니다.

| 열 | 내용 |
|---|---|
| `attributes` | 다른 열에 들어가지 않는 그 요소의 속성입니다. XML 속성 문법으로 씁니다. 예: `files="F1" dcml="0"` |
| `xml` | 다른 열에 들어가지 않는 자식 요소들의 XML 원문입니다. 예: `<location fileid="F1"/><sumStat type="vald">580</sumStat>` |

- 내용이 행에 붙어 다니므로, 행을 옮겨도 다른 시트와의 연결이 끊기지 않습니다.
- 변수의 `ID`를 고칠 때는 그 변수를 가리키는 `Categories` 시트의 `varID`와 `Groups` 시트의 `vars`도 함께 고쳐야 합니다. `varID`를 고치지 않으면 오류가 나고, `vars`를 고치지 않으면 경고가 납니다.
- 같은 속성을 그 속성의 열과 `attributes` 셀에 함께 쓰면 오류입니다. 예: `name` 열과 `attributes` 셀의 `name="…"`
- `xml` 셀에서는 루트 `codeBook`에 선언된 접두사(예: `xhtml:`)를 쓸 수 있습니다. 선언되지 않은 접두사를 쓰면 셀 주소와 함께 경고가 나옵니다. 네임스페이스를 따지는 프로그램은 그 출력을 거부하기 때문입니다. `Document` 시트의 `element` 셀도 같습니다.
- 셀 내용이 잘 구성된 XML이 아니면 셀 주소와 함께 오류가 납니다.
- 셀에 넣은 주석과 처리 명령(PI)은 셀 주소와 함께 경고를 내고 버립니다.
- 요소에 텍스트만 있으면(예: `<catgry>x</catgry>`), 그 텍스트가 `xml` 셀에 들어갑니다. 이때 그 행의 다른 내용 열과, 그 요소를 가리키는 다른 시트의 행은 비어 있어야 합니다.

### `Variables`: `var`마다 한 행

행 순서가 출력의 `var` 순서입니다.

| 열 | DDI | 열에 들어가는 조건 |
|---|---|---|
| `ID` | `var/@ID` | `var`와 `varGrp`를 통틀어 고유해야 합니다. 비워 두면 도구가 붙입니다("ID 자동 부여" 참고). |
| `name` | `var/@name` | |
| `label` | `labl` | `labl`이 하나이고, 속성이 없고, 텍스트만 있을 때 |
| `question` | `qstn/qstnLit` | `qstn`이 하나이고, 그 자식이 속성 없는 텍스트 `qstnLit` 하나뿐이고, `qstn`의 속성이 `ID`와 `responseDomainType`뿐일 때 |
| `questionID` | `qstn/@ID` | `question`과 같은 조건 |
| `responseDomainType` | `qstn/@responseDomainType` | `question`과 같은 조건 |
| `intrvl` | `var/@intrvl` | |
| `format` | `varFormat/@type` | `varFormat`이 하나이고, 내용이 없고, 속성이 있으며 그 속성이 `type`·`schema`·`category`뿐일 때 |
| `formatSchema` | `varFormat/@schema` | `format`과 같은 조건 |
| `formatCategory` | `varFormat/@category` | `format`과 같은 조건 |
| `notes` | `notes` | 속성이 없는 한 줄 텍스트 `notes`입니다. 한 줄에 하나씩 씁니다. 조건에 맞지 않는 `notes`가 나오면, 그것과 그 뒤의 `notes`는 `xml` 셀로 갑니다. |
| `attributes`, `xml` | 공통 열 | |

조건에 맞지 않는 요소는 `xml` 셀로 갑니다. `catgry`는 `Categories` 시트로 갑니다.

### `Categories`: `catgry`마다 한 행

| 열 | DDI | 열에 들어가는 조건 |
|---|---|---|
| `varID` | 소속 `var`의 ID | 비워 두면 `varName`으로 변수를 찾습니다. |
| `varName` | 소속 `var`의 `name` | 도구가 채웁니다. `varID`가 비어 있을 때만 읽습니다. |
| `code` | `catValu` | `catValu`가 하나이고, 속성이 없고, 텍스트만 있을 때 |
| `label` | `labl` | `Variables`의 `label`과 같은 조건 |
| `missing` | `catgry/@missing` | 값을 그대로 씁니다(`Y`, `N`). 빈칸은 속성이 없다는 뜻입니다. |
| `freq` | `catStat` | 첫 번째 `catStat`이 `type="freq"` 하나만 속성으로 가지고, 그런 `catStat`이 하나뿐일 때 |
| `attributes`, `xml` | 공통 열 | `missType`, `excls`, 가중 `catStat` 등이 들어갑니다. |

- 한 변수 안에서 `catgry`의 순서는 행 순서를 따릅니다.
- 도구가 만든 워크북은 변수를 이름이 아니라 ID로 가리킵니다. 파일이 여러 개인 데이터셋에서는 변수 이름이 겹치기 때문입니다. 이름으로 가리키는 규칙은 "이름으로 연결"을 참고해 주세요.

### `Groups`: `varGrp`마다 한 행

| 열 | DDI |
|---|---|
| `ID` | `varGrp/@ID`: 비워 두면 도구가 붙입니다("ID 자동 부여" 참고). |
| `type` | `varGrp/@type` |
| `name` | `varGrp/@name` |
| `label` | `labl` (`Variables`의 `label`과 같은 조건) |
| `vars` | `varGrp/@var`: 변수 ID를 공백으로 구분해 씁니다. 변수 이름을 쓰면 ID로 바꿔 씁니다. |
| `attributes`, `xml` | 공통 열 |

`vars`의 항목이 `Variables` 시트의 ID도 아니고, 정확히 한 변수의 이름도 아니면 경고를 냅니다. 오류는 아닙니다.

### `Document`: 나머지 문서 전체

위 세 시트에 들어가지 않는 내용을 개요(outline) 형태로 담습니다. 루트 속성, `docDscr`, `stdyDscr`, `fileDscr`, `otherMat`, 그리고 `dataDscr` 안의 `var`·`varGrp`가 아닌 자식이 여기에 들어갑니다.

| 열 | 뜻 |
|---|---|
| `level` | 깊이입니다. `codeBook`이 0입니다. |
| `element` | 요소 이름입니다. |
| `attributes` | XML 속성 문법으로 쓴 속성입니다. |
| `text` | 텍스트 값입니다. 원문 보관 요소면 안쪽 XML 원문입니다. |
| `kind` | 빈칸이거나, 원문 보관이면 `xml`입니다. |

- 행 순서가 곧 출력 순서입니다. 예외는 두 가지입니다. `dataDscr`의 자식(이 목록의 마지막 항목)과, 떨어져 있는 같은 이름의 형제("알려진 손실과 변경" 참고)입니다.
- `element` 셀은 `level`만큼 들여쓰기되어 보입니다. 보기 위한 것입니다.
- `Document` 표는 정렬하지 말아 주세요. 행 순서가 개요의 구조이므로, 정렬하면 요소가 다른 부모 아래로 옮겨집니다.
- 첫 행은 `level` 0의 `codeBook`이고, `xmlns="ddi:codebook:2_5"` 속성이 있어야 합니다.
- 어떤 행의 `level`은 바로 위 행보다 2 이상 깊을 수 없습니다.
- `text`가 있는 행 아래에는 자식 행을 둘 수 없습니다.
- `dataDscr` 안의 `var`와 `varGrp`는 다른 시트의 내용을 스키마가 정한 자리에 맞춰 끼워 넣습니다. 이때 `Document` 시트에 있는 `dataDscr`의 다른 자식도 함께 스키마 순서로 정렬됩니다.

### 자주 하는 편집

#### 라벨 고치기

변수 라벨은 `Variables` 시트에서 그 변수 행의 `label` 셀을 고칩니다. 값 라벨은 `Categories` 시트의 `label` 셀을 고칩니다.

`label` 셀이 비어 있다면, 그 라벨이 열의 조건에 맞지 않아 `xml` 셀에 있는 것입니다. 예를 들어 `labl`이 둘 이상이거나 `xml:lang` 같은 속성이 있는 경우입니다. 이때는 `xml` 셀 안의 `<labl>` 원문을 고칩니다.

#### 범주 추가

`Categories` 시트에서 그 변수의 마지막 범주 행 아래에 행을 넣습니다. `varID`에 변수의 ID를 쓰고, `code`와 `label`에 값과 라벨을 씁니다. 결측값이면 `missing`에 `Y`를 씁니다. `varID`를 쓰면 `varName`은 비워 두어도 됩니다.

#### 키워드 추가

`Document` 시트에서 `stdyDscr` → `stdyInfo` → `subject` 행을 찾습니다. 그 아래 마지막 `keyword` 행 다음에 행을 넣습니다. `level`은 `subject`보다 1 크게, `element`는 `keyword`로, `text`에는 키워드를 씁니다.

```text
level  element   text
3      subject
4      keyword   households
4      keyword   새 키워드        (추가한 행)
```

`subject` 행이 없다면 두 경우로 나뉩니다.

- `stdyInfo` 행이 이미 있으면, 그 아래에 `subject`(level 3)와 `keyword`(level 4) 행만 넣습니다. `subject`는 `stdyInfo`의 첫 자식 자리입니다. `studyBudget`이 있으면 그 다음입니다.
- `stdyInfo` 행도 없으면, `stdyInfo`(level 2), `subject`(level 3), `keyword`(level 4) 행을 넣습니다. `stdyInfo`는 `stdyDscr` 안에서 `citation` 다음 자리입니다. `studyAuthorization`이 있으면 그 다음입니다.

두 경우 모두, 새 행은 앞 요소의 더 깊은 행들이 모두 끝난 뒤에 넣습니다. 예를 들어 `citation` 다음에 넣는다면, `citation` 아래의 `titlStmt` 같은 행들이 모두 끝난 다음 줄에 넣습니다.

`Document` 시트는 행 순서가 곧 출력 순서이므로, 스키마가 정한 자리에 넣어야 출력이 XSD를 통과합니다. 허용되지 않는 부모 아래에 요소를 넣으면 경고가 나옵니다. 하지만 형제 사이의 순서는 검사하지 않습니다. 결과는 아래 "개발"의 `xmllint` 명령을 저장소 루트에서 실행해 확인할 수 있습니다.

#### 다시 계산하지 않는 값

`fileDscr/fileTxt/dimensns/varQnty`는 그대로 둡니다. 변수를 추가하거나 지워도 이 값은 바뀌지 않습니다. 필요하면 `Document` 시트에서 직접 고쳐 주세요.

## 왕복 보장

이 도구는 아래 다섯 가지를 보장합니다.

1. **고정점**: 이 도구가 만든 XML/JSON → xlsx → XML/JSON은 바이트 단위로 같다.
2. **무손실**: `var`와 `varGrp`에 ID가 있는 DDI 2.5 입력이라면, DDI → xlsx → DDI 결과가 입력과 같은 정규 트리를 가진다.
   - 정규 트리는 §5.3의 공백 규칙을 거치고 속성 순서를 무시한 트리다.
   - 예외 1: 주석, PI, DOCTYPE은 경고와 함께 버린다.
   - 예외 2: 스키마 순서를 어긴 입력에서는 `var`, `catgry`, `varGrp`의 자식이 스키마 순서로 돌아온다.
3. **XML과 JSON 일치**: 같은 xlsx에서 나온 XML과 JSON은 같은 트리를 담는다.
4. **유효성**: XSD를 통과하는 입력에서 열의 값만 고쳤다면 출력도 XSD를 통과한다. 이것은 테스트로 확인한다. `Document`나 `xml` 셀에 넣은 요소에 대해서는 배치 경고만 낸다.
5. **Walla 파일**: 설문 도구 Walla가 이 도구의 정규 레이아웃과 JSON 규약으로 쓴 파일은 1번에 따라 바이트 단위로 같다. 그보다 이전 형식의 Walla 내보내기 파일은 2번 수준으로 보장한다.

2번의 "§5.3의 공백 규칙"은 다음과 같습니다. 텍스트 값은 앞뒤 공백과 둘째 줄부터의 줄 앞 공백을 지우고, CR을 LF로 바꿉니다. 자식 요소 사이에 있는 줄바꿈이 든 공백은 들여쓰기로 보고 버립니다.

## 알려진 손실과 변경

입력과 출력이 다를 수 있는 경우입니다.

- **주석, 처리 명령(PI), DOCTYPE**: 경고를 내고 버립니다. 단, 원문 보관 요소 안에 있는 것은 원문과 함께 남습니다.
- **텍스트 안의 들여쓰기**: 텍스트 값의 앞뒤 공백을 지우고, 두 번째 줄부터는 각 줄의 앞 공백도 지웁니다. 그래서 텍스트 안에 일부러 넣은 들여쓰기는 사라집니다.
- **줄바꿈 문자**: 텍스트 값의 CR(`\r\n`, 단독 `\r`)은 LF로 바뀝니다. 속성 값의 CR은 `&#13;`로 써서 그대로 남습니다.
- **XML 레이아웃**: 출력은 언제나 2칸 들여쓰기의 같은 레이아웃을 씁니다. 입력의 공백 배치는 남지 않습니다.
- **XML 1.0에서 금지된 문자**: C0 제어문자(탭, LF, CR은 남깁니다), U+FFFE, U+FFFF, 짝 없는 서로게이트는 출력에서 지웁니다.
- **스키마 순서를 어긴 입력**: `var`, `catgry`, `varGrp`의 자식은 스키마 순서로 돌아옵니다. 스키마에 없는 요소는 자기 목록(`xml` 셀, 또는 `Document` 시트에 있는 `dataDscr`의 자식 행들) 안에서 바로 앞에 있던 요소를 따라갑니다. 목록의 맨 앞에 있던 요소는 맨 앞으로 갑니다.
- **`Document` 시트에 직접 넣은 같은 이름의 행**: 같은 부모 아래에서 같은 이름의 형제가 떨어져 있으면, 처음 나온 자리로 모입니다. 예를 들어 `p, list, p` 순서로 넣은 행은 `p, p, list`로 나옵니다. XML과 JSON이 같은 트리를 담게 하기 위해서입니다.
- **숫자로 붙여 넣은 값**: 다른 곳에서 숫자로 붙여 넣으면 Excel이 앞의 0을 지웁니다(`01` → `1`). 도구는 표시된 문자열을 읽으므로 이것은 되돌릴 수 없습니다. 코드 값을 붙여 넣은 뒤에는 꼭 확인해 주세요.
- **`var`, `catgry`, `varGrp`, `dataDscr`의 자식 사이 공백**: 줄바꿈이 없는 공백도 버립니다. 이 네 요소는 요소만 담으므로 그 공백은 내용이 아닙니다("원문 보관" 참고).
- **텍스트만 있는 `catgry`**: 한 변수 안에서 텍스트만 있는 `catgry`가 하나 나오면, 그것과 그 뒤의 `catgry`는 모두 `Categories` 시트가 아니라 그 변수의 `xml` 셀로 갑니다. 범주의 순서를 지키기 위해서입니다.
- **`varGrp/@var`의 변수 이름**: `var` 속성의 항목이 변수 ID가 아니고 정확히 한 변수의 `name`과 같으면, 워크북을 되돌릴 때 그 변수의 ID로 바뀝니다("이름으로 연결" 참고).
- **받지 않는 입력**: `ID`가 없는 `var`·`varGrp`, `var`와 `varGrp`를 통틀어 겹치는 `ID`, DDI 1.2.2, UTF-8이 아닌 입력, 256단계보다 깊은 중첩, 자식 요소 사이에 텍스트가 있어 원문 보관했지만 셀 하나(32,767자)에 들어가지 않는 `var`·`catgry`·`varGrp`·`dataDscr`는 오류로 거부합니다. 자세한 내용은 "받는 입력"과 "원문 보관"을 참고해 주세요.

## 스프레드시트 앱

- 공식 지원은 Microsoft Excel뿐입니다.
- Microsoft Excel for Mac (365)에서 텍스트 서식 셀에 300자와 2,000자를 넣어 확인했습니다. 두 경우 모두 `####`이 아니라 내용이 그대로 보였습니다.
- LibreOffice와 Google Sheets에서도 동작하도록 만들었지만, 테스트하지는 않았습니다.
- Excel 셀 하나에는 32,767자까지 들어갑니다. 이보다 긴 값이 생기면 `to-xlsx`가 셀 주소와 함께 오류를 냅니다.

## 확인한 결과

- 저장소의 fixture 코드북 두 개(`testdata/kofac_*`)는 DDI → xlsx → DDI 왕복 결과가 golden 파일(`canonical.xml`, `canonical.json`)과 바이트 단위로 같습니다. `xmllint`가 설치되어 있으면 Go 테스트가 두 코드북을 XML 입력과 JSON 입력으로 모두 확인합니다. npm 테스트는 `xmllint`와 상관없이 그중 하나를 확인합니다.
- 브라우저(wasm)에서도 같은 두 코드북을 각각 약 0.8–0.9초에 왕복했고, 결과가 golden 파일과 바이트 단위로 같았습니다.

## 개발

필요한 도구는 Go 1.25 이상, Node(22에서 테스트함), `xmllint`입니다. `xmllint`가 없으면 일부 Go 테스트를 건너뜁니다.

저장소 구조는 아래와 같습니다.

| 경로 | 내용 |
|---|---|
| `cmd/ddi-xlsx/` | CLI입니다. 파일 입출력은 여기서만 합니다. |
| `ddi/` | 트리 모델과 XML·JSON 읽기·쓰기입니다. |
| `workbook/` | 트리와 xlsx 사이의 변환입니다. |
| `internal/schema/` | `codebook.xsd`에서 생성한 규칙 표와 그 생성기(`gen/`)입니다. |
| `wasm/` | WebAssembly 진입점입니다. |
| `npm/` | npm 패키지입니다. |
| `xsd/ddi-codebook-2.5/` | 함께 넣어 둔 DDI Codebook 2.5 XSD입니다. |
| `testdata/` | 테스트 fixture입니다. |

`internal/schema/table.go`는 XSD에서 생성한 파일입니다. XSD나 생성기를 바꾼 뒤에는 다시 생성합니다.

```bash
go generate ./internal/schema
```

정적 검사를 합니다.

```bash
go vet ./...
```

Go 테스트를 실행합니다. `xmllint`가 없으면 XSD 검사를 포함한 테스트는 통째로 건너뜁니다. 그 테스트 안의 golden 비교도 함께 건너뛰므로, 전체를 확인하려면 `xmllint`가 있어야 합니다.

```bash
go test ./...
```

CLI 실행 파일 여섯 개를 `dist/`에, wasm을 `npm/ddi-xlsx.wasm`에 빌드합니다. 같은 Go 툴체인의 `wasm_exec.js`도 `npm/`에 복사합니다. wasm과 `wasm_exec.js`는 같은 Go 버전에서 나와야 하기 때문입니다.

```bash
scripts/build.sh
```

npm 테스트를 실행합니다. wasm이 필요하므로 `scripts/build.sh`를 먼저 실행해야 합니다.

```bash
node --test npm/test/*.test.mjs
```

변환 결과가 XSD를 통과하는지 직접 확인할 수도 있습니다. 아래 명령은 저장소 루트에서 실행합니다. `codebook-edited.xml` 자리에는 확인할 파일의 경로를 씁니다.

```bash
xmllint --nonet --noout --schema xsd/ddi-codebook-2.5/codebook.xsd codebook-edited.xml
```

## 라이선스

### 함께 넣어 둔 XSD

`xsd/ddi-codebook-2.5/`의 파일은 출처가 여러 곳입니다. 각 파일의 머리말에 적힌 내용은 아래와 같습니다.

| 파일 | 머리말의 내용 |
|---|---|
| `codebook.xsd` | Copyright (c) 2014 DDI Alliance. GNU Lesser General Public License 버전 3 또는 그 이후 버전으로 배포합니다. |
| `ddi-xhtml11.xsd`, `ddi-xhtml11-model-1.xsd`, `ddi-xhtml11-modules-1.xsd` | Copyright (c) 2009 DDI Alliance(DDI 3.1 XML Schema의 일부). GNU Lesser General Public License 버전 3 또는 그 이후 버전으로 배포합니다. |
| `XHTML/*.xsd` | W3C XHTML 모듈화(Modularization) 스키마 모듈입니다. `xhtml-copyright-1.xsd`를 저작권 문서로 가리키지만, 그 파일은 함께 들어 있지 않습니다. 라이선스는 확인 필요입니다. |
| `XHTML/*.ent` | W3C XHTML 문자 엔터티 파일입니다. ISO 8879에 맞는 SGML 시스템에서 쓰는 조건으로 복사를 허락한다는 ISO 고지가 있습니다. 파일 전체의 라이선스는 확인 필요입니다. |
| `dc.xsd`, `dcterms.xsd`, `dcmitype.xsd` | Dublin Core 스키마입니다. 라이선스 문구가 없습니다. 라이선스는 확인 필요입니다. |
| `xml.xsd` | W3C의 XML 네임스페이스 스키마입니다. 라이선스 문구가 없습니다. 라이선스는 확인 필요입니다. |

GNU Lesser General Public License의 전문은 아직 저장소에 넣지 않았습니다.

### 이 도구

이 도구 자체의 라이선스는 아직 정하지 않았습니다.

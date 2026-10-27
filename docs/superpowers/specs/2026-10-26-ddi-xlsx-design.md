# ddi-xlsx 설계

작성 2026-10-26. 상태: 초안 (빌더 검토 반영, 사용자 승인 전). 이전 제안(`ddi-xlsx-design-proposal.md`)을 대체한다.

## 1. 목적

DDI Codebook XML과 xlsx 워크북을 양방향으로 변환하는 독립 도구다. 연구자와 아카이브 담당자가 코드북을 스프레드시트에서 읽고 고친 뒤, 그 결과를 DDI로 되돌릴 수 있게 한다.

- 입력은 Walla만이 아니다. 아카이브나 다른 도구(NADA, Nesstar, Colectica 등)가 만든 DDI 2.5도 받는다. (사용자 결정)
- CLI, Node(npm), 웹 브라우저에서 모두 쓸 수 있어야 한다. (사용자 결정)
- 도구는 Walla 밖으로도 배포한다. (사용자 결정)

## 2. 목표·비목표·반목표

**목표**
- DDI → xlsx → DDI에서 내용을 하나도 잃지 않는다. 열에 들어가지 않는 내용은 같은 행의 `xml` 셀에 XML 원문으로 보관한다.
- 흔한 편집은 XML을 몰라도 셀 편집만으로 할 수 있다. 흔한 편집이란 변수 라벨, 질문 문구, 값 라벨, 결측 표시, 노트, 그룹 구성, 연구 메타데이터 수정이다.
- XML과 JSON은 같은 트리를 담는다.

**비목표**
- 응답 데이터 파일(CSV, SPSS) 변환.
- DDI 3.x.
- 입력 XML의 공백 레이아웃을 보존하는 것. 출력은 언제나 하나의 정규 레이아웃을 쓴다.
- 주석, 처리 명령(PI), DOCTYPE 보존. 이런 것은 경고를 내고 버린다.
- XSD 검증. 도구는 입력을 XSD로 거부하지 않는다. 스키마상 배치가 맞지 않으면 경고만 낸다.

**반목표 (기술적으로는 돌아가도 실패로 본다)**
- 내용을 조용히 잃는 것. 받을 수 없는 입력은 위치를 짚어 오류로 거부한다.
- Excel이 값을 자동으로 바꾸는 것. 예를 들어 코드 `01`이 `1`이 되거나, `3/4`가 날짜가 되면 안 된다.

## 3. 언어·배포 형태 (내 판단, 사용자가 Go를 먼저 제안함)

**Go**로 만든다. Go와 TS의 최소 빌드를 비교했다(2026-10-26, scratchpad).

| 형태 | Go | TS |
|---|---|---|
| CLI | 네이티브 5.9MB | bun 단일 실행 파일 57.7MB |
| 웹 | wasm gzip 2.6MB | gzip 0.28MB |

주 사용자는 R, Python, SPSS를 쓰는 연구자라서 작은 CLI 바이너리가 가장 넓게 쓰인다. Node와 웹은 같은 wasm 하나로 처리한다.

- 의존성은 `github.com/xuri/excelize/v2` 하나다. XML과 JSON은 표준 라이브러리로 처리한다.
- 배포물은 세 가지다.
  - OS별 CLI 바이너리: darwin/linux/windows × amd64/arm64.
  - npm 패키지 `ddi-xlsx`: wasm, Go의 `wasm_exec.js`, 얇은 JS 래퍼, `.d.ts`로 구성한다.
  - Go 모듈.

## 4. 저장소 구조

```
ddi-xlsx/
  go.mod
  cmd/ddi-xlsx/          CLI (os 파일 입출력은 여기만)
  ddi/                   트리 모델, XML·JSON 읽기/쓰기
  workbook/              트리 ↔ xlsx
  internal/schema/       codebook.xsd 에서 생성한 규칙 표 (+ gen/ 생성기, go generate)
  wasm/                  GOOS=js 진입점: ddiToXlsx, xlsxToDdi 를 전역에 등록
  npm/                   package.json, index.js, index.d.ts, (빌드 산출물) ddi-xlsx.wasm, wasm_exec.js
  xsd/ddi-codebook-2.5/  vendored XSD (Walla 저장소의 사본, xml.xsd 로컬 import)
  testdata/              fixture
  README.md
```

- 위치는 `~/Paprika/dev/ddi-xlsx`다.
- `git init`만 하고 remote 생성, commit, push는 하지 않는다. (사용자 규칙)
- Go 모듈 경로는 임시로 `github.com/teamPaprika/ddi-xlsx`를 쓰고, 공개하기 전에 확정한다.

## 5. DDI 읽기

### 5.1 버전

- 네임스페이스가 `ddi:codebook:2_5`인, 잘 구성된(well-formed) 문서를 받는다. XSD 유효성은 요구하지 않는다.
- 그 밖의 네임스페이스는 오류로 처리한다. (사용자 결정: 2.5만)
- DDI 1.2.2(`http://www.icpsr.umich.edu/DDI`, NADA의 기본 내보내기)는 "DDI 1.2.2 는 지원하지 않음, 2.5 로 변환해 넣을 것"이라는 별도 오류 문구로 안내한다.

### 5.2 트리

```go
type Attr struct{ Name, Value string }   // Name 은 접두사 포함 ("xml:lang", "xsi:schemaLocation")
type Node struct {
    Name     string   // DDI 요소는 접두사 없음, 다른 네임스페이스는 입력의 접두사 유지 ("xhtml:p")
    Attrs    []Attr   // 입력 순서
    Text     string   // 텍스트만 가진 요소
    Children []*Node  // 요소 자식
    Raw      string   // 원문 보관 요소의 inner XML (§5.3). 있으면 Text/Children 은 비어 있다
}
```

`xmlns:*` 선언은 루트 속성으로 유지한다.

### 5.3 원문 보관과 공백

요소마다 아래 순서로 판정한다.

1. **원문 보관(Raw)**: 다음 중 하나에 해당하면 inner XML을 해석하지 않고 원문 그대로 둔다.
   - 공백이 아닌 텍스트와 요소 자식이 함께 있다. 예: `<p>a <emph>b</emph></p>`.
   - 요소 자식 사이에, 줄바꿈 없이 공백만 있는 텍스트가 있다. 예: `<emph>a</emph> <emph>b</emph>`의 가운데 공백은 실제 내용이다.
   - 같은 이름의 요소 자식이 연속하지 않고 다시 나온다. 예: `p, list, p`나 `qstnLit, ivuInstr, qstnLit`. 그대로 두면 JSON에서 이름별로 모이면서 순서가 바뀌기 때문이다.
2. **요소만**: 요소 자식이 있으면, 자식 사이의 공백 텍스트는 버린다. 1번에 걸리지 않았으므로 이 공백은 모두 줄바꿈을 포함한 레이아웃이다. 그래서 NADA의 `qstn`처럼 들여쓰기 공백만 낀 요소는 원문 보관이 아니라 요소로 읽힌다(wb_3761의 `qstn` 308건 확인).
3. **텍스트**: 텍스트 값의 `\r\n`과 단독 `\r`는 `\n`으로 바꾼다(문자 참조 `&#13;`로 들어온 CR도 포함). CR을 그대로 쓰면 다시 읽을 때 LF가 되어 고정점이 깨지기 때문이다. 속성 값의 CR은 `&#13;`로 써서 보존한다. 그다음 텍스트 값은 앞뒤 공백을 지운다. 두 번째 줄부터는 각 줄의 앞 공백도 지운다. NADA는 텍스트를 `<catValu>\n          1\n        </catValu>`처럼 내보내기 때문이다. 이 규칙 때문에 텍스트 안에 일부러 넣은 들여쓰기는 사라진다. 이것은 알려진 손실로 문서화한다.

판정은 XSD가 아니라 실제 내용으로 한다.

### 5.4 스키마 표 (생성물)

`internal/schema/gen`이 `xsd/.../codebook.xsd`를 읽어 표를 만든다.

- 처리해야 할 XSD 구조:
  - 전역 요소와 `ref`.
  - 타입 안에서 지역으로 선언된 요소 9개. 예: `description`, `outcome`, `codeListSchemeURN`. 지역 `codeListSchemeURN`은 전역과 타입이 다르다.
  - `xs:group` 참조(`PHRASE`, `FORM` 등).
  - `complexContent`의 extension과 restriction.
  - XHTML 쪽 내용 모델은 따라가지 않는다. 혼합 내용은 원문 보관(§5.3)이 처리하기 때문이다.
- 표가 담는 정보:
  - **반복 가능한 요소 이름의 집합**: 어떤 부모 아래에서든 maxOccurs가 1보다 크거나, 반복되는 choice나 group 안에 있으면 반복 가능으로 본다. JSON은 이 집합을 이름 단위로 적용한다. 그래서 같은 요소 이름은 문서 어디서든 같은 JSON 모양을 가진다.
  - **자식 정렬 위치**: (부모 타입, 자식 이름)마다 sequence 안의 위치를 둔다. 반복되는 choice의 구성원들은 모두 같은 위치 값을 받는다. 그래서 정렬해도 `item`과 `range`, `p`와 `list`의 상대 순서가 바뀌지 않는다.
  - **허용 자식 집합**: (부모 타입)마다 둔다. 배치 경고에 쓴다.

### 5.5 정규 XML 출력

- 첫 줄은 `<?xml version="1.0" encoding="UTF-8"?>`이고, 파일은 개행으로 끝난다.
- 들여쓰기는 2칸이다.
  - 요소만 담는 요소는 자식을 한 줄에 하나씩 쓴다.
  - 텍스트 요소와 원문 보관 요소는 같은 줄에 그대로 쓴다.
  - 비어 있는 요소는 `<x/>`로 쓴다.
- 이스케이프 규칙:
  - 텍스트에서는 `&`, `<`, `>`를 바꾼다.
  - 속성에서는 `&`, `<`, `>`, `"`와 탭(`&#9;`), LF(`&#10;`), CR(`&#13;`)을 바꾼다.
- XML 1.0에서 금지된 문자는 지운다. 대상은 C0 제어문자, U+FFFE, U+FFFF, 짝 없는 서로게이트이며, Walla의 `clean`과 같은 규칙이다.
- 원문 보관 요소의 원문은 트리에 들어올 때(XML·JSON·xlsx 읽기) 반드시 `ParseFragment`를 거쳐 잘 구성되었는지 확인된다. 그래서 출력할 때는 다시 파싱하지 않는다.

### 5.6 JSON 규약

- 루트는 `{"codeBook": …}`다.
- 속성은 `@` 키로 쓴다(`@xml:lang`처럼 접두사를 포함).
- 텍스트만 있고 속성이 없는 요소는 문자열로 쓴다. 텍스트와 속성이 함께 있으면 텍스트를 `#text`에 넣는다.
- 원문 보관 요소의 내용은 `#xml`(inner XML 원문)에 넣는다.
- **배열 여부는 §5.4의 반복 가능 이름 집합으로 정한다.** 요소가 한 번만 나와도 그 이름이 반복 가능하면 배열로 쓴다.
- 키 순서는 속성(입력 순서), 그다음 자식(첫 등장 순서)이다. §5.3 규칙 덕분에 같은 이름의 형제는 언제나 연속해 있으므로, 이름별로 모아도 순서를 잃지 않는다.
- 출력은 2칸 들여쓰기에 개행으로 끝난다. 문자열 이스케이프는 `JSON.stringify`와 같게 한다. Go의 `encoding/json` 기본값과 다른 점은 아래와 같다.
  - `<`, `>`, `&`, U+2028, U+2029를 이스케이프하지 않는다.
  - 제어문자는 `\b \f \n \r \t`와 `\u00xx`(소문자 16진수)로 쓴다.
- 읽을 때는 관대하게 받는다. 배열 자리에 스칼라가 와도 받는다. 따라서 현재 Walla 규약으로 만든 `survey.json`도 읽을 수 있다.
- 이 규약은 현재 Walla 규약(`var`, `varGrp`, `catgry`, `notes` 넷만 배열)과 다르다. 예를 들어 이 규약에서는 `labl`, `IDNo`, `software`가 배열이 된다. **사용자 결정: Walla 내보내기를 이 규약과 §5.5 레이아웃에 맞춘다.** 이 작업은 Walla 쪽 후속 작업이다.

## 6. 워크북

시트는 네 개다: `Variables`, `Categories`, `Groups`, `Document`.

- 모든 셀은 문자열로 쓴다.
- 모든 열에 텍스트 서식(`@`)을 열 단위로 걸어, 빈 행에 새로 입력한 값도 Excel이 자동 변환하지 않게 한다.
- 머리글 행은 틀 고정한다.
- 읽을 때는 셀에 표시된 문자열을 쓴다. 숫자 셀도 받는다.

**행마다 있는 공통 열 두 개**

| 열 | 내용 |
|---|---|
| `attributes` | 다른 열에 들어가지 않는 그 요소의 속성. XML 속성 문법으로 쓴다(`files="F1" dcml="0"`) |
| `xml` | 다른 열에 들어가지 않는 자식 요소들의 XML 원문 조각(`<location .../><sumStat type="vald">580</sumStat>`) |

- 내용이 행에 붙어 다니므로, 행을 옮기거나 ID를 고쳐도 다른 시트와의 연결이 끊기지 않는다.
- 셀 내용이 잘 구성되지 않았으면 셀 주소와 함께 오류를 낸다.

### 6.1 `Variables`: `var`마다 한 행, 문서 순서

| 열 | DDI | 들어가는 조건 |
|---|---|---|
| ID | `var/@ID` | 필수, 고유 |
| name | `var/@name` | |
| label | `labl` | `labl`이 정확히 하나이고, 속성이 없고, 텍스트 요소일 때 |
| question | `qstn/qstnLit` | `qstn`이 하나이고, 그 자식이 속성 없는 텍스트 요소 `qstnLit` 하나뿐일 때 |
| questionID | `qstn/@ID` | 위 조건을 만족하는 `qstn`의 속성 |
| responseDomainType | `qstn/@responseDomainType` | 위와 같음. `qstn`에 이 두 속성 외의 속성이 있으면 `qstn` 전체를 `xml`로 보낸다 |
| intrvl | `var/@intrvl` | |
| format | `varFormat/@type` | `varFormat`이 하나이고, 텍스트가 없고, 속성이 type/schema/category뿐일 때 |
| formatSchema | `varFormat/@schema` | |
| formatCategory | `varFormat/@category` | |
| notes | `notes` | 속성이 없는 텍스트 요소 `notes`들. 한 줄에 하나씩 쓴다. 그 밖의 `notes`는 `xml`로 간다 |
| attributes, xml | 공통 열 | |

- `catgry`는 `Categories` 시트로 간다.
- 같은 이름의 자식이 열과 `xml`로 나뉘면, 되돌릴 때 열 쪽을 앞에 둔다.

### 6.2 `Categories`: `catgry`마다 한 행

| 열 | DDI | 조건 |
|---|---|---|
| varID | 소속 `var`의 ID | 필수 |
| varName | (참고용) | 읽을 때 무시한다. 도구가 채운다 |
| code | `catValu` | `catValu`가 하나이고, 속성 없는 텍스트 요소일 때 |
| label | `labl` | §6.1 label과 같은 조건 |
| missing | `@missing` | 값을 그대로 쓴다(`Y`, `N`, 빈칸 = 속성 없음) |
| freq | `catStat` | `type="freq"`이고 속성이 그것 하나뿐인 `catStat`이 하나일 때 |
| attributes, xml | 공통 열 | `missType`, `excls`, 가중 `catStat` 등 |

- 같은 변수 안에서 `catgry`의 순서는 행 순서를 따른다. 범주 행은 다른 곳에서 참조하지 않으므로 별도 키가 필요 없다.
- **변수를 ID로 가리키는 이유**: 파일이 여러 개인 데이터셋에서는 변수 이름이 중복된다. NADA 샘플(파일 35개)에는 `hhid`, `age` 같은 중복 이름이 460개 있었다.

### 6.3 `Groups`: `varGrp`마다 한 행

| 열 | DDI |
|---|---|
| ID | `@ID` |
| type | `@type` |
| name | `@name` |
| label | `labl` (§6.1과 같은 조건) |
| vars | `@var`: 공백으로 구분한 변수 ID |
| attributes, xml | 공통 열 |

### 6.4 `Document`: 위 세 시트에 들어가지 않는 문서 전체의 개요

문서 전체를 개요(outline) 형태로 담는다. 대상은 루트 속성, `docDscr`, `stdyDscr`, `fileDscr`, `otherMat`, 그리고 `dataDscr`의 `var`·`varGrp`가 아닌 자식이다.

| 열 | 뜻 |
|---|---|
| level | 깊이. `codeBook` = 0 |
| element | 요소 이름 |
| attributes | XML 속성 문법 |
| text | 텍스트 값. 원문 보관 요소면 inner XML 원문 |
| kind | 빈칸, 또는 원문 보관이면 `xml` |

- 행 순서가 곧 출력 순서다. 이 시트는 정렬하지 않는다.
- 셀 들여쓰기는 level만큼 준다(보기용).
- 요소를 추가하려면 원하는 위치에 행을 넣고 level을 맞추면 된다.
- `dataDscr` 행 아래에 오는 `var`와 `varGrp`는 다른 시트의 내용을 스키마 위치에 맞춰 끼워 넣는다.
- `codeBook` 아래에 `dataDscr`가 둘 이상이면 v1에서는 오류로 거부한다.

### 6.5 되돌리기 (xlsx → 트리)

**병합**: 정렬은 열과 `xml` 셀의 자식을 합칠 때만 한다. 대상은 `var`, `catgry`, `varGrp`의 자식과, `dataDscr`에 다시 끼워 넣는 `var`·`varGrp`다.
- §5.4의 위치 값으로 안정 정렬한다.
- 스키마에 없는 요소는 바로 앞 형제의 위치 값을 물려받는다.
- 스키마 순서를 지킨 입력이라면 이 정렬로 입력 순서가 그대로 재현된다.

**검사**: 아래는 오류로 처리한다.
- ID가 고유하지 않다.
- `Categories.varID`가 없는 변수를 가리킨다.
- `attributes`나 `xml` 셀이 잘 구성되지 않았다.
- `Document`의 level이 앞 행보다 2 이상 깊다.

스키마 배치가 맞지 않는 경우와 `Groups.vars`가 없는 변수를 가리키는 경우는 오류가 아니라 경고다. 입력 DDI에 이미 그런 참조가 있을 수 있고, 그 입력을 받아 놓고 되돌릴 때 거부하면 안 되기 때문이다.

**오류 메시지**는 셀 주소를 짚는다. 예: `Categories!A12: varID V999 에 해당하는 변수가 없음`.

**`fileDscr/fileTxt/dimensns/varQnty`**는 그대로 둔다. Walla는 이 값을 변수 수로 쓰지만, 일반 DDI에서는 파일별 값이라 다시 계산하지 않는다.

## 7. 왕복 보장 (README에 그대로 쓴다)

1. **고정점**: 이 도구가 만든 XML/JSON → xlsx → XML/JSON은 바이트 단위로 같다.
2. **무손실**: `var`와 `varGrp`에 ID가 있는 DDI 2.5 입력이라면, DDI → xlsx → DDI 결과가 입력과 같은 정규 트리를 가진다.
   - 정규 트리는 §5.3의 공백 규칙을 거치고 속성 순서를 무시한 트리다.
   - 예외 1: 주석, PI, DOCTYPE은 경고와 함께 버린다.
   - 예외 2: 스키마 순서를 어긴 입력에서는 `var`, `catgry`, `varGrp`의 자식이 스키마 순서로 돌아온다.
3. **XML과 JSON 일치**: 같은 xlsx에서 나온 XML과 JSON은 같은 트리를 담는다.
4. **유효성**: XSD를 통과하는 입력에서 열의 값만 고쳤다면 출력도 XSD를 통과한다. 이것은 테스트로 확인한다. `Document`나 `xml` 셀에 넣은 요소에 대해서는 배치 경고만 낸다.
5. **Walla 파일**: Walla가 이 도구의 레이아웃과 JSON 규약을 쓰도록 바뀌면(사용자 결정, Walla 쪽 후속 작업), Walla 파일도 1번에 따라 바이트 단위로 같다. 바뀌기 전에 나온 Walla 파일은 2번 수준으로 보장한다.

## 8. 인터페이스

**CLI**
```
ddi-xlsx to-xlsx   <in.xml|in.json> [-o out.xlsx]
ddi-xlsx from-xlsx <in.xlsx> [--xml out.xml] [--json out.json]
```
- 입력 형식은 내용으로 판별한다. 첫 비공백 문자가 `{`이면 JSON, `<`이면 XML이다.
- `-o`를 생략하면 입력 파일 이름의 확장자를 바꿔 쓴다. `from-xlsx`에서 둘 다 생략하면 XML만 쓴다.
- 경고는 stderr로 내보내고, 오류가 나면 exit 1로 끝낸다.

**Go 라이브러리**: `workbook.ToXLSX(ddi []byte) ([]byte, []string, error)`, `workbook.FromXLSX(x []byte) (xml, json []byte, warnings []string, err error)`.

**npm (Node와 브라우저 공통)**
```ts
export function ddiToXlsx(input: string | Uint8Array): Promise<{ xlsx: Uint8Array; warnings: string[] }>;
export function xlsxToDdi(xlsx: Uint8Array): Promise<{ xml: string; json: string; warnings: string[] }>;
```
- wasm은 처음 호출할 때 한 번 로드한다.
- Node에서는 `fs`로, 브라우저에서는 `fetch(new URL('./ddi-xlsx.wasm', import.meta.url))`로 로드한다.

## 9. 테스트 (done 조건)

`go test ./...`와 npm 스모크 테스트(`node --test`)로 확인한다.

| 항목 | 확인 내용 |
|---|---|
| KOFAC 왕복 | `kofac_2024_public`, `kofac_2025_edu`의 survey.xml/json(현재 Walla 형식)에 대해: (a) §7-2 정규 트리가 같다. (b) 1회 변환한 출력은 고정점이다. 이 출력이 Walla가 맞춘 뒤 내보낼 바이트이므로, (b)가 Walla 파일의 바이트 왕복을 보장한다. 정규 출력 fixture(`testdata/kofac_*/canonical.xml`·`.json`)를 golden으로 둔다 |
| 기관형 DDI 왕복 | §11의 합성 fixture로 아래를 확인한다: 들여쓴 텍스트, NADA형 `qstn`, `sumStat`·`catStat`, 원문 보관 세 경우(혼합 내용, 인라인 공백, 비연속 형제), 다중 파일 중복 이름, 스키마 순서를 어긴 입력. 1.2.2 입력은 안내 오류로 거부된다 |
| XSD | xlsx → DDI 출력을 `xmllint --nonet --noout --schema xsd/ddi-codebook-2.5/codebook.xsd`로 검사한다. xmllint가 없으면 건너뛴다 |
| XML·JSON 일치 | 같은 xlsx에서 나온 JSON을 트리로 읽은 결과와 XML을 트리로 읽은 결과가 같다. 원문 보관 요소가 있는 fixture를 포함한다 |
| 손 편집 | 워크북을 열어 라벨을 바꾸고, 범주 행을 추가하고, `Document`에 `keyword` 행을 추가한 뒤 xlsx → DDI를 실행하면 결과가 XSD를 통과하고 그 변경이 반영된다. ID를 고친 경우와 `xml` 셀을 깨뜨린 경우에는 셀 주소가 담긴 오류가 난다 |
| Excel 서식 | 모든 열에 `@` 서식이 있다. 코드 `01`이 그대로 유지된다. 숫자 셀로 저장된 값도 읽힌다 |
| 스키마 표 | 생성기가 지역 요소, group, extension을 처리한다. `labl`, `IDNo`, `software`는 반복 가능이다. `item`과 `range`는 같은 위치 값을 가진다 |
| JSON 이스케이프 | U+2028, `<`, 제어문자 출력이 `JSON.stringify`와 같다(Node로 기대값을 만든다) |
| wasm·npm | Node에서 npm 래퍼로 KOFAC 왕복이 된다. 브라우저 확인은 내장 브라우저에서 한 번 수동으로 한다 |

## 10. 개인정보와 fixture

- KOFAC zip의 `survey.csv`에는 실제 응답이 있다. 이 파일은 저장소에 복사하지 않는다. 지금까지 압축도 풀지 않았다.
- 저장소는 당분간 비공개이고, KOFAC 코드북 두 개(survey.xml/json)를 fixture로 둔다. (사용자 결정) 공개를 결정하면 이 fixture를 다시 검토한다.

## 11. 빌더 재량 (내 판단으로 정한 것, 바꿔도 되는 것)

- 기관형 fixture는 NADA 샘플의 특징만 흉내 낸 합성 파일로 만든다. World Bank 원본 파일은 라이선스를 확인하기 전까지 저장소에 넣지 않는다.
- vendored XSD의 라이선스(DDI Alliance)는 공개 전에 확인해서 README에 적는다.
- **지원 스프레드시트 앱**: Excel을 기준으로 한다. 로컬에 Excel과 Numbers가 있다.
  - 열 단위 `@` 서식과 "표시 문자열로 읽기"로 LibreOffice와 Google Sheets에서도 동작하도록 만들되, 공식 지원은 Excel만으로 둔다.
  - 숫자로 붙여 넣어 `01`이 `1`이 된 경우는 되돌릴 수 없으므로 README에 적는다.
- **ID 없는 `var`·`varGrp`**: v1에서는 위치를 짚은 오류로 거부한다. 실제 파일에서 필요해지면 합성 키를 추가한다.
- 릴리스 자동화(goreleaser 등)는 공개를 결정할 때 추가한다. 지금은 스크립트로 교차 빌드한다.
- zip 입력과 파일을 끌어다 놓는 웹 페이지는 v1에서 뺀다.

## 12. 결정 기록 (2026-10-26)

- Walla 내보내기를 이 도구의 XML 레이아웃과 JSON 배열 규칙에 맞춘다. 이 작업은 Walla 쪽 후속 작업이며 이 저장소의 범위가 아니다.
- 입력은 DDI 2.5만 받는다. 1.2.2는 안내 오류로 거부한다.
- 저장소는 비공개로 두고, KOFAC 코드북을 fixture로 쓴다.

## 13. 후속 (이 저장소 밖)

- Walla `packages/domain/lib/ddi/ddiXml.ts`를 §5.5 레이아웃과 §5.6 규약으로 바꾼다. 반복 가능 이름 집합은 이 도구의 생성 표를 그대로 옮긴다. 바꾼 뒤에는 KOFAC 내보내기가 이 저장소의 `canonical.*` golden과 바이트 단위로 같아야 한다.

## 14. 양식과 Excel 표 (2026-10-27 추가, 사용자 결정)

연구자가 빈 양식을 내려받아 처음부터 채운 뒤 DDI로 바꾸는 흐름을 지원한다. 사용자가 고른 범위는 "빈 양식 + 표·목록"과 "ID를 비우면 자동으로 붙이기"다.

### 14.1 Excel 표와 드롭다운 (모든 워크북)
- 네 시트의 데이터 영역을 각각 Excel 표(Table)로 만든다. 표 이름은 `ddiVariables`, `ddiCategories`, `ddiGroups`, `ddiDocument`다.
  - 표 안에서 정렬하면 Excel이 머리글 행을 빼고 정렬한다.
  - 표 위에 제목 행을 넣어도 표 머리글을 표 이름으로 찾을 수 있다.
- 읽을 때 표를 찾는 규칙:
  - 머리글 행은 그 시트에 있는 표의 첫 행이다.
  - 데이터는 머리글 아래 모든 행이다. 표 범위 밖에 입력된 행도 읽어서, 표가 늘어나지 않은 경우에도 내용을 잃지 않는다.
  - 표가 없으면(범위로 변환했거나 다른 도구로 만든 파일) 1행을 머리글로 본다.
  - 어느 경우든 핵심 머리글이 없으면 오류다. 이 규칙은 §6.5에 따르며, 최종 리뷰에서 고친 내용이다.
- 정해진 값만 들어가는 열에는 드롭다운 목록(데이터 유효성 검사)을 둔다. 목록 값은 XSD의 열거값에서 생성한다.
  - 대상 열은 `Variables.intrvl`, `Variables.responseDomainType`, `Variables.format`, `Variables.formatSchema`, `Variables.formatCategory`, `Categories.missing`, `Groups.type`이다.
  - 오류 방식은 "경고"로 한다. 사용자가 목록에 없는 값을 넣으면 경고하되 막지는 않는다. 입력 DDI에 있던 비표준 값을 그대로 보존하기 위해서다.
- 머리글 잠금(시트 보호)은 하지 않는다. 사용자가 자기 메모 열이나 행을 자유롭게 넣을 수 있어야 하기 때문이다.

### 14.2 빈 양식
- CLI: `ddi-xlsx template [-o template.xlsx]`.
  - `-o`를 생략하면 현재 폴더의 `template.xlsx`에 쓴다.
  - 그 파일이 이미 있으면 덮어쓰지 않고 오류를 낸다(§8의 기본 경로 규칙).
- Go: `workbook.Template() ([]byte, error)`. npm: `templateXlsx(): Promise<Uint8Array>`.
- 내용: 아래 최소 골격을 `ToXLSX`로 바꾼 워크북이다. `Variables`, `Categories`, `Groups`에는 머리글만 있다.
  ```xml
  <codeBook xmlns="ddi:codebook:2_5" version="2.5">
    <stdyDscr><citation><titlStmt><titl/></titlStmt></citation></stdyDscr>
    <dataDscr/>
  </codeBook>
  ```
- 아무것도 채우지 않은 양식을 DDI로 바꿔도 XSD를 통과해야 한다. 변수 몇 개와 범주만 채운 양식도 마찬가지다.

### 14.3 ID 자동 부여와 이름으로 연결 (xlsx → DDI 방향만)
- `Variables.ID`가 비어 있으면 `V<k>`를 붙인다.
  - k는 1부터 시작해 행 순서대로 매기며, 이미 쓰인 ID(변수와 그룹 모두)는 건너뛴다.
  - `Groups.ID`가 비어 있으면 같은 방식으로 `VG<k>`를 붙인다.
- `Categories.varID`가 비어 있으면 `varName`으로 변수를 찾는다.
  - 같은 이름의 변수가 정확히 하나일 때만 연결한다.
  - 없거나 둘 이상이면 셀 주소와 함께 오류를 낸다.
  - `varID`가 채워져 있으면 지금처럼 ID로만 찾는다.
- `Groups.vars`의 각 항목은 먼저 변수 ID로 찾는다.
  - ID가 아니면, 유일한 변수 이름으로 찾아 그 ID로 바꾼다.
  - 둘 다 아니면 지금처럼 경고를 낸다.
- DDI → xlsx 방향은 바뀌지 않는다. 입력 DDI의 `var`와 `varGrp`에 ID가 없으면 지금처럼 거부한다.
- 고정점: 자동으로 붙은 ID는 출력에 들어가므로, 출력 → xlsx → 출력은 여전히 바이트 단위로 같다.

# KOSAC 코드북 엔진 설계

작성 2026-10-28. 상태: 초안 (빌더 검토 전, 사용자 승인 전).

근거 자료:
- 재단 요청서 v1.2: 2장(양식), 4장(XML 구조), 6-3(나) C01~C18, 6-5 F5, 6-6 F6, 부록 A~E.
- 양식 v3.2 작성 실물: 2024 과학기술 국민인식도 작성 예시 xlsx.
- WBS 총괄 회신(2026-10-28)의 결정과 추가 요구.
- ddi-xlsx 코드: HEAD `b9e7ddc`.

## 1. 목적과 범위

ddi-xlsx 저장소에 재단 코드북 엔진을 추가한다. 이 엔진은 다음 두 가지를 다룬다.
- 재단 양식 KOSAC-SURVEY-META v3.2(xlsx)
- kosac 확장 네임스페이스 `urn:kosac:survey-meta:3.3`를 쓰는 DDI 2.5 코드북 XML

datajam ④는 이 엔진을 Go 모듈로 import해서 쓴다.

엔진이 하는 일:
1. 용역사가 올린 v3.2 양식을 읽고 S05와 C01~C18로 검증한다.
2. 분석 메타 초안을 자동으로 판정한다(부록 B). 후보 제안도 여기에 포함한다.
3. datajam이 넘긴 담당자 입력과 시스템 정보를 받아 4장 구조의 코드북 XML을 조립한다(F6).
4. 조립한 XML을 다시 모델로 읽는다. 이전 버전 ID를 물려받거나 다시 내보낼 때 쓴다.
5. 네이티브 경로: walla의 DDI export를 모델로 바꾼다.
6. XSD 검증을 두 투영으로 나누어 준비한다. 실제 xmllint 실행은 호출자가 주입한다.
7. 담당자가 볼 수 있도록 모델을 v3.2 양식 xlsx로 쓴다.

엔진이 하지 않는 일(datajam 범위):
- S01~S04, S06, S07
- X01~X15(응답 데이터 검사)
- K01~K07(공통표준코드 값 검증)
- F5 화면, F11 클린징 실행, cleansing.json
- 해시 계산, 파일 저장, S3, 이력, API

엔진은 바이트를 받아 모델과 바이트를 돌려준다. 파일 시스템, 네트워크, 외부 프로세스는 쓰지 않는다.

## 2. 결정 기록

| # | 결정 | 출처 |
|---|---|---|
| D1 | XSD는 **두 투영**으로 검증한다. 공식 DDI 2.5 XSD는 수정하지 않는다. kosac 3.3 XSD 초안과 변경 기록을 저장소에 둔다. | WBS 회신 1 |
| D2 | TEMPLATE_ID 허용 목록: `KOFAC-SURVEY-META`(기본, 통과)와 `KOSAC-SURVEY-META`(W로 통과). 이 외의 값은 R이다. | WBS 회신 2 |
| D3 | 빈 v3.2 원본이 오기 전까지 작성 예시로 만든 **임시 내장본**을 쓴다. `_meta`에 "임시 내장본, 원본 교체 예정"을 표시한다. | WBS 회신 3 |
| D4 | GitHub teamPaprika 조직에 사설 저장소 `ddi-xlsx`를 만들고, 모듈 경로는 `github.com/teamPaprika/ddi-xlsx`를 그대로 쓴다. datajam 빌드에 GOPRIVATE를 준다. 저장소를 만들고 처음 push하는 일은 이 저장소 사용자의 확인을 받은 뒤에 한다. 그전까지 datajam은 go.work나 replace로 개발한다. | 사용자 결정 (WBS 경유, 2026-10-28) |
| D5 | walla 이름 규칙(`f_…` 변수명, `walla_*` 시스템 변수 6개)은 한 함수에 모아 갈아 끼울 수 있게 한다. 매핑 테스트는 현재 testdata로 만든다. | WBS 회신 5 |
| D6 | RESP_UNIVERSE 파서와 평가기를 공개 API로 둔다. 문법은 C10에 `!=`와 `pattern:`을 더한 것이다. | WBS 추가 |
| D7 | `AssignIDs(cb, prev)`: 같은 (FILE_NO, VAR_NAME)은 이전 ID를 물려받는다. 시스템 변수는 S{n}으로 고정한다. | WBS 추가 |
| D8 | FromWallaDDI는 측정수준 힌트 맵을 선택 인자로 받는다. | WBS 추가 |
| D9 | Issue에 기계용 키 FileNo·VarName·Code를 넣는다. | WBS 추가 |
| D10 | DeriveDefaults가 결측 코드 -8·-7·-9를 추가한다. 04 시트에 이미 있는 결측 행은 라벨만 규정 문구로 맞춘다. | WBS 추가, F6 |
| D11 | 범용 계층(`ddi`, `workbook`)은 kosac을 모른다. 의존 방향은 kosac → ddi 하나다. 모듈은 하나로 둔다. | 설계 요지 §1 |
| D12 | 외부 의존성은 계속 excelize 하나다. 순수 Go XSD 검증기는 들이지 않는다. | 기존 원칙 |

## 3. 패키지

```
xsd/                    DDI 2.5 XSD (기존) + embed.go: FS 공개 (신규)
kosac/                  모델, XML 조립·읽기, 자동 판정, ID 부여, walla 변환, Issue
kosac/template/         v3.2 xlsx 읽기·쓰기, 임시 내장본 blank_v3.2.xlsx (embed)
kosac/check/            S05 결과 합치기, C01~C18, 두 투영과 배치 검사
kosac/universe/         RESP_UNIVERSE·클린징 condition 파서와 평가기 (의존성 없음)
kosac/xsd/              kosac-3.3.xsd, CHANGES.md(재단 승인용 변경 기록), embed.go
cmd/ddi-xlsx            하위 명령 kosac-check, kosac-xml, kosac-template 추가
```

- `Issue` 타입은 `kosac`에 둔다. template과 check가 모두 반환하므로, 한쪽에 두면 import 순환이 생긴다.
- `kosac`은 `ddi`만 import한다. `workbook`과 `internal/schema`는 쓰지 않는다. 범용 Variables/Categories 양식은 v3.2와 모양이 다르기 때문이다.
- wasm·npm에는 이번 단계에서 kosac을 노출하지 않는다. datajam은 Go 서버이고, 웹 소비자는 아직 없다. 필요해지면 `wasm/main.go`에 함수 세 개를 더하면 된다.

## 4. 모델

모델은 시트를 그대로 비춘다. 양식 값은 KEY로 찾는 레코드에 담는다. 엔진이 붙이는 값은 타입이 있는 필드에 담는다.

- 01 시트 44개 필드를 구조체 필드 44개로 만들지 않는다. 읽기, 쓰기, 필수 검사, 형식 검사가 모두 KEY 목록 하나로 돌아가기 때문이다.

```go
type Loc struct{ Sheet, Cell string }          // 양식에서 읽은 값만. 예: "03_변수정의", "G57"
type Field struct{ Value string; Loc Loc }
type Record map[string]Field                   // 양식 KEY → 셀 값(표시 문자열)

type Codebook struct {
    Template TemplateMeta   // _meta: ID, Version, Created (+Loc)
    Study    Record         // 01_조사정보 44 KEY
    Admin    StudyAdmin     // CategoryCode, Round{Year, WaveType, WaveNo}
    Files    []*File        // 02 행 순서
    Vars     []*Var         // 03 행 순서. 시스템 변수는 담지 않는다(조립 때 생성)
    Codes    []*Code        // 04 행 순서. 평평한 목록이다. 변수와는 (FILE_NO, VAR_NAME)으로 잇는다
    Package  *Package       // 시스템 부여. nil이면 docDscr를 통째로 쓰지 않는다
}
type StudyAdmin struct {
    CategoryCode string
    Round        struct{ Year, WaveType, WaveNo string }
    WallaID      string     // titlStmt/IDNo@agency="walla". 네이티브는 FromWallaDDI가 채운다
}
type Package struct {        // 요청서 4-2 kosac:package 표와 부록 D의 속성을 그대로 담는다
    ExportID, CodebookVersion, PreviousVersion, ApprovedBy, ApprovedAt string // @cleansed는 클린징 규칙에서 조립 때 정한다
    ExportedAt, Software, SoftwareVersion string   // docDscr prodDate·software
    Source     struct{ Kind, PublishedVersion, TemplateVersion string } // kosac:sourceSurvey(@wallaId는 Admin.WallaID)
    Files      []PackageFile // 주어진 순서대로 쓴다
    Validation *Validation
}
type PackageFile struct{ Role, FileRef, Name, Bytes, SHA256, Pages string }
type Validation struct{ PassedAt, Ruleset, WarningCount string; Notes []string }
type Suggestion struct {
    Kind            string  // "STD_ID" | "FK"
    FileNo, VarName string
    Value           string  // STD_ID 또는 코드셋 ID
    Confidence      float64
    Level           string  // "high" | "check"
    CodeMap         map[string]string // FK만: 원천 코드 → 표준코드
}
type File struct {
    Rec       Record        // 02 KEY
    LinkLevel, SensLevel string
    PackageName, SHA256, Bytes, RawName, RawSHA256, RawBytes string
    Cleansing *Cleansing    // 규칙이 없으면 nil
}
type Var struct {
    ID  string              // V{n}. AssignIDs가 채운다
    Rec Record              // 03 KEY
    Analysis                // Role, DimMeasure, AggRules, PhysicalName, LogicalName, Synonyms,
                            // IncludeYn, CodeSetID, MRGroup, NoneOption, GridGroup, RowLabel, RankNo,
                            // Scale{Min,Max,Dir}, TextFormat, ParentVar
                            // NoneOption은 MR 묶음 멤버마다 같은 값을 둔다. 조립은 첫 멤버 값을 쓴다.
                            // 파일이 달라도 묶음 키가 겹치지 않게 하려고 맵 대신 변수에 둔다
    FK  *FK
}
type Code struct {
    Rec     Record          // 04 KEY
    Type    string          // VALID|MISSING|DK|NA
    StdCode, StdSet string
}
```

- **값은 셀에 적힌 그대로** 둔다. 읽을 때 고치지 않는다. 검증은 원래 값과 위치를 보고 판정해야 하기 때문이다.
- 조립할 때 정해지는 값은 저장하지 않는다. 다음 값들이 그렇다.
  - `kosac:measLevel`: MEAS_LEVEL에서 정한다.
  - `var@kosac:cleansed`, `package@cleansed`: 클린징 규칙에서 정한다.
  - `fkRef`: FK에서 정한다.
  - varGrp: 변수 필드에서 정한다.
  - 시스템 변수
- **변수의 키는 (FILE_NO, VAR_NAME)**이다. 조회 함수 `cb.Var(fileNo, name)`과 `cb.File(fileNo)`를 둔다. 담당자 입력은 반환된 포인터의 필드에 직접 쓴다. setter는 만들지 않는다.
- 요지에 있던 `Origin` 맵(값 출처)은 만들지 않는다. 출처는 4장 표에서 KEY별로 고정되어 있으므로 datajam이 화면에 표시할 수 있다.
- 요지에 있던 `extra` 노드 보관은 만들지 않는다. 엔진이 다시 읽는 XML은 엔진이 쓴 XML뿐이다(§7).

## 5. 양식 v3.2 읽기·쓰기 (kosac/template)

### 5.1 실물 구조 (작성 예시로 확인)

| 시트 | 모양 | 읽는 법 |
|---|---|---|
| 00_작성안내 | 안내문 | 읽지 않는다 |
| 01_조사정보 | 세로형. 1행 머리: `KEY, 영역, 항목, 구분, 입력값, 작성 안내, DDI 2.5`. 2~45행에 필드가 하나씩 있다. | 머리글 `KEY`와 `입력값`으로 열을 찾는다. 행은 KEY 값으로 찾는다. |
| 02_데이터파일 | 1행 KEY(10개), 2행 한글명, 3행 안내. 4행부터 값 | 1행 KEY로 열을 찾는다. 4행부터 읽고, 모든 KEY 셀이 빈 행은 건너뛴다. |
| 03_변수정의 | 같은 모양. KEY 15개 | 같음 |
| 04_코드값 | 같은 모양. KEY 5개 | 같음 |
| _meta | A열 키, B열 값: TEMPLATE_ID, TEMPLATE_VERSION, CREATED | 키로 찾는다. 모르는 키는 무시한다. |
| 예시_설명 | 작성 예시에만 있다 | 무시한다 |

- 셀 값은 표시 문자열로 읽는다. ddi-xlsx의 기존 규칙과 같다. 실물에서 날짜와 숫자는 문자열 셀이었다.
- 숫자 셀에 표시 형식이 걸려 "10,000"처럼 보이면, 형식 검사(C02)에서 천 단위 쉼표를 허용한다.

### 5.2 S05 (template.Read가 낸다)

- 시트 01~04와 `_meta`가 없으면 R이다.
- 02~04 시트 1행의 KEY 집합이 양식과 다르면 R이다. 모자라도, 남아도 R이다. 순서는 보지 않는다.
- 01 시트에 KEY 행이 빠졌거나 모르는 KEY가 있으면 R이다.
- TEMPLATE_ID: `KOFAC-SURVEY-META`는 통과, `KOSAC-SURVEY-META`는 W, 그 밖의 값은 R이다(D2).
- TEMPLATE_VERSION은 허용 목록에 없으면 R이다. 허용 목록의 기본값은 `["3.2"]`이고, 호출자가 `ReadOptions`로 바꿀 수 있다.
- S05에 R이 하나라도 있으면 `Read`는 `nil` 모델과 Issue 목록을 돌려준다. 이 경우 C 검사는 하지 않는다(요청서 6-3: S에서 반려하면 C·X를 실행하지 않는다).
- 파일을 열 수 없는 경우(zip이 아니거나 크기 한도를 넘는 경우)는 `error`다. 크기 한도는 workbook 패키지의 압축 해제 한도와 같다.

### 5.3 쓰기

- 내장 빈 양식에 값만 채운다.
  - 01 시트는 E열에, KEY 행을 찾아 쓴다.
  - 02~04 시트는 4행부터 쓴다.
  - 셀 색, 목록, 안내 행, 필터 범위는 원본 그대로 둔다.
- 목록(데이터 유효성)은 원본 범위까지만 걸려 있다: 02 시트 23행, 03 시트 3003행. 04 시트에는 목록이 없다. 이보다 긴 행에는 목록이 없다. 이 사실은 README에 적는다.
- 자동 필터 범위(02 A2:I23, 03 A2:O3003, 04 A2:E15003)도 원본 그대로 둔다.
- 압축 해제 한도는 workbook과 같은 값(`max(64MiB, 100×len)`)을 쓴다. workbook의 함수는 비공개라서 kosac/template에 같은 함수를 둔다.
- v3.2에 열이 없는 값(분석 메타, FK, 담당자 입력)은 쓰지 않는다. 양식은 용역사 값만 보여 준다.
- 왕복 보장: `Read(Write(cb))`의 Record 값은 `cb`의 값과 같다. Loc는 새 위치가 된다.

### 5.4 임시 내장본 (D3)

- 작성 예시에서 한 번 만들어 `kosac/template/blank_v3.2.xlsx`로 커밋한다. 만드는 절차는 Go 테스트 헬퍼(`-update`)로 남겨 재현할 수 있게 한다.
  1. 01 시트 E2:E45의 값을 비운다. 단, RULE_VERSION의 목록 값 `v1.0`은 비우지 않는다.
  2. 02~04 시트 4행 이하의 값을 비운다. 스타일은 남긴다.
  3. `예시_설명` 시트를 지운다.
  4. 00_작성안내 1행의 "[작성예시] 2024 …" 제목을 일반 제목으로 바꾼다.
  5. `_meta`에 `NOTE | 임시 내장본, 원본 교체 예정` 행을 더한다.
- 공식 원본이 오면 이 파일만 바꾼다. 테스트는 이 파일의 머리 행과 목록이 §5.1과 같은지 확인한다.

## 6. 코드북 XML 조립 (F6, `cb.XML()` / `cb.JSON()`)

### 6.1 원칙

- 트리는 `ddi.Node`로 만들고, 쓰기는 `ddi.WriteXML`/`WriteJSON`에 맡긴다. 정규 레이아웃, 고정점, XML/JSON 한 트리라는 성질을 그대로 물려받는다.
- 루트 요소: `<codeBook xmlns="ddi:codebook:2_5" xmlns:kosac="urn:kosac:survey-meta:3.3" version="2.5" ID="…">`.
- **DDI 요소는 XSD 순서대로** 만든다. 조립 코드가 순서대로 붙이므로 정렬 코드는 두지 않는다. 테스트가 xmllint로 순서를 확인한다.
- **kosac 요소는 같은 부모의 DDI 자식 뒤에** 붙인다. 순서는 §6.6 표의 순서다. 투영하면 사라지는 요소라서 DDI 순서에 영향이 없고, 규칙이 하나라서 단순하다. 요청서 부록 D는 sampProc 바로 뒤처럼 중간에 끼워 넣었다. 이 차이는 재단 보고 목록(§13)에 적는다.
- kosac 속성은 DDI 속성 뒤에 붙인다.
- 값이 빈 칸이면 해당 요소나 속성을 쓰지 않는다. 자식도 속성도 없는 컨테이너(rspStmt, prodStmt, method, anlyInfo 등)도 쓰지 않는다. 예외는 두 가지이다.
  - XSD가 요구하는 요소(stdyDscr/citation/titlStmt/titl 등)
  - 사유가 빈 `kosac:unverified`
- **TITLE이 비면 조립하지 않는다.** titl은 XSD 필수 요소이기 때문이다.
- §6.4~§6.7 표의 행 순서는 출력 순서가 아니다. DDI 요소는 XSD 순서로, kosac 요소는 §6.6 순서로 쓴다.
- 반복되는 것들의 순서는 다음과 같다.
  - otherMat: report → questionnaire → extra(EXTRA_FILES 순서)
  - varGrp: 같은 유형 안에서는 첫 멤버의 03 순서를 따른다.
  - kosac:file의 자식: idVar → systemIdVar → link
  - kosac:package의 자식: sourceSurvey → file(주어진 순서) → validation
  - kosac:unverified: 01 KEY 순서 → 02(파일 순서, KEY 순서) → 03(변수 순서, KEY 순서)
- **정수 자리**(sampleSize, caseQnty, varQnty, 정수형 kosac 값)에 넣는 값은 정규화한다.
  - 천 단위 쉼표를 지운다.
  - `^\d+\.0+$`는 정수로 바꾼다.
  - C02가 "10,000"을 허용하므로, 이 정규화가 없으면 XSD를 통과하지 못한다.
- 텍스트 값은 `ddi.NormalizeText`를 거쳐 넣는다. 그래야 다시 읽어도 같은 트리가 된다.
- "확인 불가" 처리:
  - 값이 `^확인\s*불가\s*[:：]\s*(.*)$`에 맞으면 확인 불가로 본다.
  - 텍스트 자리(labl, universe, cleanOps 등)에는 원문을 그대로 넣는다.
  - 형식이 있는 자리(날짜, 정수, 목록 값, 변수 참조)에는 값을 쓰지 않는다.
  - 어느 경우든 `kosac:unverified`에 KEY와 사유를 남긴다.
- 조립 전 조건이 하나라도 어긋나면 `XML()`은 `error`를 돌려준다. 조건은 다음과 같다.
  - 모든 조사 변수에 ID가 있다.
  - 이름으로 참조한 변수(ID_VAR, LINK_VAR, WEIGHT_VAR, SPLIT_VAR, STRATA_VARS, CLUSTER_VAR, 클린징 rule.var, ParentVar)가 존재한다.
    - 이름은 그 파일의 조사 변수에서 먼저 찾고, 없으면 그 파일의 시스템 변수에서 찾는다.
    - 예: 네이티브의 ID_VAR=walla_response_id는 그 파일의 S{n}으로 바뀐다.
    - 01 시트 참조(WEIGHT_VAR 등)는 파일을 지정하지 않는다. 그래서 FILE_NO 순서로 처음 찾은 변수를 쓴다.
  - FILE_NO 참조가 맞다.
  - 오류 메시지에는 어긋난 항목 목록을 담는다. datajam은 C 검사를 통과한 뒤에 조립하므로, 정상 흐름에서는 이 오류가 나지 않는다.

### 6.2 조립 순서 (F6)

docDscr(package) → stdyDscr → fileDscr ×n(+cleansing) → dataDscr[varGrp(section → grid → multipleResp → rank) → var(시스템 변수 → 03 순서)] → otherMat.

### 6.3 ID

| 대상 | 규칙 |
|---|---|
| codeBook@ID | `CB-{DATA_NO}-{Round.Year}[-{waveType}{waveNo}]-v{codebookVersion}`. waveType이 main이면 wave 부분을 뺀다. 재료가 하나라도 없으면 ID 속성을 쓰지 않는다. |
| fileDscr@ID | `F{FILE_NO}` |
| 조사 변수 | `V{n}`. AssignIDs(§8.3)가 부여한다 |
| 시스템 변수 | `S{n}`. 파일 순서 × 고정 목록 순서로 1부터 매긴다. 파일이 2개면 F1은 S1~S6, F2는 S7~S12이다 |
| varGrp | `G{n}`. 조립 순서대로 매긴다. 버전 간 안정성은 보장하지 않는다(요청서도 요구하지 않음) |
| qstn | ID를 쓰지 않는다. Q_NO는 `qstn@seqNo`에 넣는다(§13 항목 7) |

### 6.4 01 시트 → docDscr·stdyDscr·otherMat

| KEY | 위치 | 비고 |
|---|---|---|
| (시스템) | docDscr/citation/titlStmt/titl = `{TITLE} 코드북 v{codebookVersion}` | Package가 있고 CodebookVersion이 있을 때만 |
| (시스템) | docDscr/citation/prodStmt/prodDate@date + 같은 텍스트, software@version + 텍스트 Package.Software | Package.ExportedAt, Software, SoftwareVersion |
| TITLE | stdyDscr/citation/titlStmt/titl | |
| PROJECT_TITLE | titlStmt/altTitl | |
| PROJECT_NO | titlStmt/IDNo@agency="kosac" | |
| (담당자·시스템) | titlStmt/IDNo@agency="walla" = Admin.WallaID | |
| PI_ORG, PI_NAME | rspStmt/AuthEnty. PI_NAME이 있으면 `@affiliation=PI_ORG`(PI_ORG가 비어도 빈 속성을 쓴다), 텍스트는 PI_NAME. PI_NAME이 없으면 텍스트 PI_ORG, affiliation 없음 | 거꾸로 읽기: affiliation 속성이 있으면 PI_ORG=속성, PI_NAME=텍스트. 없으면 PI_ORG=텍스트 |
| FIELD_ORG | prodStmt/producer | |
| FUNDER | prodStmt/fundAg | |
| KEYWORDS | stdyInfo/subject/keyword ×n | `;`로 나누고 앞뒤 공백을 지운다. 빈 항목은 버린다 |
| PURPOSE | stdyInfo/abstract@contentType="purpose" | |
| SUMMARY | stdyInfo/abstract@contentType="abstract" | 요청서의 "summary"는 XSD 목록에 없다(§13 항목 1) |
| FIELD_START, FIELD_END | sumDscr/collDate@event="start"/"end"@date (텍스트 없음) | 부록 D 모양 |
| ANALYSIS_UNIT | sumDscr/anlyUnit | XSD 순서상 universe 앞이다 |
| UNIVERSE | sumDscr/universe@clusion="I" | |
| SAMPLING_DESC | method/dataColl/sampProc | |
| TARGET_N | dataColl/targetSampleSize/sampleSize | 텍스트를 바로 넣을 수 없다(§13 항목 4) |
| MODE | dataColl/collMode | |
| TOOL | dataColl/resInstru | |
| WEIGHT_METHOD | dataColl/weight | |
| EDIT_METHOD | dataColl/cleanOps | |
| SAMPLING_TYPE, SAMPLE_FRAME, STRATA_VARS, CLUSTER_VAR | dataColl/kosac:sampling@type @frame @strataVars @clusterVar | 변수는 ID로 쓴다. STRATA_VARS는 공백으로 구분한 ID 목록 |
| SPLIT_BALLOT, SPLIT_VAR | dataColl/kosac:splitBallot@yn @var | |
| WEIGHT_YN, WEIGHT_VAR | dataColl/kosac:weight@yn @var | 가중치 변수에는 `var@wgt="wgt"`도 붙인다 |
| EXCLUDE_N | dataColl/kosac:excludeN (정수 텍스트) | |
| EXCLUDE_RULE | dataColl/kosac:excludeRule | 요청서는 cleanOps에 함께 넣으려 했다. 두 값을 한 요소에 넣으면 거꾸로 읽을 수 없어서 나누었다 |
| RESPONSE_RATE | method/anlyInfo/respRate | respRate는 dataColl이 아니라 anlyInfo 아래에 있다(§13 항목 5) |
| CONTACT_N, RR_FORMULA | anlyInfo/kosac:contactN, kosac:rrFormula | |
| RULE_VERSION, SURVEY_TYPE, DATA_NO, VALID_N, SCHOOL_CODE_YEAR, BIZ_YEAR | stdyDscr/kosac:study@ruleVersion @surveyType @dataNo @validN @schoolCodeYear @bizYear | |
| (담당자) CATEGORY_CODE | kosac:study@categoryCode | |
| (담당자) round | kosac:study/kosac:round@year @waveType @waveNo | |
| BIZ_NAME | kosac:study/kosac:bizName (원문 한 요소, 나누지 않는다) | |
| POP_SOURCE | kosac:study/kosac:popSource | 요청서 4-2 표는 sumDscr 옆이라고 했다. 연구 단위 확장 값이 한곳에 모이도록 여기에 둔다(§13) |
| (확인 불가) | kosac:study/kosac:unverified@key [@file] [@var] = 사유 | 02 시트 값이면 @file="F{n}", 03 시트 값이면 @var에 변수 ID |
| QUESTIONNAIRE, REPORT_FILE, EXTRA_FILES | codeBook/otherMat@type=questionnaire\|report\|extra @level="study" @URI, labl=셀 원문 | §6.8 |

### 6.5 02 시트 → fileDscr

| KEY·값 | 위치 |
|---|---|
| FILE_NO | fileDscr@ID="F{n}", kosac:file@fileNo |
| FILE_NAME | fileTxt/fileName = File.PackageName이 있으면 그 값, 없으면 FILE_NAME. 원래 이름은 kosac:file@sourceName에 둔다 |
| CONTENT | fileTxt/fileCont |
| VALID_N, (시스템) 변수 수 | fileTxt/dimensns/caseQnty, varQnty(시스템 변수를 포함한 수) |
| FILE_FORMAT | fileTxt/fileType@charset="UTF-8" |
| NOTE | fileDscr/notes. 요청서의 fileTxt/notes는 XSD에 없다(§13 항목 6) |
| UNIT, SensLevel, 해시들 | kosac:file@unit @sensLevel @sha256 @bytes @rawSha256 @rawBytes @rawName |
| ID_VAR | kosac:file/kosac:idVar@ref = 변수 ID |
| (시스템) | kosac:file/kosac:systemIdVar@ref = 그 파일의 walla_response_id 시스템 변수 ID |
| LINK_VAR, LINK_FILE_NO, LinkLevel | kosac:file/kosac:link@var(ID) @fileRef(F{n}) @level |
| Cleansing | fileDscr/kosac:cleansing (요청서 4-2 표 그대로). rule@var는 변수 ID로 바꿔 쓴다. @rawFileRef 같은 패키지 쪽 값은 datajam이 준 값을 그대로 쓴다. 요청서 안에서도 `F1_raw`와 `F1`이 섞여 있기 때문이다 |

### 6.6 kosac 요소 배치와 순서

| DDI 부모 | 뒤에 붙는 kosac 자식 (이 순서) |
|---|---|
| docDscr | package |
| stdyDscr | study |
| dataColl | sampling, splitBallot, weight, excludeN, excludeRule |
| anlyInfo | contactN, rrFormula |
| fileDscr | file, cleansing |
| varGrp | noneOption |
| var | fk, scale, missingCodes, codeSetId, analysis |
| catgry | codeType, stdCode |
| otherMat | pages, sha256, bytes |

- 이 표는 `check`의 배치 검사(§10.3) 표와 같은 것이다. 코드에서는 표 하나를 두 곳이 같이 쓴다.
- 요청서의 `kosac:codeHistory`(네이티브 코드값 이력)는 이번 범위에서 뺀다. 모델에 담을 원천(walla 게시본 이력)이 export에 없기 때문이다.
- kosac 속성:
  - var: seq, role, measLevel, pii, sensitive, reverse, mrType, mrGroup, rowLabel, rankNo, cleansed, textFormat, parentVar
  - varGrp: mrGroup

### 6.7 03·04 시트 → var, catgry

| KEY·값 | 위치 |
|---|---|
| VAR_NAME, FILE_NO | var@name, var@files="F{n}", location@fileid="F{n}" |
| (시스템) 열 순서 | var@kosac:seq. 파일마다 1부터 센다. 시스템 변수가 앞이다 |
| MEAS_LEVEL | var@intrvl: 연속이면 contin, 나머지는 discrete. kosac:measLevel = nominal·ordinal·scale·text·date. varFormat@type: 문자·날짜는 character(날짜는 @category="date"), 나머지는 numeric |
| VAR_LABEL | labl |
| Q_NO, Q_TEXT | qstn@seqNo, qstn/qstnLit |
| RESP_UNIVERSE | universe@clusion="I" (원문) |
| DERIVATION | derivation/drvdesc |
| STD_ID | concept@vocab="KOSAC-STD" |
| NOTE | notes |
| PII, SENSITIVE, REVERSE, MR_TYPE | kosac:pii, sensitive, reverse, mrType. 값이 있으면 셀 값 그대로 쓴다. 양식의 REVERSE 목록은 Y뿐이지만, 용역사가 N을 적어도 왕복이 그대로 되게 하기 위해서다 |
| Analysis | kosac:role 속성, kosac:analysis@physicalName @dimMeasure @aggRules @includeYn @logicalName @fkRef + synonym ×n, kosac:codeSetId, kosac:scale@min @max @dir, kosac:mrGroup·rowLabel·rankNo 속성, textFormat·parentVar |
| FK | kosac:fk (요청서 4-2 표의 속성 전부) |
| 04 CODE, CODE_LABEL, NOTE | catgry/catValu, labl, txt |
| Code.Type, StdCode | catgry@missing="Y"(결측 코드일 때), kosac:codeType, kosac:stdCode@set |
| 결측 코드 목록 | kosac:missingCodes = 그 변수의 실제 결측 코드 집합. 결측 코드는 Code.Type≠VALID인 코드이다. 순서는 -8, -7, -9를 먼저 쓰고, 그 밖의 결측 코드(예: 네이티브의 -6)를 catgry 순서로 잇는다. 예: "-8;-7;-9;-6". X04(값 ∈ 코드 ∪ 결측)와 맞추기 위해서다. 결측 코드가 없는 변수에는 쓰지 않는다 |

- catgry 순서는 04 시트 행 순서다. DeriveDefaults가 더한 결측 코드는 그 뒤에 붙는다.
- 시스템 변수 6개는 아래 표의 고정 정의로 만든다. 요청서 4-2의 walla_device와 walla_channel은 walla 새 export에 없으므로 넣지 않는다(D5). 목록은 표 하나라서 더하기 쉽다.

| 이름 | role | measLevel | pii | varFormat |
|---|---|---|---|---|
| walla_response_id | SYSTEM | text | N | character |
| walla_row_no | SYSTEM | scale | N | numeric |
| walla_customer_key | SYSTEM | text | Y | character |
| walla_submitted_at | SYSTEM | date | Y | character, category=date |
| walla_started_at | SYSTEM | date | Y | character, category=date |
| walla_duration_sec | SYSTEM | scale | N | numeric |

- 시스템 변수:
  - analysis: physicalName은 이름 그대로, dimMeasure는 KEY, includeYn은 N이다.
  - labl: 왈라 응답 ID, 왈라 행 번호, 왈라 고객 키, 왈라 제출 시각, 왈라 시작 시각, 왈라 응답 소요 시간(초) (표 순서대로)
  - 시스템 변수에는 결측 catgry와 missingCodes를 붙이지 않는다.
- **varGrp**는 조립 때 변수 필드로부터 만든다.
  - section: 서로 다른 VAR_GROUP 값마다 하나. 처음 나온 순서를 따른다. labl은 그룹 이름이다.
  - grid: 같은 (FILE_NO, Analysis.GridGroup)을 가진 변수 묶음. labl은 첫 멤버 Q_TEXT의 줄기(§8.1 GridGroup)이다.
  - multipleResp: 같은 (FILE_NO, Analysis.MRGroup)을 가지고 MR_TYPE이 보기별 열인 묶음.
    - `@kosac:mrGroup`을 붙인다. `kosac:noneOption@yn`은 첫 멤버의 Analysis.NoneOption에서 가져온다.
    - labl은 **라벨 접두 규칙**으로 정한다. 공통 접두가 비면 mrGroup 값을 쓴다.
  - rank: 같은 (FILE_NO, MRGroup)이고 MR_TYPE이 순위별 열인 묶음.
    - `@type="other" @otherType="rank"`로 쓰고, `@kosac:mrGroup`도 붙인다. 요청서의 `type="rank"`는 XSD 목록에 없다(§13 항목 2).
    - labl 규칙은 multipleResp와 같다.
  - @var는 멤버 변수 ID 목록이고, 03 순서를 따른다.
- **라벨 접두 규칙**(MR·rank labl과 grid RowLabel이 같이 쓴다):
  1. 멤버 VAR_LABEL들의 공통 접두를 rune 단위로 구한다.
  2. 그 접두를 마지막 구분자 경계까지 줄인다. 구분자는 `: `, ` - `, `·`, ` (`, ` [`이다. 경계가 없으면 공통 접두는 없는 것으로 본다. 그래야 "과학기술과 …"가 "과 …"로 잘리지 않는다.
  3. 끝의 공백·`:`·`-`·`(`·`[`를 지운다.

### 6.8 otherMat과 package

- REPORT_FILE은 otherMat 하나가 된다.
  - type="report"
  - URI는 Package에 role=report 파일이 있으면 그 패키지 이름이고, 없으면 셀 값이다.
  - labl은 셀 원문이다.
  - kosac:sha256과 kosac:bytes는 그 Package 파일의 값이다.
- QUESTIONNAIRE는 세 경우로 나뉜다.
  - 파일명이면 report와 같은 방식으로 쓴다.
  - 보고서 수록 형식(§10.2 C18의 쪽 범위 해석)이면 URI를 report의 URI로 두고, `kosac:pages`에 정규형 범위를 넣는다. 정규형은 `{a}`, `{a}-{b}`, `{a}-`(끝쪽 없음)이다. 작성 예시 `최종보고서 부록 수록 (PDF 336쪽~)`은 `336-`이다.
  - 둘 다 아니면 labl만 쓴다. URI와 pages는 쓰지 않는다. 이 경우 C18이 R을 낸다.
- EXTRA_FILES는 `;`로 나눈 항목마다 otherMat type="extra"를 하나씩 만든다. Package의 role=extra 파일과는 순서로 짝짓는다. 패키지 이름이 `…_extra_{k}`로 바뀌기 때문이다.
- 모든 otherMat에 `@level="study"`를 붙인다. XSD 필수 속성이다(§13 항목 3).
- kosac:package는 요청서 4-2 표와 부록 D를 따른다. 단, 코드북 파일 자신의 행(role=codebook)에는 name만 쓴다. 자기 해시는 자기 안에 넣을 수 없기 때문이다(§13 항목 9). validation 아래에 `kosac:validationNote` ×n을 둔다.

## 7. 다시 읽기와 고정점 (`kosac.Parse`)

- `Parse(b []byte) (*Codebook, error)`: XML이나 JSON을 받는다(`ddi.Parse`). 그다음 §6의 역방향으로 모델을 만든다.
- 받는 범위는 이 엔진이 쓴 모양이다. 다음 경우는 `error`이고, 오류 메시지에 경로를 담는다.
  - 매핑 표에 없는 DDI 요소나 kosac 요소가 있는 경우
  - kosac 네임스페이스가 `kosac` 접두로 루트에 선언되지 않은 경우
- 왜 이렇게 좁히는가: 다른 도구가 만든 kosac 문서는 없다. 모르는 노드를 보관하는 장치(요지의 `extra`)는 코드가 크고, 쓰일 곳이 없다.
- 버전이 바뀔 때:
  - 엔진이 kosac 3.4로 올라가 매핑이 늘어나도 3.3 문서는 그대로 읽는다. 매핑이 더해지는 방향이라 문제가 없다.
  - 반대로 엔진이 모르는 kosac 요소가 든 문서는 오류이다. 예: 재단이 손으로 요소를 더한 문서, 더 새 엔진이 쓴 문서.
  - datajam이 다시 내보낼 때 prev를 읽다가 이 오류가 나면, 원인은 둘 중 하나이다. 엔진 버전이 prev보다 낮거나, prev를 누가 손으로 고쳤다.
- 형식 있는 자리에서 빠진 값은 `kosac:unverified`로 되살린다. 되살린 값은 "확인 불가: {사유}"이다.
- 목록 값(KEYWORDS, EXTRA_FILES, STRATA_VARS)은 `; `로 이어 붙인다. BIZ_NAME은 원문 한 값이라 그대로 둔다.
- 보장(테스트로 고정):
  1. **XML 고정점**: `Build(Parse(Build(cb)))`는 `Build(cb)`와 바이트 단위로 같다. XML과 JSON 모두 그렇다.
  2. **값 보존**: Parse 결과의 Record 값은 원래 값과 같다. 다음 네 가지만 예외이다.
     - 텍스트 정규화(`NormalizeText`)
     - 목록 구분자 정규화
     - 정수 자리 정규화(§6.1: "10,000" → "10000")
     - 형식 자리의 확인 불가 값은 `확인 불가: {사유}` 모양으로 되살아난다. 예: `확인불가：x` → `확인 불가: x`

## 8. 자동 판정과 ID (kosac)

### 8.1 `DeriveDefaults(cb, ref RefData) []Suggestion`

- **비어 있는 칸만 채운다.** 분석 메타와 담당자 입력 항목 모두 같은 규칙이다. 두 번 불러도 결과가 같다. 다음 두 경우에 이 규칙이 필요하다.
  - 담당자가 고친 값을 다시 불러도 덮어쓰지 않는다.
  - FromWallaDDI가 walla 묶음 정보로 미리 채운 MRGroup·GridGroup·RankNo·ParentVar가 그대로 남는다(§12).
- 판정에 쓴 값(예: Role)을 담당자가 나중에 바꿔도, 그 값에서 나온 칸(DimMeasure 등)은 자동으로 다시 계산하지 않는다. 다시 계산하려면 그 칸을 비우고 다시 부르면 된다. 이 규칙은 README에 적는다.
- FK를 확정하는 일은 따로 `ApplyFK`가 한다(§8.2).

| 항목 | 규칙 (부록 B와 F5(나)) |
|---|---|
| Role | 앞에서부터 처음 맞는 것: ID_VAR→ID, WEIGHT_VAR→WEIGHT, SPLIT_VAR→FILTER, DERIVATION 있음→DERIVED, LINK_VAR→LINK, 문자→TEXT, 나머지→ITEM |
| DimMeasure | role WEIGHT→WEIGHT, role ID·LINK→KEY, 보기별 열→MEASURE, 명목→DIM, 서열→BOTH, 연속→MEASURE, 문자·날짜→ATTR |
| AggRules | MR_TYPE을 먼저 본다: 보기별 열→`SUM;RATIO_RESP`, 순위별 열→`COUNT;RATIO_RANK`. 그다음 DimMeasure를 본다: DIM→`COUNT;RATIO`, BOTH→`AVG;COUNT;RATIO`(같은 파일에 role=WEIGHT 변수가 있으면 `;WAVG`를 붙인다), 연속 MEASURE→`AVG;MIN;MAX;SUM`, 나머지→빈 값 |
| PhysicalName | STD_ID가 사전에 있으면 사전 물리명을 쓴다. 없으면 VAR_NAME을 소문자로 바꾸고 `[a-z0-9_]` 밖 문자를 `_`로 바꾼다. 예약어이거나 `walla_`로 시작하면 앞에 `q_`를 붙인다. 64자로 자른다. 파일 안에서 겹치면 `_2`, `_3`을 붙인다(뒤에 붙일 자리를 남기고 자른다) |
| IncludeYn | TEXT 역할이거나 PII=Y이면 N, 그 밖은 Y |
| LogicalName | VAR_LABEL. 유사어는 비운다. AI 제안은 datajam 몫이다 |
| CodeSetID | 코드가 있는 변수마다 하나씩 준다. 묶이는 상대가 없는 집합도 준다. 유효 코드의 (코드, 정규화 라벨) 집합이 같은 변수끼리 같은 ID를 받는다. 정규화 라벨은 공백을 모두 지우고 소문자로 바꾼 값이다("보통 이다" = "보통이다"). ID는 다음 순서로 정한다: ① 이 집합이 사전 코드셋의 (코드, 정규화 라벨) 집합과 같으면 그 코드셋 ID ② 서열이면 `LIKERT{n}_{k}` ③ 그 밖은 `CODES{n}_{k}`. n은 유효 코드 수, k는 같은 접두 안에서 처음 나온 순서이다. 부록 B의 "대표 라벨 약어"는 결정적으로 만들 수 없어 번호로 대신한다 |
| **접두** | MRGroup, GridGroup, C11, C12가 같이 쓴다. 변수명에서 마지막 `_` 앞부분이다. `_`가 없는 이름은 접두가 없다. 그런 이름은 grid·C11·C12 묶음에 들지 않고, MRGroup은 이름 전체가 된다 |
| MRGroup | MR_TYPE이 있는 변수를 (FILE_NO, VAR_GROUP, 접두)로 묶는다. 키는 그 접두이다. 작성 예시에서는 7묶음(MR 3, 순위 4)이 나와 27개 변수와 정확히 맞는다 |
| NoneOption | 묶음 멤버 VAR_LABEL 중 하나가 "없음"으로 끝나면 Y. 멤버 모두에 같은 값을 둔다 |
| RankNo | 순위별 열 묶음 안에서 03 순서로 1부터 매긴다 |
| GridGroup | 키는 (FILE_NO, VAR_GROUP, 접두, Q_TEXT의 줄기, MEAS_LEVEL)이다. **줄기**는 Q_TEXT 끝의 행 표시 부분과 앞뒤 공백을 뗀 문장이다.<br>- 행 표시 패턴은 상수 목록 하나(`gridRowMarkers`)에 모은다. 지금은 2024 작성 예시의 관례인 `\s*▶\s*\[([^\]]*)\]\s*$` 하나뿐이다. 다른 용역사의 표기(`문항 - 행`, `문항(행)`)는 확인한 뒤에 더한다.<br>- 어떤 패턴에도 맞지 않으면 줄기는 Q_TEXT 전체이다. 즉 요청서 원문 규칙(같은 Q_TEXT, 같은 측정수준)으로 떨어진다. 다만 (FILE_NO, VAR_GROUP, 접두) 조건은 남긴다. 이 조건을 빼면 QA6~QA9처럼 문장만 같은 서로 다른 문항이 다시 묶인다.<br>- 잘못 묶는 것보다 안 묶는 편이 안전하다. GRID_GROUP은 요청서에서도 담당자가 확정하는 항목이다. 이 한계는 README의 "자동 판정 한계"에 적는다.<br>- 그 밖의 조건:  명목·서열이고 MR·파생이 아닌 변수만 들어간다. 멤버가 2개 이상일 때만 묶음이 된다. 묶음 키 값은 첫 멤버의 이름이다. 작성 예시에서는 grid 22묶음이 정확히 나온다(QA1, QA3, QA5_1, QA5_2, QA11, QA12, QA13, QA15, QA23, QA26, QA35, QA36, QB2, QB5, QB9, QC6, QC7, QC8, A1, C7, C8, C9). 이전 규칙("같은 Q_TEXT")은 서로 다른 이지선다 4문항(QA6~QA9)을 묶고, 실제 grid는 하나도 찾지 못했다.<br>- 이미 GridGroup이 있는 멤버가 하나라도 있는 묶음은 채우지 않는다. 그래서 다시 계산하려면 묶음 전체를 비워야 한다.<br>- **네이티브 코드북(`Codebook.Native`)에서는 이름으로 grid를 판정하지 않는다.** walla 이름 `f_{F}`는 접두가 모두 `f`라서, 같은 페이지에서 문장이 같은 단일선택 문항이 가짜 grid로 묶이기 때문이다. walla의 grid는 varGrp 멤버십(§12)이 정답이다 |
| RowLabel | Q_TEXT가 행 표시 패턴에 맞으면 그 캡처(x)이다. 없으면 VAR_LABEL에서 라벨 접두 규칙(§6.7)으로 구한 공통 접두를 뺀 나머지이다. 그 결과가 비면 VAR_LABEL 그대로다 |
| Scale | 서열 변수의 유효 코드 최소·최대. 수로 비교한다. REVERSE=Y이면 dir=NEG, 아니면 POS |
| 결측 코드 (D10) | 명목·서열·연속 **조사** 변수마다 -8 MISSING, -7 DK, -9를 확인한다. 04에 없으면 Codes 끝에 행을 더한다(Loc 없음). 이미 있으면 다음처럼 한다. ① 라벨이 RefData의 허용 라벨 중 하나이면 규정 문구(첫 값)로 바꾼다. ② 허용 라벨이 아니면 라벨을 바꾸지 않고, Code.Type이 비어 있을 때만 MISSING으로 둔다(담당자가 정한 Type은 그대로 둔다). ②가 필요한 이유: walla 옛 export는 -7을 "현재 게시본에 없는 값"으로 쓴다(2024 156건, 2025 351건). 이 라벨을 "모르겠다"(DK)로 바꾸면 오류 없이 의미가 바뀐다. 업로드형은 C15가 허용 밖 라벨을 R로 막으므로 F6의 "라벨만 규정 문구로 맞춘다"와 어긋나지 않는다. -9의 codeType은 열린 질문 Q1이다 |
| Code.Type | 결측 코드가 아니면 VALID |
| Admin 기본값 | CategoryCode가 비면 DATA_NO의 첫 `-` 앞, Round.Year가 비면 FIELD_START의 연도, WaveType이 비면 main, 파일 SensLevel이 비면 3 |

- 사전·코드셋이 없는 규칙은 사전 없이 정할 수 있는 결과만 낸다. 사전이 필요한 판정 결과는 빈 값으로 둔다.
- **Suggestion**(적용하지 않는 제안):
  - STD_ID 후보: 라벨과 문항을 토큰으로 나누고 정규화한다.
    - 소문자로 바꾸고, 글자·숫자가 아닌 문자로 자른다.
    - 2자 이상 토큰의 끝에 붙은 조사(은·는·이·가·을·를·의·에·에서·으로·로·와·과·도)를 뗀다.
    - 사전 항목과 Jaccard를 계산한다. 0.8 이상이면 high, 0.6 이상이면 check이다.
  - FK 코드셋 후보:
    - STD_ID가 있으면 사전의 기본 코드셋을 확신도 1.0으로 낸다.
    - 없으면 04 라벨 집합과 코드셋 라벨·대체 라벨 집합의 Jaccard가 0.8 이상인 코드셋을 낸다.
    - 코드별 매핑 제안은 라벨이 일치하는지로 정한다.

### 8.2 `ApplyFK(cb *Codebook, v *Var, def CodeSetDef) error`

- FK가 정해진 변수에서 다음 값을 덮어쓴다.
  - PhysicalName: 코드셋 표준 물리명. 같은 파일에 같은 물리명이 이미 있으면 `_2`, `_3`을 붙인다. 예: 두 변수를 모두 CC_SIDO로 지정한 경우. 이 충돌을 보려고 cb를 받는다.
  - DimMeasure: DIM
  - FK.DimTable, FK.KeyColumn: 비어 있으면 코드셋 정의에서 채운다.
- 이 기능은 F6의 "FK 반영"에 해당한다. K01~K07 값 검증은 datajam이 한다.

### 8.3 `AssignIDs(cb, prev *Codebook)` (D7)

- 03 순서대로 처리한다.
  - prev에 같은 (FILE_NO, VAR_NAME)이 있으면 그 ID를 쓴다.
  - 없으면 `max(prev와 지금 쓰인 V 번호) + 1`을 쓴다.
- prev가 nil이면 V1부터 매긴다.
- 이미 ID가 있는 변수는 건드리지 않는다. 다만 ID가 겹치면 `error`를 돌려준다.
- 시스템 변수 ID는 §6.3 규칙으로 정해지므로 여기서 다루지 않는다.

## 9. RefData와 Issue

```go
type Issue struct {
    Rule    string // "S05", "C01"…"C18", "XSD-DDI", "XSD-KOSAC", "PLACE", "REF"
    Level   Level  // R | W
    Sheet, Cell string // 양식에서 읽은 값일 때
    Path    string     // XML 경로. 예: "dataDscr/var[@ID='V12']". ValidateXML의 Issue에만 채운다
    FileNo, VarName, Code string // 기계용 키 (D9)
    Actual, Expected, Message string // Message는 한국어 한 문장
}
type RefData struct {
    StdItems      map[string]StdItem    // 부록 C std_items.csv 한 행
    CodeSetDefs   map[string]CodeSetDef // code_set_defs.csv
    CodeSets      map[string][]StdCode  // code_sets.csv
    MissingLabels map[string][]string   // "-8" → 허용 라벨들. 첫 값이 규정 문구. nil이면 기본값
    UploadedFiles []string              // nil이면 C05·C18의 파일 대조를 건너뛴다
    PDFPages      map[string]int        // 파일명 → 쪽수. 없으면 쪽 범위 검사를 건너뛴다
    ReservedWords []string              // nil이면 기본 목록
}
```

- 기본 결측 라벨(열린 질문 Q2):
  - -8: "묻지 않음"
  - -7: "모르겠다" (같은 말로 "무응답", "모르겠다/무응답"도 받는다)
  - -9: "유효하지 않음"
- 기본 예약어 목록은 PostgreSQL의 reserved 키워드이다. DW DBMS가 다르면 RefData로 바꾼다(열린 질문 Q3).
- Issue 정렬 순서: Rule 순서 → 시트 순서 → 행 → 열. 같은 입력이면 같은 순서이다.

## 10. 검증 (kosac/check)

### 10.1 `Check(cb, ref) []kosac.Issue`

- C01~C18을 모두 돌린다. R이 나와도 멈추지 않는다.
- 판정 기준(필수 구분, 목록 값)은 엔진 안의 표에서 가져온다. 업로드한 파일의 "구분" 열이나 데이터 유효성에서 읽지 않는다. 용역사가 셀을 고쳐 기준을 바꿀 수 있기 때문이다.
- "확인 불가: 사유" 값을 다루는 규칙:
  - 01 시트 값은 구분과 상관없이 모든 KEY에서 C01 W를 낸다. 작성 예시의 4건 중 TOOL은 '선택' 항목이다.
  - 조건부 필수를 판정할 때 조건 값이 확인 불가이면 그 조건은 판정하지 않는다. 예: EXCLUDE_N이 확인 불가이면 EXCLUDE_RULE을 요구하지 않는다.
  - 02~04 시트 값은 그 칸을 검사하는 규칙 번호로 W를 낸다. 예: RESP_UNIVERSE는 C10 W.
  - 같은 셀의 형식·문법 검사(C02, C10 등)는 건너뛴다.
  - 실물에는 01 시트에 4건, 03 RESP_UNIVERSE에 1건(QC2)이 있다.

### 10.2 규칙 해석 (요청서 6-3(나)에서 열어 둔 부분만)

| ID | 엔진의 해석 |
|---|---|
| C01 | 필수 목록은 01 시트 "구분"=필수인 KEY이다. 조건부 필수도 C01에서 R로 판정한다. 조건은 다음과 같다. ① STRATA_VARS: SAMPLING_TYPE이 확률-층화이거나 할당(비확률)일 때 ② CLUSTER_VAR: 확률-집락·확률-다단계일 때 ③ CONTACT_N·RESPONSE_RATE: SAMPLING_TYPE이 `확률-`로 시작할 때 ④ RR_FORMULA: RESPONSE_RATE가 있을 때 ⑤ EXCLUDE_RULE: EXCLUDE_N ≥ 1일 때 ⑥ SCHOOL_CODE_YEAR: STD_ID=SCHOOL.ID인 변수가 있을 때 ⑦ BIZ_NAME·BIZ_YEAR: SURVEY_TYPE=사업성과일 때. WEIGHT_*는 C03, SPLIT_VAR는 C04가 맡는다 |
| C02 | 날짜는 실제 날짜인 YYYY-MM-DD. 정수 항목(TARGET_N, VALID_N, CONTACT_N, EXCLUDE_N)은 천 단위 쉼표를 허용한다(조립 때 정규화, §6.1). RESPONSE_RATE는 소수를 허용한다. BIZ_YEAR와 SCHOOL_CODE_YEAR는 YYYY. 목록 항목의 목록은 양식의 데이터 유효성과 같은 값을 엔진 상수로 둔다(RULE_VERSION, ANALYSIS_UNIT, SAMPLING_TYPE, SPLIT_BALLOT, MODE, WEIGHT_YN, SURVEY_TYPE). DATA_NO는 `^[\p{L}\p{N}._-]+$`이어야 한다. DATA_NO는 codeBook@ID(xs:ID)에 들어간다. 그런데 검증기와 버전에 따라 루트 ID 형식을 검사하는 경우도 있고 검사하지 않는 경우도 있다. 그래서 양식 단계에서 먼저 막는다 |
| C03 | WEIGHT_YN=Y이면 WEIGHT_VAR가 있고 연속이어야 하며, **WEIGHT_METHOD도 비어 있지 않아야 한다**. N이면 WEIGHT_VAR가 비어 있어야 한다 |
| C03·C04·C05 | 변수 존재는 03 시트에서 이름으로 확인한다. 01 시트 참조는 FILE_NO와 상관없이 찾는다. 01 시트는 파일을 지정하지 않기 때문이다. 02 시트 참조(ID_VAR, LINK_VAR)는 그 파일 안에서 찾는다. **시스템 변수 이름은 모든 파일에 있는 것으로 본다**(네이티브의 ID_VAR=walla_response_id) |
| C05 | FILE_NO 집합이 {1..n}이어야 한다. FILE_NO의 모양은 `^[1-9][0-9]*$`이다(조립과 같은 검사기 하나를 쓴다. "01"과 "+1"은 R). LINK_FILE_NO는 LINK_VAR가 비어 있어도 값이 있으면 검사한다. 02 VALID_N은 정수다(천 단위 쉼표 허용, caseQnty에 들어간다). UNIT은 양식 목록 값({개인,학교,기관,기타})이어야 한다. 파일명 대조는 RefData.UploadedFiles가 있을 때만 한다 |
| C06 보충 | PII와 SENSITIVE는 {Y,N} 중 하나여야 한다. REVERSE는 값이 있으면 {Y,N} 중 하나여야 하고, 이것은 C12로 낸다. 공백을 지우지 않고 그대로 비교한다. kosac 속성은 XSD에서 정확히 일치해야 하기 때문이다 |
| C03·C05 보충 | WEIGHT_YN이 Y도 N도 아니면(확인 불가 등), WEIGHT_VAR에 값이 있을 때 그 변수가 있어야 한다. 02·03 행의 FILE_NO가 "확인 불가"이면 §10.1의 예외로 R이다. 파일을 정할 수 없으면 조립할 수 없기 때문이다 |
| C04 보충 | CLUSTER_VAR는 이름 하나이다(열린 질문 Q8). SPLIT_VAR는 SPLIT_BALLOT 값과 상관없이, 형식이 있는 값이면 03 시트에 있어야 한다. 조립이 항상 이 이름을 풀기 때문이다 |
| C-통과 ⇒ 조립 성공 | C 검사에 R이 없는 모델은 `XML()`이 오류를 내지 않고, ValidateXML에서 XSD·PLACE·REF가 0건이어야 한다. 테스트로 고정한다 |
| C07 | 코드값 행 수는 결측 코드(-6~-9)를 뺀 유효 코드로 센다 |
| C08 | 조건마다 Issue를 하나씩 낸다. 작성 예시에서는 30자 초과 78건과 라벨=문항 67건, 합계 145건이 나온다. 길이는 rune 수로 센다 |
| C09 | 패턴은 `^(문항|[Qq])\s*\d+`이다. 길이는 rune 수로 센다 |
| C10 | universe 파서(§11)의 전체 문법(`!=`, `pattern:` 포함)을 받는다. 참조 변수는 같은 FILE_NO에 있어야 한다. 값 대조는 참조 변수에 유효 코드가 있을 때만 한다. 범위의 경계값은 대조하지 않는다. 작성 예시 29종은 모두 통과한다 |
| C11 | 보기별 열 코드값 검사는 유효 코드만 본다. "같은 묶음 MR_TYPE 동일"은 (FILE_NO, VAR_GROUP, 접두)가 같은 **명목** 변수끼리만 본다. `Q5_other` 같은 문자 변수가 잘못 걸리지 않게 하기 위해서다. 접두가 없는 이름은 건너뛴다 |
| C12 | "같은 VAR_GROUP"을 (FILE_NO, VAR_GROUP, 접두)로 해석한다. 접두가 없는 이름은 건너뛴다. 글자 그대로 적용하면 작성 예시에서 0건이다. 이 해석을 쓰면 요청서가 든 C8_1·C8_2·C9_1·C9_2 4건과 정확히 같아진다 |
| C13 | 산출식에서 변수명처럼 생긴 토큰만 뽑는다. 정규식은 `[A-Za-z][A-Za-z0-9_]*`이고, 숫자나 `_`가 들어 있어야 한다. "PCA"나 "NA" 같은 말이 오탐되지 않게 하기 위해서다. `~` 범위는 양 끝만 본다. 토큰이 같은 파일에 없으면 R이다. 실물 2건("QA1_1, QA1_2를 …")은 통과한다 |
| C15 | 라벨 대조만 한다. "양수 코드와 충돌 없음"의 뜻은 열린 질문 Q4이다 |
| C16 | RefData.StdItems가 nil이면 건너뛴다 |
| C17 | 기관 식별자 변수는 STD_ID=SCHOOL.ID이거나, VAR_LABEL이 `학교\s*(ID|코드)|기관\s*코드`에 맞는 변수이다. NOTE에 4자리 연도가 없으면 W이다 |
| C18 | 판정 순서는 쪽 범위 형식 → 파일명이다. C18과 조립(otherMat)이 같은 함수 하나를 쓴다. 파일명은 확장자(`\.[A-Za-z0-9]{2,5}$`)로 끝나는 값이다. 쪽 범위는 두 형식을 받는다. ① `^REPORT:p(\d+)(?:-(\d+)?)?$` ② `수록`이 들어 있고 `(\d+)\s*쪽?\s*(?:[~\-–～∼〜]\s*(?:(\d+)\s*쪽?)?)?\s*\)?\s*$`로 끝나는 값. 범위 기호에는 한글 문서에서 자주 들어오는 전각 물결 `～`, `∼`, `〜`도 포함한다. 끝쪽은 생략할 수 있다. 정규형은 `{a}`, `{a}-{b}`, `{a}-`이다. 작성 예시 `최종보고서 부록 수록 (PDF 336쪽~)`은 `336-`이 된다. 셋 다 아니면 R이다. 쪽 수 대조는 RefData.PDFPages에 REPORT_FILE이 있을 때만 한다 |

### 10.3 `ValidateXML(xml []byte, v Validator) []kosac.Issue` (D1)

```go
type Schema int // SchemaDDI | SchemaKosac
type Validator func(s Schema, doc []byte) (problems []string, err error)
```

1. **배치 검사(PLACE)**: kosac 요소가 §6.6 표의 부모 아래에만 있는지, kosac 속성이 var·varGrp에만 있는지 Go 코드로 확인한다.
   **참조 검사(REF)**: XSD 검증기에 맡기지 않고 Go 코드로 확인한다. libxml2는 IDREF가 가리키는 대상을 확인하지 않고, 루트 ID 검사도 버전마다 다르기 때문이다. 같은 문제를 XSD-DDI와 REF가 함께 보고할 수도 있다.
   - ID 형식(NCName)과 문서 안 유일성. 루트 codeBook@ID도 포함한다.
   - DDI 참조: var@files, location@fileid, varGrp@var, var@wgt-var
   - kosac 참조: idVar@ref, systemIdVar@ref, link@var·@fileRef, sampling@strataVars·@clusterVar, splitBallot@var, weight@var, rule@var, unverified@var·@file, parentVar
   - 끊긴 참조는 하나마다 R이다.
2. **DDI 투영**: 문서에서 `kosac:` 요소와 속성, `xmlns:kosac` 선언을 모두 걷어낸다. 이 투영을 `Validator(SchemaDDI, …)`에 넘긴다. 결과는 XSD-DDI로 낸다.
3. **kosac 투영**: `<kosac:projection>` 아래에 `<kosac:place path="…">`를 둔다. 그 안에 원래 kosac 요소를 그대로 복사하고, kosac 속성은 place 요소의 속성으로 옮긴다. 이 투영을 `Validator(SchemaKosac, …)`에 넘긴다. 결과는 XSD-KOSAC로 낸다.

- Validator 구현은 호출자 몫이다. datajam은 xmllint나 libxml2를 감싼다. 저장소 테스트는 `internal/xsdtest`를 쓴다. CLI는 PATH에 xmllint가 있으면 쓰고, 없으면 "XSD 검증 건너뜀"을 출력한다.
- XSD 파일은 embed로 공개한다: `xsd.FS`(DDI 2.5)와 `kosac/xsd.FS`(kosac 3.3).
  - 두 패키지 이름이 모두 `xsd`이므로 import할 때 별칭을 쓴다.
  - codebook.xsd는 xml.xsd, ddi-xhtml11.xsd(+XHTML 21개), dcterms.xsd를 상대 경로로 import한다. 그래서 Validator 구현은 FS 전체를 한 디렉터리에 풀어야 한다.

### 10.4 kosac 3.3 XSD 초안 (kosac/xsd)

- targetNamespace는 `urn:kosac:survey-meta:3.3`이고, elementFormDefault는 qualified이다.
- **전역 선언은 두 가지뿐이다.**
  - §6.6 표의 최상위 kosac 요소(package, study, sampling, …, codeType, stdCode, pages, sha256, bytes)
  - var·varGrp에 붙는 kosac 속성. `place`에서 검사하려면 전역이어야 한다.
- 하위 요소는 지역으로 선언한다. 예: package/file, sourceSurvey, validation, idVar, link, rule, map/entry, synonym, round, unverified.
  - `kosac:file`(fileDscr 자식)과 `package/file`처럼 이름은 같고 모양이 다른 요소가 있다. 둘 다 전역으로 선언하면 스키마가 컴파일되지 않는다(xmllint로 재현함).
  - 이 이름 겹침은 재단 쪽 XPath(`//kosac:file`)에도 영향을 주므로 CHANGES.md에 적는다.
- 참조 속성은 xs:IDREF가 아니라 `xs:NCName`과 `xs:NMTOKENS`로 선언한다. kosac 투영에는 대상 ID가 없어서, IDREF로 선언하면 XSD 1.0 검증기(Xerces 등)가 실패한다. 참조가 맞는지는 REF 검사가 본다.
- `projection`과 `place`도 선언한다. place는 다음 두 가지를 strict로 받는다. 따라서 kosac 이름은 하나하나 선언과 대조된다.
  - `xs:any namespace="##targetNamespace" processContents="strict"`
  - `xs:anyAttribute namespace="##targetNamespace" processContents="strict"`
- 값 형식:
  - 목록 값(role, measLevel, dimMeasure, codeType, mapping, action, waveType, level, Y/N)은 enumeration이다.
  - 개수와 쪽수는 정수이다.
  - sha256은 64자리 16진수이다.
- `CHANGES.md`(재단 승인용)에 요청서 v1.2와 다른 점을 적는다. 대상은 §6.4·§6.6·§13의 모든 차이와, 요소·속성 목록 전체이다.

## 11. 응답대상 문법과 평가기 (kosac/universe, D6)

```
expr    := and ("OR" and)*
and     := cond ("AND" cond)*
cond    := VAR ("=" | "!=") values | VAR "=" pattern
values  := item ("," item)*
item    := VALUE | VALUE "~" VALUE
pattern := "pattern:" REGEX          (RE2 문법)
```

- 키워드 AND와 OR는 대문자만 받고, 앞뒤에 공백이 있어야 한다. 괄호는 받지 않는다. 실물 30건에는 괄호식이 없다. 괄호가 오면 "괄호는 지원하지 않음" 오류를 낸다.
- VAR는 `[A-Za-z][A-Za-z0-9_]*`이다. `=`, `!=`, `,`, `~` 앞뒤의 공백은 허용하고 버린다.
- VALUE는 공백·`,`·`~`가 없는 토큰이다. 비교 규칙:
  - 양쪽이 모두 수이면 수로 비교한다.
  - 그 밖은 문자열로 비교한다. 예: `B100203`
  - 범위(`~`)는 수끼리만 쓸 수 있다.
- `pattern:`의 정규식은 조건의 끝까지이다. 끝은 문자열 끝이거나, 그다음에 처음 나오는 AND·OR 앞이다. AND·OR 앞뒤에는 공백 문자(탭·줄바꿈 포함)가 있어야 한다.
  - 정규식 앞뒤의 공백은 지운다. 엑셀 셀 끝의 공백 때문에 조건이 조용히 아무것도 맞히지 않는 일을 막기 위해서다. 의미 있는 공백은 `[ ]`로 쓴다.
  - 정규식 안에는 공백과 쉼표가 들어가도 된다.
  - 정규식 자체에 ` AND `나 ` OR `가 필요하면, 괄호식이 없는 지금 문법으로는 경계를 정할 수 없다. 그래서 "pattern 안의 AND/OR는 지원하지 않음" 오류를 낸다. 해당 글자를 `[ ]AND[ ]`처럼 문자 클래스로 쓰면 피할 수 있다.
- 클린징 condition은 `ParseCondition(s, target)`으로 받는다. 단독 `pattern:{정규식}`을 쓰면 대상 변수에 대한 조건이 된다.
- 평가기:
  - API는 `Expr.Eval(row Row) (bool, error)`이고, `type Row interface{ Value(name string) (string, bool) }`이다.
  - 변수가 없으면 오류이다.
  - 빈 값은 어떤 값과도 같지 않다. `!=`이면 참이다.
- `Expr.Vars()`는 참조 변수 목록이다. C10이 쓴다.

## 12. 네이티브 경로 (`FromWallaDDI(b []byte, opt WallaOptions) (*Codebook, []Issue, error)`)

- walla export(XML·JSON)를 `ddi.Parse`로 읽은 뒤 모델로 바꾼다. 바꾸면서 일부 정보는 버린다.
- 01 시트: TITLE은 stdyDscr titl에서 가져온다. 나머지는 비운다.
- Admin.WallaID는 `IDNo@agency="walla"` 값이다. 두 fixture에 모두 있다.
- Package는 nil로 둔다. 나머지 패키지 정보는 내보낼 때 datajam이 채운다.
- 대신 `Codebook.Native=true`로 표시한다. 조립은 Native이면 Package가 nil이어도 `kosac:package/kosac:sourceSurvey@kind="native"`를 쓴다. Package가 있으면 Source.Kind 자리에 native를 쓴다.
  - 이렇게 하면 datajam이 Package를 언제 채우든 Native 표시가 XML에 남는다. Parse는 kind=native를 보고 Native를 되살린다.
  - Native인데 Source.Kind가 native 아닌 다른 값이면 조립 오류로 낸다.
  - 버전이 없으므로 docDscr 제목("코드북 v")은 생기지 않는다(§6.4).
  - 이 규칙은 최종 리뷰에서 바꿨다. 이전 규칙에서는 Package 없이 저장한 네이티브 초안을 다시 읽으면 Native가 사라졌다. 그 뒤 판정을 다시 돌리면 KOFAC 실제 설문에서 가짜 grid가 4개(2024), 25개(2025) 생겼다.
- 02 시트:
  - 파일 행은 하나이다. FILE_NO=1, FILE_FORMAT=CSV, FILE_NAME은 F1의 fileName이다. walla의 F2(Parquet)는 같은 데이터를 다른 형식으로 담은 것이라 버린다.
  - VALID_N은 caseQnty에서 가져온다. 현재 fixture처럼 caseQnty가 없으면 비운다.
  - ID_VAR는 `walla_response_id`이다.
- 03 시트:
  - var마다 한 행을 만든다. VAR_NAME, VAR_LABEL(labl), Q_TEXT(qstnLit, 없으면 labl), NOTE(notes를 줄바꿈으로 이음)를 채운다.
  - PII와 SENSITIVE는 N이다.
  - VAR_GROUP과 Q_NO는 비운다. walla export의 varGrp는 grid와 multipleResp뿐이라 영역 정보가 없다.
    - datajam이 Record에 직접 채운다. 넣을 값은 walla 폼의 그룹(페이지) 라벨이다.
    - 채우지 않으면 C06 R이 난다.
  - MR_TYPE은 walla varGrp에서 정한다. multipleResp 멤버는 보기별 열, 순위 묶음은 순위별 열이다.
- MEAS_LEVEL은 다음 순서로 정한다.
  1. `opt.MeasHints[변수명]`. 값은 측정수준(명목·서열·연속·문자·날짜) 그대로다. walla 필드 타입을 측정수준으로 바꾸는 표는 walla 필드 타입을 아는 datajam이 가진다. 그러면 엔진이 walla 필드 타입 이름을 몰라도 된다.
  2. 힌트가 없으면: 코드가 있으면 명목, varFormat@category가 date이면 날짜, varFormat이 character이면 문자, 그 밖은 연속
- 결정 D8은 "힌트가 없으면 nominal"이었다. 위 규칙은 그것을 조금 다듬은 것이다. 문자 변수를 명목으로 두면 C07(코드 ≥ 2)이 오판하기 때문이다(빌더 재량, 사용자 확인 대상).
- 04 시트: catgry마다 한 행을 만든다.
- 시스템 변수 처리: `walla_` 이름 6개는 조사 변수로 가져오지 않는다. 조립 때 시스템 변수로 다시 만들어지기 때문이다.
  - **옛 export 모드**: 문서에 `response_id`라는 변수가 있고 `walla_response_id`가 없으면 옛 export(현재 testdata)로 본다. 판정은 문서마다 한 번 한다. 옛 모드에서는 다음처럼 처리한다.
    - 시스템 변수는 `response_id`, `submitted_at`, `customer_key` 세 개이다.
    - 순위 변수는 `_rank{k}`(밑줄 하나)로 끝나는 이름이다. 옛 export에는 순위 varGrp가 없다(예: Q30_rank1·2).
    - 기타 원문은 `_other`로 끝나는 이름이다.
    - 새 샘플이 오면 옛 모드 분기를 통째로 지운다. 새 이름 규칙에서는 밑줄 하나 뒤의 `rank1`이 보기 ID일 수 있으므로, 이 규칙을 새 모드에 섞으면 안 된다.
  - walla CSV의 시스템 열 순서는 재단 표 순서(§6.7)와 같다.
  - walla_started_at과 walla_duration_sec는 기록되지 않으면 빈칸이다. 이 값은 응답 데이터의 일이라 엔진과 관계가 없다.
- **walla 이름 규칙** (2026-10-28 개정, DDI 내보내기 세션의 반박 검토 반영. 앞선 규칙을 대체한다):
  - ID 조각은 `[A-Za-z0-9]`로만 되어 있다. 조각에는 `_`가 없다.
    - 예산은 필드 10자, 보기·행·커스텀 키 9자, 열 8자이다.
    - 예산 안의 영숫자 ID는 그대로 쓴다. 그 밖의 ID는 sha1 hex 앞부분을 예산 길이만큼 쓴다.
  - 조각 사이에는 `_` 하나를 쓰고, 고정 접미사 앞에는 `__` 둘을 쓴다.
    - `f_{F}`, `f_{F}_{보기}`, `f_{F}_{행}`, `f_{F}_{행}_{열}`
    - `f_{F}__other`, `f_{F}__custom`, `f_{F}__rank{k}`, `f_{F}__lat`, `__lon`, `__amount`, `__orderid`, `__status`
    - 숨은 필드: `h_{ID}`. 이 규칙은 바뀔 수 있다.
  - 이름은 최장 31자로, Stata 32자 제한 안에 든다.
  - **VAR_NAME은 walla 이름을 그대로 쓴다.** 재게시해도 이름이 변하지 않게 하려는 것이고, 이 이름은 이미 S04를 만족한다.
  - 묶음(MR·grid)은 이름으로 정하지 않고 walla varGrp 멤버십으로 정한다. walla 쪽 이름 규칙이 바뀌어도 영향을 받지 않게 하기 위해서다.
    - **묶음 키는 walla `varGrp@name`이다.** 새 샘플에서는 `f_{F}`, 옛 export에서는 `Q7`이다. 두 모드 모두 이 속성이 있다.
    - multipleResp 멤버: MR_TYPE=보기별 열, MRGroup=varGrp@name. 보기 ID가 `other`·`custom`인 보기 열(`f_{F}_other`)과 직접입력 플래그(`f_{F}__custom`)도 멤버이면 그대로 멤버이다.
    - grid 멤버: GridGroup=varGrp@name. 단, 순위 변수는 grid에 넣지 않는다(아래).
  - 고정 접미사는 마지막 `__`에서 자른다. ID 조각에는 `_`가 없다. 커스텀 속성 키에는 `_`가 들어갈 수 있지만 연속 `__`는 없다(`[A-Za-z0-9]+(_[A-Za-z0-9]+)*`). 그래서 이 경계는 하나로 정해진다. 밑줄 하나 뒤의 `other`·`custom`은 보기 ID이고 접미사가 아니다.
    - `__rank{k}`: MR_TYPE=순위별 열, RankNo=k, MRGroup은 `__` 앞부분이다. GridGroup은 비운다.
      - walla 새 export는 순위 묶음을 `type="other" otherType="rank"` varGrp로 낸다(샘플 VG4, d4589f0176).
      - 엔진은 varGrp 유형과 상관없이 이름 접미사로 판정한다. 그래서 순위 변수가 grid varGrp에 들어 와도 결과가 같다.
    - `__other`: ParentVar는 `__` 앞부분이다(그 이름의 변수가 있을 때만). 다중선택처럼 기본 변수가 없으면 ParentVar는 비운다.
      - textFormat은 walla가 담는 모양을 따른다.
      - `__` 앞부분과 같은 이름의 변수가 있으면 TEXT이다(단일선택 기타 원문).
      - 그 이름이 multipleResp varGrp@name이면 JSON_ARR이다(다중선택 기타 원문 배열).
      - 그 밖(grid, 순위, 커스텀 필드)은 JSON_OBJ이다.
      - 합성 샘플 `variables.tsv`의 분류와 맞는다: otherText 4, otherList 1, otherGrid 3, otherKeys 1.
    - `__custom`, `__lat`, `__lon`, `__amount`, `__orderid`, `__status`: ParentVar는 `__` 앞부분이다(그 변수가 있을 때만).
  - 새 모드 fixture는 `testdata/walla_synthetic/`(DDI 내보내기 세션이 만든 합성 샘플, 실데이터 없음)이다.
    - 변수 52개 중 시스템 6, 조사 46
    - 18개 필드 타입
    - multipleResp 1(멤버 4), grid 3, other/rank 1(멤버 2)
    - -6 결측 7건
    - fileDscr 2(csv, parquet). caseQnty는 없다
    - IDNo walla = `SampleForm0000000001`
  - 이 값들은 FromWallaDDI가 미리 채운다. DeriveDefaults는 빈 칸만 채우므로 이 값을 덮어쓰지 않는다(§8.1).
- **결측과 빈칸**:
  - walla는 지금 분기로 안 보인 문항과 건드리지 않은 문항을 모두 빈칸으로 낸다.
  - 빈칸을 -8·-7로 바꾸는 일은 응답 데이터를 바꾸는 일이므로 datajam F6(CSV 생성)의 몫이다. 엔진은 코드북에 결측 코드를 선언만 한다(DeriveDefaults, D10).
  - walla의 -6(이전 게시본 값) catgry는 그대로 가져온다. Code.Type은 MISSING으로 둔다(열린 질문 Q7).
- walla 이름 규칙과 결측 처리는 함수 하나(`wallaKind`)와 상수 몇 개에 모은다(D5). walla 규칙이 바뀌면 이곳만 고친다.
- 네이티브 초안은 비어 있는 필수 칸 때문에 C01·C06에 걸린다. 어떤 규칙을 네이티브에 적용할지는 datajam이 정한다.

## 13. 요청서와 다르게 정한 것 (재단 보고 목록, CHANGES.md의 원본)

| # | 요청서 | 엔진 | 이유 |
|---|---|---|---|
| 1 | abstract@contentType="summary" | "abstract" | XSD 목록은 abstract·purpose·mixed뿐이다 |
| 2 | varGrp@type="rank" | type="other" otherType="rank" | XSD 목록에 rank가 없다 |
| 3 | otherMat에 level 없음 | level="study" | XSD 필수 속성이다 |
| 4 | `<targetSampleSize>10000</…>` | `<targetSampleSize><sampleSize>10000</sampleSize></…>` | targetSampleSize는 요소만 담는다 |
| 5 | dataColl/respRate | anlyInfo/respRate | respRate의 부모는 anlyInfo이다 |
| 6 | fileTxt/notes | fileDscr/notes | fileTxt에는 notes가 없다 |
| 7 | 그리드가 qstn@ID 공유, Q_NO→qstn@ID | qstn에 ID 없음, Q_NO→qstn@seqNo | xs:ID는 문서에서 유일해야 하고 숫자로 시작할 수 없다 |
| 8 | 부록 D의 sumDscr: universe 다음 anlyUnit | anlyUnit 다음 universe | XSD 순서 |
| 9 | role=codebook 파일에 bytes·sha256 | name만 | 자기 해시는 자기 안에 넣을 수 없다 |
| 10 | kosac 요소를 DDI 순서 중간에 둠 | 같은 부모의 DDI 자식 뒤에 둠 | 규칙 하나. 공식 XSD는 어느 위치든 받지 않으므로 두 투영으로 검증한다(D1) |
| 11 | EXCLUDE_RULE·EDIT_METHOD를 cleanOps 하나에 | cleanOps=EDIT_METHOD, kosac:excludeRule 추가 | 거꾸로 읽을 수 있게 |
| 12 | 확인 불가 값을 원래 자리에 둠(부록 D cleanOps) | 텍스트 자리는 원문을 둔다. 형식 있는 자리는 비우고 kosac:unverified만 남긴다 | 정수·날짜 자리에 문장이 들어가면 XSD를 통과하지 못한다 |
| 13 | 시스템 변수 8개(device·channel 포함) | 6개 | walla 새 export 기준(D5). datajam F6가 device·channel을 null 열로 채울 수 있다 |
| 14 | kosac:popSource 위치: sumDscr 옆(4-2 표) | kosac:study 아래 | 연구 단위 확장 값을 한곳에 모은다 |
| 15 | kosac:validN 요소(4-2 표) | kosac:study@validN 속성 | 부록 D 모양을 따른다 |
| 16 | 원래 업로드 파일명 자리 없음 | kosac:file@sourceName. otherMat는 labl에 셀 원문을 둔다 | 패키지 이름으로 바뀐 뒤에도 Parse가 FILE_NAME·REPORT_FILE을 되살리도록 |
| 17 | unverified@key만 | @file(02 값), @var(03 값) 추가 | 파일·변수 단위 값을 되살리도록 |
| 18 | kosac:codeHistory | 이번 범위에서 뺌 | 원천인 게시본 이력이 export에 없다 |
| 19 | 같은 이름 kosac:file 두 가지(fileDscr 자식, package 자식) | 그대로 둔다. package/file은 지역 선언이다 | 4-2 표를 따른다. 재단의 `//kosac:file` XPath는 둘을 함께 잡으므로 부모로 구분해야 한다. **제안**: package 쪽을 `kosac:artifact`로 바꾸면 겹침이 없어진다. 4장 구조를 바꾸는 일이라 재단과 협의해 정하고, 엔진이 먼저 바꾸지는 않는다 |

- 12행 보충: DDI XSD에서 실패하는 형식 자리는 sampleSize 같은 정수 자리이다. collDate@date는 xs:string이라 DDI XSD를 통과한다. 그래도 날짜 자리를 비우는 것은 kosac 형식과 ETL 기준에 맞추기 위해서다.
- 11행 보충: cleanOps는 여러 번 나올 수 있어서 두 개를 두어도 XSD는 통과한다. 그래도 나눈 이유는 거꾸로 읽을 수 있게 하기 위해서다. XSD가 강제한 결정은 아니다.

## 14. API 요약 (datajam 흐름)

```go
// 업로드형
cb, issues, err := template.Read(xlsx, template.ReadOptions{})   // S05. S05 R이면 cb는 nil
if cb != nil {
    issues = append(issues, check.Check(cb, ref)...)             // C01~C18 (nil cb도 panic하지 않고 빈 결과)
}
// S05 R이면 여기서 멈춘다. 아래 단계는 cb가 nil이 아니라는 것을 전제로 한다
sug := kosac.DeriveDefaults(cb, ref)                             // 초안 + 제안
// … F5: cb.Admin, cb.File(n).SensLevel, v := cb.Var("1", "SQ4"); v.FK = &kosac.FK{…}; err = kosac.ApplyFK(cb, v, def) …
// … F11: cb.File(n).Cleansing = &kosac.Cleansing{…}
err = kosac.AssignIDs(cb, prev)                                  // prev는 kosac.Parse(이전 XML) 또는 nil
cb.Package = &kosac.Package{…}                                   // 해시는 datajam이 계산
xml, err := cb.XML()
issues = check.ValidateXML(xml, xmllintValidator)

// 네이티브형
cb, issues, err := kosac.FromWallaDDI(export, kosac.WallaOptions{MeasHints: hints})
draft, err := template.Write(cb)                                 // 담당자 열람용 v3.2

// 클린징·X07 등
e, err := universe.ParseCondition("SQ3!=1 AND SQ3_1!=-8", "SQ3_1")
ok, err := e.Eval(row)
```

## 15. CLI

- `ddi-xlsx kosac-check 양식.xlsx`: S05와 C01~C18을 돌려 Issue를 한 줄에 하나씩 출력한다. R이 있으면 종료 코드는 1이다. 용역사가 제출 전에 스스로 확인할 때 쓴다.
- `ddi-xlsx kosac-xml 양식.xlsx [-o out.xml]`: Read → DeriveDefaults → AssignIDs → XML 순서로 실행한다. 미리보기와 디버깅용이다. Package는 없다. xmllint가 있으면 ValidateXML도 돌린다.
- `ddi-xlsx kosac-template [-o 파일]`: 내장 빈 양식을 쓴다. 기존 template 명령처럼, 기본 경로에 파일이 이미 있으면 덮어쓰지 않는다.

## 16. 테스트 (done 조건)

fixture:
- `testdata/kosac/v32_sample_2024.xlsx`: 재단 작성 예시의 사본이다. 응답 데이터가 아니라 코드북이다.
  - 고객 문서이므로 비공개 저장소라는 전제로만 둔다. 이 전제는 커밋 전에 사용자 확인을 받는다.
  - 저장소를 공개하기로 하면 이 파일과, 이 파일에서 만든 골든(`v32_sample_2024.*`, `blank_v3.2.xlsx`)을 빼야 한다.
  - 이 사실을 `testdata/kosac/README.md`에 적는다.
- 기존 `testdata/kofac_*`

| 테스트 | 기대 |
|---|---|
| 작성 예시 읽기 | 변수 214개, 코드 828개, 파일 1개. 01 시트의 확인 불가 4건 |
| 작성 예시 검사 | 빌더 점검에서 이 설계 규칙을 손으로 돌려 본 예상값은 다음과 같다. C07 R 1건(SQ1), C01 W 4건, C08 W 145건, C12 W 4건(C8_1·C8_2·C9_1·C9_2), C10 W 1건(QC2 확인 불가), C18 0건(쪽 범위 `336-`), 그 밖의 규칙은 0건. Issue 전체를 골든 파일(`v32_sample_2024.issues.json`)로 고정하고, 처음 만들 때 사람이 검토한다 |
| 작성 예시 판정 | MR 묶음 3, 순위 묶음 4, grid 22(§8.1 목록과 같다), NoneOption은 QA19 Y, QA21 Y, QD1_1 N. RowLabel이 단어 중간에서 잘리지 않는다 |
| 양식 왕복 | Read → Write → Read에서 Record 값이 같다 |
| 임시 내장본 | 머리 행, 목록, `_meta` NOTE를 확인한다. 빈 양식을 Read하면 S05 R이 없다 |
| 조립 골든 | 작성 예시 → DeriveDefaults → AssignIDs → 고정 Package·Admin·Cleansing·FK fixture → XML 골든(`v32_sample_2024.xml`, `.json`) |
| XSD | 골든의 DDI 투영이 공식 XSD를, kosac 투영이 kosac XSD를 xmllint로 통과한다. 일부러 망가뜨린 문서는 각각 XSD-DDI, XSD-KOSAC, PLACE Issue를 낸다 |
| 고정점 | Parse → XML이 골든과 바이트 단위로 같다(XML, JSON) |
| AssignIDs | 변수 추가, 삭제, 순서 변경, 파일 2개 경우에 같은 이름은 같은 ID를 받고, 새 변수는 max+1을 받는다 |
| universe | 실물 RESP_UNIVERSE 30종이 모두 파싱된다(확인 불가 1건 제외). `!=`, pattern, 범위, 문자열 값, 오류(괄호, 빈 값, `~`에 문자열)를 확인하고, Eval 표 테스트를 한다 |
| C 규칙 | 규칙마다 통과 1건과 실패 1건 이상의 작은 표 테스트 |
| 네이티브(새 모드) | `walla_synthetic/survey.xml`과 `.json` → 모델 → XML이 두 투영과 REF 검사를 통과한다. 조사 변수 46, MR 1(멤버 4), grid 3, rank 1(멤버 2), -6 결측 7, ParentVar·접미사 판정이 §12와 같다 |
| 네이티브(새 모드, 실제 설문) | `kofac_b01/2024`, `kofac_b01/2025`(KOFAC 두 설문을 새 규칙으로 다시 낸 코드북, CSV 없음) → 조사 변수 286·634(옛 모드와 같은 수), 2024 rank 4·MR 3·grid 22, 2025 MR 14·grid 67, -6 결측 156·351, 두 투영과 REF 통과 |
| 네이티브(옛 모드) | `kofac_2024_public`, `kofac_2025_edu` → 모델 → XML이 통과한다. 조사 변수 286·634(시스템 3 제외), 순위는 `_rank{k}`로, -7 "현재 게시본에 없는 값"은 라벨 유지·MISSING(§8.1 결측 ②) |
| CLI | kosac-check 종료 코드, kosac-template이 덮어쓰지 않는 것 |
| 기존 | `go vet ./... && go test ./...`, npm 테스트가 그대로 녹색이다 |

## 17. 열린 질문 (기본값으로 구현하고, 재단 회의에서 확인)

| # | 질문 | 기본값 |
|---|---|---|
| Q1 | -9(유효하지 않음)의 codeType은 MISSING인가, NA인가? | MISSING |
| Q2 | -7의 규정 라벨은 "모르겠다"인가, "무응답"인가? 둘 다 받는가? | 규정 문구는 "모르겠다", 둘 다 받는다 |
| Q3 | DW의 DBMS는 무엇인가? (예약어 목록이 달라진다) | PostgreSQL reserved |
| Q4 | C15의 "양수 코드와 충돌 없음"은 무슨 뜻인가? | 검사하지 않는다 |
| Q5 | 재단 ETL은 혼합 문서를 공식 XSD 하나로 검증하는가? (D1 회의 안건) | 두 투영 |
| Q6 | TEMPLATE_ID의 정본은 KOFAC과 KOSAC 중 무엇인가? | D2 |
| Q7 | -6(이전 게시본 값)의 codeType은 무엇인가? 목록(VALID·MISSING·DK·NA)에 맞는 값이 없다 | MISSING |
| Q8 | CLUSTER_VAR에 변수를 여러 개 적을 수 있는가? 다단계 집락 추출이면 여럿일 수 있다. 지금은 한 이름만 받고(kosac:sampling@clusterVar는 NCName), 여럿이면 C04 R이다 | 한 이름 |

## 18. 범위 밖

- wasm·npm 노출(§3)
- K01~K07, X 검사, S01~S04, S06, S07
- 버전 간 담당자 판정 이월: datajam이 prev 모델에서 (FILE_NO, VAR_NAME)으로 복사하면 된다.
- v3.3 양식(부록 E의 추가 열)
- walla 쪽 export 변경

## 19. 공수

| 작업 | MD |
|---|---|
| 모델, 조립, Parse, 고정점 | 3–4 |
| 양식 읽기·쓰기, 임시 내장본 | 2 |
| C01~C18, universe | 2–3 |
| kosac XSD, 두 투영, 배치 검사, CHANGES.md | 1.5–2 |
| DeriveDefaults, ApplyFK, AssignIDs | 1–1.5 |
| 네이티브 | 1 |
| CLI, README, 리뷰 | 1–1.5 |
| **합계** | **11.5–15** |

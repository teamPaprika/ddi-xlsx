# ddi-xlsx

`ddi-xlsx`는 DDI Codebook 2.5 문서(XML 또는 JSON)를 편집할 수 있는 xlsx 워크북으로 바꾸고, 고친 워크북을 다시 DDI로 되돌리는 도구입니다.

**문서: <https://teampaprika.github.io/ddi-xlsx/>**

연구자와 아카이브 담당자가 코드북을 스프레드시트에서 읽고 고칠 수 있게 하는 것이 목적입니다. 변수 라벨, 질문 문구, 값 라벨, 결측 표시, 노트, 그룹 구성, 연구 메타데이터는 셀만 고쳐서 바꿀 수 있습니다. 열에 들어가지 않는 내용은 같은 행의 `xml` 셀에 XML 원문으로 보관합니다. 그래서 DDI → xlsx → DDI 왕복에서 내용을 잃지 않습니다(예외는 [왕복 보장과 손실](https://teampaprika.github.io/ddi-xlsx/docs/roundtrip/) 참고).

- 명령줄 도구(CLI): macOS, Linux, Windows용 실행 파일입니다.
- Go 라이브러리: `workbook.ToXLSX`, `workbook.FromXLSX`, `workbook.Template`입니다.
- npm 패키지: Node와 웹 브라우저에서 쓰는 WebAssembly 모듈입니다. npm 레지스트리에는 아직 게시하지 않았습니다.

## 기능과 현재 지원 범위

| 기능 | 입력과 출력 | 현재 범위 |
|---|---|---|
| 범용 변환기 | DDI 2.5 XML·JSON ↔ 편집용 Excel | `Variables`, `Categories`, `Groups`, `Document` 시트를 사용합니다. |
| 재단 양식 엔진 | 재단 v3.2 Excel ↔ 재단 모델·XML | 양식 읽기·쓰기와 코드북 규칙 검사를 제공합니다. 현재 XML은 `kosac:` 확장을 포함합니다. |
| Node·브라우저 SDK | DDI XML·JSON ↔ 편집용 Excel | 범용 변환과 빈 양식 생성 함수를 WASM으로 제공합니다. |

**재단 엔진의 현재 XML은 공식 DDI 2.5 XSD를 단독으로 통과하지 않습니다.** 자세한 내용은 [재단 코드북 엔진](https://teampaprika.github.io/ddi-xlsx/docs/kosac/)에 있습니다.

이 도구는 코드북을 변환하고 검사합니다. 응답 데이터 정제, Parquet 생성, 승인, 데이터 플랫폼 적재는 호출하는 시스템에서 처리합니다.

## 빠른 시작

Go 1.25 이상과 Git이 필요합니다. 다음은 macOS·Linux에서 소스로 CLI를 빌드하는 예시입니다.

```bash
git clone https://github.com/teamPaprika/ddi-xlsx.git
cd ddi-xlsx
go build -o dist/ddi-xlsx ./cmd/ddi-xlsx
./dist/ddi-xlsx template -o sample-codebook.xlsx
./dist/ddi-xlsx from-xlsx sample-codebook.xlsx --xml sample-codebook.xml --json sample-codebook.json
```

`-o`, `--xml`, `--json`으로 지정한 기존 파일은 덮어씁니다.

## 문서

| 페이지 | 내용 |
|---|---|
| [설치](https://teampaprika.github.io/ddi-xlsx/docs/install/) | CLI 실행 파일, Go 모듈, 로컬 npm 패키지 |
| [CLI 사용법](https://teampaprika.github.io/ddi-xlsx/docs/cli/) | 변환 명령, 출력 경로 규칙, 종료 코드 |
| [Go 라이브러리](https://teampaprika.github.io/ddi-xlsx/docs/go/) | `workbook` 패키지 |
| [Node·브라우저 SDK](https://teampaprika.github.io/ddi-xlsx/docs/npm/) | WASM 함수, 브라우저 사용, 큰 코드북 |
| [받는 입력](https://teampaprika.github.io/ddi-xlsx/docs/input/) | 입력 조건, 원문 보관, JSON 형식 |
| [빈 양식으로 시작하기](https://teampaprika.github.io/ddi-xlsx/docs/blank-template/) | 빈 양식, ID 자동 부여, 이름으로 연결 |
| [워크북 구조](https://teampaprika.github.io/ddi-xlsx/docs/workbook/) | 시트와 열, 자주 하는 편집 |
| [왕복 보장과 손실](https://teampaprika.github.io/ddi-xlsx/docs/roundtrip/) | 보장 범위, 알려진 손실, 확인한 결과 |
| [재단 코드북 엔진](https://teampaprika.github.io/ddi-xlsx/docs/kosac/) | KOSAC CLI, Go API, XML 검사 |
| [기여하기](https://teampaprika.github.io/ddi-xlsx/docs/contributing/) | 개발 환경, 검사 명령, 문서 규칙 |
| [라이선스](https://teampaprika.github.io/ddi-xlsx/docs/license/) | 제3자 자료의 저작권과 라이선스 |

문서 원본은 [`site/content/docs/`](site/content/docs/)에 있습니다. 기능을 바꾸면 같은 브랜치에서 문서도 고칩니다.

## 라이선스

소스코드 저장소는 공개 상태입니다. 이 도구 자체의 라이선스는 아직 정하지 않았습니다. CLI 바이너리는 LGPL-3.0-or-later 스키마를 내장합니다. 그래서 도구의 라이선스는 LGPL-3.0 4절의 조건과 맞아야 합니다.

저장소와 배포물에 들어 있는 제3자 자료는 아래와 같습니다. 조사 기준일은 2026-11-03입니다. 저작권 줄, 출처, 변경 사항, 의무는 [라이선스](https://teampaprika.github.io/ddi-xlsx/docs/license/) 페이지에 있습니다.

| 구성 요소 | 파일 | 라이선스(SPDX) | 배포 |
|---|---|---|---|
| DDI Codebook 2.5 XSD | `xsd/ddi-codebook-2.5/codebook.xsd` | `LGPL-3.0-or-later`, Copyright (c) 2014 DDI Alliance | git, CLI 내장 |
| DDI XHTML 스키마 | `xsd/ddi-codebook-2.5/ddi-xhtml11*.xsd` | `LGPL-3.0-or-later`(Copyright (c) 2009 DDI Alliance) + W3C XHTML 스키마 고지 | git, CLI 내장 |
| W3C XHTML 모듈 스키마 | `xsd/ddi-codebook-2.5/XHTML/*.xsd` 18개 | W3C XHTML 스키마 고지(SPDX 없음), W3C 1998-2002 | git, CLI 내장 |
| XHTML 문자 엔터티 | `xsd/ddi-codebook-2.5/XHTML/*.ent` 3개 | `ISO-permission`, W3C 부분은 `W3C-20150513`로 취급 | git, CLI 내장 |
| W3C XML 네임스페이스 스키마 | `xsd/ddi-codebook-2.5/xml.xsd` | 표시 없음, `W3C-20150513`로 취급 | git, CLI 내장 |
| Dublin Core 스키마 | `xsd/ddi-codebook-2.5/dc.xsd`, `dcterms.xsd`, `dcmitype.xsd` | `CC-BY-4.0`, Copyright © 2011 DCMI. `dc.xsd`와 `dcterms.xsd`는 `xml.xsd` 경로 한 줄을 바꾸었습니다. | git, CLI 내장 |
| 생성한 규칙 표 | `internal/schema/table.go` | `codebook.xsd`의 2차적 저작물인지 정하지 않음 | git, CLI, wasm |
| 재단 양식 자산 | `kosac/xsd/kosac-3.3.xsd`, `kosac/xsd/CHANGES.md`, `kosac/template/blank_v3.2.xlsx` | 표시 없음 | git, CLI 내장(`.xsd`, `.xlsx`) |
| excelize | `github.com/xuri/excelize/v2` v2.11.0 | `BSD-3-Clause` | CLI, wasm |
| efp, nfp | `github.com/xuri/efp` v0.0.1, `github.com/xuri/nfp` | `BSD-3-Clause` | CLI, wasm |
| mscfb, msoleps | `github.com/richardlehane/mscfb` v1.0.7, `msoleps` v1.0.6 | `Apache-2.0` | CLI, wasm |
| go-deepcopy | `github.com/tiendc/go-deepcopy` v1.7.2 | `MIT` | CLI, wasm |
| Go 확장 모듈 | `golang.org/x/crypto`, `x/net`, `x/text` | `BSD-3-Clause` + `PATENTS`. `x/text`의 표는 `Unicode-3.0` 데이터에서 생성 | CLI, wasm |
| Go 표준 라이브러리 | Go 툴체인 | `BSD-3-Clause` + `PATENTS`. 일부 파일은 Inferno `MIT`, `SunPro`, `Unicode-3.0` | CLI, wasm |
| Go wasm 로더 | `npm/wasm_exec.js` | `BSD-3-Clause`, Copyright 2018 The Go Authors | npm |
| 테스트 데이터(재단) | `testdata/kofac_*`, `testdata/kosac/` | 표시 없음 | git |
| 테스트 데이터(합성) | `testdata/walla_*`, `testdata/archive/` | 이 도구의 라이선스를 따름 | git |
| 문서 사이트 | Hextra, Tailwind CSS, modern-normalize, Heroicons, Chroma 스타일 | `MIT` | 문서 사이트 |
| 문서 사이트 검색 | FlexSearch v0.8.143 | `Apache-2.0` | 문서 사이트 |

- "CLI 내장"은 `go:embed`로 CLI 바이너리에 파일을 그대로 넣는다는 뜻입니다. wasm(npm 패키지)에는 XSD와 재단 양식이 들어가지 않습니다.
- `go.sum`에만 있는 testify(`MIT`), go-spew(`ISC`), go-difflib(`BSD-3-Clause`), `golang.org/x/image`(`BSD-3-Clause`), yaml.v3(`MIT AND Apache-2.0`)은 링크되지 않습니다.
- CLI가 실행하는 `xmllint`(libxml2, `MIT`)는 함께 배포하지 않습니다.
- CLI 바이너리와 npm 패키지는 아직 배포하지 않았습니다.

확인이 필요한 항목은 아래와 같습니다.

1. LGPL-3.0과 GPL-3.0 전문이 저장소에 없습니다.
2. W3C 고지 파일 `xhtml-copyright-1.xsd`가 저장소에 없습니다. XHTML 모듈 스키마 18개가 이 파일을 가리킵니다.
3. `xml.xsd`에는 라이선스 표시가 없습니다.
4. `internal/schema/table.go`가 LGPL 스키마의 2차적 저작물인지 정하지 않았습니다.
5. 재단 자료의 이용 조건이 저장소에 적혀 있지 않습니다.
6. CLI 바이너리나 npm 패키지를 배포하기 전에, 고지와 라이선스 전문을 배포물에 넣어야 합니다.
7. 이 도구의 라이선스를 정해야 합니다.

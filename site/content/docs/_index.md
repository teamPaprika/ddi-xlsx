---
title: 개요
weight: 1
---

`ddi-xlsx`는 DDI Codebook 2.5 문서(XML 또는 JSON)를 편집용 xlsx 워크북으로 바꿉니다. 고친 워크북은 다시 DDI로 바꿉니다.

이 도구의 목적은 연구자와 아카이브 담당자가 코드북을 스프레드시트에서 읽고 고치는 것입니다. 변수 라벨, 질문 문구, 값 라벨, 결측 표시, 노트, 그룹, 연구 메타데이터는 셀을 고쳐서 바꿉니다.

열에 들어가지 않는 내용은 같은 행의 `xml` 셀에 XML 원문으로 남습니다. 그래서 DDI → xlsx → DDI 왕복에서 내용을 잃지 않습니다. 예외는 [왕복 보장과 손실](roundtrip.md)에 있습니다.

## 제공 형태

| 형태 | 설명 | 문서 |
|---|---|---|
| CLI | macOS, Linux, Windows용 실행 파일입니다. | [CLI 사용법](cli.md) |
| Go 라이브러리 | `workbook.ToXLSX`, `workbook.FromXLSX`, `workbook.Template`입니다. | [Go 라이브러리](go.md) |
| npm 패키지 | Node와 브라우저에서 쓰는 WebAssembly 모듈입니다. | [Node·브라우저 SDK](npm.md) |

npm 레지스트리에는 아직 게시하지 않았습니다.

## 기능과 지원 범위

| 기능 | 입력과 출력 | 현재 범위 |
|---|---|---|
| 범용 변환기 | DDI 2.5 XML·JSON ↔ 편집용 Excel | `Variables`, `Categories`, `Groups`, `Document` 시트를 씁니다. |
| 재단 양식 엔진 | 재단 v3.2·v4.0 Excel ↔ 재단 모델 → DDI 2.5 XML·JSON | 양식 검사와 표준 요소 매핑을 제공합니다. |
| Node·브라우저 SDK | DDI XML·JSON ↔ 편집용 Excel | 범용 변환과 빈 양식 생성을 WASM으로 제공합니다. |

{{< callout type="warning" >}}
재단 출력은 DDI 2.5 표준 요소와 속성을 사용합니다. 전체 XSD 검증에는 검증기가 필요합니다. 새 출력은 `ddi.Parse`로 읽으세요. 원본 재단 양식을 다시 편집하려면 XLSX와 모델을 보관하세요. 자세한 내용은 [재단 코드북 엔진](kosac/_index.md)에 있습니다.
{{< /callout >}}

## 이 도구가 하지 않는 일

이 도구는 코드북을 변환하고 검사합니다. 아래 작업은 호출하는 시스템이 합니다.

- 응답 데이터 정제
- Parquet 생성
- 승인
- 데이터 플랫폼 적재

## 개발 중인 기능

아래 기능은 로컬 개발 작업에 있습니다. 현재 공개 버전에는 없습니다.

- `checkCodebook`, `kosacTemplateXlsx` WASM API
- 단일 HTML로 실행하는 브라우저 검사 화면과 JavaScript REPL

구현과 테스트를 반영한 뒤에 사용 방법을 추가합니다.

## 문서 목록

{{< cards >}}
  {{< card link="getting-started" title="빠른 시작" icon="play" >}}
  {{< card link="install" title="설치" icon="download" >}}
  {{< card link="cli" title="CLI 사용법" icon="terminal" >}}
  {{< card link="go" title="Go 라이브러리" icon="code" >}}
  {{< card link="npm" title="Node·브라우저 SDK" icon="globe-alt" >}}
  {{< card link="input" title="받는 입력" icon="document-text" >}}
  {{< card link="blank-template" title="빈 양식으로 시작하기" icon="document-add" >}}
  {{< card link="workbook" title="워크북 구조" icon="table" >}}
  {{< card link="roundtrip" title="왕복 보장과 손실" icon="refresh" >}}
  {{< card link="kosac" title="재단 코드북 엔진" icon="office-building" >}}
  {{< card link="contributing" title="기여하기" icon="user-group" >}}
  {{< card link="license" title="라이선스" icon="scale" >}}
{{< /cards >}}

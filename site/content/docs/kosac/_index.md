---
title: 재단 코드북 엔진
weight: 11
---

현재 공개 버전은 재단 코드북 양식 KOSAC-SURVEY-META v3.2(xlsx)를 다룹니다. 재단 양식은 [범용 워크북](../workbook.md)과 다른 형식입니다.

{{< callout type="warning" >}}
**현재 구현:** 출력 XML에는 확장 네임스페이스 `urn:kosac:survey-meta:3.3`이 있습니다. 양식 버전 3.2와 XML 확장 버전 3.3은 다른 번호입니다. 현재 출력은 재단의 최종 제출 XML로 확정되지 않았습니다.

**후속 작업:** 재단 양식의 항목을 DDI 2.5 표준 요소에 매핑하고, 최종 XML을 공식 XSD로 직접 검사해야 합니다. `kosac:` 확장은 이 작업의 완료 기준이 아닙니다.
{{< /callout >}}

## 엔진이 하는 일

- 용역사가 채운 v3.2 양식을 읽고 검사합니다. 양식 구조 검사는 S05이고, 내용 규칙은 C01~C18입니다.
- 분석 메타의 초안을 채웁니다. 분석 메타는 역할, 차원·측정 구분, 집계 규칙, 다중응답·grid 묶음입니다.
- 결측 코드의 초안을 채웁니다.
- 변수에 ID를 붙이고 코드북 XML을 조립합니다. `kosac.Parse`는 조립한 XML을 다시 모델로 읽습니다.
- Walla 내보내기(DDI)를 모델로 바꿉니다. 이 경로를 네이티브 경로라고 합니다.
- XML을 두 투영으로 나누어 XSD로 검사할 수 있게 합니다.
- 모델을 v3.2 양식 xlsx로 씁니다.

엔진 패키지는 파일, 네트워크, 외부 프로세스를 쓰지 않습니다. 바이트를 받아 모델과 바이트를 돌려줍니다. `xmllint`를 실행하는 곳은 CLI뿐입니다.

## 패키지

| 경로 | 내용 |
|---|---|
| `kosac/` | 모델, XML 조립과 읽기, 자동 판정, ID 부여, Walla 변환 |
| `kosac/template/` | v3.2 양식 읽기·쓰기와 내장 빈 양식 |
| `kosac/check/` | C01~C18과 XML 검사(PLACE, REF, 두 투영) |
| `kosac/universe/` | 응답대상(RESP_UNIVERSE)과 클린징 조건의 파서·평가기 |
| `kosac/xsd/` | kosac 3.3 XSD 초안과 재단 승인용 변경 기록(`CHANGES.md`) |

## 이 절의 페이지

{{< cards >}}
  {{< card link="cli" title="KOSAC CLI" icon="terminal" >}}
  {{< card link="go-api" title="재단 Go API" icon="code" >}}
  {{< card link="validation" title="XML 검사와 양식 파일" icon="shield-check" >}}
{{< /cards >}}

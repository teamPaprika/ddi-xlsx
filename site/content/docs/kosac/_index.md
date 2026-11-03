---
title: 재단 코드북 엔진
weight: 11
---

재단 KOSAC-SURVEY-META v3.2·v4.0 양식을 읽고 모델로 보관합니다.
재단 양식은 앞의 범용 DDI 편집 워크북과 다른 형식입니다.
`Codebook.Tree`, `XML`, `JSON`은 DDI Codebook 2.5 표준 요소만 출력합니다.
필드별 대상과 해석 조건은 [108개 항목 매핑표](https://github.com/teamPaprika/ddi-xlsx/blob/main/docs/kosac-ddi25-mapping.md)에 있습니다.

| 경로 | 역할 |
|---|---|
| `kosac/` | 재단 모델, 표준 XML·JSON 생성, 자동 초안, ID 부여, Walla 입력 변환 |
| `kosac/template/` | 버전별 양식 읽기·쓰기, 공식 v4.0 빈 양식 |
| `kosac/check/` | 양식 내용 규칙과 전체 XML 검증 |
| `kosac/universe/` | 기계식 응답대상·정제 조건의 파서와 평가기 |
| `kosac/xsd/` | 과거 확장 형식의 XSD와 설계 기록. 새 출력 검증에는 사용하지 않음 |

엔진은 바이트와 모델만 처리합니다. 파일 입출력과 `xmllint` 실행은 CLI가 맡습니다.
응답 정제 실행, 코드표 조회, 데이터 적재는 호출 시스템의 책임입니다.


전체 필드의 대상은 [108개 항목 매핑표](https://github.com/teamPaprika/ddi-xlsx/blob/main/docs/kosac-ddi25-mapping.md)에 있습니다.

## 이 절의 페이지

- [KOSAC CLI](cli.md)
- [재단 Go API](go-api.md)
- [XML 검사와 양식 파일](validation.md)

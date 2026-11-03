---
title: XML 검사와 양식 파일
weight: 3
---

`check.ValidateXML`은 먼저 ID 형식·중복과 지원하는 참조의 대상을 확인합니다.
`kosac:` 확장이 있으면 `DDI-EXTENSION`으로 반려합니다.
검증기가 있으면 **입력 XML 전체를 그대로** 공식 DDI 2.5 XSD로 한 번 검사합니다.
확장을 제거한 투영 문서의 통과를 표준 문서의 통과로 보고하지 않습니다.

`validator == nil`이면 XSD 검증을 실행하지 않습니다.
이 결과를 표준 적합성 확인으로 사용하지 마세요.
XSD는 `xsd.FS` 전체를 같은 디렉터리에 풀어 상대 경로 import가 작동하게 해야 합니다.
구현은 `cmd/ddi-xlsx/xmllint.go`를 참고하세요.

## 양식과 예제

`template.Blank()`와 `kosac-template`은 재단의 2026-10-19 개정 공식 v4.0 빈 양식을 반환합니다.
`template.Write`는 모델의 명시된 버전으로 씁니다. 버전이 없으면 v4.0입니다.
v4.0 항목을 v3.2로 쓰려 하면 유실을 막기 위해 거부합니다.

원본 Excel의 목록 검증 범위에는 빈 구간이 있을 수 있습니다.
Excel 입력 목록만으로 값의 타당성을 판단하지 마세요.
제공 예제, 표준 출력 기준 파일, 과거 형식 fixture의 출처는 [예제 안내](https://github.com/teamPaprika/ddi-xlsx/blob/main/testdata/kosac/README.md)에 있습니다.

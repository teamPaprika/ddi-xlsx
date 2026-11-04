---
title: 브라우저에서 실행
weight: 3
width: wide
toc: false
---

이 페이지에서 ddi-xlsx를 설치 없이 실행합니다. 이 페이지는 [Node·브라우저 SDK](npm.md)와 Go 엔진(WASM)을 Web Worker에서 실행합니다. 검사와 변환은 파일을 브라우저 안에서만 처리합니다.

{{< playground >}}

## 입력별 작업

| 입력 | 작업 | 같은 CLI 명령 |
|---|---|---|
| DDI XML·JSON | 검사, 편집용 Excel로 변환 | `to-xlsx` |
| 편집용 Excel | 검사, DDI XML·JSON으로 변환 | `from-xlsx` |
| 재단 양식 Excel(v3.2, v4.0) | 검사, DDI XML·JSON으로 변환 | `kosac-check`, `kosac-xml` |
| 없음 | 편집용 빈 양식 받기, 재단 v4.0 빈 양식 받기 | `template`, `kosac-template` |

- 입력 종류에 맞는 작업만 켜집니다.
- 결과 파일의 "입력으로 쓰기"를 누르면, 그 파일이 새 입력이 됩니다. 그래서 DDI XML → 편집용 Excel → DDI XML 왕복을 이 페이지에서 확인합니다.
- 결과 아래의 "같은 작업을 코드로"는 같은 작업의 CLI, JavaScript, Go 코드를 보여 줍니다.
- "코드 실행" 탭은 JavaScript 코드로 [SDK 함수](npm.md#함수)를 호출합니다.

## 검사 범위

| 검사 | 실행하는 곳 |
|---|---|
| 문서 읽기, ID·참조(READ, REF) | 이 페이지 |
| 재단 양식 구조(S05), 내용 규칙(C) | 이 페이지 |
| DDI 2.5 XSD | CLI와 `xmllint`. 방법은 [CLI 사용법](cli.md#출력-xml-검사)에 있습니다. |
| 공통코드·기관 기준정보 대조 | Go API(`check.Check`에 `kosac.RefData`를 줍니다) |
| 응답 CSV 교차 검사, 재코딩 실행 | 응답 데이터가 있는 시스템 |

결과에서 "실행 안 함" 단계는 위 표의 다른 곳에서 검사하는 단계입니다.

## 참고

- 이 페이지를 처음 열 때, 브라우저가 엔진 파일(약 13 MB)을 받습니다. 그다음에는 브라우저 캐시를 씁니다.
- 예시 입력은 모두 합성 자료나 빈 양식입니다.
- 재단 양식을 DDI XML·JSON으로 바꾸려면, 내용을 채운 양식을 입력으로 넣으세요. 빈 양식을 넣으면, 필수 항목 `TITLE`이 비어 있다는 오류가 나옵니다.
- "코드 실행" 탭의 코드는 별도의 Web Worker에서 실행합니다. 이 코드는 브라우저 JavaScript이므로, 네트워크 요청도 보냅니다.
- [브라우저 코드북 검사실](browser-check.md)은 검사, 범용 변환, 코드 실행을 HTML 파일 하나로 만듭니다.

{{< callout type="warning" >}}
다른 사람이 준 코드를 "코드 실행" 탭에서 실행하기 전에, 코드의 내용을 확인하세요.
{{< /callout >}}

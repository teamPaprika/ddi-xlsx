---
title: Go 라이브러리
weight: 5
---

범용 변환기는 `workbook` 패키지에 있습니다. 함수는 파일을 읽거나 쓰지 않습니다. 바이트를 받아 바이트를 돌려줍니다.

```go
import "github.com/teamPaprika/ddi-xlsx/workbook"

// src는 DDI 2.5 문서(XML 또는 JSON)의 바이트입니다.
xlsx, warnings, err := workbook.ToXLSX(src)

// 워크북을 XML과 JSON으로 되돌립니다. 둘은 같은 트리를 담습니다.
xmlOut, jsonOut, warnings, err := workbook.FromXLSX(xlsx)

// 빈 양식 워크북을 만듭니다.
blank, err := workbook.Template()
```

## 함수

| 함수 | 입력 | 출력 |
|---|---|---|
| `ToXLSX` | DDI 2.5 XML 또는 JSON 바이트 | xlsx 바이트, 경고 목록 |
| `FromXLSX` | xlsx 바이트 | XML 바이트, JSON 바이트, 경고 목록 |
| `Template` | 없음 | 빈 양식 xlsx 바이트 |

입력 형식은 내용으로 판별합니다. 입력 조건은 [받는 입력](input.md)에 있습니다. 빈 양식의 내용은 [빈 양식으로 시작하기](blank-template.md)에 있습니다.

## 경고와 오류

- `warnings`는 경고 문구의 목록입니다. 경고가 있어도 변환은 성공한 것입니다.
- `FromXLSX`가 셀 내용 때문에 실패하면, 오류 문구는 셀 주소로 시작합니다. 예: `Categories!A12: …`
- 변환이 성공해도 출력의 XSD 유효성은 보장하지 않습니다. 고친 워크북의 출력은 따로 검사하세요. 방법은 [CLI 사용법](cli.md)의 "출력 XML 검사"에 있습니다.

## 재단 코드북 엔진

재단 양식을 다루는 패키지는 `kosac`, `kosac/check`, `kosac/template`, `kosac/universe`입니다. 사용 방법은 [재단 Go API](kosac/go-api.md)에 있습니다.

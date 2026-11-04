---
title: Node·브라우저 SDK
weight: 7
---

npm 패키지는 같은 변환기를 WebAssembly로 실행합니다. 설치 방법은 [설치](install.md)에 있습니다.

## 함수

```ts
ddiToXlsx(input: string | Uint8Array): Promise<{ xlsx: Uint8Array; warnings: string[] }>
xlsxToDdi(xlsx: Uint8Array): Promise<{ xml: string; json: string; warnings: string[] }>
templateXlsx(): Promise<Uint8Array>
checkCodebook(input: string | Uint8Array, options?: { format?: 'ddi' | 'ddi-xlsx' | 'kosac-xlsx' }): Promise<CheckReport>
kosacTemplateXlsx(): Promise<Uint8Array>
kosacToDdi(xlsx: Uint8Array): Promise<{ xml: string | null; json: string | null; issues: CheckIssue[] }>
```

[브라우저에서 실행](playground.md) 페이지에서 이 함수를 설치 없이 실행합니다.

- `ddiToXlsx`는 DDI XML이나 JSON을 워크북으로 바꿉니다.
- `xlsxToDdi`는 워크북을 XML과 JSON으로 바꿉니다. 둘은 같은 트리를 담습니다.
- `templateXlsx`는 빈 양식 워크북의 바이트를 돌려줍니다. 양식의 내용은 [빈 양식으로 시작하기](blank-template.md)에 있습니다.
- `kosacTemplateXlsx`는 재단 v4.0 빈 양식을 반환합니다. 일반 DDI 편집용 양식과 구분합니다.
- `kosacToDdi`는 재단 v3.2·v4.0 양식을 표준 DDI XML과 JSON으로 바꿉니다. 변환 순서는 CLI의 `kosac-xml`과 같습니다. XSD 검사는 CLI에서 합니다.
  - 양식 구조 검사(S05)가 반려하면, `xml`과 `json`은 `null`입니다. `issues`에 반려 이유가 있습니다.
  - 변환하면, `issues`에 S05 경고와 ID·참조 검사 결과가 있습니다. 내용 규칙(C)은 `checkCodebook`으로 검사하세요.
  - 코드북을 조립할 수 없으면(예: 필수 항목 `TITLE`이 비어 있으면), `Error`로 거부됩니다.

## 코드북 검사

`checkCodebook`은 현재 Go 엔진의 검사를 실행합니다. `format`의 기본값은 `ddi`입니다.
XML·JSON은 문자열이나 `Uint8Array`로, Excel은 `Uint8Array`로 전달하세요.

| 형식 | 실행하는 검사 |
|---|---|
| `ddi` | XML·JSON 읽기, 지원하는 ID·변수·파일 참조 검사, `kosac:` 확장 반려 |
| `ddi-xlsx` | 편집용 워크북을 DDI로 변환한 뒤 위 검사 실행 |
| `kosac-xlsx` | 재단 v3.2·v4.0 양식 읽기(S05)와 코드북 규칙(C) 검사 |

```js
import { checkCodebook } from 'ddi-xlsx';

const report = await checkCodebook(input, { format: 'kosac-xlsx' });
console.log(report.issues, report.steps);
```

결과에는 `format`, `templateVersion`, `rejections`, `warnings`, `issues`, `steps`가 있습니다.
`templateVersion`은 양식을 읽은 경우에만 있습니다.
각 `issues` 항목은 규칙(`Rule`), 등급(`Level`), 위치, 실제값, 기대값, 메시지를 담습니다.
`steps`의 상태는 `pass`, `fail`, `warning`, `not_run`입니다.
양식 구조가 반려되면 C 검사는 `not_run`입니다.

WASM이 실행하는 단계는 READ, REF, S05, C입니다.
XSD, REFERENCE, RESPONSE, RECODING 단계는 `not_run`으로 표시합니다. 각 단계의 `detail`에 그 단계를 실행하는 곳이 있습니다.
`kosac:` 확장은 REF 단계가 `DDI-EXTENSION`으로 반려합니다. 자세한 내용은 [XML 검사](kosac/validation.md)에 있습니다.

문서 오류는 `issues`로 반환합니다.
지원하지 않는 `format`이나 입력 자료형, WASM 실행 실패는 예외로 거부합니다.
브라우저에서 파일 검사와 JavaScript 코드를 실행하려면 [코드북 검사실](browser-check.md)을 사용하세요.

## Node 예시

```js
import { readFile, writeFile } from 'node:fs/promises';
import { ddiToXlsx, xlsxToDdi } from 'ddi-xlsx';

const { xlsx, warnings } = await ddiToXlsx(await readFile('codebook.xml'));
await writeFile('codebook-npm.xlsx', xlsx);

const out = await xlsxToDdi(await readFile('codebook-npm.xlsx'));
await writeFile('codebook-npm-edited.xml', out.xml);
await writeFile('codebook-npm-edited.json', out.json);
```

`writeFile`은 같은 이름의 파일을 덮어씁니다.

## 오류

| 원인 | 결과 | 조치 |
|---|---|---|
| 변환할 수 없는 입력 | 변환기의 오류 문구를 담은 `Error`로 거부됩니다. 그 호출만 실패합니다. | 오류 문구에 있는 셀이나 요소를 고치세요. |
| 메모리 부족으로 변환기 정지 | `ddi-xlsx converter stopped; try again` 오류로 거부됩니다. | 다시 호출하세요. 다음 호출은 변환기를 새로 로드합니다. |

같은 입력은 다시 실패할 수 있습니다. 같은 입력으로 다시 실패하면, 그 코드북은 CLI로 변환하세요.

## wasm 로드

- wasm은 처음 호출할 때 한 번 로드합니다.
- Node에서는 파일에서 읽습니다.
- 브라우저에서는 `fetch`로 받습니다.

## 브라우저에서 쓸 때

- 패키지는 ES 모듈입니다.
- `index.js`는 `ddi-xlsx.wasm`을 같은 위치에서 찾습니다(`new URL('./ddi-xlsx.wasm', import.meta.url)`). 번들러를 쓰면, 이 파일이 함께 배포되는지 확인하세요.
- 서버는 `.wasm` 파일을 `Content-Type: application/wasm`으로 보내야 합니다. 다른 타입이면 브라우저가 모듈을 로드하지 않습니다.
- WASM 파일 크기는 빌드 버전에 따라 다릅니다. gzip 압축을 쓰면 전송 크기가 줄어듭니다.
- 변환은 호출한 스레드에서 끝날 때까지 실행됩니다. 메인 스레드에서 호출하면, 그동안 화면이 멈춥니다.

## 큰 코드북

브라우저에서 큰 코드북을 변환하면, Web Worker 안에서 실행하세요. 아래는 합성 코드북 두 개로 측정한 결과입니다.

- 파일 A: 변수 700개, 변수마다 범주 40개입니다. `Categories` 시트가 10.5 MB입니다.
- 파일 B: 변수 3,358개, 변수마다 범주 10개입니다.

| 런타임 | 파일 | 시간 | 메모리 |
|---|---|---|---|
| Node 22의 wasm | A | DDI → xlsx 약 1.3초, xlsx → DDI 약 2.6초 | 약 0.7 GB |
| CLI(네이티브) | A | DDI → xlsx 약 0.22초, xlsx → DDI 약 0.44초 (왕복 약 0.7초) | 측정하지 않음 |
| CLI(네이티브) | B | 왕복 약 0.95초 | 측정하지 않음 |

브라우저에서는 큰 코드북의 시간과 메모리를 측정하지 않았습니다.

wasm 안에서 메모리가 부족하면, 그 호출은 `ddi-xlsx converter stopped; try again` 오류로 거부됩니다. 다음 호출은 wasm을 새로 로드합니다. 그래서 페이지를 새로 고치거나 프로세스를 다시 시작할 필요가 없습니다.

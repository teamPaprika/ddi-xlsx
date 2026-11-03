---
title: Node·브라우저 SDK
weight: 6
---

npm 패키지는 같은 변환기를 WebAssembly로 실행합니다. 설치 방법은 [설치](install.md)에 있습니다.

## 함수

```ts
ddiToXlsx(input: string | Uint8Array): Promise<{ xlsx: Uint8Array; warnings: string[] }>
xlsxToDdi(xlsx: Uint8Array): Promise<{ xml: string; json: string; warnings: string[] }>
templateXlsx(): Promise<Uint8Array>
```

- `ddiToXlsx`는 DDI XML이나 JSON을 워크북으로 바꿉니다.
- `xlsxToDdi`는 워크북을 XML과 JSON으로 바꿉니다. 둘은 같은 트리를 담습니다.
- `templateXlsx`는 빈 양식 워크북의 바이트를 돌려줍니다. 양식의 내용은 [빈 양식으로 시작하기](blank-template.md)에 있습니다.

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

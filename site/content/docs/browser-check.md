---
title: 브라우저 코드북 검사실
weight: 8
---

{{< callout type="info" >}}
문서 사이트의 [브라우저에서 실행](/docs/playground) 페이지에서 같은 검사와 모든 변환을 바로 실행합니다. 이 페이지는 검사실을 HTML 파일 하나로 빌드하는 방법을 설명합니다.
{{< /callout >}}

검사실은 Go WASM과 [SDK](npm.md)를 Web Worker에서 실행합니다.
파일 검사, 코드북 변환, JavaScript REPL을 제공합니다.
검사·변환의 기본 코드는 입력 파일을 외부로 보내지 않습니다.
사용자가 작성한 REPL 코드는 브라우저 JavaScript로 실행합니다.

## 빌드와 실행

Go 1.25 이상과 Node.js 18 이상이 필요합니다.
저장소 루트에서 다음 명령을 실행하세요.

```bash
node demo/build.mjs
```

`demo/dist/index.html`을 브라우저에서 여세요.
이 HTML은 WASM, 같은 Go 버전의 로더, SDK, Worker를 포함합니다.
Node 서버나 CDN이 필요하지 않습니다. 정적 호스팅에도 같은 HTML을 사용합니다.
브라우저는 gzip 압축을 푸는 `DecompressionStream`을 지원해야 합니다.

빌드는 `npm/ddi-xlsx.wasm`과 `npm/wasm_exec.js`도 생성합니다.
Go 모듈이 캐시에 없으면 빌드할 때 다운로드합니다.
생성된 HTML과 WASM·로더는 Git에서 제외합니다.

## 입력과 실행

XML·JSON 파일을 고르면 DDI 형식을 선택합니다.
Excel 파일을 고르면 일반 DDI 편집용 양식과 재단 양식 중 하나를 선택하세요.
내장 예시는 합성 DDI와 빈 양식입니다. 고객 응답을 포함하지 않습니다.
화면은 20 MiB 이하 파일을 받습니다.

```js
return await api.checkCodebook(input, { format });
```

REPL은 `api`, `input`, `format`, `examples`, `state`, `console`, `download`를 제공합니다.
`state`는 실행 사이에 값을 유지합니다. 런타임 초기화는 Worker와 `state`를 새로 만듭니다.
오래 실행하는 코드는 런타임 초기화로 중지하세요.
`download(name, data)`는 파일 받기 버튼을 만듭니다.
이 함수는 REPL에서만 쓰는 보조 함수입니다.

## 검사 범위

실행 범위는 SDK의 [코드북 검사](npm.md#코드북-검사)와 같습니다.
XSD·응답 CSV·외부 기준정보·재코딩 실행은 검사하지 않습니다.
재단 양식 검사 결과는 최종 제출 가능 여부를 확정하지 않습니다.
표준 DDI 출력은 현재 Go 코어를 따르며 과거 `kosac:` 확장 출력으로 바꾸지 않습니다.

## 검사 명령

빌드한 뒤 다음 명령을 실행하세요.

```bash
node --test npm/test/check.test.mjs demo/bundle.test.mjs demo/app.test.mjs
```

SDK 검사는 실제 WASM을 호출합니다.
번들 검사는 내장 Worker와 REPL을 Node에서 실행합니다.
화면 검사는 DOM 대역으로 입력 형식 선택과 결과 표시를 확인합니다.
이 검사는 실제 브라우저의 화면·접근성 검사를 대신하지 않습니다.

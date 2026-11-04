# WASM 코드북 검사실

검사와 JavaScript REPL을 브라우저의 Web Worker에서 실행합니다. API 서버, Node 서버, CDN은 필요하지 않습니다. 회의 자료와 별도 페이지입니다.

## 빌드와 실행

Go 1.25 이상, Node.js 18 이상이 있는 환경에서 레포 루트의 명령을 실행합니다.

```bash
node demo/build.mjs
```

`demo/dist/index.html`을 브라우저에서 엽니다. 이 파일 하나에 Go WASM, 해당 Go 버전의 로더, 현재 npm SDK와 Worker·UI를 담습니다. 정적 호스팅에 이 HTML만 올려도 됩니다. 입력 파일을 받는 API는 없습니다.

WASM은 gzip으로 압축해 넣고 Worker에서 브라우저의 `DecompressionStream`으로 풉니다. 이 API를 지원하는 최신 브라우저가 필요합니다. 큰 인라인 스크립트가 브라우저에서 잘리는 일을 줄이면서 단일 HTML 구성을 유지합니다.

빌드는 `npm/ddi-xlsx.wasm`, `npm/wasm_exec.js`도 생성합니다. 세 파일은 빌드 산출물이며 Git에서 제외합니다. npm 설치는 필요하지 않습니다. Go 모듈이 캐시에 없다면 빌드할 때 다운로드합니다.

## 검사 범위

| 입력 형식 | 실행하는 기존 엔진 | 이 데모에서 실행하지 않는 검사 |
| --- | --- | --- |
| `ddi` | DDI XML·JSON 파서, `check.ValidateXML(..., nil)`의 지원 ID·변수·파일 참조 검사, kosac 확장 반려 | XSD, 모든 DDI 참조의 완전성, 응답 CSV |
| `ddi-xlsx` | `workbook.FromXLSX`, 위 DDI 검사 | XSD, 응답 CSV |
| `kosac-xlsx` | `template.Read`, `check.Check(cb, kosac.RefData{})` | 재코딩 실행·영향 건수 재현, 외부 기준정보·코드표, 응답 CSV, XSD |

재단 v4.0과 v3.2를 읽는 절차는 현재 Go 엔진을 따릅니다. 양식 구조(S05)가 반려되면 코드북 규칙(C)을 실행하지 않습니다. 미실행 단계는 결과에 `not_run`으로 표시합니다. 오류가 없어도 전체 제출 검사 통과를 뜻하지 않습니다.

XML·JSON 파일을 고르면 DDI 입력으로 설정합니다. Excel을 올리면 입력 형식을 다시 선택해야 코드 실행이 켜집니다. 재단 코드북과 일반 DDI 편집용 Excel을 구분하며, 이전 파일의 형식을 새 Excel에 자동 적용하지 않습니다. 내장 예시 버튼은 각 예시의 형식을 함께 선택합니다.

v4의 `TRANSFORM_TYPE`은 필수 자유 입력입니다. 이 데모는 `RefData.TransformTypes` 승인 목록을 공급하지 않습니다. Go 호출자가 승인 목록을 명시하면 엔진은 그 목록을 검사합니다. 05 이력 `STATUS`는 빈 양식의 6개 값을 받고, 03·04 `RECODE_STATUS`의 두 허용값은 유지합니다. 코드북 수정·추가 매핑·포괄적 검토 이력의 수용과 실제 실행·재현 검증은 구분합니다.

제공하는 예시는 모두 합성 자료 또는 빈 양식입니다. 실제 고객의 설문·응답은 포함하지 않습니다. UI는 20 MiB 이하 파일을 받습니다. 큰 파일이나 오래 실행하는 코드는 초기화 버튼으로 중지할 수 있습니다. 파일을 수정하거나 자동으로 다운로드하지 않습니다.

## REPL

`api`, `input`, `format`, `examples`, `state`, `console`, `download`를 제공합니다. `return`한 값과 로그를 표시합니다. `state` 객체는 실행 사이에 유지되며, 런타임 초기화 때 지워집니다. 매 실행의 지역 변수는 다음 실행으로 넘어가지 않습니다.

```js
return await api.checkCodebook(input, { format });
```

```js
const { xlsx, warnings } = await api.ddiToXlsx(input);
download('codebook.xlsx', xlsx);
return { bytes: xlsx.length, warnings };
```

`download`는 이 REPL의 보조 함수이며 파일 받기 버튼을 만듭니다. `ddi-xlsx` 패키지의 API는 아닙니다. 검사·변환의 기본 코드는 파일을 외부로 보내지 않습니다. 사용자가 작성한 REPL 코드는 브라우저 JavaScript로 실행합니다.

## 앱에서 사용

빌드한 뒤 쓰려는 프로젝트에서 로컬 패키지를 설치합니다. npm 레지스트리에는 게시하지 않았습니다.

```bash
npm install /path/to/ddi-xlsx/npm
```

```js
import { checkCodebook } from 'ddi-xlsx';

const input = new Uint8Array(await file.arrayBuffer());
const report = await checkCodebook(input, { format: 'kosac-xlsx' });
console.log(report.issues);
```

일반 앱의 WASM 자산 배포와 Worker 사용은 [SDK 문서](../site/content/docs/npm.md)를 따릅니다. 단일 HTML 빌드는 SDK의 자산 로딩 부분에서 압축을 풀고, 동일한 SDK·Go WASM을 사용합니다. 압축을 푼 바이트가 npm WASM과 일치하는지 테스트합니다.

## 검증

```bash
node --test npm/test/check.test.mjs
node --test --test-skip-pattern='a call that stops the Go program' npm/test/roundtrip.test.mjs
node --test demo/bundle.test.mjs
node --test demo/app.test.mjs
```

서버 기반 검사·저장·재단 제출 기능은 포함하지 않습니다.

집중 검증에서는 강제 4 GiB 메모리 중단 테스트를 제외합니다. 단일 HTML의 묶인 코드 검증은 Node Worker에서 브라우저의 process·fetch 환경을 모사합니다. 재단 코드북 테스트 파일은 검사 때만 읽으며 HTML에는 포함하지 않습니다. `app.test.mjs`는 빌드한 HTML의 화면 로직을 모의 DOM에서 검사합니다. 실제 브라우저의 렌더링·파일 선택 창·Worker 실행을 확인한 결과와 구분합니다.

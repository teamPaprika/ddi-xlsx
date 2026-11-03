---
title: 기여하기
weight: 12
---

이 페이지는 코드와 문서를 바꾸는 절차를 설명합니다. 사람과 AI 에이전트(Claude Code, Codex)에게 같은 규칙을 적용합니다.

## 개발 환경

| 도구 | 버전 | 용도 |
|---|---|---|
| Go | 1.25 이상 | 빌드와 테스트 |
| Node | 22에서 테스트함 | npm 패키지 테스트 |
| `xmllint` | | XSD 검사. 없으면 일부 Go 테스트를 건너뜁니다. |
| Hugo extended | 0.157.0에서 확인함 | 문서 사이트 미리보기 |

## 저장소 구조

| 경로 | 내용 |
|---|---|
| `cmd/ddi-xlsx/` | CLI입니다. 파일 입출력은 이곳에서만 합니다. |
| `ddi/` | 트리 모델과 XML·JSON 읽기·쓰기 |
| `workbook/` | 트리와 xlsx 사이의 변환 |
| `kosac/` | 재단 코드북 엔진. 하위 패키지는 [재단 코드북 엔진](/docs/kosac)에 있습니다. |
| `internal/schema/` | `codebook.xsd`에서 생성한 규칙 표와 생성기(`gen/`) |
| `wasm/` | WebAssembly 진입점 |
| `npm/` | npm 패키지 |
| `xsd/ddi-codebook-2.5/` | 함께 넣어 둔 DDI Codebook 2.5 XSD |
| `testdata/` | 테스트 fixture |
| `docs/superpowers/` | 설계 문서(`specs/`)와 구현 계획(`plans/`) |
| `site/` | 이 문서 사이트(Hugo, Hextra 테마) |

## 작업 절차

{{% steps %}}

### 브랜치 만들기

`main`에서 브랜치를 만드세요.

```bash
git switch -c my-change main
```

### 코드와 테스트 고치기

동작을 바꾸면, 그 동작을 확인하는 테스트도 추가하세요. 큰 기능은 먼저 `docs/superpowers/specs/`의 설계 문서를 읽으세요.

### 문서 고치기

바뀐 동작을 `site/content/docs/`의 해당 페이지에 쓰세요. 규칙은 아래 [문서 규칙](#문서-규칙)에 있습니다.

### 검사 실행

아래 [검사 명령](#검사-명령)을 실행하세요.

### 커밋과 PR

커밋 메시지 규칙을 따르세요. 그다음 브랜치를 push하고 PR을 만드세요.

{{% /steps %}}

## 검사 명령

| 명령 | 실행하는 때 |
|---|---|
| `go generate ./internal/schema` | XSD나 생성기(`internal/schema/gen/`)를 바꾼 다음 |
| `go vet ./...` | 항상 |
| `go test ./...` | 항상 |
| `scripts/build.sh` | 배포용 빌드, 또는 npm 테스트 전 |
| `node --test npm/test/*.test.mjs` | `npm/`이나 `wasm/`을 바꾼 다음 |
| `scripts/check-docs.sh` | push 전 |

- `internal/schema/table.go`는 생성한 파일입니다. 직접 고치지 마세요.
- `xmllint`가 없으면, XSD 검사를 포함한 테스트를 통째로 건너뜁니다. 그 테스트 안의 golden 비교도 건너뜁니다. 전체를 확인하려면 `xmllint`를 설치하세요.
- `scripts/build.sh`는 CLI 실행 파일 여섯 개를 `dist/`에, wasm을 `npm/ddi-xlsx.wasm`에 빌드합니다. 같은 Go 툴체인의 `wasm_exec.js`도 `npm/`에 복사합니다. wasm과 `wasm_exec.js`는 같은 Go 버전에서 나와야 하기 때문입니다.
- npm 테스트에는 wasm이 필요합니다. 먼저 `scripts/build.sh`를 실행하세요.

## 커밋 메시지

- 제목은 한국어 한 줄입니다. 바뀐 내용을 씁니다.
- 제목 앞에 종류를 붙일 수 있습니다. 예: `fix:`, `test:`, `docs:`
- 본문은 `- `로 시작하는 목록입니다. 이유와 영향 범위를 씁니다.
- 문서를 바꿀 필요가 없는 기능 코드 변경이면, 본문에 `[no-docs]`와 이유를 쓰세요.

```text
workbook 셀 읽기 코드를 한 함수로 합침

- read.go와 template.go에 같은 코드가 있었다.
- [no-docs] 동작은 바뀌지 않는다.
```

## PR

PR 설명에 아래 내용을 쓰세요.

- 무엇을 바꾸었는지
- 어떻게 확인했는지(실행한 검사 명령)
- 바꾼 문서 페이지, 또는 `[no-docs]`의 이유

CI가 PR마다 문서 검사와 사이트 빌드를 실행합니다. `main`에 merge하면, CI가 사이트를 GitHub Pages에 배포합니다.

## 문서 규칙

### 문서를 고쳐야 하는 변경

아래 경로의 파일(테스트 파일 제외)을 바꾸면, 문서도 같은 브랜치에서 고치세요.

| 경로 | 예 |
|---|---|
| `cmd/`, `ddi/`, `workbook/`, `kosac/`, `wasm/`의 `.go` 파일 | CLI 옵션, 변환 규칙, 오류 문구 |
| `npm/index.js`, `npm/index.d.ts` | SDK 함수 |
| `kosac/xsd/*.xsd` | kosac XSD 초안 |
| `kosac/template/*.xlsx` | 내장 양식 |

문서는 `site/content/` 아래의 페이지와 `README.md`입니다.

### 문서 검사

`scripts/check-docs.sh`는 기본 브랜치(`origin/main`)와 현재 `HEAD` 사이의 변경을 봅니다. 위 표의 파일이 바뀌었고 문서가 바뀌지 않았으면, 검사가 실패합니다.

검사가 실패하면, 아래 중 하나를 하세요.

1. 바뀐 동작을 문서에 쓰고 커밋하세요.
2. 문서를 바꿀 필요가 없으면(리팩터링, 내부 수정, 테스트만 바꾼 경우), 커밋 메시지에 `[no-docs]`와 이유를 쓰세요.

검사는 아래 세 곳에서 실행됩니다.

| 실행 위치 | 실행 시점 | 설정 파일 |
|---|---|---|
| Claude Code | `git push`, `gh pr create` 전 | `.claude/settings.json` |
| Codex | `git push`, `gh pr create` 전 | `.codex/hooks.json` |
| GitHub Actions | PR마다 | `.github/workflows/docs.yml` |

에이전트 훅이 실패하면, 에이전트는 push나 PR 생성을 하지 않고 실패 이유를 받습니다.

{{< callout type="info" >}}
Codex는 신뢰한 프로젝트에서만 프로젝트 훅을 로드합니다. 처음에 Codex CLI에서 `/hooks`를 실행하고 이 훅을 신뢰하세요.
{{< /callout >}}

### 쓰기 규칙

문서는 쉬운 한국어로 씁니다. 기준은 ASD-STE100 Simplified Technical English의 규칙입니다.

- 한 문장에 한 가지 사실을 씁니다.
- 절차에서는 한 문장에 한 가지 지시를 씁니다. 지시는 "~하세요"로 끝냅니다.
- 설명은 "~합니다"로 끝냅니다.
- 조건을 지시 앞에 씁니다. 예: "파일이 있으면, 도구가 덮어씁니다."
- 주어를 씁니다. "도구가", "CLI가"처럼 누가 하는지 씁니다.
- "~수도 있습니다", "아마", "보통"처럼 뜻이 흐린 말을 쓰지 않습니다. 사실이나 조건을 씁니다.
- "등"을 쓰지 않습니다. 항목을 모두 쓰거나 "같은"을 씁니다.
- 세 가지 이상을 나열하면 목록이나 표로 씁니다.
- 코드, 명령, 파일 경로, 오류 문구는 바꾸지 않고 `코드`로 씁니다.

한 개념에는 한 단어를 씁니다.

| 단어 | 뜻 |
|---|---|
| 검사 | 도구, 테스트, `xmllint`가 하는 자동 확인 |
| 확인 | 사람이 직접 보는 일 |
| 워크북 | 도구가 읽고 쓰는 xlsx 파일 |
| 양식 | 빈 워크북. 재단 양식은 KOSAC v3.2·v4.0 xlsx |
| 오류 | 도구가 입력을 거부하는 결과 |
| 경고 | 도구가 알리지만 변환은 끝나는 결과 |

### 미리보기

`site/`에서 Hugo 개발 서버를 실행하세요.

```bash
cd site
hugo server
```

브라우저에서 `http://localhost:1313/ddi-xlsx/`를 여세요.

### 새 페이지 추가

1. `site/content/docs/`에 Markdown 파일을 만드세요.
2. front matter에 `title`과 `weight`를 쓰세요. `weight`는 사이드바 순서입니다.
3. 다른 페이지는 상대 경로의 `.md` 링크로 가리키세요. 예: `[CLI 사용법](cli.md)`

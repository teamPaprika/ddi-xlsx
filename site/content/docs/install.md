---
title: 설치
weight: 4
---

{{< tabs >}}

{{< tab name="CLI" >}}
`scripts/build.sh`는 `dist/`에 실행 파일 여섯 개를 빌드합니다. 빌드에는 Go 1.25 이상이 필요합니다.

```bash
scripts/build.sh
```

| 파일 | 운영체제 |
|---|---|
| `ddi-xlsx-darwin-arm64` | macOS (Apple silicon) |
| `ddi-xlsx-darwin-amd64` | macOS (Intel) |
| `ddi-xlsx-linux-arm64`, `ddi-xlsx-linux-amd64` | Linux |
| `ddi-xlsx-windows-arm64.exe`, `ddi-xlsx-windows-amd64.exe` | Windows |

1. 운영체제에 맞는 파일을 `PATH`에 있는 디렉터리로 복사하세요.
2. 파일 이름을 `ddi-xlsx`로 바꾸세요. Windows에서는 `ddi-xlsx.exe`로 바꾸세요.

이 문서의 명령 예시는 `ddi-xlsx`라는 이름을 씁니다. 따로 설치할 런타임은 없습니다. 실행 파일 크기는 빌드 버전에 따라 다릅니다.

현재 플랫폼용 CLI만 빌드하려면, 아래 명령을 쓰세요.

```bash
go build -o dist/ddi-xlsx ./cmd/ddi-xlsx
```
{{< /tab >}}

{{< tab name="Go" >}}
Go 모듈 경로는 `github.com/teamPaprika/ddi-xlsx`입니다. 저장소는 공개 상태입니다. 비공개 저장소 인증은 필요하지 않습니다.

```bash
go get github.com/teamPaprika/ddi-xlsx/workbook
```

사용 방법은 [Go 라이브러리](/docs/go)에 있습니다.
{{< /tab >}}

{{< tab name="npm" >}}
`npm/` 디렉터리가 npm 패키지입니다. npm 레지스트리에는 아직 게시하지 않았습니다.

`ddi-xlsx.wasm`과 `wasm_exec.js`는 빌드 산출물이라서 저장소에 없습니다.

1. 저장소 루트에서 `scripts/build.sh`를 실행하세요.
2. 패키지를 쓸 프로젝트에서 로컬 경로로 설치하세요.

```bash
npm install /path/to/ddi-xlsx/npm
```

사용 방법은 [Node·브라우저 SDK](/docs/npm)에 있습니다.
{{< /tab >}}

{{< /tabs >}}

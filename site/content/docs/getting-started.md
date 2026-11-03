---
title: 빠른 시작
weight: 2
---

이 절차는 소스에서 CLI를 빌드합니다. 그다음 빈 양식을 만들고 DDI XML·JSON으로 바꿉니다. 예시 명령은 macOS와 Linux용입니다.

## 준비

- Go 1.25 이상
- Git

## 절차

{{% steps %}}

### 저장소 받기

```bash
git clone https://github.com/teamPaprika/ddi-xlsx.git
cd ddi-xlsx
```

### CLI 빌드

```bash
go build -o dist/ddi-xlsx ./cmd/ddi-xlsx
```

### 빈 양식 만들기

```bash
./dist/ddi-xlsx template -o sample-codebook.xlsx
```

### 양식을 DDI로 바꾸기

```bash
./dist/ddi-xlsx from-xlsx sample-codebook.xlsx --xml sample-codebook.xml --json sample-codebook.json
```

{{% /steps %}}

{{< callout type="warning" >}}
`-o`, `--xml`, `--json`으로 지정한 경로에 파일이 있으면, 도구가 그 파일을 덮어씁니다.
{{< /callout >}}

## 다음 단계

- 이미 DDI 파일이 있으면, [CLI 사용법](cli.md)에서 `to-xlsx` 명령을 읽으세요.
- 양식을 채워서 DDI를 만들려면, [빈 양식으로 시작하기](blank-template.md)를 읽으세요.
- 재단 빈 양식은 `kosac-template` 명령으로 만듭니다. [KOSAC CLI](kosac/cli.md)를 읽으세요.
- Windows 실행 파일은 [설치](install.md)에 있습니다.

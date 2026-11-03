---
title: KOSAC CLI
weight: 1
---

```text
ddi-xlsx kosac-check    <양식.xlsx>
ddi-xlsx kosac-xml      <양식.xlsx> [-o out.xml]
ddi-xlsx kosac-template [-o kosac-template.xlsx]
```

## `kosac-check`

`kosac-check`는 양식에 S05와 C01~C18 검사를 실행합니다. 용역사는 제출 전에 이 명령으로 양식을 확인합니다.

```bash
ddi-xlsx kosac-check 양식.xlsx
```

```text
W C01 01_조사정보!E19 POP_SOURCE의 값이 확인 불가로 적혀 있습니다.
R C07 03_변수정의!B5 명목·서열 변수에는 결측 코드가 아닌 코드가 두 개 이상 있어야 합니다.
R XSD-DDI DDI 투영이 DDI 2.5 XSD를 통과하지 못했습니다.: -:46: element bogus: Schemas validity error : …
```

### 출력 형식

- 결과를 stdout에 한 줄에 하나씩 씁니다. 형식은 `<수준> <규칙> <시트>!<셀> <메시지>`입니다.
- 수준은 `R`(반려)이나 `W`(경고)입니다.
- 셀이 없는 결과는 `<시트>!<셀>` 자리에 XML 경로를 씁니다. 셀과 경로가 모두 없으면 그 자리를 뺍니다.
- XSD-DDI와 XSD-KOSAC 결과는 메시지 뒤에 `: <검사기 메시지>`를 붙입니다. 이 두 규칙의 메시지는 항상 같습니다. 그래서 무엇이 틀렸는지는 검사기(xmllint)의 원문만 알려 줍니다. 검사를 실행하지 못했으면, 그 원인이 붙습니다.

### 종료 코드와 건너뛰는 검사

- `R`이 하나라도 있으면 종료 코드는 1이고, 없으면 0입니다.
- S05에서 `R`이 나오면, C01~C18을 실행하지 않습니다. 양식 구조가 틀리면 다른 칸을 잘못 읽기 때문입니다.
- CLI는 재단 기준 자료를 받지 않습니다. 기준 자료는 표준 항목 사전, 코드셋, 업로드 파일 목록, PDF 쪽수입니다. 그래서 이 자료가 필요한 검사는 건너뜁니다.

## `kosac-xml`

`kosac-xml`은 양식을 읽고, 초안을 채우고(`DeriveDefaults`), ID를 붙여(`AssignIDs`) 코드북 XML을 씁니다. 이 명령은 미리보기와 디버깅용입니다.

```bash
ddi-xlsx kosac-xml 양식.xlsx -o preview.xml
```

- FK, 클린징, 패키지 정보(`kosac:package`)는 넣지 않습니다. 그래서 출력에 `docDscr`가 없습니다.
- 담당자 입력 항목에는 `DeriveDefaults`가 채운 자동 기본값만 들어갑니다. 예: 조사 분류 코드, 조사 연도, 차수 유형 `main`, 파일 민감도 `3`
- C01~C18은 실행하지 않습니다. 내용 검사는 `kosac-check`로 하세요.
- S05에서 `R`이 나오면, XML을 쓰지 않습니다. 그 결과만 출력하고 종료 코드 1로 끝납니다.
- XML을 쓴 다음, S05 경고와 XML 검사 결과를 `kosac-check`와 같은 형식으로 출력합니다. `R`이 있으면 종료 코드는 1입니다. 쓴 파일은 남습니다.
- `PATH`에 `xmllint`가 있으면, 두 투영의 XSD 검사까지 합니다. 없으면 stderr에 `XSD 검증 건너뜀: xmllint 없음`을 쓰고, PLACE와 REF만 검사합니다.
- `-o`가 없으면, 입력 파일의 확장자를 `.xml`로 바꾼 경로에 씁니다.

## `kosac-template`

`kosac-template`은 내장 빈 양식을 씁니다. `-o`가 없으면, 현재 디렉터리의 `kosac-template.xlsx`에 씁니다.

```bash
ddi-xlsx kosac-template -o my-kosac.xlsx
```

내장 빈 양식은 임시본입니다. [XML 검사와 양식 파일](validation.md#양식-파일)을 읽으세요.

## 출력 경로 규칙

`kosac-xml`과 `kosac-template`의 출력 경로 규칙은 다른 명령과 같습니다.

- 기본 경로에 파일이 이미 있으면, 덮어쓰지 않고 오류로 끝납니다.
- `-o`로 지정한 경로는 덮어씁니다.

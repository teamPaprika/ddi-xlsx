---
title: KOSAC CLI
weight: 1
---

```bash
ddi-xlsx kosac-check survey.xlsx
ddi-xlsx kosac-xml survey.xlsx -o codebook.xml --json codebook.json
ddi-xlsx kosac-template -o blank-v4.xlsx
```

`kosac-check`는 S05 구조 검사 후 내용 규칙을 실행합니다.
v3.2는 C01~C18, v4.0은 버전별 추가 규칙을 적용합니다.
규칙은 재단 프로파일의 요구사항이며 DDI 자체의 필수 조건과 구분합니다.

- 결과 형식은 `<수준> <규칙> <시트>!<셀> <메시지>`입니다.
- `R`은 반려, `W`는 경고입니다. `R`이 있으면 종료 코드 1, 없으면 0입니다.
- S05가 구조를 반려하면 내용 규칙을 실행하지 않습니다.
- CLI는 외부 코드표, 응답 원자료, 업로드 파일 목록, PDF 쪽수를 받지 않습니다. 해당 자료가 필요한 검사는 실행하지 않습니다.

`kosac-xml`은 구조를 읽고, `DeriveDefaults`와 `AssignIDs`를 거쳐 표준 XML을 씁니다.
`--json`을 주면 같은 트리의 JSON도 씁니다. JSON은 이 라이브러리의 트리 표현입니다. JSON 규칙은 [받는 입력](/docs/input#json-형식)에 있습니다.

- 내용 규칙은 별도 `kosac-check`로 실행합니다. XML 생성 성공은 제출 승인과 다릅니다.
- 결측 여부가 미확정인 v4.0 코드는 오류로 거부합니다. DDI의 기본값 `missing="N"`으로 추정하지 않습니다.
- 출력 후 REF와 전체 XSD를 검사합니다. 검증 반려 시에도 검토용 출력은 남습니다.
- `xmllint`가 없으면 XSD를 건너뛰었다고 stderr에 표시합니다. 참조 검사는 계속합니다.
- `-o`를 생략하면 입력과 같은 이름의 `.xml`을 씁니다. 그 파일이 있으면 오류로 멈춥니다.
- 명시한 출력 경로는 덮어씁니다. 입력·XML·JSON이 같은 파일을 가리키면 거부합니다.

# kosac 3.3 XSD 초안 — 변경 내역

이 문서는 재단 승인용이다. `kosac-3.3.xsd`의 구조와, 요청서 v1.2(4장 코드북 XML 최종 구조)와 다르게 정한 점을 적는다.

- 네임스페이스: `urn:kosac:survey-meta:3.3`. 접두는 항상 `kosac`이다.
- `elementFormDefault="qualified"`이다. kosac 요소의 속성은 접두 없이 쓰고, var·varGrp에 붙는 kosac 속성만 `kosac:` 접두를 쓴다.

## 개요

### 두 투영으로 검증하는 이유

공식 DDI 2.5 XSD(`codebook.xsd`)에는 다른 네임스페이스의 요소나 속성을 받는 자리가 없다. 그래서 kosac 요소가 들어간 코드북은 공식 XSD를 그대로 통과할 수 없다. 엔진은 코드북 하나를 두 문서로 나누어 각각 검증한다.

1. **DDI 투영**: `kosac:` 요소와 속성, `xmlns:kosac` 선언을 모두 걷어낸 문서. 공식 DDI 2.5 XSD로 검증한다. 결과 규칙은 XSD-DDI이다.
2. **kosac 투영**: `kosac:projection` 아래에, kosac 내용이 있는 DDI 요소마다 `kosac:place`를 하나씩 둔다.
   - `kosac:place/@path`는 그 DDI 요소의 경로이다. 형식은 검사 결과의 경로와 같다. 예: `dataDscr/var[@ID='V12']`.
   - kosac 자식 요소는 그대로 복사한다.
   - kosac 속성은 place 요소의 속성으로 옮긴다.
   - 이 문서를 `kosac-3.3.xsd`로 검증한다. 결과 규칙은 XSD-KOSAC이다.

`kosac:place`는 `xs:any namespace="##targetNamespace" processContents="strict"`와 `xs:anyAttribute namespace="##targetNamespace" processContents="strict"`로 내용을 받는다. 그래서 place 안의 kosac 이름은 하나하나 전역 선언과 대조된다. 선언되지 않은 요소나 속성(예: `kosac:foo`)은 오류이다.

투영 예:

```xml
<kosac:projection xmlns:kosac="urn:kosac:survey-meta:3.3">
  <kosac:place path="dataDscr/var[@ID='V2']" kosac:seq="8" kosac:role="ITEM" kosac:measLevel="nominal">
    <kosac:codeSetId>CODES2_1</kosac:codeSetId>
    <kosac:analysis physicalName="sq1" dimMeasure="DIM" includeYn="Y"/>
  </kosac:place>
</kosac:projection>
```

### XSD 밖에서 하는 검사 (PLACE, REF)

kosac 투영에서는 kosac 내용이 원래 DDI 부모와 떨어진다. 그래서 다음 두 가지는 XSD가 아니라 엔진 코드가 확인한다.

- **PLACE (배치 검사)**: kosac 요소가 아래 배치 표의 부모 아래에만 있는지, kosac 속성이 정해진 요소(var, varGrp)에만 있는지 확인한다.
- **REF (참조 검사)**: libxml2(xmllint)는 IDREF·IDREFS가 가리키는 ID가 문서에 있는지 확인하지 않는다. 예를 들어 `var@files="F9"`는 파일 F9가 없어도 DDI XSD를 통과한다. 또 kosac 투영에는 참조 대상 ID가 없으므로, kosac의 참조 속성은 `xs:IDREF`가 아니라 `xs:NCName`(하나)과 `xs:NMTOKENS`(목록)로 선언했다. `xs:IDREF`로 선언하면 XSD 1.0 검증기가 모든 참조를 실패로 본다. REF는 다음을 확인한다.
  - 모든 ID(루트 `codeBook@ID` 포함)가 NCName 형식이고 문서 안에서 한 번만 쓰인다.
  - DDI 참조: `var@files`, `location@fileid`, `varGrp@var`, `var@wgt-var`.
  - kosac 참조: `kosac:idVar/@ref`, `kosac:systemIdVar/@ref`, `kosac:link/@var`, `kosac:link/@fileRef`, `kosac:sampling/@strataVars`, `kosac:sampling/@clusterVar`, `kosac:splitBallot/@var`, `kosac:weight/@var`, `kosac:rule/@var`, `kosac:unverified/@var`, `kosac:unverified/@file`, `@kosac:parentVar`.
  - 변수 참조는 문서의 `var@ID`를, 파일 참조는 `fileDscr@ID`를 가리켜야 한다. 끊긴 참조는 하나마다 R이다.

### 전역 선언과 지역 선언

- 전역 선언은 place가 받는 것뿐이다. 그리고 검증용 루트인 `kosac:projection`이 있다.
  - 아래 배치 표의 kosac 요소 22개
  - var·varGrp에 붙는 kosac 속성 13개
- 그 밖의 하위 요소는 모두 지역 선언이다. 예: `kosac:sourceSurvey`, package 아래 `kosac:file`, `kosac:validation`, `kosac:round`, `kosac:unverified`, `kosac:idVar`, `kosac:link`, `kosac:rule`, `kosac:map`, `kosac:entry`, `kosac:synonym`.
- 지역 요소를 DDI 요소 바로 아래에 두면(예: stdyDscr 아래 `kosac:round`) place에서 전역 선언을 찾지 못해 오류가 난다.
- 이름은 같고 모양이 다른 `kosac:file`이 두 가지 있다(fileDscr 자식, package 자식). 둘 다 전역으로 선언하면 스키마가 컴파일되지 않는다(xmllint로 확인함). 그래서 package 쪽을 지역으로 선언했다. 아래 "요청서 v1.2와 다른 점" 19행을 본다.

### 배치 표

kosac 요소는 같은 부모의 DDI 자식 뒤에, 아래 순서로 붙는다.

| DDI 부모 | 뒤에 붙는 kosac 자식 (이 순서) |
|---|---|
| docDscr | package |
| stdyDscr | study |
| dataColl | sampling, splitBallot, weight, excludeN, excludeRule |
| anlyInfo | contactN, rrFormula |
| fileDscr | file, cleansing |
| varGrp | noneOption |
| var | fk, scale, missingCodes, codeSetId, analysis |
| catgry | codeType, stdCode |
| otherMat | pages, sha256, bytes |

kosac 속성은 DDI 속성 뒤에 붙는다.

- var: seq, role, measLevel, pii, sensitive, reverse, mrType, mrGroup, rowLabel, rankNo, cleansed, textFormat, parentVar
- varGrp: mrGroup

### 값 형식

- 목록 값은 enumeration이다. 대소문자와 공백까지 그대로 비교한다. 값의 출처는 아래 "요청서가 정하지 않아 초안에서 정한 것"에 적는다.
- 개수·크기·행 수는 `xs:nonNegativeInteger`(0 이상 정수)이다. 순번(열 순서, 순위, 파일 번호, 차수, 코드북 버전)은 `xs:positiveInteger`(1 이상 정수)이다.
- sha256은 64자리 16진수이다.
- 쪽 범위는 `{a}`, `{a}-{b}`, `{a}-`(끝쪽 없음) 중 하나이다. 예: `336-`.
- 연도는 네 자리 숫자이다.
- 참조는 `xs:NCName`(변수·파일 ID 하나)이나 `xs:NMTOKENS`(공백으로 구분한 ID 목록)이다.
- 그 밖의 값은 `xs:string`이다.

### 요청서가 정하지 않아 초안에서 정한 것

재단 회의에서 확인할 항목이다. 형식 이름은 XSD의 `type` 값이다. 문자열이 아닌 형식은 datajam이 넘길 수 있는 값을 좁히므로, 모두 여기에 적는다.

- **목록 값의 비교 방식**: 모든 enumeration은 `xs:string`을 바탕으로 한다. 그래서 대소문자와 공백까지 그대로 비교한다. 앞뒤 공백을 지우지 않으므로 `" Y"`는 오류이다. 목록의 값은 요청서 4-2 표에서 가져왔다. 대상 형식은 다음과 같다.
  - `kosac:ynType`, `kosac:roleType`, `kosac:measLevelType`, `kosac:dimMeasureType`, `kosac:codeTypeType`, `kosac:mappingType`, `kosac:actionType`, `kosac:waveTypeType`, `kosac:linkLevelType`, `kosac:sensLevelType`, `kosac:fileRoleType`, `kosac:sourceKindType`, `kosac:scaleDirType`, `kosac:textFormatType`
  - `kosac:mrTypeType`: 요청서 4-2 표에는 이 값의 목록이 없다. 양식 목록 값(`보기별 열`, `순위별 열`)을 enumeration으로 두었다.
- **정수 구분**:
  - `xs:positiveInteger`(1 이상 정수): `@kosac:seq`, `@kosac:rankNo`, `kosac:file/@fileNo`, `kosac:round/@waveNo`, `kosac:package/@codebookVersion`. codebookVersion은 정수만 받는다. "1.1" 같은 값은 XSD-KOSAC 오류이다.
  - `xs:nonNegativeInteger`(0 이상 정수): 개수·크기·행 수이다. 대상은 `bytes`, `rawBytes`, `cellsChanged`, `rowsAffected`, `rows`, `warningCount`, `matchedRows`, `unmatchedRows`, `kosac:excludeN`, `kosac:contactN`, `kosac:study/@validN`, `kosac:bytes`이다.
- **연도** `kosac:yearType`(네 자리 숫자):
  - `kosac:study/@schoolCodeYear`와 `kosac:study/@bizYear`는 엔진 C02 검사의 YYYY 규칙을 따른다.
  - `kosac:round/@year`와 `kosac:fk/@refYear`에도 같은 형식을 썼다. 이 둘은 요청서에도 C02에도 형식 규정이 없다.
- **sha256** `kosac:sha256Type`: 64자리 16진수이다. 소문자뿐 아니라 대문자 16진수도 받는다.
- **쪽 범위** `kosac:pagesType`: `{a}`, `{a}-{b}`, `{a}-`이다. 요청서에는 예("336-")만 있고, 형식은 엔진 C18 검사의 정규형이다.
- **참조 형식**: `xs:NCName`(하나)과 `xs:NMTOKENS`(공백으로 구분한 목록)이다.
  - 문서 안 ID를 가리키는 참조는 REF가 대상이 있는지까지 확인한다.
  - datajam이 준 값을 그대로 쓰는 두 속성도 `xs:NCName`으로 두었다: `kosac:cleansing/@rawFileRef`(예: `F1_raw`), package 쪽 `kosac:file/@fileRef`(예: `F1`). 이 두 값에 공백이 있거나 숫자로 시작하면 XSD-KOSAC 오류이다.
- **십진수** `xs:decimal`: `kosac:scale/@min`, `@max`. 코드값은 정수이지만 더 넓게 받는다.
- 문자열로 둔 값: 일시(`approvedAt`, `passedAt`, `appliedAt`, `validatedAt`), 버전 문자열(`publishedVersion`, `templateVersion`, `codeSetVersion`, `ruleset`), `aggRules`(요청서는 "AVG;COUNT;RATIO 등"으로 목록을 닫지 않았다), `kosac:rule/@source`(X07·K04·MANUAL 등), `kosac:missingCodes`.
- 양식 목록이지만 요청서 4장에 값이 없어 문자열로 둔 값: `kosac:sampling/@type`(SAMPLING_TYPE), `kosac:study/@ruleVersion`(RULE_VERSION), `kosac:study/@surveyType`(SURVEY_TYPE). 이 값들은 엔진의 C02 검사가 양식 목록으로 확인한다.
- 필수 속성: `kosac:place/@path`, `kosac:unverified/@key`, `kosac:idVar/@ref`, `kosac:systemIdVar/@ref`, `kosac:rule/@reason`(요청서가 사유를 필수로 정함). 나머지는 모두 선택이다. 엔진은 빈 값을 쓰지 않기 때문이다.

## 요소·속성 목록

- 이름 열: 요소는 `kosac:요소`, 요소의 속성은 `kosac:요소/@속성`, var·varGrp에 붙는 전역 속성은 `@kosac:속성`으로 적는다.
- 출처 열: 양식 KEY, 또는 (담당자)·(시스템). 담당자 입력 항목은 요청서 4-2 표의 이름을 쓴다.

### 검증용 요소

| 이름 | 부모 | 형식 | 출처 |
|---|---|---|---|
| `kosac:projection` | (검증 문서의 루트) | `kosac:place` ×n | 검증용. 코드북에는 쓰지 않는다 |
| `kosac:place` | `kosac:projection` | kosac 요소 ×n, kosac 속성 | 검증용 |
| `kosac:place/@path` | `kosac:place` | 문자열, 필수 | 검증용. 원래 DDI 요소의 경로 |

### docDscr

| 이름 | 부모 | 형식 | 출처 |
|---|---|---|---|
| `kosac:package` | docDscr | sourceSurvey?, file ×n, validation? | (시스템) 패키지 정보가 아직 없는 네이티브 초안에는 `kind="native"`인 sourceSurvey만 둔다(@cleansed 없음) |
| `kosac:package/@exportId` | `kosac:package` | 문자열 | (시스템) 내보내기 ID `exp_{YYYYMMDD}_{seq}` |
| `kosac:package/@codebookVersion` | `kosac:package` | 1 이상 정수 | (시스템) 코드북 버전 |
| `kosac:package/@previousVersion` | `kosac:package` | 문자열 | (시스템) 이전 버전 ID |
| `kosac:package/@approvedBy` | `kosac:package` | 문자열 | (시스템) 내보내기를 실행한 담당자 ID |
| `kosac:package/@approvedAt` | `kosac:package` | 문자열 | (시스템) 내보낸 일시 |
| `kosac:package/@cleansed` | `kosac:package` | 목록: Y, N | (시스템) 클린징 규칙이 하나라도 있으면 Y |
| `kosac:sourceSurvey` | `kosac:package` | 빈 요소 | (시스템) |
| `kosac:sourceSurvey/@wallaId` | `kosac:sourceSurvey` | 문자열 | (시스템) 왈라 설문 ID |
| `kosac:sourceSurvey/@kind` | `kosac:sourceSurvey` | 목록: upload, native | (시스템) |
| `kosac:sourceSurvey/@publishedVersion` | `kosac:sourceSurvey` | 문자열 | (시스템) 게시본 |
| `kosac:sourceSurvey/@templateVersion` | `kosac:sourceSurvey` | 문자열 | (시스템) 양식 버전 |
| `kosac:file` | `kosac:package` | 빈 요소 | (시스템) 패키지 파일 하나 |
| `kosac:file/@role` | `kosac:package/kosac:file` | 목록: codebook, data, data_raw, cleansing, questionnaire, report, extra | (시스템) |
| `kosac:file/@fileRef` | `kosac:package/kosac:file` | NCName | (시스템) 데이터 파일 참조 |
| `kosac:file/@name` | `kosac:package/kosac:file` | 문자열 | (시스템) 패키지 안 파일명 |
| `kosac:file/@bytes` | `kosac:package/kosac:file` | 0 이상 정수 | (시스템) |
| `kosac:file/@sha256` | `kosac:package/kosac:file` | 64자리 16진수 | (시스템) |
| `kosac:file/@pages` | `kosac:package/kosac:file` | 쪽 범위 | (시스템) 보고서 안 설문지의 쪽 |
| `kosac:validation` | `kosac:package` | validationNote ×n | (시스템) 검증 통과 정보 |
| `kosac:validation/@passedAt` | `kosac:validation` | 문자열 | (시스템) |
| `kosac:validation/@ruleset` | `kosac:validation` | 문자열 | (시스템) 검증 규칙 집합 버전 |
| `kosac:validation/@warningCount` | `kosac:validation` | 0 이상 정수 | (시스템) |
| `kosac:validationNote` | `kosac:validation` | 문자열 | (시스템) 경고 한 건 |

### stdyDscr

| 이름 | 부모 | 형식 | 출처 |
|---|---|---|---|
| `kosac:study` | stdyDscr | round?, bizName?, popSource?, unverified ×n | |
| `kosac:study/@ruleVersion` | `kosac:study` | 문자열 | RULE_VERSION |
| `kosac:study/@surveyType` | `kosac:study` | 문자열 | SURVEY_TYPE |
| `kosac:study/@dataNo` | `kosac:study` | 문자열 | DATA_NO |
| `kosac:study/@categoryCode` | `kosac:study` | 문자열 | (담당자) CATEGORY_CODE |
| `kosac:study/@validN` | `kosac:study` | 0 이상 정수 | VALID_N |
| `kosac:study/@schoolCodeYear` | `kosac:study` | 네 자리 연도 | SCHOOL_CODE_YEAR |
| `kosac:study/@bizYear` | `kosac:study` | 네 자리 연도 | BIZ_YEAR |
| `kosac:round` | `kosac:study` | 빈 요소 | (담당자) 조사 실시 단위 |
| `kosac:round/@year` | `kosac:round` | 네 자리 연도 | (담당자) SURVEY_YEAR |
| `kosac:round/@waveType` | `kosac:round` | 목록: main, pre, post | (담당자) WAVE_TYPE |
| `kosac:round/@waveNo` | `kosac:round` | 1 이상 정수 | (담당자) WAVE_NO |
| `kosac:bizName` | `kosac:study` | 문자열 | BIZ_NAME |
| `kosac:popSource` | `kosac:study` | 문자열 | POP_SOURCE |
| `kosac:unverified` | `kosac:study` | 문자열(사유) | (시스템) "확인 불가"로 적은 셀 |
| `kosac:unverified/@key` | `kosac:unverified` | 문자열, 필수 | (시스템) 그 셀의 양식 KEY |
| `kosac:unverified/@file` | `kosac:unverified` | NCName | (시스템) 02 시트 값이면 그 파일의 ID |
| `kosac:unverified/@var` | `kosac:unverified` | NCName | (시스템) 03 시트 값이면 그 변수의 ID |

### dataColl, anlyInfo

| 이름 | 부모 | 형식 | 출처 |
|---|---|---|---|
| `kosac:sampling` | dataColl | 빈 요소 | |
| `kosac:sampling/@type` | `kosac:sampling` | 문자열 | SAMPLING_TYPE |
| `kosac:sampling/@frame` | `kosac:sampling` | 문자열 | SAMPLE_FRAME |
| `kosac:sampling/@strataVars` | `kosac:sampling` | NMTOKENS (변수 ID 목록) | STRATA_VARS |
| `kosac:sampling/@clusterVar` | `kosac:sampling` | NCName (변수 ID) | CLUSTER_VAR |
| `kosac:splitBallot` | dataColl | 빈 요소 | |
| `kosac:splitBallot/@yn` | `kosac:splitBallot` | 목록: Y, N | SPLIT_BALLOT |
| `kosac:splitBallot/@var` | `kosac:splitBallot` | NCName (변수 ID) | SPLIT_VAR |
| `kosac:weight` | dataColl | 빈 요소 | |
| `kosac:weight/@yn` | `kosac:weight` | 목록: Y, N | WEIGHT_YN |
| `kosac:weight/@var` | `kosac:weight` | NCName (변수 ID) | WEIGHT_VAR |
| `kosac:excludeN` | dataColl | 0 이상 정수 | EXCLUDE_N |
| `kosac:excludeRule` | dataColl | 문자열 | EXCLUDE_RULE |
| `kosac:contactN` | anlyInfo | 0 이상 정수 | CONTACT_N |
| `kosac:rrFormula` | anlyInfo | 문자열 | RR_FORMULA |

### fileDscr

| 이름 | 부모 | 형식 | 출처 |
|---|---|---|---|
| `kosac:file` | fileDscr | idVar?, systemIdVar?, link? | |
| `kosac:file/@fileNo` | fileDscr/`kosac:file` | 1 이상 정수 | FILE_NO |
| `kosac:file/@unit` | fileDscr/`kosac:file` | 문자열 | UNIT |
| `kosac:file/@sensLevel` | fileDscr/`kosac:file` | 목록: 1, 2, 3 | (담당자) SENS_LEVEL |
| `kosac:file/@sha256` | fileDscr/`kosac:file` | 64자리 16진수 | (시스템) 전처리본의 해시 |
| `kosac:file/@bytes` | fileDscr/`kosac:file` | 0 이상 정수 | (시스템) 전처리본의 크기 |
| `kosac:file/@rawSha256` | fileDscr/`kosac:file` | 64자리 16진수 | (시스템) 원본의 해시 |
| `kosac:file/@rawBytes` | fileDscr/`kosac:file` | 0 이상 정수 | (시스템) 원본의 크기 |
| `kosac:file/@rawName` | fileDscr/`kosac:file` | 문자열 | (시스템) 원본 파일명 |
| `kosac:file/@sourceName` | fileDscr/`kosac:file` | 문자열 | FILE_NAME (업로드한 원래 이름) |
| `kosac:idVar` | fileDscr/`kosac:file` | 빈 요소 | |
| `kosac:idVar/@ref` | `kosac:idVar` | NCName (변수 ID), 필수 | ID_VAR |
| `kosac:systemIdVar` | fileDscr/`kosac:file` | 빈 요소 | |
| `kosac:systemIdVar/@ref` | `kosac:systemIdVar` | NCName (변수 ID), 필수 | (시스템) 그 파일의 walla_response_id |
| `kosac:link` | fileDscr/`kosac:file` | 빈 요소 | |
| `kosac:link/@var` | `kosac:link` | NCName (변수 ID) | LINK_VAR |
| `kosac:link/@fileRef` | `kosac:link` | NCName (파일 ID) | LINK_FILE_NO |
| `kosac:link/@level` | `kosac:link` | 목록: RESPONDENT, SCHOOL, HOUSEHOLD | (담당자) LINK_LEVEL |
| `kosac:cleansing` | fileDscr | rule ×n | (담당자·시스템) 클린징 기록 |
| `kosac:cleansing/@appliedBy` | `kosac:cleansing` | 문자열 | (시스템) 확정한 담당자 ID |
| `kosac:cleansing/@appliedAt` | `kosac:cleansing` | 문자열 | (시스템) 확정 일시 |
| `kosac:cleansing/@rawFileRef` | `kosac:cleansing` | NCName | (시스템) 원본 파일 참조(패키지 쪽 값) |
| `kosac:cleansing/@cellsChanged` | `kosac:cleansing` | 0 이상 정수 | (시스템) |
| `kosac:cleansing/@rowsAffected` | `kosac:cleansing` | 0 이상 정수 | (시스템) |
| `kosac:cleansing/@logFile` | `kosac:cleansing` | 문자열 | (시스템) cleansing.json 파일명 |
| `kosac:rule` | `kosac:cleansing` | map? | (담당자·시스템) 클린징 규칙 하나 |
| `kosac:rule/@id` | `kosac:rule` | 문자열 | (시스템) `CL-{nn}` |
| `kosac:rule/@var` | `kosac:rule` | NCName (변수 ID) | (담당자) 대상 변수 |
| `kosac:rule/@action` | `kosac:rule` | 목록: REPLACE, MISSING_9, MISSING_8, MISSING_7, MASK, SWAP | (담당자) |
| `kosac:rule/@condition` | `kosac:rule` | 문자열 | (담당자) 대상 행 조건 |
| `kosac:rule/@rows` | `kosac:rule` | 0 이상 정수 | (시스템) 바뀐 행 수 |
| `kosac:rule/@source` | `kosac:rule` | 문자열 | (시스템) 검증 규칙 ID 또는 MANUAL |
| `kosac:rule/@reason` | `kosac:rule` | 문자열, 필수 | (담당자) 사유 |
| `kosac:rule/@evidence` | `kosac:rule` | 문자열 | (담당자) 근거 자료 |
| `kosac:map` | `kosac:rule` | entry ×n | (담당자) REPLACE·SWAP의 매핑표 |
| `kosac:entry` | `kosac:map` | 빈 요소 | (담당자) |
| `kosac:entry/@from` | `kosac:entry` | 문자열 | (담당자) 원본 값 |
| `kosac:entry/@to` | `kosac:entry` | 문자열 | (담당자) 전처리 값 |

### varGrp, var

| 이름 | 부모 | 형식 | 출처 |
|---|---|---|---|
| `kosac:noneOption` | varGrp | 빈 요소 | |
| `kosac:noneOption/@yn` | `kosac:noneOption` | 목록: Y, N | (시스템→담당자) NONE_OPTION_YN |
| `@kosac:mrGroup` | var, varGrp | 문자열 | (시스템→담당자) MR_GROUP |
| `@kosac:seq` | var | 1 이상 정수 | (시스템) CSV 열 순서 |
| `@kosac:role` | var | 목록: ID, WEIGHT, ITEM, DERIVED, FILTER, LINK, SYSTEM, TEXT | (시스템→담당자) VAR_ROLE |
| `@kosac:measLevel` | var | 목록: nominal, ordinal, scale, text, date | MEAS_LEVEL |
| `@kosac:pii` | var | 목록: Y, N | PII |
| `@kosac:sensitive` | var | 목록: Y, N | SENSITIVE |
| `@kosac:reverse` | var | 목록: Y, N | REVERSE |
| `@kosac:mrType` | var | 목록: 보기별 열, 순위별 열 | MR_TYPE |
| `@kosac:rowLabel` | var | 문자열 | (시스템→담당자) GRID_ROW_LABEL |
| `@kosac:rankNo` | var | 1 이상 정수 | (시스템→담당자) MR_RANK_NO |
| `@kosac:cleansed` | var | 목록: Y, N | (시스템) 클린징 규칙이 적용된 변수 |
| `@kosac:textFormat` | var | 목록: TEXT, JSON_OBJ, JSON_ARR | (시스템) TEXT_FORMAT |
| `@kosac:parentVar` | var | NCName (변수 ID) | (시스템) PARENT_VAR |
| `kosac:fk` | var | 빈 요소 | (담당자) 공통표준코드 지정 |
| `kosac:fk/@codeSet` | `kosac:fk` | 문자열 | (담당자) 공통코드 집합 |
| `kosac:fk/@dimTable` | `kosac:fk` | 문자열 | (시스템) 차원 테이블 |
| `kosac:fk/@keyColumn` | `kosac:fk` | 문자열 | (시스템) 키 열 |
| `kosac:fk/@mapping` | `kosac:fk` | 목록: CODE, LABEL, PASSTHRU | (시스템→담당자) 매핑 방식 |
| `kosac:fk/@codeSystem` | `kosac:fk` | 문자열 | (담당자) 기관 코드 체계 |
| `kosac:fk/@refYear` | `kosac:fk` | 네 자리 연도 | (담당자) 기준 연도 |
| `kosac:fk/@linkKey` | `kosac:fk` | 목록: Y, N | (담당자) 결합 키 겸용 |
| `kosac:fk/@codeSetVersion` | `kosac:fk` | 문자열 | (시스템) 코드셋 버전 |
| `kosac:fk/@validatedAt` | `kosac:fk` | 문자열 | (시스템) 값 검증 일시 |
| `kosac:fk/@matchedRows` | `kosac:fk` | 0 이상 정수 | (시스템) |
| `kosac:fk/@unmatchedRows` | `kosac:fk` | 0 이상 정수 | (시스템) |
| `kosac:scale` | var | 빈 요소 | (시스템→담당자) 서열 변수 |
| `kosac:scale/@min` | `kosac:scale` | 십진수 | SCALE_MIN |
| `kosac:scale/@max` | `kosac:scale` | 십진수 | SCALE_MAX |
| `kosac:scale/@dir` | `kosac:scale` | 목록: POS, NEG | SCALE_DIR |
| `kosac:missingCodes` | var | 문자열(`;`로 이은 코드, 예: `-8;-7;-9`) | (시스템) 결측 코드 |
| `kosac:codeSetId` | var | 문자열 | (시스템) CODE_SET_ID |
| `kosac:analysis` | var | synonym ×n | (시스템→담당자) 분석 메타 |
| `kosac:analysis/@physicalName` | `kosac:analysis` | 문자열 | PHYSICAL_NAME |
| `kosac:analysis/@dimMeasure` | `kosac:analysis` | 목록: DIM, MEASURE, BOTH, ATTR, WEIGHT, KEY | DIM_MEASURE |
| `kosac:analysis/@aggRules` | `kosac:analysis` | 문자열(`;`로 이은 집계) | AGG_RULES |
| `kosac:analysis/@includeYn` | `kosac:analysis` | 목록: Y, N | INCLUDE_YN |
| `kosac:analysis/@logicalName` | `kosac:analysis` | 문자열 | LOGICAL_NAME |
| `kosac:analysis/@fkRef` | `kosac:analysis` | 문자열 | (시스템) `kosac:fk/@codeSet`과 같은 값 |
| `kosac:synonym` | `kosac:analysis` | 문자열 | SYNONYMS |

### catgry, otherMat

| 이름 | 부모 | 형식 | 출처 |
|---|---|---|---|
| `kosac:codeType` | catgry | 목록: VALID, MISSING, DK, NA | (시스템) CODE_TYPE |
| `kosac:stdCode` | catgry | 문자열(표준코드) | (담당자) STD_CODE |
| `kosac:stdCode/@set` | `kosac:stdCode` | 문자열 | (담당자) 공통코드 집합 |
| `kosac:pages` | otherMat | 쪽 범위 | QUESTIONNAIRE (보고서 수록 형식일 때) |
| `kosac:sha256` | otherMat | 64자리 16진수 | (시스템) |
| `kosac:bytes` | otherMat | 0 이상 정수 | (시스템) |

## 요청서 v1.2와 다른 점

| # | 요청서 | 엔진 | 이유 |
|---|---|---|---|
| 1 | `abstract@contentType="summary"` | `"abstract"` | XSD 목록은 abstract·purpose·mixed뿐이다 |
| 2 | `varGrp@type="rank"` | `type="other" otherType="rank"` | XSD 목록에 rank가 없다 |
| 3 | otherMat에 level 없음 | `level="study"` | XSD 필수 속성이다 |
| 4 | `<targetSampleSize>10000</…>` | `<targetSampleSize><sampleSize>10000</sampleSize></…>` | targetSampleSize는 요소만 담는다 |
| 5 | dataColl/respRate | anlyInfo/respRate | respRate의 부모는 anlyInfo이다 |
| 6 | fileTxt/notes | fileDscr/notes | fileTxt에는 notes가 없다 |
| 7 | 그리드가 qstn@ID 공유, Q_NO→qstn@ID | qstn에 ID 없음, Q_NO→qstn@seqNo | xs:ID는 문서에서 유일해야 하고 숫자로 시작할 수 없다 |
| 8 | 부록 D의 sumDscr: universe 다음 anlyUnit | anlyUnit 다음 universe | XSD 순서 |
| 9 | role=codebook 파일에 bytes·sha256 | name만 | 자기 해시는 자기 안에 넣을 수 없다 |
| 10 | kosac 요소를 DDI 순서 중간에 둠 | 같은 부모의 DDI 자식 뒤에 둠 | 규칙 하나. 공식 XSD는 어느 위치든 받지 않으므로 두 투영으로 검증한다 |
| 11 | EXCLUDE_RULE·EDIT_METHOD를 cleanOps 하나에 | cleanOps=EDIT_METHOD, `kosac:excludeRule` 추가 | 거꾸로 읽을 수 있게 |
| 12 | 확인 불가 값을 원래 자리에 둠(부록 D cleanOps) | 텍스트 자리는 원문을 둔다. 형식 있는 자리는 비우고 `kosac:unverified`만 남긴다 | 정수·날짜 자리에 문장이 들어가면 XSD를 통과하지 못한다 |
| 13 | 시스템 변수 8개(device·channel 포함) | 6개 | walla 새 export 기준. datajam이 device·channel을 빈 열로 채울 수 있다 |
| 14 | kosac:popSource 위치: sumDscr 옆(4-2 표) | `kosac:study` 아래 | 연구 단위 확장 값을 한곳에 모은다 |
| 15 | kosac:validN 요소(4-2 표) | `kosac:study/@validN` 속성 | 부록 D 모양을 따른다 |
| 16 | 원래 업로드 파일명 자리 없음 | `kosac:file/@sourceName`. otherMat는 labl에 셀 원문을 둔다 | 패키지 이름으로 바뀐 뒤에도 FILE_NAME·REPORT_FILE을 되살리도록 |
| 17 | unverified@key만 | `@file`(02 값), `@var`(03 값) 추가 | 파일·변수 단위 값을 되살리도록 |
| 18 | kosac:codeHistory | 이번 범위에서 뺌 | 원천인 게시본 이력이 export에 없다 |
| 19 | 같은 이름 kosac:file 두 가지(fileDscr 자식, package 자식) | 그대로 둔다. package/file은 지역 선언이다 | 4-2 표를 따른다. 재단의 `//kosac:file` XPath는 둘을 함께 잡으므로 부모로 구분해야 한다 |

- 11행 보충: cleanOps는 여러 번 나올 수 있어서 두 개를 두어도 XSD는 통과한다. 그래도 나눈 이유는 거꾸로 읽을 수 있게 하기 위해서다. XSD가 강제한 결정은 아니다.
- 12행 보충: DDI XSD에서 실패하는 형식 자리는 sampleSize 같은 정수 자리이다. collDate@date는 xs:string이라 DDI XSD를 통과한다. 그래도 날짜 자리를 비우는 것은 kosac 형식과 ETL 기준에 맞추기 위해서다.

### 19행: 재단이 정할 제안

package 쪽 `kosac:file`을 `kosac:artifact`로 바꾸면 이름 겹침이 없어진다. 그러면 `//kosac:file`은 데이터 파일(fileDscr 자식)만 잡고, package 쪽도 전역으로 선언할 수 있다. 다만 이것은 요청서 4장의 구조를 바꾸는 일이다. 그래서 재단과 협의해 정하고, 엔진이 먼저 바꾸지는 않는다.

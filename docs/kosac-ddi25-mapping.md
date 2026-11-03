# 재단 v4.0 → DDI Codebook 2.5 매핑

이 문서는 재단 2026-10-19 개정 빈 양식의 105개 업무 항목과 `_meta` 3개 항목을 다룬다.
실제 출력 구현은 `kosac/standard.go`다. 출력은 표준 DDI 요소와 속성만 사용한다.
기준 스키마는 [DDI Alliance Codebook 2.5 XSD](https://ddialliance.org/Specification/DDI-Codebook/2.5/XMLSchema/codebook.xsd)이며 저장소의 같은 XSD로 전체 문서를 검사한다.

## 출력 계약

- 구조가 대응하는 값은 표준 요소·속성으로 기록한다. `MEAS_LEVEL`은 `nature`, `DDI_INTERVAL`은 별도로 `intrvl`에 대응한다.
- 숫자형 코드와 문자형 코드를 구분한다. `01`을 `1`로 바꾸지 않는다. 물리 저장 형식이나 열 너비는 추정하지 않는다.
- v4.0 `IS_MISSING`이 예·아니오(Y·N)로 확정되어야 내보낸다. 미확정은 오류이며 `missing=N`으로 추정하지 않는다.
- `MISSING_REASON=해당 없음`은 결측 사유다. 비적용 표시로 버리지 않는다.
- `DDI_INTERVAL`이 비적용·미확인이면 속성을 생략하고 원문을 설명에 남긴다. 표준 XSD의 생략 기본값은 `discrete`이므로 수신 시스템은 원문도 확인해야 한다.
- 관리 상태·보안 정책·표준코드 연결 결과는 설명이다. 전용 XML 확장이나 JSON 역변환 프로토콜을 정의하지 않는다.
- `05_재코딩이력` 행마다 `dataProcessing`과 `codingInstructions/txt`를 연결한다. 실행 언어·코드가 없으므로 `command`를 만들지 않는다.
- 정제·재코딩 기록은 제출자의 설명이다. 실제 응답을 처리하거나 결과를 검증했다는 뜻이 아니다.
- JSON은 `ddi-xlsx`의 XML 트리 표현이다. 별도의 DDI 표준 JSON 스키마로 주장하지 않는다.

## 이전 형식과 호환성

`Codebook.Tree/XML/JSON`의 기본 출력은 v3.2·v4.0 모두 표준 전용이다.
과거 출력의 `kosac:` 요소와 v4.0 보존용 JSON 노트는 생성하지 않는다.
`kosac.Parse`는 과거 확장 형식 읽기용으로 남긴다. 새 출력은 `ddi.Parse`로 읽는다.
새 출력에서 원본 재단 양식의 모든 항목을 구조적으로 복원할 수는 없다.
원본 XLSX와 재단 모델을 저장하고 `template.Read/Write`로 양식 편집을 이어간다.
범용 `workbook.ToXLSX/FromXLSX`는 표준 문서를 별도의 DDI 편집 워크북으로 왕복한다.

외부 Excel에는 Walla 시스템 변수를 생성하지 않는다.
Walla 경로는 입력 DDI에 존재하는 시스템 변수만 별도 모델 항목으로 보존한다.
코드북 자신의 최종 해시와 크기는 자기 문서에 넣지 않는다. 외부 manifest에서 관리한다.

## 항목별 구현

표의 경로는 `codeBook` 기준이다. `notes`와 `txt`의 값은 사람이 읽는 설명이며 역변환 키가 아니다.
조건부 구조는 실제 입력에서 해석 가능한 부분만 구조화한다. 대상이 미확정이면 같은 범위의 설명으로 남긴다.

### 01_조사정보

| 원본 KEY | 출력 경로 | 해석 |
|---|---|---|
| `RULE_VERSION` | `docDscr/notes` | 가이드라인 버전이다. codeBook/@version의 DDI 버전과 구분해 설명으로 보존한다. |
| `TITLE` | `stdyDscr/citation/titlStmt/titl` | 공식 조사명이다. |
| `PROJECT_TITLE` | `stdyDscr/citation/notes` | 계약상 과제 정보다. 조사 대체 제목이나 조사 식별자와 동일하다고 가정하지 않는다. |
| `PROJECT_NO` | `stdyDscr/citation/notes` | 계약상 과제 정보다. 조사 대체 제목이나 조사 식별자와 동일하다고 가정하지 않는다. |
| `PURPOSE` | `stdyDscr/stdyInfo/abstract` | 목적과 조사 내용을 별도 abstract로 보존한다. 분류값은 적용 프로파일에서 정한다. |
| `SUMMARY` | `stdyDscr/stdyInfo/abstract` | 목적과 조사 내용을 별도 abstract로 보존한다. 분류값은 적용 프로파일에서 정한다. |
| `FUNDER` | `stdyDscr/citation/prodStmt/fundAg` | 발주·지원 기관을 기록한다. |
| `PI_ORG` | `stdyDscr/citation/rspStmt/AuthEnty/@affiliation` | 책임자 성명과 소속을 연결한다. 성명 미확인 시 기관만 기재하고 미확인 사유를 설명한다. |
| `FIELD_ORG` | `stdyDscr/method/dataColl/dataCollector` | 실사 기관을 기록한다. 자료 발행 주체 producer와 구분한다. |
| `PI_NAME` | `stdyDscr/citation/rspStmt/AuthEnty` | 성명은 본문, 소속은 affiliation이다. 성명 미확인 사유는 citation/notes에 남긴다. |
| `FIELD_START` | `stdyDscr/stdyInfo/sumDscr/collDate/@date` | 시작·종료를 event 속성으로 구분한다. |
| `FIELD_END` | `stdyDscr/stdyInfo/sumDscr/collDate/@date` | 시작·종료를 event 속성으로 구분한다. |
| `KEYWORDS` | `stdyDscr/stdyInfo/subject/keyword` | 세미콜론으로 구분한 키워드를 반복한다. 개수 제한은 재단 규칙이다. |
| `QUESTIONNAIRE` | `otherMat` | 실제 자료의 파일명·URI와 설명을 구분한다. 복수 설문지·쪽수는 임의로 한 위치로 축약하지 않는다. |
| `REPORT_FILE` | `otherMat` | 실제 자료의 파일명·URI와 설명을 구분한다. 복수 설문지·쪽수는 임의로 한 위치로 축약하지 않는다. |
| `EXTRA_FILES` | `otherMat` | 실제 자료의 파일명·URI와 설명을 구분한다. 복수 설문지·쪽수는 임의로 한 위치로 축약하지 않는다. |
| `UNIVERSE` | `stdyDscr/stdyInfo/sumDscr/universe` | 연구 전체의 조사 대상이다. 변수별 응답 대상과 구분한다. |
| `POP_SOURCE` | `stdyDscr/method/dataColl/sampleFrame/txt` | 모집단 기준 자료와 시점을 표본틀 설명으로 보존한다. 날짜를 임의 추출하지 않는다. |
| `ANALYSIS_UNIT` | `stdyDscr/stdyInfo/sumDscr/anlyUnit` | 조사 전체 분석 단위다. |
| `SAMPLING_TYPE` | `stdyDscr/method/dataColl/sampProc` | 표본설계를 설명한다. 층화·집락 변수는 Link로 연결할 수 있지만 전용 층화·집락 속성은 아니다. |
| `SAMPLING_DESC` | `stdyDscr/method/dataColl/sampProc` | 표본설계를 설명한다. 층화·집락 변수는 Link로 연결할 수 있지만 전용 층화·집락 속성은 아니다. |
| `SAMPLE_FRAME` | `stdyDscr/method/dataColl/sampleFrame/sampleFrameName` | 표본틀 이름을 기록한다. |
| `STRATA_VARS` | `stdyDscr/method/dataColl/sampProc` | 표본설계를 설명한다. 층화·집락 변수는 Link로 연결할 수 있지만 전용 층화·집락 속성은 아니다. |
| `CLUSTER_VAR` | `stdyDscr/method/dataColl/sampProc` | 표본설계를 설명한다. 층화·집락 변수는 Link로 연결할 수 있지만 전용 층화·집락 속성은 아니다. |
| `TARGET_N` | `stdyDscr/method/dataColl/targetSampleSize/sampleSize` | 목표 수와 실제 응답 수를 구분한다. 숫자 슬롯에는 숫자만 쓴다. |
| `VALID_N` | `stdyDscr/notes` | 조사 전체 유효 수와 중복 설명이다. 파일별 caseQnty와 동일하다고 간주하지 않는다. |
| `SPLIT_BALLOT` | `stdyDscr/method/dataColl/resInstru` | 분할 여부와 배정 변수를 설문 도구 설명에 기록한다. 실행 분기 규칙은 아니다. |
| `SPLIT_VAR` | `stdyDscr/method/dataColl/resInstru` | 분할 여부와 배정 변수를 설문 도구 설명에 기록한다. 실행 분기 규칙은 아니다. |
| `MODE` | `stdyDscr/method/dataColl/collMode` | 조사 수집 방법을 기록한다. 선택 목록은 재단 규칙이다. |
| `TOOL` | `stdyDscr/method/notes` | 빈 양식 정의는 온라인 플랫폼 이름이다. resInstru는 조사 도구 유형, ConOps는 자료 통제 작업이므로 이름만 보고 치환하지 않는다. |
| `CONTACT_N` | `stdyDscr/method/notes` | 접촉 수와 응답률 산식의 직접 전용 숫자 슬롯을 가정하지 않는다. |
| `RESPONSE_RATE` | `stdyDscr/method/anlyInfo/respRate` | 응답률의 설명이다. 원본의 미확인 사유를 실제 비율로 바꾸지 않는다. |
| `RR_FORMULA` | `stdyDscr/method/notes` | 접촉 수와 응답률 산식의 직접 전용 숫자 슬롯을 가정하지 않는다. |
| `EXCLUDE_N` | `stdyDscr/method/dataProcessing` | 사례 제외 수와 기준을 처리 설명으로 보존한다. 확정되지 않은 수를 0으로 바꾸지 않는다. |
| `EXCLUDE_RULE` | `stdyDscr/method/dataProcessing` | 사례 제외 수와 기준을 처리 설명으로 보존한다. 확정되지 않은 수를 0으로 바꾸지 않는다. |
| `EDIT_METHOD` | `stdyDscr/method/dataColl/cleanOps` | 오류 점검과 정제 이력을 기록한다. 원문 주장과 실행 검증은 구분한다. |
| `WEIGHT_YN` | `stdyDscr/method/dataColl/weight; dataDscr/var/@wgt` | 가중치 설명과 가중치 변수 표식을 쓴다. 적용 대상은 파일별로 확인해야 한다. |
| `WEIGHT_VAR` | `stdyDscr/method/dataColl/weight; dataDscr/var/@wgt` | 가중치 설명과 가중치 변수 표식을 쓴다. 적용 대상은 파일별로 확인해야 한다. |
| `WEIGHT_METHOD` | `stdyDscr/method/dataColl/weight; dataDscr/var/@wgt` | 가중치 설명과 가중치 변수 표식을 쓴다. 적용 대상은 파일별로 확인해야 한다. |
| `SCHOOL_CODE_YEAR` | `stdyDscr/notes` | 대상 변수와 기준연도·코드 버전의 범위를 확정하기 전에는 stdCatgry/@date로 승격하지 않는다. |
| `SURVEY_TYPE` | `stdyDscr/notes` | 재단의 조사 종류와 연결 사업이다. 표준 전용 속성을 만들지 않고 설명으로 보존한다. |
| `DATA_NO` | `stdyDscr/citation/titlStmt/IDNo` | 재단 데이터 인벤토리 번호이며 XML 내부 ID와 다르다. |
| `BIZ_NAME` | `stdyDscr/notes` | 재단의 조사 종류와 연결 사업이다. 표준 전용 속성을 만들지 않고 설명으로 보존한다. |
| `BIZ_YEAR` | `stdyDscr/notes` | 재단의 조사 종류와 연결 사업이다. 표준 전용 속성을 만들지 않고 설명으로 보존한다. |
| `SERIES_NAME` | `stdyDscr/citation/serStmt/serName` | 반복조사 시리즈명을 기록한다. |
| `STUDY_ROUND` | `stdyDscr/stdyInfo/sumDscr/collDate/@cycle` | 조사 차수를 수집일과 연결한다. 수집일이 없으면 notes로 보존한다. |
| `ACCESS_RESTRICTIONS` | `stdyDscr/dataAccs/useStmt/restrctn` | 이용 제한과 접근 조건의 설명이다. 서비스의 인가 집행을 대신하지 않는다. |
| `CLEANING_LOG` | `stdyDscr/method/dataColl/cleanOps` | 오류 점검과 정제 이력을 기록한다. 원문 주장과 실행 검증은 구분한다. |
| `SCALE_EXCEPTION` | `stdyDscr/notes` | 재단 만족도 정책의 적용 예외다. DDI가 모든 조사를 5점 척도로 강제하지 않는다. |
| `SOURCE_FILES` | `otherMat/txt` | 원본 파일 목록과 버전·보관 정보를 보존한다. 실제 파일·해시는 별도 확인한다. |

### 02_데이터파일

| 원본 KEY | 출력 경로 | 해석 |
|---|---|---|
| `FILE_NO` | `fileDscr/@ID; dataDscr/var/@files` | F+파일번호로 내부 ID를 만들고 변수에서 참조한다. 원본 번호도 보존한다. |
| `FILE_NAME` | `fileDscr/fileTxt/fileName` | 02는 제출 파일을 기술한다. 실제 파일이 존재한다고 보증하지 않는다. |
| `FILE_FORMAT` | `fileDscr/fileTxt/fileType` | SAV·CSV·XLSX 등 물리 형식이다. 최종 Parquet 패키지 정책과 구분한다. |
| `CONTENT` | `fileDscr/fileTxt/fileCont` | 파일 내용의 설명이다. |
| `UNIT` | `fileDscr/notes` | 파일별 단위다. 조사 전체 단위와 같다는 가정 없이 보존한다. |
| `ID_VAR` | `fileDscr/notes` | 원본 식별변수 설명이다. recGrp/@recidvar는 실제 계층·관계 구조가 확인된 경우의 후속 후보다. |
| `LINK_VAR` | `fileDscr/notes` | 연결 변수 설명을 보존한다. 제출 양식만으로 관계형 레코드 구조를 가정하지 않는다. |
| `LINK_FILE_NO` | `fileDscr/notes` | 연결 파일 번호를 보존한다. 대상 파일·변수는 모델 조립 시 확인한다. |
| `NOTE` | `fileDscr/notes` | 파일별 비고·조사 시점 원문을 보존한다. 임의로 날짜를 확정하지 않는다. |
| `VALID_N` | `fileDscr/fileTxt/dimensns/caseQnty` | 제출자가 적은 행 수다. 정수로 해석할 수 없는 값은 notes에 남긴다. 실제 파일 교차 검사는 별도다. |
| `SURVEY_TIME` | `fileDscr/notes` | 파일별 비고·조사 시점 원문을 보존한다. 임의로 날짜를 확정하지 않는다. |
| `TIME_VAR` | `fileDscr/notes` | 시점 변수 이름을 설명으로 보존한다. Link/@refs는 생성하지 않는다. |
| `UNIQUE_KEY` | `fileDscr/notes` | 고유 키의 제출자 기록이다. 실제 응답 중복을 확인했다는 의미가 아니다. |
| `VARIABLE_N` | `fileDscr/fileTxt/dimensns/varQnty` | 유효한 정수 선언을 사용한다. 비어 있거나 정수가 아니면 모델에 선언된 변수 수를 쓴다. 원문은 notes에 보존한다. |

### 03_변수정의

| 원본 KEY | 출력 경로 | 해석 |
|---|---|---|
| `FILE_NO` | `dataDscr/var/@files; location/@fileid` | 파일 안에서 변수명이 유일해야 한다. 파일 간 같은 이름은 내부 ID로 구분한다. |
| `VAR_NAME` | `dataDscr/var/@name` | 물리 변수명이다. ID와 분리하고 숫자처럼 변환하지 않는다. |
| `VAR_LABEL` | `dataDscr/var/labl` | 변수의 설명 라벨이다. |
| `Q_NO` | `dataDscr/var/qstn/@seqNo` | 문항 번호다. |
| `Q_TEXT` | `dataDscr/var/qstn/qstnLit` | 질문 원문이다. |
| `VAR_GROUP` | `dataDscr/varGrp/@var; varGrp/labl` | 그룹을 만들고 변수 ID를 연결한다. 20자와 콜론 형식은 재단 규칙이다. |
| `MEAS_LEVEL` | `dataDscr/var/@nature` | 명목→nominal, 서열→ordinal, 등간→interval, 비율→ratio. intrvl과 구분한다. |
| `RESP_UNIVERSE` | `dataDscr/var/universe` | 자연어·조건식 모두 응답 대상 설명으로 기록할 수 있다. 자동 평가식으로 실행되지는 않는다. |
| `MR_TYPE` | `dataDscr/varGrp/@type; var/notes` | DeriveDefaults에서 확인한 다중응답·순위 묶음을 varGrp로 기록한다. 입력 저장 방식은 변수 설명에도 남긴다. |
| `REVERSE` | `dataDscr/var/codInstr` | 반대 방향 여부는 코더 설명이다. 실제 역채점 변환과 그 원본·산식을 별도로 기록한다. |
| `DERIVATION` | `dataDscr/var/derivation/drvdesc` | 산출 설명을 보존한다. 실행 언어가 없는 설명을 drvcmd로 위장하지 않는다. |
| `STD_ID` | `dataDscr/var/concept` | 표준 데이터 요소의 의미 식별자로 참조한다. 코드북 변수 ID나 표준 코드값과 혼동하지 않는다. 사전 URI는 미제공이다. |
| `PII` | `dataDscr/var/security` | 준식별자·민감정보·접근 수준을 보안 설명으로 보존한다. 재단 Y/N 플래그가 DDI 전용 속성인 것은 아니다. |
| `SENSITIVE` | `dataDscr/var/security` | 준식별자·민감정보·접근 수준을 보안 설명으로 보존한다. 재단 Y/N 플래그가 DDI 전용 속성인 것은 아니다. |
| `NOTE` | `dataDscr/var/notes` | 변수 비고다. |
| `DATA_TYPE` | `dataDscr/var/varFormat/@type; var/@representationType` | 숫자→numeric, 문자→character/text. 날짜·일시는 datetime과 varFormat으로 설명한다. 실제 저장 형식은 별도 확인한다. |
| `DDI_INTERVAL` | `dataDscr/var/@intrvl` | discrete·contin만 쓴다. 해당 없음·확인 불가는 원문 설명이 필요하다. 속성 생략 시 XSD 기본값 discrete가 적용되는 한계가 있다. |
| `MISSING_STATUS_VAR` | `dataDscr/var/notes` | 결측 상태 변수명을 설명으로 보존한다. DDI 전용 결측상태 속성은 정의하지 않는다. |
| `ACCESS_LEVEL` | `dataDscr/var/security` | 준식별자·민감정보·접근 수준을 보안 설명으로 보존한다. 재단 Y/N 플래그가 DDI 전용 속성인 것은 아니다. |
| `CODE_SYSTEM` | `dataDscr/var/stdCatgry` | 외부 분류 체계·기준연도를 설명한다. 확인된 코드 목록 URI가 있으면 URI를 붙일 수 있다. |
| `SOURCE_VAR` | `dataDscr/var/derivation/drvdesc 또는 var/notes` | 원본 파일 변수명을 기록한다. 같은 문서의 ID와 연결할 근거가 없어 derivation/@var를 추정하지 않는다. |
| `RECODE_STATUS` | `dataDscr/var/notes` | 검토·처리 상태다. 재코딩을 실제 수행했다는 보증이 아니다. |

### 04_코드값

| 원본 KEY | 출력 경로 | 해석 |
|---|---|---|
| `FILE_NO` | `dataDscr/var/catgry` | 파일번호·변수명 조합으로 범주를 연결한다. |
| `VAR_NAME` | `dataDscr/var/catgry` | 파일번호·변수명 조합으로 범주를 연결한다. |
| `CODE` | `dataDscr/var/catgry/catValu` | 문자열로 보존한다. 01과 1을 임의로 합치지 않는다. |
| `CODE_LABEL` | `dataDscr/var/catgry/labl` | 코드 의미를 보존한다. |
| `NOTE` | `dataDscr/var/catgry/txt` | 범주의 비고다. |
| `IS_MISSING` | `dataDscr/var/catgry/@missing` | 예→Y, 아니오→N. 확인 불가는 최종 확정이 필요하다. 속성 생략도 XSD 기본값 N이므로 미확인 상태를 대신하지 못한다. |
| `MISSING_REASON` | `dataDscr/var/catgry/@missType` | 결측 유형을 기술한다. -9·-8·-7 숫자 체계는 재단 프로파일이며 DDI가 강제하지 않는다. |
| `RECODE_STATUS` | `dataDscr/var/catgry/txt` | 코드값 검토 상태를 설명한다. |

### _meta

| 원본 KEY | 출력 경로 | 해석 |
|---|---|---|
| `TEMPLATE_ID` | `docDscr/notes` | 입력 Excel 양식 식별자·버전·작성일을 설명한다. DDI 문서 버전·생성일과 혼동하지 않는다. |
| `TEMPLATE_VERSION` | `docDscr/notes` | 입력 Excel 양식 식별자·버전·작성일을 설명한다. DDI 문서 버전·생성일과 혼동하지 않는다. |
| `CREATED` | `docDscr/notes` | 입력 Excel 양식 식별자·버전·작성일을 설명한다. DDI 문서 버전·생성일과 혼동하지 않는다. |

### 05_재코딩이력

| 원본 KEY | 출력 경로 | 해석 |
|---|---|---|
| `FILE_NO` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |
| `SOURCE_FILE` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |
| `SOURCE_VAR` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |
| `SOURCE_CODE_OR_CONDITION` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |
| `TARGET_VAR` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |
| `TARGET_CODE_OR_RULE` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |
| `TRANSFORM_TYPE` | `stdyDscr/method/codingInstructions/@type` | type은 자유 문자열이다. 재단의 승인 목록은 별도 운영 규칙이다. |
| `BASIS` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |
| `RECORD_COUNT` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |
| `STATUS` | `stdyDscr/method/codingInstructions/txt` | 미착수부터 검증 완료까지의 업무 상태다. DDI 표준 상태 열거형은 아니다. |
| `NOTE` | `stdyDscr/method/codingInstructions/txt` | 원문에 있는 변환 조건·대상·근거·영향 수를 처리 설명으로 보존한다. 실행 언어·명령은 없으므로 command는 만들지 않는다. |

## 검증과 한계

회귀 테스트는 실제 2024·2025 작성 예제를 읽고 전체 XSD, 변수·범주·재코딩 행 수,
XML·JSON 일치, 원본 레코드의 XLSX 왕복을 확인한다.
범용 DDI 편집 워크북 왕복은 속성 순서를 제외한 동일 트리와 전체 XSD를 확인한다.
내용 규칙 결과와 XSD 결과는 별개다. XSD 통과는 재단 작성 내용의 승인이나 표준코드 실재성 검증이 아니다.

NEIS·공통코드 조회, 실제 응답의 중복·결측·재코딩 실행, PDF 파일 내용,
Datajam 저장·승인, 플랫폼 ETL과 Wide·Long Parquet 계약은 이 모듈에서 검증하지 않는다.

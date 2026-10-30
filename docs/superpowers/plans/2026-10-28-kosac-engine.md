# KOSAC Codebook Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a KOSAC codebook engine to ddi-xlsx. It reads and writes the foundation's KOSAC-SURVEY-META v3.2 workbook and checks it (S05, C01–C18). It derives the analysis meta and assembles the DDI 2.5 + `urn:kosac:survey-meta:3.3` codebook XML. It reads that XML back at a fixed point. It prepares two-projection XSD validation plus Go placement and reference checks. datajam imports it as a Go module.

**Architecture:** `kosac` holds a sheet-shaped model:
- template values are `Record`s keyed by template KEY;
- engine values are typed fields.

Assembly builds a `*ddi.Node` tree in XSD order. Writing is left to `ddi.WriteXML`/`WriteJSON`, so the layout and fixed-point guarantees are inherited. `kosac.Parse` is the exact inverse for engine-written XML. The other packages:
- `kosac/template`: v3.2 xlsx I/O, using an embedded temporary blank.
- `kosac/check`: the C rules, plus `ValidateXML` (PLACE and REF in Go, and two XSD projections through a caller-injected validator).
- `kosac/universe`: the RESP_UNIVERSE / cleansing-condition parser and evaluator.

**Tech Stack:** Go 1.25, `github.com/xuri/excelize/v2` v2.11.0 (still the only dependency), the existing `ddi` package, and `xmllint` in tests.

**Spec:** `docs/superpowers/specs/2026-10-28-kosac-engine-design.md` (Korean; § numbers below refer to it). The generic layer's spec is `docs/superpowers/specs/2026-10-26-ddi-xlsx-design.md`. The builder check that shaped the spec is in the notes folder (`ddi-xlsx-kosac-builder-check.md`); the spec already contains its resolutions.

## Global Constraints

- Repository: `~/Paprika/dev/ddi-xlsx`. Module path: `github.com/teamPaprika/ddi-xlsx`.
- **Do not commit, push or add a remote** (user rule). Skip all commit steps. Report `git status --short` at the end of each task.
- Only one non-stdlib Go dependency: `github.com/xuri/excelize/v2`.
- Dependency direction: `kosac/...` may import `ddi` and `xsd`. `ddi`, `workbook` and `internal/schema` must not import `kosac`. Do not modify `ddi/` or `workbook/` (D11).
- The engine does no file, network or process I/O. The only exception is `cmd/ddi-xlsx`, which may run `xmllint`.
- `Issue.Message` is one Korean sentence. `Issue.Rule` is one of `S05`, `C01`…`C18`, `XSD-DDI`, `XSD-KOSAC`, `PLACE`, `REF`. `Issue.Level` is `R` or `W`.
- Template KEYs, sheet names and dropdown values are exact (§5.1, §10.2). Sheet names: `00_작성안내`, `01_조사정보`, `02_데이터파일`, `03_변수정의`, `04_코드값`, `_meta`. Rule standards (구분, lists) are engine constants, never read from the uploaded file (§10.1).
- The kosac namespace URI is `urn:kosac:survey-meta:3.3` and the prefix is always `kosac`.
- **Defaults for the open questions (§17) live in one place each**, so a foundation answer changes one constant:
  - Q1/Q7 codeType;
  - Q2 missing labels (`kosac.DefaultMissingLabels`);
  - Q3 reserved words.
- Code comments explain only WHY. No ticket markers, task numbers or doc paths in code.
- Never open or copy any `survey.csv`.
- Fixtures:
  - `testdata/kosac/v32_sample_2024.xlsx` is a copy of `/private/tmp/claude-501/-Users-kimuyb-Paprika-dev-walla-next--claude-worktrees-pr-1348-review-b1e3de/cce1d6dd-4e2c-4de9-87bb-db7d9a69afe6/scratchpad/kosac/v32_sample.xlsx`. Task 2 copies it.
  - `testdata/walla_synthetic/` (new walla export, already copied).
  - `testdata/kofac_*` (old walla export).

## Review Focus

1. **A contractor types dates and numbers as real Excel values.** Examples: a date cell shown as `2024/10/04`, and a number shown as `10,000`.
   - Date cells must read as `2024-10-04`.
   - C02 must accept `10,000`.
   - The assembled `sampleSize` must be `10000`, so the XSD passes.
   - The tests are in Task 2, Task 3 and Task 4.
2. **Headers and rows a person edits.** Examples: a KEY header with a trailing space, blank rows between data rows, and a row with only NOTE filled.
   - Headers are trimmed before matching.
   - All-blank rows are skipped.
   - A row with only NOTE is a row, so C06 reports its empty required cells.
   - The test is in Task 2.
3. **The same VAR_NAME in two files.** A linkage variable in F1 and F2.
   - The vars get two distinct IDs.
   - 02-sheet references resolve within their own file.
   - 01-sheet references resolve to the lowest FILE_NO.
   - NoneOption and group keys do not collide across files.
   - AssignIDs keeps both IDs across versions.
   - The tests are in Task 4 and Task 7.
4. **"확인 불가: …" in any template cell.** This includes 01 optional fields (TOOL), 02 integer fields and 03 RESP_UNIVERSE.
   - It gives a W under the rule that owns the cell, and no format R.
   - The typed slot is omitted, the text slot keeps the text, and `kosac:unverified` records it with `@file`/`@var`.
   - `Parse` restores `확인 불가: {사유}`.
   - The tests are in Task 3, Task 4 and Task 5.
5. **Large codebooks.**
   - Test size: 3,500 vars and 30,000 codes. This is beyond the template's dropdown rows (02: 23, 03: 3003).
   - Read, Check, DeriveDefaults, AssignIDs and XML must finish in under 10s.
   - Write must not fail past the dropdown range.
   - `TestLargeCodebook` is in Task 7. It is skipped under `-short`.

---

### Task 1: `kosac/universe` — condition parser and evaluator (§11)

**Files:**
- Create: `kosac/universe/universe.go`
- Test: `kosac/universe/universe_test.go`

**Interfaces:**
- Produces:
  ```go
  type Row interface{ Value(name string) (string, bool) }
  type Expr struct{ /* unexported */ }
  func Parse(s string) (*Expr, error)
  func ParseCondition(s, target string) (*Expr, error) // also accepts a bare "pattern:{regex}" about target
  func (e *Expr) Vars() []string                       // referenced names, first-appearance order, unique
  func (e *Expr) Eval(r Row) (bool, error)
  ```
- Errors are Korean messages and include the 1-based rune position.

- [ ] **Step 1: Write the failing tests.**
  - `TestParseSample`: every string in this list parses.
    - Values: `SQ2_1=2`, `SQ3_1=2`, `QA5=1,2`, `QA5=4,5`, `QA28=6`, `SQ3_1=1,2`, `QA37_1=1~10`, `C1=2~4`, `SQ2_1=3~7`, `DQ4a=2~5`, `DQ4a=3~5`, `DQ4a=4~5`, `SQ3!=1 AND SQ3_1!=-8`, `SQ2_1_3=B100203,B100217,B100231`, `Random_Ballot=1`, `Q5 = 1, 2`.
    - `Vars()` of the AND example is `["SQ3","SQ3_1"]`.
  - `TestParseErrors`: each of these fails. `""`, `Q5=`, `Q5=1,`, `=1`, `(Q5=1)`, `Q5=a~b`, `Q5=1 and Q6=2` (lowercase keyword), `Q5=1 AND`, `Q5=pattern:[` (bad regex), `1Q=1`.
  - `TestEval` (table):
    - `Q5=1,2` with Q5=`2` is true, and with `3` is false.
    - `SQ2=14~18` with `14`, `18` and `13.5` gives true, true, false.
    - `Q5!=1` with `""` is true, and `Q5=1` with `""` is false.
    - `A=1 OR B=2 AND C=3` follows AND-before-OR: with A=`1`, B=`0`, C=`0` it is true.
    - `X=B100203` compares as a string.
    - `Q5=01` with Q5=`1` is true (numeric).
    - A var missing from the row is an error.
  - `TestParseConditionPattern`:
    - `ParseCondition("pattern:^[0-9]{7}$","SCH")` gives true for SCH=`7230115` and false for `B100203`.
    - `Parse("pattern:x")` is an error.
    - §11 boundary: `ParseCondition("pattern:^a b,c$ AND X=1","T")` has the regex `^a b,c$` and `Vars()` is `["T","X"]`.
    - `ParseCondition("pattern:a AND b","T")` is an error whose message says that AND/OR inside a pattern is not supported.
- [ ] **Step 2:** Run `go test ./kosac/universe/`. Expected: FAIL (package missing).
- [ ] **Step 3: Implement.** Use a hand-written tokenizer over runes and the §11 grammar. Compare numbers with `strconv.ParseFloat`. Use `regexp` (RE2) for patterns.
- [ ] **Step 4:** Run `go test ./kosac/universe/`. Expected: PASS.

### Task 2: Model and `kosac/template` read/write with the temporary blank (§4, §5)

**Files:**
- Create: `kosac/model.go`, `kosac/issue.go`, `kosac/refdata.go`
- Create: `kosac/template/read.go`, `kosac/template/write.go`, `kosac/template/blank.go` (embed), `kosac/template/blank_v3.2.xlsx`, `kosac/template/unzip.go` (the same limit as workbook's private helper; §5.3)
- Create: `testdata/kosac/v32_sample_2024.xlsx` (copy, see Global Constraints)
- Create: `testdata/kosac/README.md`, in Korean. It states:
  - the sample is the foundation's example codebook, a customer document and not response data;
  - it is kept only on the premise that the repository is private;
  - if the repository is made public, this file, every golden built from it (`v32_sample_2024.*`) and `kosac/template/blank_v3.2.xlsx` must be removed or replaced first.
- Test: `kosac/template/template_test.go`, `kosac/model_test.go`

**Interfaces:**
- Produces in `kosac`:
  - Types:
    - `Loc`, `Field`, `Record` (map), `Level` (`R`, `W` consts), `Issue` (§9, all fields string except Level)
    - `TemplateMeta{ID, Version, Created Field}`, `StudyAdmin` (§4, incl. `WallaID`)
    - `File`, `Var`, `Analysis` (§4 list, incl. `NoneOption`), `Scale{Min, Max, Dir string}`, `FK` (request 4-2 kosac:fk attributes), `Code`
    - `Cleansing`, `CleansingRule`, `MapEntry{From, To string}`
    - `Package`, `PackageFile`, `Validation`, `Suggestion` (§4 verbatim), `Codebook`
    - Every scalar is `string` except `Suggestion.Confidence`, so values pass through as written.
  - `func (r Record) Get(key string) string` and `func (r Record) Set(key, value string)`. Set keeps the Loc.
  - `func (cb *Codebook) Var(fileNo, name string) *Var`, `func (cb *Codebook) File(fileNo string) *File`, `func (cb *Codebook) CodesOf(v *Var) []*Code`
  - `func Unverified(value string) (reason string, ok bool)`. It uses the regex `^확인\s*불가\s*[:：]\s*(.*)$`, applied after trimming.
  - `var StudyKeys, FileKeys, VarKeys, CodeKeys []string`, in template order (44, 10, 15 and 5 keys).
  - `var StudyRequired map[string]string`. It maps KEY to `필수`, `선택`, `조건부` or `재단`.
  - RefData types: `RefData`, `StdItem`, `CodeSetDef`, `StdCode` (§9 and request appendix C columns).
- Produces in `kosac/template`:
  ```go
  type ReadOptions struct{ AllowedVersions []string } // nil → []string{"3.2"}
  func Read(xlsx []byte, opt ReadOptions) (*kosac.Codebook, []kosac.Issue, error)
  func Write(cb *kosac.Codebook) ([]byte, error)
  func Blank() []byte
  ```

- [ ] **Step 1: Write the failing tests.**
  - `TestReadSample`: Read `v32_sample_2024.xlsx` and check:
    - 1 file, 214 vars and 828 codes;
    - `Study.Get("VALID_N")` is `"10000"`;
    - `Study["POP_SOURCE"].Loc` is `{01_조사정보, E19}`;
    - `Vars[0].Rec["VAR_NAME"]` is `IDX` at `{03_변수정의, B4}`;
    - there are no Issues (TEMPLATE_ID is KOFAC);
    - `Template.ID.Value` is `KOFAC-SURVEY-META`.
  - `TestReadS05`. Each case is built by editing the sample with excelize.
    - A missing `04_코드값` sheet gives R and a nil model.
    - An extra KEY header `FOO` on 03 gives R.
    - A missing 01 KEY row (`TOOL` deleted) gives R.
    - TEMPLATE_ID `KOSAC-SURVEY-META` gives one W and a non-nil model.
    - TEMPLATE_ID `X` gives R.
    - TEMPLATE_VERSION `3.1` gives R, and with `AllowedVersions: {"3.1"}` it gives no issue.
  - `TestReadEditedCells` (Review Focus 1–2):
    - E12 set by `SetCellValue(time.Date(2024,10,4,…))` with a `yyyy/mm/dd` style reads `2024-10-04`.
    - Header `VAR_NAME ` (trailing space) is found.
    - Two blank rows inserted between data rows are skipped.
    - A row with only NOTE is kept.
  - `TestWriteRoundTrip`: `Read(Write(Read(sample)))` has the same Record values for 01, 02, 03 and 04, and the same counts.
  - `TestBlank`:
    - `Read(Blank())` gives no R;
    - `_meta` holds `NOTE` = `임시 내장본, 원본 교체 예정`;
    - there is no `예시_설명` sheet;
    - every data validation of the sample is still present with the same lists and ranges (01: E2, E20, E21, E28, E30, E38, E42; 02: C4:C23, E4:E23; 03: G, I, J, L, M, N over 4:3003);
    - the autoFilter ranges are kept (02 A2:I23, 03 A2:O3003, 04 A2:E15003);
    - 01 E2 still holds `v1.0`, and all other E cells are empty.
  - `TestBlankRegenerates` (runs only with `-update`): rebuilds `blank_v3.2.xlsx` from the sample by the five steps in §5.4.
- [ ] **Step 2:** Run `go test ./kosac/...`. Expected: FAIL.
- [ ] **Step 3: Implement.**
  - Read display strings with `GetCellValue`.
  - For a cell whose number format is a date, convert the raw serial to `YYYY-MM-DD`. Use `excelize.ExcelDateToTime`.
  - S05 issues carry Sheet and Cell.
  - Write sets values into a copy of `Blank()`.
- [ ] **Step 4:** Run `go test ./kosac/... && go vet ./...`. Expected: PASS.

### Task 3: `kosac/check` — C01–C18 (§10.1, §10.2)

**Files:**
- Create: `kosac/check/check.go` (driver and sort) and `kosac/check/rules.go` (one func per rule)
- Modify: `kosac/model.go` (add `DefaultMissingLabels`, and `Prefix(name string) (string, bool)`: the part before the last `_`, false when there is none; §8.1 "접두")
- Test: `kosac/check/check_test.go`, golden `testdata/kosac/v32_sample_2024.issues.json`

**Interfaces:**
- Consumes: Task 1 `universe.Parse`, `Expr.Vars`. Task 2 model, `Unverified` and RefData.
- Produces:
  - `func Check(cb *kosac.Codebook, ref kosac.RefData) []kosac.Issue`, sorted per §9.
  - `var kosac.DefaultMissingLabels = map[string][]string{"-8": {"묻지 않음"}, "-7": {"모르겠다", "무응답", "모르겠다/무응답"}, "-9": {"유효하지 않음"}}`

- [ ] **Step 1: Write the failing tests.**
  - `TestRules`: a table with at least one pass and one fail per rule, C01…C18. Each case builds a tiny model in Go. Pin the §10.2 interpretations:
    - C01:
      - conditional STRATA_VARS on `할당(비확률)`;
      - `확인 불가` in optional TOOL gives W;
      - EXCLUDE_N `확인 불가` does not require EXCLUDE_RULE;
    - C02:
      - `10,000` passes, `2024-02-30` fails, FIELD_END < FIELD_START fails;
      - DATA_NO `문화-01-01` passes and `문화 01/01` fails;
    - C03 WEIGHT_YN=Y with empty WEIGHT_METHOD gives R;
    - C05 ID_VAR `walla_response_id` with no such 03 row passes (system var);
    - C07 counts only non-missing codes;
    - C08 gives one Issue per condition;
    - C09 `Q 3` gives R;
    - C10:
      - `Q5=1,2` where Q5 has codes {1,2} passes;
      - `Q5=9` gives R;
      - a reference to a var in another file gives R;
      - a range on a scale var passes;
    - C11 `Q5_other` (문자) does not join the MR group;
    - C12 C8_1 (REVERSE=Y) and C8_2 in group `C8` with another 서열 var gives W, and a name with no `_` is skipped;
    - C13:
      - "QA1_1, QA1_2를 0·50·100점으로 환산한 평균" passes;
      - `PCA 점수` passes (no digit or `_`);
      - `QX9 평균` with no QX9 gives R;
    - C17 STD_ID=SCHOOL.ID with no year in NOTE gives W;
    - C18:
      - `REPORT:p336-` with PDFPages 300 gives R;
      - `최종보고서 부록 수록 (PDF 336쪽~)` gives pages `336-` and no R;
      - `보고서 참조` gives R.
  - `TestUnverifiedEverywhere`:
    - 01 POP_SOURCE `확인 불가: x` gives a C01 W and no C02;
    - 03 RESP_UNIVERSE `확인 불가: y` gives a C10 W and no parse R.
  - `TestRefDataNil`: with a zero RefData, C05 and C18 skip the file-name checks and C16 skips. There are no panics.
  - `TestSampleGolden`: `Check(Read(sample), RefData{})` is compared with `v32_sample_2024.issues.json` (`-update` rewrites it). It must contain exactly:
    - C07 R 1 (SQ1);
    - C01 W 4;
    - C08 W 145;
    - C12 W 4 (C8_1, C8_2, C9_1, C9_2);
    - C10 W 1 (QC2);
    - no other issues.
    - **Pause after the first `-update`.** Report the counts per rule in the task report for human review before accepting the golden.
- [ ] **Step 2:** Run `go test ./kosac/check/`. Expected: FAIL.
- [ ] **Step 3: Implement** each rule as §10.2 states. Use `utf8.RuneCountInString` for lengths.
- [ ] **Step 4:** Run `go test ./kosac/...`. Expected: PASS. Report the per-rule counts on the sample.

### Task 4: IDs, assembly and the DDI projection (§6, §8.3)

**Files:**
- Create: `kosac/ids.go`, `kosac/build.go` (tree assembly), `kosac/system.go` (system var table, §6.7), `kosac/place.go` (the §6.6 table), `kosac/project.go` (`DDIProjection`), `kosac/labels.go` (the §6.7 label-prefix rule)
- Test: `kosac/build_test.go`, `kosac/ids_test.go`

**Interfaces:**
- Consumes: Task 2 model.
- Produces:
  ```go
  func AssignIDs(cb *Codebook, prev *Codebook) error
  func (cb *Codebook) Tree() (*ddi.Node, error)
  func (cb *Codebook) XML() ([]byte, error)  // ddi.WriteXML(Tree())
  func (cb *Codebook) JSON() ([]byte, error) // ddi.WriteJSON(Tree())
  func DDIProjection(root *ddi.Node) *ddi.Node // deep copy without kosac:* elements, attributes and xmlns:kosac
  func LabelPrefix(labels []string) string     // §6.7 label-prefix rule; "" when no boundary
  var KosacChildren = map[string][]string{ /* §6.6, DDI parent → ordered kosac child names without prefix */ }
  var KosacAttrs = map[string][]string{"var": {...}, "varGrp": {"mrGroup"}}
  var SystemVars = []SystemVar{ /* §6.7 table, with the labl texts */ }
  ```

- [ ] **Step 1: Write the failing tests.**
  - `TestAssignIDs`:
    - With nil prev, IDs are V1…Vn in 03 order.
    - With prev, kept names keep their IDs. A new var gets max+1. A deleted var's ID is not reused. Reordering keeps IDs.
    - Two files with the same name give two IDs (Review Focus 3).
    - A preset duplicate ID is an error.
  - `TestLabelPrefix`:
    - `{"과학기술분야의 업무는 …", "과학기술과 관련하여 …"}` gives `""`;
    - `{"STEM 학위·자격: 대학원 학위", "STEM 학위·자격: 학사"}` gives `STEM 학위·자격`;
    - `{"경쟁력 있는 과학기술 분야 (1순위)", "경쟁력 있는 과학기술 분야 (2순위)"}` gives `경쟁력 있는 과학기술 분야`.
  - `TestTreeSmall`: build a hand-made model.
    - It has:
      - 2 files;
      - a grid pair whose Q_TEXT ends with `▶ [가]` / `▶ [나]`;
      - an MR group of 3 with one label ending in `없음`;
      - a rank pair;
      - a weight var;
      - STRATA_VARS of 2 names;
      - a cleansing rule on file 2 whose var name also exists in file 1;
      - file 2's ID_VAR `walla_response_id` (not in Vars);
      - a Package with role=codebook and role=report files;
      - QUESTIONNAIRE `최종보고서 부록 수록 (PDF 336쪽~)`;
      - TARGET_N `10,000`.
    - Assert:
      - DDI children are in XSD order, and kosac children follow them in `KosacChildren` order;
      - otherMat order is report → questionnaire → extra;
      - S1–S6 for F1 and S7–S12 for F2, with the §6.7 labl texts and no missing catgry;
      - kosac:seq restarts per file;
      - the rank group is `type="other" otherType="rank"` with `kosac:mrGroup`;
      - `kosac:noneOption yn="Y"`;
      - the grid labl is the stem without `▶ […]`;
      - `var@wgt="wgt"`;
      - the strataVars IDs;
      - the cleansing `rule@var` is the file-2 ID;
      - file 2's ID_VAR and `kosac:systemIdVar` are both `S7`;
      - the codebook package file has no sha256;
      - `codeBook@ID` follows the §6.3 format;
      - `<sampleSize>10000</sampleSize>`;
      - the questionnaire has `kosac:pages` `336-` and the report's URI;
      - there is no empty `rspStmt`/`anlyInfo`.
  - `TestTreeUnverified`:
    - POP_SOURCE `확인 불가: a` gives `kosac:unverified key="POP_SOURCE"`.
    - FIELD_START `확인 불가: b` gives no start collDate and an unverified entry.
    - 02 VALID_N `확인 불가: c` on file 2 gives no caseQnty and `unverified key="VALID_N" file="F2"`.
    - RESP_UNIVERSE `확인 불가: d` keeps the universe text and adds an unverified entry with `@var`.
    - `확인 불가:` with an empty reason still writes the unverified element.
  - `TestTreeErrors`. Each of these makes `Tree()` return an error naming the item: an empty TITLE, a var without an ID, an unknown WEIGHT_VAR, a cleansing rule on an unknown var.
  - `TestAuthEnty`:
    - (org, name) gives `affiliation=org` and text name;
    - ("", name) gives `affiliation=""` and text name;
    - (org, "") gives text org and no attribute.
  - `TestProjectionValidates`: `xsdtest.Validate(t, ddi.WriteXML(DDIProjection(tree)))` passes for TestTreeSmall's model, and also for a model with every optional slot filled.
- [ ] **Step 2:** Run `go test ./kosac/`. Expected: FAIL.
- [ ] **Step 3: Implement.**
  - Assemble directly in XSD order. Use a small helper `el(name string, attrs ...string) *ddi.Node` plus `add(parent, child)` that skips nil and empty children.
  - Put the text through `ddi.NormalizeText`. Normalize integer slots per §6.1.
  - Name resolution follows §6.1:
    - look in the file's survey vars first, then in its system vars;
    - 01-sheet references go to the lowest FILE_NO.
- [ ] **Step 4:** Run `go test ./kosac/... && go vet ./...`. Expected: PASS.

### Task 5: `kosac.Parse` and the fixed point (§7)

**Files:**
- Create: `kosac/parse.go`
- Test: `kosac/parse_test.go`

**Interfaces:**
- Consumes: Task 4 `Tree`, `KosacChildren` and `SystemVars`.
- Produces: `func Parse(b []byte) (*Codebook, error)`. It accepts XML or JSON through `ddi.Parse`.

- [ ] **Step 1: Write the failing tests.**
  - `TestFixedPoint`: for TestTreeSmall's model and the all-slots model, `XML(Parse(XML(cb)))` is byte-equal, in XML and in JSON.
  - `TestParseValues`:
    - Record values equal the inputs, except for the four §7 exceptions: list fields are joined with `"; "`, `10,000` becomes `10000`, and `확인불가：x` becomes `확인 불가: x`.
    - BIZ_NAME comes back verbatim.
    - The three AuthEnty cases of Task 4 come back exactly.
    - 02 and 03 unverified values come back on the right file and var.
  - `TestParseRejects`. Each case must be an error that names the path:
    - an unknown DDI element (`stdyDscr/stdyInfo/notes`) added to engine output;
    - an unknown `kosac:foo`;
    - the kosac namespace bound to prefix `k`;
    - a DDI 2.5 file from `testdata/archive/` (not engine output).
- [ ] **Step 2:** Run `go test ./kosac/`. Expected: FAIL.
- [ ] **Step 3: Implement** the inverse of every §6 row. Recover names from IDs through the var table, and system vars by name.
- [ ] **Step 4:** Run `go test ./kosac/...`. Expected: PASS.

### Task 6: kosac 3.3 XSD, the kosac projection and `ValidateXML` (§10.3, §10.4, §13)

**Files:**
- Create: `xsd/embed.go` (`package xsd`; `//go:embed ddi-codebook-2.5`; `var FS embed.FS`)
- Create: `kosac/xsd/kosac-3.3.xsd`, `kosac/xsd/CHANGES.md`, `kosac/xsd/embed.go` (`var FS embed.FS`)
- Modify: `kosac/project.go` (add `KosacProjection`)
- Create: `kosac/check/xsd.go` (ValidateXML, PLACE) and `kosac/check/ref.go` (REF)
- Modify: `internal/xsdtest/xsdtest.go` (add `ValidateSchema(t, schemaPath string, xml []byte) []string`, which returns the problems instead of failing)
- Test: `kosac/check/xsd_test.go`, `kosac/check/ref_test.go`, `kosac/project_test.go`

**Interfaces:**
- Produces:
  ```go
  func KosacProjection(root *ddi.Node) *ddi.Node // <kosac:projection> of <kosac:place path="…">, §10.3 step 3
  // package check
  type Schema int
  const (SchemaDDI Schema = iota; SchemaKosac)
  type Validator func(s Schema, doc []byte) (problems []string, err error)
  func ValidateXML(xml []byte, v Validator) []kosac.Issue
  ```
- The `place@path` format is the same as `Issue.Path`, for example `dataDscr/var[@ID='V12']`.

- [ ] **Step 1: Write the failing tests.**
  - `TestKosacXSDCompiles`: `xmllint --noout --schema kosac-3.3.xsd` on an empty projection compiles. This guards the global/local split in §10.4.
  - `TestKosacXSDAcceptsEngineOutput`: the kosac projection of TestTreeSmall's and the all-slots model validates.
  - `TestKosacXSDRejects`. Each must give problems:
    - `kosac:role="BOSS"`;
    - `kosac:fk@mapping="X"`;
    - `kosac:file@sha256="xyz"`;
    - `kosac:excludeN` text `많음`;
    - an undeclared `kosac:foo` attribute.
  - `TestValidateXML` with a Validator built on `xsdtest.ValidateSchema`:
    - engine output gives zero issues;
    - a DDI break (delete `otherMat@level`) gives one XSD-DDI R;
    - a kosac break gives one XSD-KOSAC R;
    - `kosac:package` moved under `stdyDscr` gives PLACE R;
    - `kosac:seq` on `catgry` gives PLACE R.
  - `TestRef`. Each of these gives one REF R:
    - `varGrp var="V1 V99"`;
    - `var files="F9"`;
    - `idVar ref="V99"`;
    - `rule var="V99"`;
    - `unverified file="F9"`;
    - a duplicate `var@ID`;
    - `codeBook ID="1abc"`.
  - `TestValidatorError`: a Validator that returns an error gives one R issue whose Message says the validation could not run. It does not panic.
  - `TestChangesDocCoversTable`:
    - every row 1–19 of spec §13 appears in `CHANGES.md` (match by the request-side text column);
    - every kosac element and attribute name in the XSD is listed in `CHANGES.md`.
- [ ] **Step 2:** Run `go test ./kosac/... ./internal/xsdtest/`. Expected: FAIL.
- [ ] **Step 3: Implement.**
  - XSD per §10.4:
    - Only the §6.6 top-level elements and the var/varGrp attributes are global.
    - Sub-elements are local.
    - Reference attributes are `xs:NCName`/`xs:NMTOKENS`.
    - Enumerations come from request 4-2 and §8.1.
  - `CHANGES.md` is in Korean. Its sections:
    - 개요 (two projections and why; REF exists because libxml2 does not resolve IDREF);
    - 요소·속성 목록 (name, parent, type, source KEY);
    - 요청서 v1.2와 다른 점 (§13 table and its two notes, including row 19's `kosac:artifact` proposal for the foundation to choose).
- [ ] **Step 4:** Run `go test ./... && go vet ./...`. Expected: PASS.

### Task 7: `DeriveDefaults`, `ApplyFK`, suggestions and the sample golden (§8.1, §8.2)

**Files:**
- Create: `kosac/derive.go`, `kosac/suggest.go`, `kosac/reserved.go` (PostgreSQL reserved keywords)
- Test: `kosac/derive_test.go`, `kosac/large_test.go`, goldens `testdata/kosac/v32_sample_2024.xml` and `.json`

**Interfaces:**
- Consumes: Tasks 2–6.
- Produces:
  ```go
  func DeriveDefaults(cb *Codebook, ref RefData) []Suggestion // fills only empty fields (§8.1)
  func ApplyFK(cb *Codebook, v *Var, def CodeSetDef) error   // error when v.FK is nil or names another code set
  ```

- [ ] **Step 1: Write the failing tests.**
  - `TestDeriveRules`: a table over a small model, covering every row of §8.1:
    - role priority (a var that is both ID_VAR and DERIVATION gets ID);
    - DimMeasure;
    - AggRules with MR_TYPE first (순위별 열 gives `COUNT;RATIO_RANK`, not DIM's), and `;WAVG` only when the same file has a WEIGHT var;
    - PhysicalName: `Q-1`→`q_1`, `select`→`q_select`, a 70-char name is cut to 64, and the second duplicate gets `_2`;
    - IncludeYn;
    - CodeSetID: `보통 이다`/`보통이다` share an ID, a singleton set still gets one, `LIKERT5_1`/`LIKERT5_2`, and a RefData code-set match;
    - MRGroup prefix (a name with no `_` uses the whole name), NoneOption on every member, RankNo;
    - GridGroup with the stem key and RowLabel from `▶ [x]`. The markers are one constant list, `gridRowMarkers`. A pair whose Q_TEXT has no marker falls back to the full Q_TEXT and still groups when FILE_NO, VAR_GROUP, prefix and MEAS_LEVEL match. Four no-`_` names with the same Q_TEXT (the QA6–QA9 case) do not group;
    - Scale compared as numbers (`10` > `9`) with NEG;
    - fill-only-empty: preset `MRGroup="X"` and `DimMeasure="ATTR"` survive;
    - Admin defaults are filled only when empty.
  - `TestDeriveMissingCodes`:
    - a nominal var with 04 rows 1, 2 and `-8 / 무응답(잘못된 라벨)` keeps that label, gets Type MISSING, and then has -7 and -9 appended;
    - `-7 / 무응답` is relabelled to `모르겠다` in place;
    - Types are VALID/MISSING/DK/MISSING (Q1 default);
    - a text var and the system vars get none;
    - `kosac:missingCodes` is `-8;-7;-9`, and `-8;-7;-9;-6` when a -6 code exists;
    - calling DeriveDefaults twice changes nothing the second time.
  - `TestSuggest`: with a RefData holding PERSON.GENDER and CC_GENDER:
    - a var labelled `성별` gives a high STD_ID suggestion;
    - labels {남성, 여성} give an FK suggestion CC_GENDER with CodeMap {1:M, 2:F};
    - nothing is applied to the model.
  - `TestApplyFK`:
    - PhysicalName becomes `gender_cd` and DimMeasure becomes DIM;
    - DimTable/KeyColumn are filled from the def only when empty;
    - two vars in one file mapped to CC_SIDO get `sido_cd` and `sido_cd_2`;
    - a nil FK is an error.
  - `TestSampleDerived`: on the sample, after DeriveDefaults:
    - MR groups 3, rank groups 4, grid groups exactly the 22 keys listed in §8.1;
    - NoneOption is QA19 Y, QA21 Y and QD1_1 N;
    - no RowLabel starts in the middle of a word: each is a `▶ [x]` value or a full VAR_LABEL.
  - `TestSampleGolden`:
    1. Read the sample, then DeriveDefaults, then AssignIDs(nil).
    2. Set a fixed Package, Admin, one FK with ApplyFK, and one Cleansing with two rules (fixtures in the test file).
    3. Run XML and JSON. Compare with the goldens (`-update`).
    4. ValidateXML gives zero issues, and Parse gives a fixed point.
    5. **Pause after the first `-update`.** Report these in the task report for human review:
       - section count;
       - CodeSetID count and the 5 largest sets;
       - 3 sample RowLabels;
       - MR and rank labl texts.
  - `TestLargeCodebook` (skipped under `-short`):
    - Generate 3,500 vars and 30,000 codes over 2 files.
    - Run Write, Read, Check, DeriveDefaults, AssignIDs, XML and Parse. Together they take under 10s.
    - Write must not error past the dropdown rows.
- [ ] **Step 2:** Run `go test ./kosac/...`. Expected: FAIL.
- [ ] **Step 3: Implement** per §8.1. The tokenizer and Jaccard are in `suggest.go`, with the particle list from §8.1.
- [ ] **Step 4:** Run `go test ./... && go vet ./...`. Expected: PASS. Report the counts named above.

### Task 8: Native path `FromWallaDDI` (§12)

**Files:**
- Create: `kosac/walla.go`
- Test: `kosac/walla_test.go`

**Interfaces:**
- Consumes: Tasks 2, 4, 6, 7.
- Produces:
  ```go
  type WallaOptions struct{ MeasHints map[string]string } // var name → MEAS_LEVEL value (명목·서열·연속·문자·날짜)
  func FromWallaDDI(b []byte, opt WallaOptions) (*Codebook, []Issue, error)
  ```
- Keep the walla naming and missing rules in one unexported function, `wallaKind`, plus its constants, as §12 says.
  - The legacy mode is one branch, detected once per document.
  - The branch carries a comment saying it goes when the KOFAC surveys are re-exported with the new rules.

- [ ] **Step 1: Write the failing tests.**
  - `TestWallaSynthetic` on `testdata/walla_synthetic/survey.xml`:
    - 46 survey vars (52 minus 6 `walla_*`);
    - 1 file, FILE_FORMAT `CSV`, ID_VAR `walla_response_id`, VALID_N empty (no caseQnty);
    - `Admin.WallaID` is `SampleForm0000000001`, and `Package` is nil;
    - MR: the 4 members of `f_Rk8Mn3Tq1b` (incl. `f_Rk8Mn3Tq1b_other`, `f_Rk8Mn3Tq1b_custom`, `f_Rk8Mn3Tq1b__custom`) have MR_TYPE 보기별 열 and MRGroup `f_Rk8Mn3Tq1b`;
    - grid: `f_Ux1Dz4Yf2g`, `f_Vy6Ea7Zg3h` and `f_Xa3Gc5Bi6j` members have GridGroup = that name;
    - rank: `f_Wz9Fb2Ah4i__rank1/2` have 순위별 열, RankNo 1/2, MRGroup `f_Wz9Fb2Ah4i` and an empty GridGroup;
    - suffixes:
      - `f_6558d0ad57__other` has ParentVar `f_6558d0ad57` and textFormat TEXT;
      - `f_Ux1Dz4Yf2g__other` has no ParentVar (no base var);
      - `f_Rk8Mn3Tq1b_other` has no ParentVar (option ID);
      - `f_Dg5Mi6Ho7p__lat` has no ParentVar;
    - 7 `-6` codes with Type MISSING; after DeriveDefaults those vars have missingCodes `-8;-7;-9;-6`;
    - after DeriveDefaults, the walla-filled group keys are unchanged;
    - after DeriveDefaults, AssignIDs, setting every VAR_GROUP to `기본` and XML, `ValidateXML` gives zero issues;
    - `survey.json` gives the same model as `survey.xml`;
    - with `MeasHints{"f_Yb7Hd8Cj9k":"서열"}`, that var gets 서열.
  - `TestWallaKofacB01` on `testdata/kofac_b01/2024/survey.xml` and `testdata/kofac_b01/2025/survey.xml`. These are the real KOFAC codebooks re-exported with the new rules (PR #1568 b0c99290c2).
    - The survey var count is 286 and 634 (292−6 and 640−6). This is the same as the legacy fixtures, which cross-checks the two modes.
    - 2024 has 4 rank groups (`__rank{k}`), 3 MR groups and 22 grid groups. 2025 has 0 rank, 14 MR and 67 grid groups.
    - The -6 `현재 게시본에 없는 값` codes are kept with Type MISSING: 156 in 2024 and 351 in 2025.
    - The XML and JSON inputs give the same model.
    - After filling VAR_GROUP, DeriveDefaults, AssignIDs and XML, `ValidateXML` gives zero issues.
  - `TestWallaLegacy` on `testdata/kofac_2024_public/survey.xml` and `testdata/kofac_2025_edu/survey.xml`:
    - 286 and 634 survey vars (system `response_id`, `submitted_at`, `customer_key` dropped);
    - the `Q30_rank1/2` style vars are 순위별 열;
    - after DeriveDefaults, every `-7 / 현재 게시본에 없는 값` row keeps its label with Type MISSING (156 and 351 rows);
    - XML passes `ValidateXML` after VAR_GROUP is filled.
- [ ] **Step 2:** Run `go test ./kosac/ -run Walla`. Expected: FAIL.
- [ ] **Step 3: Implement** per §12.
- [ ] **Step 4:** Run `go test ./...`. Expected: PASS. In the task report, add one line for datajam: native drafts leave VAR_GROUP empty, so C06 gives R until datajam fills each var's `VAR_GROUP` with the walla form's group (page) label.

### Task 9: CLI subcommands and README (§15)

**Files:**
- Modify: `cmd/ddi-xlsx/main.go`, `README.md`
- Create: `cmd/ddi-xlsx/xmllint.go`. It is a `check.Validator` that does three things:
  - it copies `xsd.FS` and `kosac/xsd.FS` whole into a temp dir with `os.CopyFS` (codebook.xsd imports siblings by relative path);
  - it runs `xmllint --nonet --noout --schema`;
  - it removes the dir.
- Test: `cmd/ddi-xlsx/main_test.go`

**Interfaces:**
- Produces:
  - CLI usage lines:
    - `ddi-xlsx kosac-check <양식.xlsx>`
    - `ddi-xlsx kosac-xml <양식.xlsx> [-o out.xml]`
    - `ddi-xlsx kosac-template [-o kosac-template.xlsx]`
  - Output: one issue per line, `<Level> <Rule> <Sheet>!<Cell> <Message>`. When there is no cell, `<Path>` replaces `<Sheet>!<Cell>`.
  - kosac-check exits 1 when any R exists and 0 otherwise.
  - kosac-xml writes the file and prints the ValidateXML issues. When xmllint is missing it prints `XSD 검증 건너뜀: xmllint 없음`.
  - kosac-template does not overwrite the default path, the same as `template`.

- [ ] **Step 1: Write the failing tests.**
  - `TestKosacCheck`:
    - the sample gives exit 1 (C07 R), and its output contains `R C07 03_변수정의!`;
    - a fixed sample (SQ1 codes added in a temp copy) gives exit 0.
  - `TestKosacXML`: the sample gives a file that `kosac.Parse` accepts.
  - `TestKosacTemplate`: `-o p` writes `Blank()` bytes. An existing default path gives exit 1.
- [ ] **Step 2:** Run `go test ./cmd/...`. Expected: FAIL.
- [ ] **Step 3: Implement.** Add a README section `재단 코드북(KOSAC) 엔진`. It covers:
  - what it does;
  - the three commands;
  - the Go API flow (§14);
  - DeriveDefaults' fill-only-empty rule and how to recompute a field (§8.1);
  - a "자동 판정 한계" note: the grid row marker is only `▶ [행]` for now. Other notations fall back to the full Q_TEXT, so they may stay ungrouped until the reviewer confirms GRID_GROUP in F5;
  - the two-projection validation with PLACE/REF;
  - the temporary blank;
  - the dropdown row limits (§5.3);
  - the private-fixture note (`testdata/kosac/README.md`).
- [ ] **Step 4:** Run `go vet ./... && go test ./... && scripts/build.sh && node --test npm/test/*.test.mjs`. Expected: all PASS, and the npm tests are unchanged.

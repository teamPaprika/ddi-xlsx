---
title: 재단 Go API
weight: 2
---

모든 함수는 바이트와 모델만 주고받습니다. 아래 코드는 호출 순서를 보여 줍니다.

```go
import (
	"github.com/teamPaprika/ddi-xlsx/kosac"
	"github.com/teamPaprika/ddi-xlsx/kosac/check"
	"github.com/teamPaprika/ddi-xlsx/kosac/template"
	"github.com/teamPaprika/ddi-xlsx/kosac/universe"
)

// 업로드형: 용역사가 채운 양식에서 코드북 XML을 만듭니다.
// prev는 이전 XML을 kosac.Parse로 읽은 모델이거나 nil입니다.
func upload(xlsx []byte, ref kosac.RefData, prev *kosac.Codebook, validator check.Validator) ([]byte, []kosac.Issue, error) {
	cb, issues, err := template.Read(xlsx, template.ReadOptions{}) // S05
	if err != nil {
		return nil, nil, err // xlsx가 아니거나 압축 해제 한도를 넘었습니다.
	}
	if cb == nil {
		return nil, issues, nil // S05에서 R이 나왔습니다. 양식 구조가 틀렸으므로 C01~C18을 돌리지 않습니다.
	}
	issues = append(issues, check.Check(cb, ref)...) // C01~C18
	kosac.DeriveDefaults(cb, ref)                    // 빈 칸을 채웁니다. 돌려주는 STD_ID·FK 후보는 담당자에게 보여 줍니다.

	// 담당자 확인: cb.Admin, cb.File("1").SensLevel 등을 고칩니다.
	// cb.Var와 cb.File은 해당 행이 없으면 nil을 돌려줍니다.
	if v := cb.Var("1", "SQ4"); v != nil {
		v.FK = &kosac.FK{CodeSet: "CC_SIDO"}
		if err := kosac.ApplyFK(cb, v, ref.CodeSetDefs["CC_SIDO"]); err != nil {
			return nil, nil, err
		}
	}
	if f := cb.File("1"); f != nil {
		f.Cleansing = &kosac.Cleansing{ /* … */ }
	}

	if err := kosac.AssignIDs(cb, prev); err != nil {
		return nil, nil, err
	}
	cb.Package = &kosac.Package{ /* … */ } // 해시는 호출자가 계산합니다.
	xml, err := cb.XML()                   // cb.JSON()도 있습니다.
	if err != nil {
		return nil, nil, err
	}
	return xml, append(issues, check.ValidateXML(xml, validator)...), nil
}

// 네이티브형: Walla 내보내기에서 담당자가 볼 v3.2 양식을 만듭니다.
// groups는 VAR_NAME → Walla 폼의 그룹(페이지) 라벨입니다.
func native(export []byte, hints, groups map[string]string) ([]byte, []kosac.Issue, error) {
	cb, issues, err := kosac.FromWallaDDI(export, kosac.WallaOptions{MeasHints: hints})
	if err != nil {
		return nil, nil, err
	}
	for _, v := range cb.Vars {
		v.Rec.Set("VAR_GROUP", groups[v.Rec.Get("VAR_NAME")]) // 비워 두면 C06 R이 납니다.
	}
	draft, err := template.Write(cb)
	return draft, issues, err
}

// 응답대상이나 클린징 조건이 응답 한 행에 맞는지 봅니다.
// row는 변수 이름으로 원래 값을 주는 universe.Row입니다.
func applies(row universe.Row) (bool, error) {
	e, err := universe.ParseCondition("SQ3!=1 AND SQ3_1!=-8", "SQ3_1")
	if err != nil {
		return false, err
	}
	return e.Eval(row)
}
```

## 입력과 결과

- `ref`는 `kosac.RefData`입니다. 비어 있는 필드가 있으면, 그 자료가 필요한 검사와 판정을 건너뜁니다.
- `issues`의 각 항목에는 아래 필드가 있습니다.

| 필드 | 내용 |
|---|---|
| `Rule` | 규칙 |
| `Level` | 수준 |
| `Sheet`·`Cell` 또는 `Path` | 위치 |
| `FileNo`·`VarName`·`Code` | 프로그램이 읽는 키 |
| `Message` | 한 문장 메시지 |

## 자동 판정(`DeriveDefaults`)

- 빈 칸만 채웁니다. 그래서 담당자가 고친 값은 다시 호출해도 남습니다. 두 번 호출해도 결과가 같습니다.
- 판정에 쓴 값(예: 역할)을 나중에 바꿔도, 그 값에서 나온 칸(예: 차원·측정 구분)은 다시 계산하지 않습니다. 다시 계산하려면, 그 칸을 비우고 `DeriveDefaults`를 다시 호출하세요.
- `GridGroup`은 묶음 단위로 계산합니다. 멤버 중 하나라도 값이 있는 묶음은 바꾸지 않습니다. 다시 계산하려면, 그 묶음의 모든 멤버에서 `GridGroup`을 비우세요.

### 자동 판정의 한계

- grid를 찾을 때, 문항 문장(Q_TEXT) 끝의 행 표시를 떼고 비교합니다. 지금 아는 행 표시는 2024 작성 예시의 `▶ [행]` 하나뿐입니다.
- 다른 표기(예: `문항 - 행`, `문항(행)`)는 행 표시로 보지 않고, Q_TEXT 전체를 비교합니다. 그래서 실제 grid 문항이 묶이지 않는 경우가 있습니다. 잘못 묶는 것보다 묶지 않는 것이 안전하기 때문입니다.
- 묶이지 않은 grid는 담당자가 확인 화면(F5)에서 GRID_GROUP을 확정할 때까지 묶이지 않은 상태로 남습니다.

## 네이티브 코드북(Walla 내보내기)

- `kosac.FromWallaDDI`가 만든 모델은 `Codebook.Native`가 참입니다.
- 네이티브 코드북에서는 변수 이름으로 grid를 판정하지 않습니다. Walla 변수 이름은 접두가 모두 `f`입니다. 그래서 같은 페이지에서 문장이 같은 단일선택 문항이 가짜 grid로 묶이기 때문입니다. 묶음은 Walla의 varGrp 멤버십으로만 정합니다.
- `Codebook.Native`는 XML에 `kosac:package/kosac:sourceSurvey@kind="native"`로 남습니다.
- `FromWallaDDI`는 `cb.Package`를 nil로 둡니다. 이때도 조립은 `docDscr` 아래에 이 표시만 씁니다. 그래서 `kosac.Parse`로 다시 읽으면 `Native`가 참이고 `Package`는 nil입니다.
- 나중에 `cb.Package`를 채우면, `Source.Kind` 자리에 `native`가 들어갑니다. `Source.Kind`는 비워 두거나 `"native"`로 두세요. 다른 값(예: `"upload"`)이면 `cb.XML()`이 오류를 냅니다.

{{< callout type="warning" >}}
네이티브 초안은 VAR_GROUP(영역)을 비워 둡니다. Walla 내보내기에는 영역 정보가 없기 때문입니다. 비워 두면 C06 `R`이 나옵니다. 호출자가 Walla 폼의 그룹(페이지) 라벨을 채우세요. 예: `v.Rec.Set("VAR_GROUP", pageLabel)`
{{< /callout >}}

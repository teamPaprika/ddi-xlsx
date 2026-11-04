---
title: 라이선스
weight: 15
---

이 페이지는 저장소와 배포물에 들어 있는 제3자 자료의 저작권과 라이선스를 설명합니다. 조사 기준일은 2026-11-03입니다.

{{< callout type="info" >}}
이 페이지는 조사 결과입니다. 법률 판단이 필요하면, 법률 전문가에게 확인하세요.
{{< /callout >}}

## 요약

| 구성 요소 | 라이선스(SPDX) | git 저장소 | CLI 바이너리 | npm 패키지 | 문서 사이트 |
|---|---|---|---|---|---|
| 이 도구의 소스코드 | 정하지 않음 | 있음 | 있음 | 있음 | |
| DDI Codebook 2.5 XSD | `LGPL-3.0-or-later` | 있음 | 내장 | | |
| DDI XHTML 스키마 | `LGPL-3.0-or-later` + W3C 고지 | 있음 | 내장 | | |
| W3C XHTML 모듈 스키마 | W3C XHTML 스키마 고지 | 있음 | 내장 | | |
| XHTML 문자 엔터티 | `ISO-permission` + W3C | 있음 | 내장 | | |
| W3C `xml.xsd` | 표시 없음 | 있음 | 내장 | | |
| Dublin Core 스키마 | `CC-BY-4.0` | 있음 | 내장 | | |
| 생성한 규칙 표 | 정하지 않음 | 있음 | 있음 | 있음 | |
| 재단 양식 자산 | 표시 없음 | 있음 | 내장 | | |
| Go 모듈 | `BSD-3-Clause`, `Apache-2.0`, `MIT` | | 링크 | 링크 | |
| Go 표준 라이브러리, `wasm_exec.js` | `BSD-3-Clause` | | 링크 | 있음 | |
| 테스트 데이터 | 아래 표 | 있음 | | | |
| 문서 사이트 테마와 라이브러리 | `MIT`, `Apache-2.0` | | | | 있음 |

- "내장"은 `go:embed`로 바이너리에 파일을 그대로 넣는다는 뜻입니다.
- git 저장소의 파일은 Go 모듈 zip(`go get`)에도 들어갑니다. `site/`는 자기 `go.mod`가 있어서 제외됩니다.
- CLI 바이너리와 npm 패키지는 아직 배포하지 않았습니다. 배포 전에 [확인이 필요한 항목](#확인이-필요한-항목)을 처리해야 합니다.

## 이 도구

소스코드 저장소는 공개 상태입니다. 이 도구 자체의 라이선스는 아직 정하지 않았습니다.

CLI 바이너리는 LGPL-3.0-or-later 스키마를 내장합니다. 그래서 도구의 라이선스는 LGPL-3.0 4절의 조건과 맞아야 합니다. 4절은 내장한 부분의 수정과, 그 수정을 디버깅하기 위한 역공학을 막지 않는 조건을 요구합니다.

## 함께 넣어 둔 스키마

`xsd/ddi-codebook-2.5/`에는 파일 29개가 있습니다. `xsd/embed.go`가 이 파일 전부를 CLI 바이너리에 내장합니다. CLI는 XSD를 검사할 때 이 파일을 임시 디렉터리에 쓰고 `xmllint`에 넘깁니다. wasm(npm 패키지)에는 들어가지 않습니다.

### DDI Codebook 2.5 XSD

| 항목 | 내용 |
|---|---|
| 파일 | `codebook.xsd` |
| 저작권 | Copyright (c) 2014 DDI Alliance |
| 라이선스 | GNU LGPL 버전 3 또는 그 이후 버전(`LGPL-3.0-or-later`) |
| 출처 | DDI Alliance의 DDI Codebook 2.5.1 XML Schema. 원본과 바이트 단위로 같습니다. |

- LGPL-3.0은 GPL-3.0에 추가 권한을 더한 라이선스입니다. 그래서 GPL-3.0의 조건도 적용합니다.
- DDI Codebook 2.6은 CC BY 4.0으로 배포합니다. 이 라이선스는 2.5에 적용하지 않습니다. DDI Alliance는 2.5.1을 지금도 LGPL로 배포합니다.
- 의무: 파일 머리말을 유지합니다. 받는 사람에게 LGPL-3.0과 GPL-3.0 전문을 줍니다. 바이너리로 배포하면, 이 스키마가 들어 있다는 고지와 두 전문을 함께 줍니다.

### DDI XHTML 스키마

| 항목 | 내용 |
|---|---|
| 파일 | `ddi-xhtml11.xsd`, `ddi-xhtml11-model-1.xsd`, `ddi-xhtml11-modules-1.xsd` |
| 저작권 | Copyright (c) 2009 DDI Alliance(DDI 3.1 XML Schema의 일부). W3C 부분은 Copyright 1998-2003 W3C(MIT, INRIA, Keio) |
| 라이선스 | DDI Alliance 부분은 `LGPL-3.0-or-later`. W3C 부분은 W3C XHTML 스키마 고지 |
| 출처 | DDI Alliance가 W3C XHTML 1.1 스키마를 고친 파일입니다. |

W3C 고지의 전문은 `ddi-xhtml11.xsd` 안에 있습니다.

### W3C XHTML 모듈 스키마

| 항목 | 내용 |
|---|---|
| 파일 | `XHTML/`의 `.xsd` 18개 |
| 저작권 | W3C(MIT, INRIA, Keio), 1998-2002. 파일 안에는 저작권 줄이 없습니다. |
| 라이선스 | W3C XHTML 스키마 고지. SPDX 식별자는 없습니다. |
| 출처 | W3C XHTML Modularization 1.1 Working Draft(2004-02-18) 스키마 |

- 각 파일은 고지 파일 `xhtml-copyright-1.xsd`를 가리킵니다. 이 파일은 저장소에 없습니다.
- 고지는 목적에 관계없이 사용, 복사, 수정, 배포를 허락합니다. 조건은 모든 사본에 저작권 고지와 허락 문단을 넣는 것입니다.
- DDI Alliance 배포본과 같은 파일입니다. W3C 원본과 비교하면 아래가 다릅니다.
  - `xhtml-attribs-1.xsd`, `xhtml-bdo-1.xsd`, `xhtml-blkphras-1.xsd`는 `../xml.xsd`를 가져옵니다.
  - `xhtml-charent-1.xsd`는 엔터티 파일을 파일 이름으로 가리킵니다.

### XHTML 문자 엔터티

| 항목 | 내용 |
|---|---|
| 파일 | `XHTML/xhtml-lat1.ent`, `XHTML/xhtml-special.ent`, `XHTML/xhtml-symbol.ent` |
| 저작권 | Portions (C) International Organization for Standardization 1986. W3C 저작권 줄은 없습니다. |
| 라이선스 | ISO 8879 엔터티 고지(`ISO-permission`). W3C 부분은 W3C Software and Document License(`W3C-20150513`)로 취급합니다. |
| 출처 | W3C XHTML Modularization 1.1 Working Draft(2006-07-05) DTD. 원본과 바이트 단위로 같습니다. |

- ISO 고지는 고지를 유지하는 조건으로 복사를 허락합니다. 수정은 허락하지 않습니다. 그래서 이 파일은 고치지 않습니다.
- 엔터티 이름 일부는 RFC 2070과 HTML 4.0에서 왔습니다. 이름과 문자 코드는 사실 정보입니다.

### W3C XML 네임스페이스 스키마

| 항목 | 내용 |
|---|---|
| 파일 | `xml.xsd` |
| 저작권 | 표시 없음. 내용상 W3C XML Schema Working Group이 만들었습니다. |
| 라이선스 | 표시 없음. W3C 원본(`2001/03`, `2001`, `2009/01`)에도 없습니다. W3C Software and Document License(`W3C-20150513`)로 취급합니다. |
| 출처 | `http://www.w3.org/2001/03/xml.xsd` |

DDI Alliance 배포본과 같은 파일입니다. W3C 원본과 비교하면 DOCTYPE과 설명 일부가 없습니다.

### Dublin Core 스키마

| 항목 | 내용 |
|---|---|
| 파일 | `dc.xsd`, `dcterms.xsd`(2008-02-11), `dcmitype.xsd`(2003-04-02) |
| 저작권 | Copyright © 2011 DCMI, the Dublin Core Metadata Initiative. 파일에는 작성자(Tim Cole, Tom Habing, Jane Hunter, Pete Johnston, Carl Lagoze)가 있습니다. |
| 라이선스 | Creative Commons Attribution 4.0 International(`CC-BY-4.0`) |
| 출처 | <https://www.dublincore.org/schemas/xmls/> |

- 변경: `dc.xsd`와 `dcterms.xsd`는 한 줄이 원본과 다릅니다. W3C `xml.xsd`의 절대 URL 대신 상대 경로 `xml.xsd`를 가져옵니다. `dcmitype.xsd`는 원본과 같습니다.
- 의무: 작성자 표시를 유지합니다. 저작권 고지, 라이선스 링크, 원본 URI, 변경 사실을 표시합니다. 이 페이지가 그 표시입니다.

### 생성한 규칙 표

`internal/schema/table.go`는 `go generate`가 `codebook.xsd`에서 만든 파일입니다. 요소 이름, 반복 횟수, 자식 순서, 속성 값 목록을 담습니다. 이 파일은 CLI 바이너리와 wasm에 링크됩니다.

이 표가 LGPL 스키마의 2차적 저작물인지는 정하지 않았습니다. 표의 내용이 저작권 보호 대상이 아닌 사실 정보이면, 제3자 라이선스는 적용하지 않습니다.

## 재단 양식 자산

| 파일 | 내용 | 라이선스 | 배포 |
|---|---|---|---|
| `kosac/xsd/kosac-3.3.xsd` | 이 프로젝트가 쓴 kosac 확장 XSD 초안입니다. 요소 이름과 값 목록은 재단 요청서를 따릅니다. | 표시 없음 | CLI 내장, `kosac/xsd.FS` |
| `kosac/xsd/CHANGES.md` | 재단 승인용 변경 기록입니다. | 표시 없음 | git 저장소 |
| `kosac/template/blank_v3.2.xlsx` | 재단 작성 예시의 값을 지운 임시 양식입니다. 재단의 안내문, 항목 정의, 목록은 남아 있습니다. | 표시 없음 | CLI 내장, v3.2 모델의 `template.Write`가 씁니다. |
| `kosac/template/blank_v4.0.xlsx` | 재단이 제공한 2026-10-19 개정 공식 빈 양식입니다. | 표시 없음 | CLI 내장, `kosac-template`, `template.Blank`, v4.0 모델의 `template.Write`가 씁니다. |

저장소에는 이 자료의 라이선스나 이용 허락이 적혀 있지 않습니다.

## Go 모듈

아래 모듈은 CLI 바이너리와 wasm(npm 패키지)에 링크됩니다. 버전은 `go.mod` 기준입니다.

| 모듈 | 버전 | 라이선스 | 저작권 |
|---|---|---|---|
| `github.com/xuri/excelize/v2` | v2.11.0 | `BSD-3-Clause` | Copyright (c) 2016-2026 The excelize Authors. Copyright (c) 2011-2017 Geoffrey J. Teale. `hsl.go`: Copyright (c) 2012 Rodrigo Moraes |
| `github.com/xuri/efp` | v0.0.1 | `BSD-3-Clause` | Copyright (c) 2017 - 2025 Ri Xu |
| `github.com/xuri/nfp` | v0.0.2-0.20250530014748-2ddeb826f9a9 | `BSD-3-Clause` | Copyright (c) 2022-2025 Ri Xu |
| `github.com/richardlehane/mscfb` | v1.0.7 | `Apache-2.0` | Copyright 2013 Richard Lehane |
| `github.com/richardlehane/msoleps` | v1.0.6 | `Apache-2.0` | Copyright 2014, 2015 Richard Lehane |
| `github.com/tiendc/go-deepcopy` | v1.7.2 | `MIT` | Copyright (c) 2023 tiendc |
| `golang.org/x/crypto` | v0.53.0 | `BSD-3-Clause` | Copyright 2009 The Go Authors |
| `golang.org/x/net` | v0.56.0 | `BSD-3-Clause` | Copyright 2009 The Go Authors |
| `golang.org/x/text` | v0.38.0 | `BSD-3-Clause` | Copyright 2009 The Go Authors |

- `golang.org/x` 모듈에는 Google의 특허 허락(`PATENTS`)이 함께 있습니다.
- `golang.org/x/text`의 표는 Unicode, CLDR, ICU, WHATWG 데이터에서 생성했습니다. Unicode 데이터의 라이선스는 Unicode License v3(`Unicode-3.0`)입니다.
- Apache-2.0 모듈 두 개에는 `NOTICE` 파일이 없습니다.

`go.sum`에는 아래 모듈의 체크섬도 있습니다. 이 모듈은 링크되지 않습니다. 의존성의 테스트에서만 씁니다.

| 모듈 | 버전 | 라이선스 |
|---|---|---|
| `github.com/stretchr/testify` | v1.11.1 | `MIT` |
| `github.com/davecgh/go-spew` | v1.1.1 | `ISC` |
| `github.com/pmezard/go-difflib` | v1.0.0 | `BSD-3-Clause` |
| `golang.org/x/image` | v0.38.0 | `BSD-3-Clause` |
| `gopkg.in/yaml.v3` | v3.0.1 | `MIT AND Apache-2.0` |

### Go 표준 라이브러리와 `wasm_exec.js`

| 구성 요소 | 라이선스 | 저작권 | 배포 |
|---|---|---|---|
| Go 표준 라이브러리와 런타임 | `BSD-3-Clause`, `PATENTS` | Copyright 2009 The Go Authors | CLI 바이너리, wasm |
| `npm/wasm_exec.js` | `BSD-3-Clause` | Copyright 2018 The Go Authors | npm 패키지. `scripts/build.sh`가 Go 툴체인에서 복사합니다. |

표준 라이브러리의 일부 파일에는 다른 고지가 있습니다.

| 파일 | 고지 | 들어가는 곳 |
|---|---|---|
| `runtime/memmove_amd64.s` | Inferno `MIT`. Copyright © 1994-1999 Lucent Technologies Inc., 2000-2007 Vita Nuova Holdings Limited | amd64 CLI 바이너리 |
| `math/log.go`, `math/exp.go` | Sun Microsystems fdlibm 고지(`SunPro`) | arm64 CLI 바이너리, wasm |
| `unicode/tables.go` | Unicode Character Database에서 생성(`Unicode-3.0`) | CLI 바이너리, wasm |

## 실행할 때 쓰는 외부 도구

| 도구 | 라이선스 | 설명 |
|---|---|---|
| `xmllint`(libxml2) | `MIT` | CLI와 테스트가 `PATH`의 `xmllint`를 실행합니다. 함께 배포하지 않습니다. |

## 테스트 데이터

`testdata/`에는 응답자별 응답 자료가 없습니다. 코드북 메타데이터만 있습니다.

| 경로 | 내용 | 라이선스 |
|---|---|---|
| `testdata/kofac_2024_public/`, `testdata/kofac_2025_edu/`, `testdata/kofac_b01/` | 재단 설문의 Walla DDI 내보내기와 정규 출력 | 표시 없음 |
| `testdata/kosac/` | 재단 작성 예시와 그 변환·검사 결과 | 표시 없음 |
| `testdata/walla_contract_v1/`, `testdata/walla_synthetic/` | Walla가 만든 합성 자료 | 이 도구의 라이선스를 따릅니다. |
| `testdata/archive/` | 이 프로젝트가 만든 합성 자료 | 이 도구의 라이선스를 따릅니다. |

`kosac/`의 테스트 코드에도 재단 작성 예시의 값 일부가 있습니다.

## 문서 사이트

아래 파일은 GitHub Pages로 배포하는 이 사이트에 들어갑니다.

| 구성 요소 | 버전 | 라이선스 | 저작권 | 들어가는 곳 |
|---|---|---|---|---|
| Hextra | v0.13.0 | `MIT` | Copyright (c) 2023 Xin | HTML 템플릿, JS, CSS, 파비콘, 로고 |
| Tailwind CSS | v4.3.0 | `MIT` | Copyright (c) Tailwind Labs, Inc. | CSS. Hextra가 미리 빌드했습니다. |
| modern-normalize | | `MIT` | Copyright (c) Nicolas Gallagher, Jonathan Neal, Sindre Sorhus | Tailwind의 기본 스타일 안 |
| Heroicons v1 | | `MIT` | Copyright (c) 2020 Refactoring UI Inc. | HTML의 SVG 아이콘, CSS, JS |
| Heroicons v2 | | `MIT` | Copyright (c) Tailwind Labs, Inc. | 맨 위로 이동 아이콘 |
| FlexSearch | v0.8.143 | `Apache-2.0` | Thomas Wilkerling | 검색 JS. Hugo 빌드가 jsDelivr에서 받아 이 사이트에서 제공합니다. |
| Chroma `github` 스타일 | | `MIT` | Copyright (C) 2017 Alec Thomas | 코드 강조 CSS |
| GitHub 마크 | | 상표 | GitHub, Inc. | 저장소 링크 아이콘. SVG 그림은 Nextra(`MIT`, Copyright (c) 2020 Shu Ding)에서 왔습니다. |

- GitHub 마크는 GitHub, Inc.의 상표입니다. 이 사이트는 GitHub 저장소로 가는 링크에만 씁니다.
- 파비콘과 로고 파일은 Hextra의 기본 파일입니다.

빌드에만 쓰는 도구는 배포물에 들어가지 않습니다.

| 도구 | 라이선스 |
|---|---|
| Hugo v0.157.0 extended | `Apache-2.0` |
| Go 툴체인 | `BSD-3-Clause` |
| GitHub Actions(`actions/checkout`, `actions/setup-go`, `actions/upload-pages-artifact`, `actions/deploy-pages`) | `MIT` |

## 확인이 필요한 항목

1. LGPL-3.0과 GPL-3.0 전문이 저장소에 없습니다. `codebook.xsd`와 DDI XHTML 스키마를 배포하려면 두 전문이 필요합니다.
2. W3C 고지 파일 `xhtml-copyright-1.xsd`가 저장소에 없습니다. XHTML 모듈 스키마 18개가 이 파일을 가리킵니다.
3. `xml.xsd`에는 라이선스 표시가 없습니다.
4. `internal/schema/table.go`가 LGPL 스키마의 2차적 저작물인지 정하지 않았습니다.
5. 재단 자료(양식, 작성 예시, 설문 내보내기, kosac XSD 초안)의 이용 조건이 저장소에 적혀 있지 않습니다.
6. CLI 바이너리나 npm 패키지를 배포하기 전에, 이 페이지의 고지와 라이선스 전문을 배포물에 넣어야 합니다.
7. 이 도구의 라이선스를 정해야 합니다.

## 라이선스 전문

| 라이선스 | 전문 |
|---|---|
| `Apache-2.0` | [Apache License 2.0](/licenses/Apache-2.0.txt) |
| `LGPL-3.0-or-later` | <https://www.gnu.org/licenses/lgpl-3.0.html> |
| `GPL-3.0` | <https://www.gnu.org/licenses/gpl-3.0.html> |
| `CC-BY-4.0` | <https://creativecommons.org/licenses/by/4.0/legalcode> |
| `W3C-20150513` | <https://www.w3.org/copyright/software-license-2023/> |
| `Unicode-3.0` | <https://www.unicode.org/license.txt> |
| Go `PATENTS` | <https://github.com/golang/go/blob/master/PATENTS> |
| `ISC` | <https://opensource.org/license/isc-license-txt> |

`MIT`와 `BSD-3-Clause`의 허락 문구는 아래와 같습니다. 각 구성 요소의 저작권 줄은 위 표에 있습니다.

{{% details title="MIT License" closed="true" %}}

```text
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

{{% /details %}}

{{% details title="BSD 3-Clause License" closed="true" %}}

```text
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of the copyright holder nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

{{% /details %}}

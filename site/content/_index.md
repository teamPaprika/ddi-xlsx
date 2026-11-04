---
title: ddi-xlsx
layout: hextra-home
---

{{< hextra/hero-badge link="https://github.com/teamPaprika/ddi-xlsx" >}}
  GitHub 저장소
{{< /hextra/hero-badge >}}

<div class="hx:mt-6 hx:mb-6">
{{< hextra/hero-headline >}}
  DDI 코드북을&nbsp;<br class="hx:sm:block hx:hidden" />스프레드시트에서 편집합니다
{{< /hextra/hero-headline >}}
</div>

<div class="hx:mb-12">
{{< hextra/hero-subtitle >}}
  DDI Codebook 2.5 XML·JSON을 Excel 워크북으로 바꾸고, Excel에서 고친 워크북을 다시 DDI로 바꿉니다.&nbsp;<br class="hx:sm:block hx:hidden" />열에 들어가지 않는 요소는 셀에 XML 원문으로 남깁니다.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx:mb-6">
{{< hextra/hero-button text="빠른 시작" link="docs/getting-started" >}}
{{< hextra/hero-button text="브라우저에서 실행" link="docs/playground" style="margin-left:.75rem;background:transparent;color:inherit;box-shadow:inset 0 0 0 1px currentColor" >}}
</div>

<div class="hx:mt-6"></div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="CLI"
    subtitle="macOS, Linux, Windows용 실행 파일입니다. 런타임을 따로 설치하지 않습니다."
    link="docs/cli"
  >}}
  {{< hextra/feature-card
    title="Go 라이브러리"
    subtitle="바이트를 받아 바이트를 돌려줍니다. 파일을 읽거나 쓰지 않습니다."
    link="docs/go"
  >}}
  {{< hextra/feature-card
    title="Node·브라우저 SDK"
    subtitle="WebAssembly 모듈로 같은 변환을 Node와 브라우저에서 실행합니다."
    link="docs/npm"
  >}}
  {{< hextra/feature-card
    title="재단 코드북 엔진"
    subtitle="재단 v3.2·v4.0 양식을 읽고 검사합니다. 표준 DDI XML·JSON을 만듭니다."
    link="docs/kosac"
  >}}
  {{< hextra/feature-card
    title="왕복 보장"
    subtitle="DDI 편집 워크북의 왕복 보장과 알려진 손실을 설명합니다."
    link="docs/roundtrip"
  >}}
  {{< hextra/feature-card
    title="기여하기"
    subtitle="개발 환경, 테스트, 문서 규칙, PR 절차를 설명합니다."
    link="docs/contributing"
  >}}
{{< /hextra/feature-grid >}}

# SEO · AEO · GEO 기반 구축 설계

> 상태: 설계 확정 (구현 대기). 작성일 2026-06-02.

## Goal

한 번의 작업으로 세 검색 표면의 기술 기반을 Quotient 웹에 심는다.

- **SEO** (전통 검색 / Google·Naver): 메타데이터·사이트맵·canonical로 순위와 리치스니펫.
- **AEO** (답변 엔진 / Google AI Overview·Bing): 구조화 데이터로 답변 박스에 우리 문장이 발췌되게.
- **GEO** (생성형 엔진 / ChatGPT·Claude·Perplexity): AI 봇 크롤 허용 + 인용 가치 있는 구조화 콘텐츠로 생성 답변의 출처에 등장.

새 콘텐츠를 쓰지 않는다. **기존 랜딩 콘텐츠를 기계가 읽을 수 있게 구조화**하는 것이 이번 범위다.

## 범위

| | 포함 (IN) | 제외 (OUT) |
|---|---|---|
| 메타 | metadataBase, title 템플릿, description, OG, Twitter 카드, canonical, robots, verification | — |
| 크롤 | `robots.ts`(AI봇 허용), `sitemap.ts`, `manifest.ts` | — |
| 구조화 | JSON-LD: Organization·WebSite·SoftwareApplication·FAQPage | 유료 Offer/Product 스키마(Pro 미출시), Review/Rating |
| 이미지 | `opengraph-image.tsx`(영문), `icon.tsx` | 종목별 동적 OG, 페이지별 OG |
| 콘텐츠 | 기존 랜딩 FAQ를 FAQPage로 구조화만 | 신규 콘텐츠(블로그·비교글·용어집) = ② 콘텐츠 엔진, 별도 안건 |
| 국제화 | 한국어 단일 (`ko_KR`) | 다국어/hreflang |

## 정체성·규제 정합성 (필수 확인)

`docs/superpowers/specs/2026-05-28-identity-3-pillars.md` 및 CLAUDE.md 규제 원칙과 충돌 없음:

- JSON-LD `SoftwareApplication`은 **"분석 도구"**로만 표기. `applicationCategory: FinanceApplication`은 카테고리 분류일 뿐, 투자자문·매매추천 표현은 description에 넣지 않는다.
- 수익률 랭킹·자금 보관·마이데이터 등 규제 위반 기능을 구조화 데이터로도 노출하지 않는다.
- description 카피는 랜딩 "안 하는 것을 분명히" 톤과 일치 — "분석/도구"는 OK, "수익 보장/추천"은 금지.

## 아키텍처 — 세 층 + 단일 도메인 소스

```
lib/seo.ts  ── siteUrl 단일 소스 ──┐
                                    ├─→ app/layout.tsx        (전역 메타 + Organization·WebSite JSON-LD)
                                    ├─→ app/page.tsx          (SoftwareApplication·FAQPage JSON-LD)
                                    ├─→ app/{pricing,privacy,terms}/page.tsx (페이지별 메타)
                                    ├─→ app/robots.ts         (크롤 규칙 + sitemap 링크)
                                    ├─→ app/sitemap.ts        (공개 라우트)
                                    ├─→ app/manifest.ts       (PWA)
                                    └─→ app/opengraph-image.tsx, app/icon.tsx (next/og)
components/JsonLd.tsx ── JSON-LD 주입 헬퍼 (모든 스키마가 공용)
```

### 1. 도메인 기준점 — `lib/seo.ts`

**Why**: 도메인이 여러 파일(메타·사이트맵·robots·JSON-LD)에 흩어지면 도메인 교체 시 누락이 생긴다. 단일 함수로 모은다.

```ts
// 해석 우선순위: 명시 env → Vercel 자동 → 로컬
export function siteUrl(): string {
  if (process.env.NEXT_PUBLIC_SITE_URL) return process.env.NEXT_PUBLIC_SITE_URL;
  if (process.env.VERCEL_PROJECT_PRODUCTION_URL)
    return `https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`;
  return "http://localhost:3000";
}
export const SITE_NAME = "Quotient";
export const SITE_DESC = "한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.";
```

도메인 구매 후엔 Vercel에 `NEXT_PUBLIC_SITE_URL` 한 줄만 추가하면 전 파일이 따라온다.

### 2. 메타데이터

**루트 `app/layout.tsx`** — `metadata` 객체 확장 (현재 title·description만 있음):

- `metadataBase: new URL(siteUrl())` — OG·canonical 상대경로의 절대화 기준
- `title: { default: "Quotient — Portfolio Intelligence Terminal", template: "%s · Quotient" }`
- `description`, `applicationName: "Quotient"`, `keywords`(한국 주식·미국 주식·포트폴리오·자산 분석·AI 분석 등), `authors`/`creator`
- `openGraph`: `type: website`, `locale: "ko_KR"`, `siteName`, `title`, `description`, `url: "/"` (이미지는 `opengraph-image.tsx`가 자동 주입)
- `twitter`: `card: "summary_large_image"`, `title`, `description`
- `robots`: `{ index: true, follow: true, googleBot: { index:true, follow:true, "max-image-preview":"large" } }`
- `alternates: { canonical: "/" }`
- `verification`: **조건부 구성**. env가 없을 때 `undefined`를 그대로 넘기면 빈 `<meta>`가 출력될 수 있으므로, 토큰이 존재하는 키만 담아 객체를 만든다:
  ```ts
  const verification: Metadata["verification"] = {};
  if (process.env.NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION)
    verification.google = process.env.NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION;
  if (process.env.NEXT_PUBLIC_NAVER_SITE_VERIFICATION)
    verification.other = { "naver-site-verification": process.env.NEXT_PUBLIC_NAVER_SITE_VERIFICATION };
  // 비어 있으면 metadata.verification 자체를 생략
  ```
- `icons`/`manifest`는 파일 컨벤션(`icon.tsx`/`manifest.ts`)으로 자동 연결

**페이지별** (`/`, `/pricing`, `/privacy`, `/terms` — 4곳 모두 서버 컴포넌트 확인 완료):
- 각 `export const metadata`에 고유 `title`(템플릿이 ` · Quotient` 자동 부착), `description`, `alternates.canonical`.
- 예: `/pricing` → `title: "요금제"`, `/privacy` → `title: "개인정보 처리방침"`, `/terms` → `title: "이용약관"`. 루트 `/`는 default title 유지.

### 3. 크롤 규칙

**`app/robots.ts`** — 함수형 `MetadataRoute.Robots` 반환:

```
rules:
  - userAgent: "*"            allow: "/"   disallow: [/app, /login, /signup,
                                                       /forgot-password, /reset-password,
                                                       /verify-email, /preview, /api]
  - userAgent: [GPTBot, OAI-SearchBot, ChatGPT-User, ClaudeBot, Claude-Web,
                anthropic-ai, PerplexityBot, Google-Extended, CCBot]
                allow: "/"   disallow: [/app, /api, /login, /signup,
                                         /forgot-password, /reset-password, /verify-email]
sitemap: `${siteUrl()}/sitemap.xml`
host:    siteUrl()
```

**Why AI봇 명시 허용**: GEO의 전제 조건. 이 봇들이 공개 페이지를 못 읽으면 생성형 답변에 인용될 수 없다. 단, 인증/대시보드(`/app`)는 개인 데이터이므로 모든 봇에 차단.

**`app/sitemap.ts`** — 공개 라우트만 (`MetadataRoute.Sitemap`):
`/`(priority 1.0, weekly), `/pricing`(0.8, monthly), `/privacy`(0.3, yearly), `/terms`(0.3, yearly). `lastModified`는 빌드 시각.

**`app/manifest.ts`** — `MetadataRoute.Manifest`: `name: "Quotient — Portfolio Intelligence Terminal"`, `short_name: "Quotient"`, `description`, `start_url: "/"`, `display: "standalone"`, `background_color`/`theme_color`, `icons`(생성된 icon 참조). *색상값은 `apps/web/app/globals.css`의 `--bg` 토큰을 구현 시 확인해 일치시킨다(`#0a0a0a`는 가정값).*

### 4. 구조화 데이터 (JSON-LD)

**`components/JsonLd.tsx`** — 타입드 주입 헬퍼. **Why 컴포넌트화**: 스키마가 4종이고 페이지마다 다르게 조합된다. 인라인 `<script>` 반복 대신 한 곳에서 직렬화·이스케이프를 관리.

```tsx
export function JsonLd({ data }: { data: Record<string, unknown> }) {
  return <script type="application/ld+json"
    dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }} />;
}
```

**전역 (layout.tsx)** — `Organization` + `WebSite`:

```jsonc
// Organization
{ "@context":"https://schema.org","@type":"Organization",
  "name":"Quotient","url":"<siteUrl>","logo":"<siteUrl>/brand-512.png",
  "description":"한국·미국 자산 통합 분석 + 자연어 AI 분석가 인터페이스.",
  "email":"sdl182975@gmail.com" }
// WebSite
{ "@context":"https://schema.org","@type":"WebSite",
  "name":"Quotient","url":"<siteUrl>",
  "inLanguage":"ko-KR" }
```
*SearchAction은 사이트 내 검색이 없으므로 넣지 않는다.*

**logo URL 주의 (Important)**: Google Organization `logo`는 **정사각형(≥112×112)**을 요구한다. 32×32 `icon`은 너무 작고, 1200×630 `opengraph-image`는 와이드라 둘 다 부적합. 별도로 **512×512 정사각 브랜드 이미지**를 `next/og`로 생성한다 — `icon.tsx`와 같은 렌더(검정 배경 + `#FFD500` "Q")를 크기만 키워 `app/brand-512/route.tsx` 같은 안정 경로로 노출하고, 그 절대 URL을 logo에 넣는다. 파일 컨벤션 `icon` 라우트는 빌드 해시가 붙어 URL이 불안정할 수 있으므로 logo로 직접 참조하지 않는다.

**랜딩 (page.tsx)** — `SoftwareApplication` + `FAQPage`:

```jsonc
// SoftwareApplication — "분석 도구"로 표기, 자문 표현 회피
{ "@context":"https://schema.org","@type":"SoftwareApplication",
  "name":"Quotient","applicationCategory":"FinanceApplication",
  "operatingSystem":"Web","url":"<siteUrl>",
  "description":"한국·미국 자산을 한 화면에서 분석하고 자연어로 질문하는 개인용 포트폴리오 분석 도구.",
  "offers":{"@type":"Offer","price":"0","priceCurrency":"KRW"},
  "featureList":["통합 포트폴리오 분석","AI 자연어 분석가","마켓 모니터","매일 아침 브리핑"] }
```

`FAQPage`는 랜딩 FAQ 4개 Q&A를 **그대로** 구조화 (`app/page.tsx`의 `FAQ()` 데이터와 1:1):

1. Q "보유 자산은 어떻게 검증하나요?" / A "검증하지 않습니다. 사용자가 직접 입력하며…"
2. Q "다른 핀테크(토스·뱅크샐러드)와 차이는?" / A "그쪽은 마이데이터로 통합 자산 관리…"
3. Q "AI는 어떤 모델인가요? 비용은?" / A "Anthropic Claude 기반입니다. 지금은 무료로 월 30회까지…"
4. Q "Paper Trading은 언제 나오나요?" / A "정식 출시 이후 순차적으로 공개할 예정입니다…"

**Why FAQPage가 핵심**: 이 4개 Q&A는 SEO(구글 FAQ 리치스니펫) + AEO(답변 박스 발췌) + GEO(AI 인용)에 동시에 먹히는 가장 강한 단일 자산이다. **단일 소스 원칙**: FAQ 텍스트를 `lib/faq.ts`로 추출해 `page.tsx`의 `<details>` 렌더와 `FAQPage` JSON-LD가 같은 배열을 참조 → 본문과 구조화 데이터의 불일치(구글 정책 위반) 방지.

### 5. 이미지 (코드 생성, `next/og`)

**`app/opengraph-image.tsx`** — `ImageResponse`, `size {width:1200,height:630}`, `contentType:"image/png"`. 영문 카피 (한글 폰트 로딩 회피):
- 배경 `#0a0a0a`, 상단 모노 태그 `PORTFOLIO · INTELLIGENCE · TERMINAL`(`#737373`), 브랜드 `QUOTIENT`(`#FFD500`), 영문 태그라인(`#e5e5e5`), 하단 3색 라인(`#FFD500`/`#00FFFF`/`#00FF7F`), 우하단 `quotient.app`.
- **제약**: `ImageResponse`는 flexbox만 지원(grid 불가), CSS 부분집합만. 레이아웃은 flex column으로.
- 시스템 기본 폰트(Latin)로 충분 — 별도 폰트 번들 없음.
- `export const alt = "Quotient — Portfolio Intelligence Terminal"` 도 함께 export(접근성·`og:image:alt`).
- **runtime 주의 (Important)**: `next/og`의 `ImageResponse`는 Next 버전에 따라 `export const runtime = "edge"`가 필요할 수 있다. 포크 문서(`opengraph-image`/`image-response`)에서 런타임 요구를 **구현 시 확인**한다.
- **Twitter 이미지**: `twitter-image.tsx`를 따로 두지 않고 `opengraph-image`에 의존. 대부분 크롤러는 `twitter:image` 미지정 시 `og:image`로 폴백한다. 명시적 `twitter:image`가 필요하면 동일 렌더를 재export하는 `app/twitter-image.tsx`를 추가(선택).

**`app/icon.tsx`** — `ImageResponse`, 32×32, `#0a0a0a` 배경에 `#FFD500` "Q" 모노그램. 기존 `favicon.ico`는 레거시 폴백으로 유지(공존 가능). `brand-512`(logo용)와 같은 디자인을 크기만 다르게 공유 — 렌더 함수를 `lib/brand-mark.tsx`로 추출해 DRY.

## 검증 방법 (구현 후 스모크)

1. 빌드 후 라우트 존재 확인: `/robots.txt`, `/sitemap.xml`, `/manifest.webmanifest`, `/opengraph-image`, `/icon`.
2. **Google Rich Results Test**에 배포 URL 입력 → FAQPage·Organization·SoftwareApplication 인식 확인.
3. **카톡/슬랙에 링크 붙여넣어** OG 카드 렌더 육안 확인.
4. `robots.txt` 응답에 AI봇 그룹 + sitemap 라인 포함 확인.
5. 각 공개 페이지 `view-source`에서 `<title>`·canonical·JSON-LD `<script>` 확인.
6. `docs/E2E_SMOKE.md`에 위 항목 추가.

## USER_ACTIONS 추가분 (구현 시 `docs/USER_ACTIONS.md` 등재)

| 액션 | 필수성 | 비고 |
|---|---|---|
| Vercel에 `NEXT_PUBLIC_SITE_URL` 설정 | 도메인 구매 후 필수 | 미설정 시 Vercel 자동 URL 사용 |
| Google Search Console 등록 + 사이트맵 제출 | 권장(SEO 핵심) | `NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION` 토큰 |
| Naver 서치어드바이저 등록 | 권장(한국 타겟) | `NEXT_PUBLIC_NAVER_SITE_VERIFICATION` 토큰 |
| Bing Webmaster Tools | 선택 | AEO(Bing) 보강 |

## 구현 전제 (Critical)

- 현재 루트 `node_modules`가 비어 있다(워크스페이스 호이스팅된 `next` 제거됨). 구현 진입 전 `npm install` 필요.
- 설치 후 **포크 Next 문서**(`node_modules/next/dist/docs/`)에서 `sitemap`·`robots`·`opengraph-image`·`manifest`·`metadata`의 정확한 export 시그니처·타입을 **재확인**한다 (`apps/web/AGENTS.md`: "This is NOT the Next.js you know"). 본 스펙의 시그니처는 설계 기준이며, 구현 시 포크 문서가 최종 권위.

## 비범위 재확인

블로그·비교글·용어집 같은 신규 콘텐츠(② 콘텐츠 엔진)는 이번에 하지 않는다. AEO·GEO의 장기 성과는 결국 "인용할 가치가 있는 콘텐츠"가 좌우하므로 별도 안건으로 다룬다.

## 검토 이력

### 2026-06-02 — 작성 + 1차 자체 검토

발견 이슈와 처리:

**Critical** — 없음 (스펙대로 구현 시 빌드/동작이 깨지는 항목 없음. 단 "구현 전제"의 `npm install` + 포크 문서 재확인은 Critical 전제로 명시).

**Important**
1. `verification` env가 `undefined`일 때 그대로 넘기면 빈 `<meta>` 출력 위험 → §2를 **조건부 객체 구성**으로 패치.
2. Organization `logo`를 `/icon`(32×32)으로 둔 것은 Google 정사각 ≥112px 요구 위반 + 파일 컨벤션 URL 해시 불안정 → **512×512 정사각 브랜드 이미지(`brand-512`)** 별도 생성으로 패치. `icon`/`brand-512`/`opengraph-image`는 `lib/brand-mark.tsx` 공용 렌더로 DRY.
3. `opengraph-image` 런타임 요구(`runtime="edge"` 가능성)가 포크 버전 의존 → §5에 구현 시 확인 명시.

**Minor**
4. Twitter 이미지 폴백 동작 명시(og:image 폴백, 필요 시 `twitter-image.tsx` 선택) → §5 보강.
5. `opengraph-image` `alt` export 누락 → §5 추가.
6. manifest 색상값(`#0a0a0a`)이 가정값 → `globals.css --bg` 확인 단서 추가.
7. `keywords`는 구글 랭킹에 무효(완전성 목적) — 의도 명확화는 plan 단계에서 처리.

처리: 위 1~6 스펙 직접 패치 완료. 7은 비범위 주석으로 충분.

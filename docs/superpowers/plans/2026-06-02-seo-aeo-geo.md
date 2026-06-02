# SEO · AEO · GEO Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 한 번의 배포로 검색엔진(SEO)·답변엔진(AEO)·생성엔진(GEO) 세 축에서 Quotient가 노출·인용되도록 메타데이터·구조화 데이터·크롤링 정책·OG/파비콘 자산을 코드로 구축한다.

**Architecture:** 도메인·카피·브랜드 색을 `lib/seo.ts` 단일 소스로 모으고, 순수 함수(`buildRootMetadata`/`pageMetadata`/`*JsonLd`)가 직렬화 가능한 객체를 반환하도록 설계해 vitest로 단위 검증한다. 파일 컨벤션(`app/robots.ts`·`app/sitemap.ts`·`app/manifest.ts`·`app/opengraph-image.tsx`·`app/icon.tsx`)은 Next 포크가 자동 라우팅한다. 이미지 라우트(`next/og`)는 vitest 대신 `next build` + 수동 fetch로 검증한다.

**Tech Stack:** Next.js 16.2.6 (포크), React 19.2.4, TypeScript strict, `next/og` (Satori/ImageResponse), vitest 4.1.7 + @testing-library/react (jsdom).

**커밋 규칙:** 모든 커밋 메시지는 다음 트레일러로 끝낸다.
```
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

**참조 스펙:** `docs/superpowers/specs/2026-06-02-seo-aeo-geo-design.md`

---

## Setup / Preconditions (코드 작성 전 필수)

> ⚠️ 이 단계를 건너뛰면 이후 모든 Task가 실패한다. 워크스페이스 루트의 `node_modules`가 제거된 상태이며 `next`와 포크 문서가 모두 없다.

- [ ] **S1: 의존성 복원**

Run (레포 루트 `/Users/yuhojin/Desktop/finance`):
```bash
npm install
```
Expected: `next`·`react`·`vitest` 등이 복원되고 `apps/web/node_modules` 혹은 hoisted root에 `next/dist/docs/`가 다시 생성됨.

- [ ] **S2: 포크 문서로 파일 컨벤션 시그니처 확인**

이 Next는 학습 데이터의 Next가 아니다. 코드 작성 전 아래 문서를 읽고 정확한 export 시그니처를 확인한다.
```bash
find /Users/yuhojin/Desktop/finance -path '*/next/dist/docs/*' -name '*.md' | grep -Ei 'sitemap|robots|opengraph|image-response|manifest|metadata|icon'
```
확인할 것:
- `sitemap.ts` / `robots.ts` 의 반환 타입(`MetadataRoute.Sitemap` / `MetadataRoute.Robots`)과 필드명(`changeFrequency`, `userAgent`, `disallow` 등)
- `opengraph-image.tsx` / `icon.tsx` 가 `export const runtime`(예: `"edge"`)를 요구하는지, `size`/`contentType`/`alt` export 규약
- `manifest.ts` 의 `MetadataRoute.Manifest` 필드(`theme_color`, `background_color`, `display`, `icons[].sizes`)
- `generateMetadata` vs 정적 `metadata` 의 `Metadata` 타입 필드(`metadataBase`, `alternates.canonical`, `robots`, `verification`, `openGraph`, `twitter`)

만약 포크 시그니처가 이 계획의 드래프트 코드와 다르면 **드래프트가 아니라 포크 문서를 따른다**. 차이를 발견하면 해당 Task 구현 시 반영한다.

- [ ] **S3: 작업 브랜치 확인**

Run:
```bash
git rev-parse --abbrev-ref HEAD
```
현재 `master`. 별도 worktree/브랜치 지시가 없으면 이 브랜치에서 진행하되, 첫 구현 커밋 전 사용자에게 커밋 시점을 확인한다(이 레포는 명시적 요청 없이는 커밋 금지).

---

## File Structure

신규 생성:
- `apps/web/lib/seo.ts` — 도메인·카피·브랜드 색 단일 소스 + `buildRootMetadata()` / `pageMetadata()`
- `apps/web/lib/faq.ts` — FAQ 항목 단일 소스 (랜딩 렌더 + FAQPage JSON-LD 공용)
- `apps/web/lib/jsonld.ts` — Organization / WebSite / SoftwareApplication / FAQPage JSON-LD 빌더
- `apps/web/lib/brand-mark.tsx` — `SquareMark` 공용 브랜드 렌더(아이콘·512 공용)
- `apps/web/components/seo/JsonLd.tsx` — `<script type="application/ld+json">` 주입 컴포넌트
- `apps/web/app/robots.ts` — 크롤링 정책(일반 + AI 봇 허용)
- `apps/web/app/sitemap.ts` — 사이트맵(공개 4개 경로)
- `apps/web/app/manifest.ts` — PWA/브랜드 매니페스트
- `apps/web/app/icon.tsx` — 32×32 파비콘(`next/og`)
- `apps/web/app/brand-512/route.tsx` — 512×512 로고(JSON-LD `logo` + manifest 용)
- `apps/web/app/opengraph-image.tsx` — 1200×630 OG 카드(영문 카피)

테스트 신규:
- `apps/web/lib/seo.test.ts`, `apps/web/lib/faq.test.ts`, `apps/web/lib/jsonld.test.ts`
- `apps/web/app/robots.test.ts`, `apps/web/app/sitemap.test.ts`, `apps/web/app/manifest.test.ts`
- `apps/web/components/seo/JsonLd.test.tsx`, `apps/web/lib/brand-mark.test.tsx`

수정:
- `apps/web/app/layout.tsx` — `metadata = buildRootMetadata()` + Organization/WebSite JSON-LD 렌더
- `apps/web/app/page.tsx` — `metadata = pageMetadata({path:"/"})` + SoftwareApplication/FAQPage JSON-LD + FAQ를 `FAQ_ITEMS`로 치환
- `apps/web/app/pricing/page.tsx`, `apps/web/app/privacy/page.tsx`, `apps/web/app/terms/page.tsx` — `pageMetadata()`로 치환(제목 중복 "— Quotient" 제거)

> 모든 테스트는 `apps/web`에서 실행: `cd apps/web && npx vitest run <path>`. `npm test`는 watch 모드이므로 단발 검증엔 `npx vitest run`을 쓴다.

---

## Task 1: 도메인·메타데이터 단일 소스 (`lib/seo.ts`)

**Files:**
- Create: `apps/web/lib/seo.ts`
- Test: `apps/web/lib/seo.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// apps/web/lib/seo.test.ts
import { afterEach, describe, expect, it, vi } from "vitest";
import { buildRootMetadata, pageMetadata, siteUrl, SITE_NAME } from "./seo";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("siteUrl", () => {
  it("prefers NEXT_PUBLIC_SITE_URL", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://quotient.app");
    vi.stubEnv("VERCEL_PROJECT_PRODUCTION_URL", "ignored.vercel.app");
    expect(siteUrl()).toBe("https://quotient.app");
  });

  it("falls back to VERCEL_PROJECT_PRODUCTION_URL with https", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "");
    vi.stubEnv("VERCEL_PROJECT_PRODUCTION_URL", "quotient.vercel.app");
    expect(siteUrl()).toBe("https://quotient.vercel.app");
  });

  it("falls back to localhost when no env", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "");
    vi.stubEnv("VERCEL_PROJECT_PRODUCTION_URL", "");
    expect(siteUrl()).toBe("http://localhost:3000");
  });
});

describe("buildRootMetadata", () => {
  it("sets metadataBase from siteUrl and title template", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://quotient.app");
    const m = buildRootMetadata();
    expect(m.metadataBase?.toString()).toBe("https://quotient.app/");
    expect(m.title).toEqual({
      default: "Quotient — Portfolio Intelligence Terminal",
      template: "%s · Quotient",
    });
    expect(m.robots).toMatchObject({ index: true, follow: true });
  });

  it("omits verification when env tokens absent", () => {
    vi.stubEnv("NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION", "");
    vi.stubEnv("NEXT_PUBLIC_NAVER_SITE_VERIFICATION", "");
    const m = buildRootMetadata();
    expect(m.verification).toBeUndefined();
  });

  it("includes verification when token present", () => {
    vi.stubEnv("NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION", "g-token");
    vi.stubEnv("NEXT_PUBLIC_NAVER_SITE_VERIFICATION", "");
    const m = buildRootMetadata();
    expect(m.verification?.google).toBe("g-token");
  });

  it("does NOT set a root canonical (avoids leaking to private routes)", () => {
    const m = buildRootMetadata();
    expect(m.alternates?.canonical).toBeUndefined();
  });
});

describe("pageMetadata", () => {
  it("sets per-page canonical and og url from path", () => {
    const m = pageMetadata({ path: "/pricing", title: "가격" });
    expect(m.alternates?.canonical).toBe("/pricing");
    expect(m.title).toBe("가격");
    expect(m.openGraph?.url).toBe("/pricing");
  });

  it("uses SITE_NAME constant", () => {
    expect(SITE_NAME).toBe("Quotient");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/seo.test.ts`
Expected: FAIL — `Cannot find module './seo'` (또는 export 미정의).

- [ ] **Step 3: Write minimal implementation**

```ts
// apps/web/lib/seo.ts
import type { Metadata } from "next";

export const SITE_NAME = "Quotient";
export const SITE_TITLE_DEFAULT = "Quotient — Portfolio Intelligence Terminal";
export const SITE_DESC =
  "한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.";

export const OG_EYEBROW = "PORTFOLIO · INTELLIGENCE · TERMINAL";
export const OG_TAGLINE_EN =
  "Korean + US assets on one screen. Ask in plain language.";

export const KEYWORDS = [
  "한국 주식",
  "미국 주식",
  "포트폴리오 분석",
  "자산 관리",
  "KOSPI",
  "NASDAQ",
  "AI 투자 분석",
  "환율",
  "포트폴리오 인텔리전스",
];

export const BRAND = {
  bg: "#0A0A0A",
  bgSubtle: "#111111",
  fg: "#E5E5E5",
  muted: "#737373",
  line: "#262626",
  accent: "#FFD500",
  info: "#00FFFF",
  up: "#00FF7F",
} as const;

export function siteUrl(): string {
  if (process.env.NEXT_PUBLIC_SITE_URL) return process.env.NEXT_PUBLIC_SITE_URL;
  if (process.env.VERCEL_PROJECT_PRODUCTION_URL)
    return `https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`;
  return "http://localhost:3000";
}

export function buildRootMetadata(): Metadata {
  const base = siteUrl();
  const google = process.env.NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION;
  const naver = process.env.NEXT_PUBLIC_NAVER_SITE_VERIFICATION;
  const verification =
    google || naver
      ? {
          ...(google ? { google } : {}),
          ...(naver ? { other: { "naver-site-verification": naver } } : {}),
        }
      : undefined;

  return {
    metadataBase: new URL(base),
    title: {
      default: SITE_TITLE_DEFAULT,
      template: "%s · Quotient",
    },
    description: SITE_DESC,
    applicationName: SITE_NAME,
    keywords: KEYWORDS,
    authors: [{ name: SITE_NAME }],
    creator: SITE_NAME,
    openGraph: {
      type: "website",
      locale: "ko_KR",
      siteName: SITE_NAME,
      title: SITE_TITLE_DEFAULT,
      description: SITE_DESC,
      url: "/",
    },
    twitter: {
      card: "summary_large_image",
      title: SITE_TITLE_DEFAULT,
      description: SITE_DESC,
    },
    robots: {
      index: true,
      follow: true,
      googleBot: {
        index: true,
        follow: true,
        "max-image-preview": "large",
        "max-snippet": -1,
      },
    },
    ...(verification ? { verification } : {}),
  };
}

export function pageMetadata(opts: {
  path: string;
  title?: string;
  description?: string;
}): Metadata {
  const { path, title, description } = opts;
  return {
    ...(title ? { title } : {}),
    ...(description ? { description } : {}),
    alternates: { canonical: path },
    openGraph: {
      url: path,
      ...(title ? { title } : {}),
      ...(description ? { description } : {}),
    },
  };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run lib/seo.test.ts`
Expected: PASS (all cases).

> 포크 주의: `Metadata` 타입에서 `verification.other`, `robots.googleBot["max-image-preview"]` 필드명이 다르면 S2에서 읽은 포크 문서를 따른다. 타입 에러 시 `npx tsc --noEmit`로 확인.

> env 테스트가 유효한 이유: vitest(esbuild/`@vitejs/plugin-react`)는 Next 빌드와 달리 `NEXT_PUBLIC_*`를 **정적 인라인하지 않으므로** `siteUrl()` 내부의 `process.env.NEXT_PUBLIC_SITE_URL`는 런타임 조회로 남고 `vi.stubEnv`가 그대로 먹는다. `stubEnv(key, "")`는 빈 문자열(falsy)을 넣어 `if (process.env.X)` 폴백 분기를 정확히 탄다. 즉 이 테스트는 vitest 한정으로 타당하며, 실제 배포에선 `NEXT_PUBLIC_SITE_URL`이 빌드 시 인라인된다(아래 Task 10 모듈-평가 시점 주의 참조).

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/seo.ts apps/web/lib/seo.test.ts
git commit -m "$(cat <<'EOF'
feat(web): SEO 도메인·메타데이터 단일 소스 (lib/seo)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: FAQ 단일 소스 (`lib/faq.ts`)

**Files:**
- Create: `apps/web/lib/faq.ts`
- Test: `apps/web/lib/faq.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// apps/web/lib/faq.test.ts
import { describe, expect, it } from "vitest";
import { FAQ_ITEMS } from "./faq";

describe("FAQ_ITEMS", () => {
  it("has 4 items, each with non-empty q and a", () => {
    expect(FAQ_ITEMS).toHaveLength(4);
    for (const item of FAQ_ITEMS) {
      expect(item.q.length).toBeGreaterThan(0);
      expect(item.a.length).toBeGreaterThan(0);
    }
  });

  it("includes the asset-verification question", () => {
    expect(FAQ_ITEMS.some((i) => i.q.includes("검증"))).toBe(true);
  });

  it("includes the Paper Trading question", () => {
    expect(FAQ_ITEMS.some((i) => i.q.includes("Paper Trading"))).toBe(true);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/faq.test.ts`
Expected: FAIL — `Cannot find module './faq'`.

- [ ] **Step 3: Write minimal implementation**

```ts
// apps/web/lib/faq.ts
export interface FaqItem {
  q: string;
  a: string;
}

export const FAQ_ITEMS: FaqItem[] = [
  {
    q: "보유 자산은 어떻게 검증하나요?",
    a: "검증하지 않습니다. 사용자가 직접 입력하며 본인 분석 도구로 제공합니다. 다른 사용자에게 노출되지 않으므로 본인 정확도가 곧 분석 정확도입니다.",
  },
  {
    q: "다른 핀테크(토스·뱅크샐러드)와 차이는?",
    a: "그쪽은 마이데이터로 통합 자산 관리. 우리는 라이선스 없이 분석에만 집중 — 정보 밀도가 높은 모노스페이스 UI, 자연어 AI 분석가, 한국+미국 자산 한 화면. 개발자·파워유저 타겟.",
  },
  {
    q: "AI는 어떤 모델인가요? 비용은?",
    a: "Anthropic Claude 기반입니다. 지금은 무료로 월 30회까지 쓸 수 있고, 이후 Pro 플랜으로 한도를 넓힐 예정입니다.",
  },
  {
    q: "Paper Trading은 언제 나오나요?",
    a: "정식 출시 이후 순차적으로 공개할 예정입니다. 가상 자금 + 백테스트 + AI 매매 일기를 한 번에 제공합니다.",
  },
];
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run lib/faq.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/faq.ts apps/web/lib/faq.test.ts
git commit -m "$(cat <<'EOF'
feat(web): FAQ 단일 소스 (lib/faq) — 랜딩·FAQPage 공용

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: JSON-LD 빌더 (`lib/jsonld.ts`)

**Files:**
- Create: `apps/web/lib/jsonld.ts`
- Test: `apps/web/lib/jsonld.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// apps/web/lib/jsonld.test.ts
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  faqPageJsonLd,
  organizationJsonLd,
  softwareApplicationJsonLd,
  webSiteJsonLd,
} from "./jsonld";
import { FAQ_ITEMS } from "./faq";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("organizationJsonLd", () => {
  it("is an Organization with absolute logo >=112px source", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://quotient.app");
    const o = organizationJsonLd();
    expect(o["@type"]).toBe("Organization");
    expect(o.name).toBe("Quotient");
    expect(o.logo).toBe("https://quotient.app/brand-512");
    expect(o.url).toBe("https://quotient.app");
  });
});

describe("webSiteJsonLd", () => {
  it("is a WebSite in Korean", () => {
    const w = webSiteJsonLd();
    expect(w["@type"]).toBe("WebSite");
    expect(w.inLanguage).toBe("ko-KR");
  });
});

describe("softwareApplicationJsonLd", () => {
  it("is a free FinanceApplication with no advisory wording", () => {
    const s = softwareApplicationJsonLd();
    expect(s["@type"]).toBe("SoftwareApplication");
    expect(s.applicationCategory).toBe("FinanceApplication");
    expect(s.offers).toMatchObject({ price: "0", priceCurrency: "KRW" });
    // 규제 정합성: 자문·추천·수익 보장 문구 금지
    expect(JSON.stringify(s)).not.toMatch(/추천|자문|수익 보장|수익률 보장/);
  });
});

describe("faqPageJsonLd", () => {
  it("maps every FAQ item to a Question/Answer", () => {
    const f = faqPageJsonLd();
    expect(f["@type"]).toBe("FAQPage");
    expect(f.mainEntity).toHaveLength(FAQ_ITEMS.length);
    expect(f.mainEntity[0]).toMatchObject({
      "@type": "Question",
      name: FAQ_ITEMS[0].q,
      acceptedAnswer: { "@type": "Answer", text: FAQ_ITEMS[0].a },
    });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/jsonld.test.ts`
Expected: FAIL — `Cannot find module './jsonld'`.

- [ ] **Step 3: Write minimal implementation**

```ts
// apps/web/lib/jsonld.ts
import { FAQ_ITEMS } from "./faq";
import { SITE_DESC, SITE_NAME, siteUrl } from "./seo";

export function organizationJsonLd() {
  const url = siteUrl();
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: SITE_NAME,
    url,
    logo: `${url}/brand-512`,
    email: "sdl182975@gmail.com",
    description: SITE_DESC,
  };
}

export function webSiteJsonLd() {
  const url = siteUrl();
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: SITE_NAME,
    url,
    inLanguage: "ko-KR",
    description: SITE_DESC,
  };
}

export function softwareApplicationJsonLd() {
  const url = siteUrl();
  return {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: SITE_NAME,
    url,
    applicationCategory: "FinanceApplication",
    operatingSystem: "Web",
    description:
      "한국·미국 자산을 한 화면에서 분석하고 자연어로 질문하는 개인용 포트폴리오 분석 도구.",
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "KRW",
    },
    featureList: [
      "통합 포트폴리오 분석",
      "AI 자연어 분석가",
      "마켓 모니터",
      "매일 아침 브리핑",
    ],
  };
}

export function faqPageJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: FAQ_ITEMS.map((item) => ({
      "@type": "Question",
      name: item.q,
      acceptedAnswer: {
        "@type": "Answer",
        text: item.a,
      },
    })),
  };
}
```

> 스펙 정합(중요): `softwareApplicationJsonLd`의 `description`·`featureList`는 스펙 §4(`2026-06-02-seo-aeo-geo-design.md` 147–153행)의 박제 카피를 **그대로** 사용한다. 임의 문구로 바꾸지 않는다. `organizationJsonLd`/`webSiteJsonLd`만 공용 `SITE_DESC`를 쓴다.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run lib/jsonld.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/jsonld.ts apps/web/lib/jsonld.test.ts
git commit -m "$(cat <<'EOF'
feat(web): JSON-LD 빌더 (Organization·WebSite·SoftwareApplication·FAQPage)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: JSON-LD 주입 컴포넌트 (`components/seo/JsonLd.tsx`)

**Files:**
- Create: `apps/web/components/seo/JsonLd.tsx`
- Test: `apps/web/components/seo/JsonLd.test.tsx`

- [ ] **Step 1: Write the failing test**

```tsx
// apps/web/components/seo/JsonLd.test.tsx
import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { JsonLd } from "./JsonLd";

describe("JsonLd", () => {
  it("renders a ld+json script with serialized data", () => {
    const data = { "@type": "WebSite", name: "Quotient" };
    const { container } = render(<JsonLd data={data} />);
    const script = container.querySelector(
      'script[type="application/ld+json"]',
    );
    expect(script).not.toBeNull();
    expect(JSON.parse(script!.innerHTML)).toEqual(data);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run components/seo/JsonLd.test.tsx`
Expected: FAIL — `Cannot find module './JsonLd'`.

- [ ] **Step 3: Write minimal implementation**

```tsx
// apps/web/components/seo/JsonLd.tsx
export function JsonLd({ data }: { data: Record<string, unknown> }) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run components/seo/JsonLd.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/seo/JsonLd.tsx apps/web/components/seo/JsonLd.test.tsx
git commit -m "$(cat <<'EOF'
feat(web): JSON-LD 주입 컴포넌트 (components/seo/JsonLd)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: 공용 브랜드 렌더 (`lib/brand-mark.tsx`)

> 아이콘(32)·512 로고가 공유하는 순수 JSX. `next/og`를 import하지 않으므로 vitest로 렌더 가능. 이미지 라우트는 이 JSX를 `ImageResponse`에 감싸기만 한다.

**Files:**
- Create: `apps/web/lib/brand-mark.tsx`
- Test: `apps/web/lib/brand-mark.test.tsx`

- [ ] **Step 1: Write the failing test**

```tsx
// apps/web/lib/brand-mark.test.tsx
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { SquareMark } from "./brand-mark";

describe("SquareMark", () => {
  it("renders the Q glyph", () => {
    render(<SquareMark size={32} />);
    expect(screen.getByText("Q")).toBeInTheDocument();
  });

  it("scales font-size with the size prop", () => {
    const { container } = render(<SquareMark size={512} />);
    const root = container.firstElementChild as HTMLElement;
    expect(root.style.width).toBe("512px");
    expect(root.style.height).toBe("512px");
    expect(root.style.display).toBe("flex");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/brand-mark.test.tsx`
Expected: FAIL — `Cannot find module './brand-mark'`.

- [ ] **Step 3: Write minimal implementation**

```tsx
// apps/web/lib/brand-mark.tsx
import { BRAND } from "./seo";

export function SquareMark({ size }: { size: number }) {
  return (
    <div
      style={{
        width: `${size}px`,
        height: `${size}px`,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: BRAND.bg,
        color: BRAND.accent,
        fontFamily: "monospace",
        fontWeight: 700,
        fontSize: Math.round(size * 0.62),
        letterSpacing: "-0.04em",
      }}
    >
      Q
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run lib/brand-mark.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/brand-mark.tsx apps/web/lib/brand-mark.test.tsx
git commit -m "$(cat <<'EOF'
feat(web): 공용 브랜드 마크 렌더 (lib/brand-mark)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: 크롤링 정책 (`app/robots.ts`)

> AI 봇(GPTBot·ClaudeBot·PerplexityBot·Google-Extended 등) 허용은 GEO 핵심 결정. 비공개 경로는 일반·AI 모두 disallow.

**Files:**
- Create: `apps/web/app/robots.ts`
- Test: `apps/web/app/robots.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// apps/web/app/robots.test.ts
import { afterEach, describe, expect, it, vi } from "vitest";
import robots from "./robots";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("robots", () => {
  it("points sitemap and host at siteUrl", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://quotient.app");
    const r = robots();
    expect(r.sitemap).toBe("https://quotient.app/sitemap.xml");
    expect(r.host).toBe("https://quotient.app");
  });

  it("disallows private routes for the wildcard agent", () => {
    const r = robots();
    const wildcard = (Array.isArray(r.rules) ? r.rules : [r.rules]).find(
      (rule) => rule.userAgent === "*",
    );
    expect(wildcard?.disallow).toEqual(
      expect.arrayContaining(["/app", "/api", "/login"]),
    );
  });

  it("explicitly allows major AI crawlers", () => {
    const r = robots();
    const rules = Array.isArray(r.rules) ? r.rules : [r.rules];
    const agents = rules.flatMap((rule) =>
      Array.isArray(rule.userAgent) ? rule.userAgent : [rule.userAgent],
    );
    expect(agents).toEqual(
      expect.arrayContaining(["GPTBot", "ClaudeBot", "PerplexityBot"]),
    );
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run app/robots.test.ts`
Expected: FAIL — `Cannot find module './robots'`.

- [ ] **Step 3: Write minimal implementation**

```ts
// apps/web/app/robots.ts
import type { MetadataRoute } from "next";
import { siteUrl } from "@/lib/seo";

const PRIVATE = [
  "/app",
  "/api",
  "/login",
  "/signup",
  "/forgot-password",
  "/reset-password",
  "/verify-email",
  "/preview",
];

const AI_BOTS = [
  "GPTBot",
  "OAI-SearchBot",
  "ChatGPT-User",
  "ClaudeBot",
  "Claude-Web",
  "anthropic-ai",
  "PerplexityBot",
  "Google-Extended",
  "CCBot",
];

export default function robots(): MetadataRoute.Robots {
  const base = siteUrl();
  return {
    rules: [
      { userAgent: "*", allow: "/", disallow: PRIVATE },
      { userAgent: AI_BOTS, allow: "/", disallow: PRIVATE },
    ],
    sitemap: `${base}/sitemap.xml`,
    host: base,
  };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run app/robots.test.ts`
Expected: PASS.

> 포크 주의: `MetadataRoute.Robots`의 `host` 필드가 포크에 없으면 제거하고 테스트도 그 줄을 뺀다(S2 문서 확인).

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/robots.ts apps/web/app/robots.test.ts
git commit -m "$(cat <<'EOF'
feat(web): robots — 비공개 경로 차단 + AI 크롤러 명시 허용(GEO)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: 사이트맵 (`app/sitemap.ts`)

**Files:**
- Create: `apps/web/app/sitemap.ts`
- Test: `apps/web/app/sitemap.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// apps/web/app/sitemap.test.ts
import { afterEach, describe, expect, it, vi } from "vitest";
import sitemap from "./sitemap";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("sitemap", () => {
  it("lists the 4 public routes as absolute URLs", () => {
    vi.stubEnv("NEXT_PUBLIC_SITE_URL", "https://quotient.app");
    const urls = sitemap().map((e) => e.url);
    expect(urls).toEqual([
      "https://quotient.app",
      "https://quotient.app/pricing",
      "https://quotient.app/privacy",
      "https://quotient.app/terms",
    ]);
  });

  it("gives the homepage (first entry) the highest priority", () => {
    const entries = sitemap();
    expect(entries[0].priority).toBe(1);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run app/sitemap.test.ts`
Expected: FAIL — `Cannot find module './sitemap'`.

- [ ] **Step 3: Write minimal implementation**

```ts
// apps/web/app/sitemap.ts
import type { MetadataRoute } from "next";
import { siteUrl } from "@/lib/seo";

export default function sitemap(): MetadataRoute.Sitemap {
  const base = siteUrl();
  const now = new Date();
  return [
    {
      url: base,
      lastModified: now,
      changeFrequency: "weekly",
      priority: 1,
    },
    {
      url: `${base}/pricing`,
      lastModified: now,
      changeFrequency: "monthly",
      priority: 0.8,
    },
    {
      url: `${base}/privacy`,
      lastModified: now,
      changeFrequency: "yearly",
      priority: 0.3,
    },
    {
      url: `${base}/terms`,
      lastModified: now,
      changeFrequency: "yearly",
      priority: 0.3,
    },
  ];
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run app/sitemap.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/sitemap.ts apps/web/app/sitemap.test.ts
git commit -m "$(cat <<'EOF'
feat(web): sitemap — 공개 4개 경로 절대 URL

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: 매니페스트 (`app/manifest.ts`)

**Files:**
- Create: `apps/web/app/manifest.ts`
- Test: `apps/web/app/manifest.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// apps/web/app/manifest.test.ts
import { describe, expect, it } from "vitest";
import manifest from "./manifest";

describe("manifest", () => {
  it("uses brand bg color and standalone display", () => {
    const m = manifest();
    expect(m.name).toContain("Quotient");
    expect(m.short_name).toBe("Quotient");
    expect(m.display).toBe("standalone");
    expect(m.background_color).toBe("#0A0A0A");
    expect(m.theme_color).toBe("#0A0A0A");
  });

  it("declares both icon sizes", () => {
    const sizes = (manifest().icons ?? []).map((i) => i.sizes);
    expect(sizes).toEqual(expect.arrayContaining(["32x32", "512x512"]));
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run app/manifest.test.ts`
Expected: FAIL — `Cannot find module './manifest'`.

- [ ] **Step 3: Write minimal implementation**

```ts
// apps/web/app/manifest.ts
import type { MetadataRoute } from "next";
import { BRAND, SITE_NAME, SITE_TITLE_DEFAULT } from "@/lib/seo";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: SITE_TITLE_DEFAULT,
    short_name: SITE_NAME,
    description:
      "한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.",
    start_url: "/",
    display: "standalone",
    background_color: BRAND.bg,
    theme_color: BRAND.bg,
    icons: [
      { src: "/icon", sizes: "32x32", type: "image/png" },
      { src: "/brand-512", sizes: "512x512", type: "image/png" },
    ],
  };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run app/manifest.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/manifest.ts apps/web/app/manifest.test.ts
git commit -m "$(cat <<'EOF'
feat(web): manifest — 브랜드 색·아이콘 2종

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: 이미지 라우트 (`app/icon.tsx`, `app/brand-512/route.tsx`, `app/opengraph-image.tsx`)

> ⚠️ `next/og`(Satori/resvg)는 vitest에서 import 시 실패할 수 있으므로 **단위 테스트하지 않는다**. 검증은 Task 12의 `next build` + 수동 fetch로 한다. Satori는 flexbox만 지원하고 다자식 div엔 `display:flex`가 필수다. 영문 카피만 쓰므로 한글 폰트 로딩이 불필요하다(설계 결정).

**Files:**
- Create: `apps/web/app/icon.tsx`
- Create: `apps/web/app/brand-512/route.tsx`
- Create: `apps/web/app/opengraph-image.tsx`

- [ ] **Step 1: 포크 규약 재확인**

S2에서 읽은 문서로 다음을 확정한다:
- `icon.tsx` / `opengraph-image.tsx`가 `export const runtime = "edge"`를 요구하는가? 요구하면 각 파일에 추가.
- `ImageResponse`의 import 경로가 `next/og`인지 포크가 다른 경로를 쓰는지.
- route handler(`brand-512/route.tsx`)에서 `ImageResponse`를 직접 반환해도 되는지(대부분 가능).

문서와 아래 드래프트가 다르면 문서를 따른다.

- [ ] **Step 2: 아이콘 구현 (`app/icon.tsx`)**

```tsx
// apps/web/app/icon.tsx
import { ImageResponse } from "next/og";
import { SquareMark } from "@/lib/brand-mark";

export const size = { width: 32, height: 32 };
export const contentType = "image/png";

export default function Icon() {
  return new ImageResponse(<SquareMark size={32} />, { ...size });
}
```

- [ ] **Step 3: 512 로고 구현 (`app/brand-512/route.tsx`)**

```tsx
// apps/web/app/brand-512/route.tsx
import { ImageResponse } from "next/og";
import { SquareMark } from "@/lib/brand-mark";

export function GET() {
  return new ImageResponse(<SquareMark size={512} />, {
    width: 512,
    height: 512,
  });
}
```

- [ ] **Step 4: OG 카드 구현 (`app/opengraph-image.tsx`)**

```tsx
// apps/web/app/opengraph-image.tsx
import { ImageResponse } from "next/og";
import { BRAND, OG_EYEBROW, OG_TAGLINE_EN } from "@/lib/seo";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "Quotient — Portfolio Intelligence Terminal";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "1200px",
          height: "630px",
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          background: BRAND.bg,
          padding: "0 96px",
          fontFamily: "monospace",
        }}
      >
        <div
          style={{
            display: "flex",
            fontSize: 24,
            letterSpacing: "0.28em",
            color: BRAND.muted,
          }}
        >
          {OG_EYEBROW}
        </div>
        <div
          style={{
            display: "flex",
            fontSize: 140,
            fontWeight: 700,
            color: BRAND.accent,
            letterSpacing: "-0.02em",
            marginTop: 12,
          }}
        >
          QUOTIENT
        </div>
        <div
          style={{
            display: "flex",
            fontSize: 30,
            color: BRAND.fg,
            marginTop: 20,
          }}
        >
          {OG_TAGLINE_EN}
        </div>
        <div style={{ display: "flex", gap: 14, marginTop: 40 }}>
          <div
            style={{ width: 72, height: 10, background: BRAND.accent }}
          />
          <div style={{ width: 72, height: 10, background: BRAND.info }} />
          <div style={{ width: 72, height: 10, background: BRAND.up }} />
        </div>
      </div>
    ),
    { ...size },
  );
}
```

- [ ] **Step 5: 타입 체크**

Run: `cd apps/web && npx tsc --noEmit`
Expected: 에러 없음. (이미지 라우트는 빌드 검증이므로 여기선 타입만 통과하면 됨. 런타임 검증은 Task 12.)

- [ ] **Step 6: Commit**

```bash
git add apps/web/app/icon.tsx apps/web/app/brand-512/route.tsx apps/web/app/opengraph-image.tsx
git commit -m "$(cat <<'EOF'
feat(web): 코드 생성 OG·파비콘·512 로고 (next/og)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: 루트 레이아웃 배선 (`app/layout.tsx`)

> 루트 metadata를 `buildRootMetadata()`로 교체하고, 사이트 전역 JSON-LD(Organization·WebSite)를 주입한다. 루트는 canonical을 두지 않는다(비공개 경로 누수 방지 — 스펙 수정점).

> 모듈-평가 시점 주의(Important): `export const metadata = buildRootMetadata()`와 `organizationJsonLd()`/`webSiteJsonLd()`는 모듈 로드/렌더 시점에 `siteUrl()`을 1회 캡처한다. `NEXT_PUBLIC_SITE_URL`은 빌드 시 인라인되어 안전하지만, 폴백인 `VERCEL_PROJECT_PRODUCTION_URL`은 `NEXT_PUBLIC_` 접두사가 없어 클라이언트에선 안 보이고 서버/빌드에서만 읽힌다. 따라서 **프로덕션에서는 `NEXT_PUBLIC_SITE_URL` 설정을 사실상 필수로 본다**(미설정 시 환경에 따라 `metadataBase`·JSON-LD `url`/`logo`가 로컬호스트로 구워질 수 있음). USER_ACTIONS에 그 취지를 반영(Task 12 Step 7).

**Files:**
- Modify: `apps/web/app/layout.tsx`

- [ ] **Step 1: 현재 파일 확인**

Run: `cd apps/web && sed -n '1,40p' app/layout.tsx` 대신 Read 도구로 `apps/web/app/layout.tsx` 전체를 읽어 import 블록·`export const metadata`·`<body>` 구조를 파악한다.

- [ ] **Step 2: metadata 교체**

기존:
```tsx
export const metadata: Metadata = {
  title: "Quotient — Portfolio Intelligence Terminal",
  description: "한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.",
};
```
교체:
```tsx
import { buildRootMetadata } from "@/lib/seo";
import { JsonLd } from "@/components/seo/JsonLd";
import { organizationJsonLd, webSiteJsonLd } from "@/lib/jsonld";

export const metadata = buildRootMetadata();
```
(`import type { Metadata }`가 더 이상 쓰이지 않으면 제거.)

- [ ] **Step 3: `<body>` 안에 JSON-LD 주입**

`<body ...>` 여는 태그 바로 다음(혹은 `<PostHogProvider>` 바깥 최상단)에 추가:
```tsx
        <JsonLd data={organizationJsonLd()} />
        <JsonLd data={webSiteJsonLd()} />
```

- [ ] **Step 4: 기존 레이아웃 테스트가 깨지지 않는지 + 타입 체크**

Run:
```bash
cd apps/web && npx tsc --noEmit && npx vitest run app/page.test.tsx
```
Expected: 타입 통과, 기존 `page.test.tsx` PASS(랜딩 h1/CTA 불변).

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/layout.tsx
git commit -m "$(cat <<'EOF'
feat(web): 루트 메타데이터·전역 JSON-LD 배선 (layout)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: 랜딩·정적 페이지 배선 (`page.tsx` + pricing/privacy/terms)

> 랜딩에 SoftwareApplication·FAQPage JSON-LD를 주입하고 FAQ를 `FAQ_ITEMS`로 단일화한다. pricing/privacy/terms는 수동 "— Quotient" 접미사를 제거하고 `pageMetadata()`로 canonical을 부여한다(루트 template `%s · Quotient`가 자동 접미).

**Files:**
- Modify: `apps/web/app/page.tsx`
- Modify: `apps/web/app/pricing/page.tsx`
- Modify: `apps/web/app/privacy/page.tsx`
- Modify: `apps/web/app/terms/page.tsx`
- Test: `apps/web/app/page.test.tsx` (기존 유지 + FAQ 항목 검증 1건 추가)

- [ ] **Step 1: 랜딩 테스트에 FAQ 단일 소스 검증 추가**

⚠️ 배치 정확히: (1) `import { FAQ_ITEMS } from "@/lib/faq";`는 **파일 상단 import 그룹**에 추가(맨 아래 붙이지 말 것 — import는 호이스팅되지만 가독성·린트 위반 방지). (2) 새 `it(...)`는 **기존 `describe("LandingPage", () => { ... })` 블록 안**, 마지막 `it` 다음·블록 닫는 `})` 앞에 넣는다(top-level `it`로 빼지 말 것).

수정 후 파일 구조(발췌):
```tsx
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import LandingPage from "./page";
import { FAQ_ITEMS } from "@/lib/faq"; // ← 추가 (import 그룹)

describe("LandingPage", () => {
  // ...기존 it 케이스들 유지...

  it("renders every FAQ question from the single source", () => {
    render(<LandingPage />);
    for (const item of FAQ_ITEMS) {
      expect(screen.getByText(item.q)).toBeInTheDocument();
    }
  }); // ← 추가 (describe 블록 내부)
});
```

- [ ] **Step 2: 테스트 실행 — 실패 확인**

Run: `cd apps/web && npx vitest run app/page.test.tsx`
Expected: 새 케이스 FAIL(아직 `page.tsx`가 인라인 `qs`를 쓰고 `FAQ_ITEMS` 미연결일 수 있음). 기존 케이스는 PASS 유지.

- [ ] **Step 3: 랜딩 `page.tsx` 수정**

Read로 `apps/web/app/page.tsx`를 열어 `FAQ()` 내부 인라인 `const qs = [...]`를 찾는다. 다음으로 교체:
```tsx
import { FAQ_ITEMS } from "@/lib/faq";
// ...
// FAQ() 컴포넌트 내부의 인라인 qs 배열을 제거하고 FAQ_ITEMS를 사용
//   {FAQ_ITEMS.map((item) => ( ...item.q / item.a... ))}
```
파일 상단에 metadata + JSON-LD 추가:
```tsx
import { pageMetadata } from "@/lib/seo";
import { JsonLd } from "@/components/seo/JsonLd";
import { faqPageJsonLd, softwareApplicationJsonLd } from "@/lib/jsonld";

export const metadata = pageMetadata({ path: "/" });
```
컴포넌트 반환 JSX 최상단(또는 첫 섹션 앞)에:
```tsx
      <JsonLd data={softwareApplicationJsonLd()} />
      <JsonLd data={faqPageJsonLd()} />
```

- [ ] **Step 4: 랜딩 테스트 통과 확인**

Run: `cd apps/web && npx vitest run app/page.test.tsx`
Expected: 전체 PASS(기존 h1/CTA + 새 FAQ 케이스).

- [ ] **Step 5: pricing/privacy/terms metadata 교체**

각 파일의 기존:
```tsx
export const metadata: Metadata = { title: "가격 — Quotient" };
```
교체(예: pricing):
```tsx
import { pageMetadata } from "@/lib/seo";

export const metadata = pageMetadata({
  title: "가격",
  description: "Quotient 요금제 — 무료로 시작하고 Pro로 한도를 넓히세요.",
  path: "/pricing",
});
```
privacy:
```tsx
export const metadata = pageMetadata({
  title: "개인정보 처리방침",
  description: "Quotient 개인정보 처리방침.",
  path: "/privacy",
});
```
terms:
```tsx
export const metadata = pageMetadata({
  title: "서비스 약관",
  description: "Quotient 서비스 약관.",
  path: "/terms",
});
```
(각 파일에서 미사용된 `import type { Metadata }`는 제거.)

- [ ] **Step 6: 타입 + 전체 web 테스트**

Run:
```bash
cd apps/web && npx tsc --noEmit && npx vitest run
```
Expected: 타입 통과, 전체 테스트 PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/page.tsx apps/web/app/page.test.tsx apps/web/app/pricing/page.tsx apps/web/app/privacy/page.tsx apps/web/app/terms/page.tsx
git commit -m "$(cat <<'EOF'
feat(web): 랜딩 JSON-LD·FAQ 단일화 + 정적 페이지 canonical/title 정리

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: 빌드 검증 + 문서 갱신

> 이미지 라우트와 파일 컨벤션은 런타임에서만 확인 가능하므로 빌드 + 수동 fetch로 마감한다. 그 후 MANDATORY 문서 갱신.

**Files:**
- Modify: `docs/STATUS.md`, `docs/ROADMAP.md`, `docs/USER_ACTIONS.md`, `docs/E2E_SMOKE.md`

- [ ] **Step 1: 전체 단위 테스트**

Run: `cd apps/web && npx vitest run`
Expected: 전체 PASS.

- [ ] **Step 2: 프로덕션 빌드**

Run: `cd apps/web && npm run build`
Expected: 빌드 성공. `/robots.txt`·`/sitemap.xml`·`/manifest.webmanifest`·`/icon`·`/opengraph-image`·`/brand-512` 라우트가 출력 목록에 보임. 빌드 실패 시(특히 `next/og`/`runtime`) S2 포크 문서로 원인 교정.

- [ ] **Step 3: 런타임 수동 확인**

Run: `cd apps/web && npm run start` (또는 `npm run preview`) 후 별도 셸에서:
```bash
curl -s http://localhost:3000/robots.txt
curl -s http://localhost:3000/sitemap.xml
curl -sI http://localhost:3000/opengraph-image | grep -i content-type
curl -sI http://localhost:3000/icon | grep -i content-type
curl -sI http://localhost:3000/brand-512 | grep -i content-type
```
Expected: robots에 AI 봇 그룹·sitemap 라인; sitemap에 4개 `<url>`; 이미지 3종 `content-type: image/png`.

- [ ] **Step 4: 구조화 데이터 육안 확인**

브라우저에서 `http://localhost:3000` 소스 보기 → `application/ld+json` 4개(Organization·WebSite·SoftwareApplication·FAQPage) 존재 확인. 배포 후에는 Google Rich Results Test(https://search.google.com/test/rich-results)로 FAQPage 인식 확인(USER_ACTIONS에 기재).

- [ ] **Step 5: `docs/STATUS.md` 갱신**

- SEO·AEO·GEO 항목을 ✅ 구현됨으로 추가/이동
- "최근 변경 이력" 맨 위에 한 줄: `- (2026-06-02) SEO·AEO·GEO 기반 구축: 메타데이터 단일소스·robots/sitemap/manifest·JSON-LD 4종·코드 생성 OG/파비콘`
- "마지막 업데이트" 날짜 갱신

- [ ] **Step 6: `docs/ROADMAP.md` 갱신**

- 완료된 SEO 항목 제거, "현재 추천 다음 작업" 재설정(예: 배포 후 Search Console 등록·콘텐츠/블로그(범위 외였던 ②) 검토)

- [ ] **Step 7: `docs/USER_ACTIONS.md` 갱신 (신규 사용자 액션 등재)**

표에 행 추가:
| 액션 | 설명 | 필수 |
|---|---|---|
| `NEXT_PUBLIC_SITE_URL` 설정 | Vercel 환경변수에 배포 절대 URL 지정. 정적 메타데이터·JSON-LD가 모듈-평가 시점에 도메인을 굽기 때문에 **프로덕션 사실상 필수**(미설정 시 환경에 따라 로컬호스트로 구워질 위험; 도메인 확정 전까지는 Vercel 자동 URL 폴백) | 프로덕션 필수 |
| Google Search Console 등록 | 속성 추가 → `NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION` 토큰 등록 → `/sitemap.xml` 제출 | 필수(노출) |
| Naver 서치어드바이저 등록 | 사이트 등록 → `NEXT_PUBLIC_NAVER_SITE_VERIFICATION` 토큰 → 사이트맵 제출 | 권장(국내) |
| Bing Webmaster (선택) | GSC 가져오기로 5분 등록 | 선택 |
| Rich Results Test 확인 | 배포 후 FAQPage 인식 검증 | 권장 |

- [ ] **Step 8: `docs/E2E_SMOKE.md` 갱신**

골든패스에 SEO 스모크 추가: 배포 도메인에서 `/robots.txt`·`/sitemap.xml` 200, 링크 공유 시 OG 카드 노출(슬랙/카톡), Rich Results Test FAQPage 통과.

- [ ] **Step 9: Lint**

Run: `cd apps/web && npm run lint`
Expected: 통과(경고 0 목표, 신규 파일 기준).

- [ ] **Step 10: Commit**

```bash
git add apps/web/app docs/STATUS.md docs/ROADMAP.md docs/USER_ACTIONS.md docs/E2E_SMOKE.md
git commit -m "$(cat <<'EOF'
docs: SEO·AEO·GEO 구축 반영 (STATUS·ROADMAP·USER_ACTIONS·E2E_SMOKE)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review (작성자 체크리스트 — 이미 반영됨)

**1. 스펙 커버리지:** 스펙의 각 요소 → Task 매핑
- 도메인 단일 소스 → Task 1 ✅
- 루트/페이지별 metadata → Task 1(빌더)·10·11(배선) ✅
- robots(AI 봇 허용) → Task 6 ✅
- sitemap → Task 7 ✅
- manifest → Task 8 ✅
- JSON-LD 4종 → Task 3(빌더)·4(주입)·10·11(배선) ✅
- FAQPage(기존 FAQ 구조화) → Task 2·3·11 ✅
- OG(영문)·icon·brand-512 → Task 5·9 ✅
- 검증 방법·USER_ACTIONS → Task 12 ✅

**2. 스펙 대비 수정점(플랜에서 교정함):**
- (교정) 루트 canonical 제거 → 페이지별 canonical만. 이유: 루트 `alternates.canonical:"/"`는 `/app`·`/login` 등 모든 경로에 "/"를 누수시킴. Task 1에서 `buildRootMetadata`는 canonical 미설정, `pageMetadata`가 경로별 부여.
- (교정) pricing/privacy/terms는 이미 `"… — Quotient"` 수동 접미를 가짐 → 루트 `template:"%s · Quotient"`와 충돌(이중 접미). Task 11에서 제목을 맨몸("가격" 등)으로 바꿔 template가 접미하게 함.
- (명시) 이미지 라우트(icon/brand-512/opengraph-image)는 `next/og` 의존으로 vitest 단위 테스트 제외 → Task 9는 타입체크, Task 12 빌드+fetch로 검증. 대신 순수 JSX(`SquareMark`)를 Task 5에서 RTL로 단위 검증.

**3. 플레이스홀더 스캔:** "TBD/적절히/유사하게" 없음. 모든 코드 step에 완전한 코드 포함. ✅

**4. 타입 일관성:** `siteUrl()`·`SITE_NAME`·`SITE_TITLE_DEFAULT`·`BRAND`·`FAQ_ITEMS`·`FaqItem`·`buildRootMetadata`·`pageMetadata`·`organizationJsonLd`·`webSiteJsonLd`·`softwareApplicationJsonLd`·`faqPageJsonLd`·`JsonLd`·`SquareMark` — 정의 Task와 사용 Task 간 시그니처 일치. ✅

**5. 포크 리스크:** S2에서 파일 컨벤션 시그니처를 먼저 확정하고, 드래프트와 다르면 포크 문서를 따르도록 각 Task에 주석. `MetadataRoute.*` 타입·`runtime` export·`verification` 필드명이 주요 변수.

---

## Subagent 검토 반영 (2026-06-02)

`general-purpose` subagent 1차 검토(직접 검토 금지 규칙 준수) 결과와 처리:

**Critical**
- **C1 — `page.test.tsx` 테스트 배치 오류**: 추가 `it`을 파일 맨 아래(=`describe` 바깥 top-level)에 두고 `import`를 본문 중간에 끼우는 모호한 지시였음 → Task 11 Step 1을 "import는 상단 그룹, `it`는 기존 `describe` 블록 내부"로 명시하고 수정 후 파일 구조 발췌를 추가.
- **C2 — `NEXT_PUBLIC_*` 인라인 vs `vi.stubEnv`**: vitest는 Next 빌드와 달리 정적 인라인을 하지 않아 `stubEnv`가 유효함을 Task 1에 주석으로 박제(테스트가 vitest 한정으로 타당한 근거).

**Important**
- **I1 — 모듈-평가 시점 도메인 캡처**: `export const metadata`/JSON-LD 빌더가 `siteUrl()`을 1회 캡처 → 프로덕션에서 `NEXT_PUBLIC_SITE_URL` 미설정 시 로컬호스트로 구워질 위험. Task 10에 주의 주석 + USER_ACTIONS 행을 "프로덕션 필수"로 격상.
- **I2 — verification 테스트 격리**: "includes verification" 케이스에 `vi.stubEnv("NEXT_PUBLIC_NAVER_SITE_VERIFICATION","")` 추가(주변 env 누수 방지).
- **I3 — SoftwareApplication 카피 드리프트**: Task 3 빌더가 스펙 §4의 박제 `description`·`featureList`와 다른 임의 문구를 썼음 → 스펙 원문으로 정렬하고 "임의 변경 금지" 주석 추가.

**Minor**
- **M2 — sitemap priority 매처 취약**: `endsWith(".app")||endsWith("3000")` → `entries[0].priority === 1`로 단순화.
- M1(OG 우하단 `quotient.app` 텍스트 생략)·M3(next/og 경계 정확)·M4(`@/*` 별칭·스크립트·BRAND 토큰 일치)는 확인용 — 조치 불요.

검토 후 미해결 Critical/Important 없음. 단 S1/S2 전제(설치·포크 문서 확인)는 실행 시점 필수로 유지.

---

## Execution Handoff

계획 완료. 두 가지 실행 옵션:

1. **Subagent-Driven (권장)** — Task마다 새 subagent 디스패치 + 스펙 리뷰 → 코드 퀄리티 리뷰 2단계
2. **Inline Execution** — 이 세션에서 체크포인트로 직접 실행

(이 레포 규칙상 Inline은 사용하지 않고 Subagent-Driven으로 진행. 단, 첫 구현 커밋 전 사용자에게 진행/커밋 승인 확인.)

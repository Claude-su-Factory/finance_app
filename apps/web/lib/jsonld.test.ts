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

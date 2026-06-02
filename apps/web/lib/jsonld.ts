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

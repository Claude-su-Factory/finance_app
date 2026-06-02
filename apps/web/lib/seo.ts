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

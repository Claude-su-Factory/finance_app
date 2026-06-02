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

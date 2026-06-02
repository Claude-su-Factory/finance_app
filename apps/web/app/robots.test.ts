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

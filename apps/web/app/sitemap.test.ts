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

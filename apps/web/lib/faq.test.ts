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

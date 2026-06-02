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

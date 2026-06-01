import { describe, it, expect } from "vitest";
import { lookupFixture, MOCKS } from "./fixtures";

describe("lookupFixture", () => {
  it("알려진 엔드포인트는 해당 픽스처를 반환한다", () => {
    expect(lookupFixture("/v1/holdings")).toBe(MOCKS["/v1/holdings"]);
  });
  it("미정의 엔드포인트는 빈 객체로 degrade한다", () => {
    expect(lookupFixture("/v1/unknown/thing")).toEqual({});
  });
});

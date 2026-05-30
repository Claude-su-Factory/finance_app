// @vitest-environment node
import { describe, it, expect, beforeEach, vi } from "vitest";
import { NextRequest } from "next/server";
import { updateSession } from "./middleware";

const { getUserMock, fromMock, singleMock } = vi.hoisted(() => {
  const singleMock = vi.fn();
  const fromMock = vi.fn(() => ({
    select: () => ({ eq: () => ({ single: singleMock }) }),
  }));
  const getUserMock = vi.fn();
  return { getUserMock, fromMock, singleMock };
});

vi.mock("@supabase/ssr", () => ({
  createServerClient: () => ({
    auth: { getUser: getUserMock },
    from: fromMock,
  }),
}));

function makeRequest(path: string, cookies?: Record<string, string>) {
  const req = new NextRequest(`https://q.test${path}`);
  if (cookies) {
    for (const [k, v] of Object.entries(cookies)) req.cookies.set(k, v);
  }
  return req;
}

beforeEach(() => {
  getUserMock.mockReset();
  fromMock.mockClear();
  singleMock.mockReset();
  getUserMock.mockResolvedValue({ data: { user: { id: "u1" } } });
});

describe("updateSession 온보딩 쿠키 캐시", () => {
  it("온보딩 완료 + 쿠키 없음: 조회 실행 후 q_onboarded 쿠키를 굽는다", async () => {
    singleMock.mockResolvedValue({ data: { onboarding_completed: true } });
    const res = await updateSession(makeRequest("/app"));
    expect(fromMock).toHaveBeenCalledTimes(1);
    expect(res.cookies.get("q_onboarded")?.value).toBe("1");
  });
});

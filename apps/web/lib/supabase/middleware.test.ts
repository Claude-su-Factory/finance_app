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

  it("온보딩 완료 + q_onboarded 쿠키: 인증은 검증하되 profiles 조회는 스킵", async () => {
    singleMock.mockResolvedValue({ data: { onboarding_completed: true } });
    await updateSession(makeRequest("/app", { q_onboarded: "1" }));
    expect(getUserMock).toHaveBeenCalled();
    expect(fromMock).not.toHaveBeenCalled();
  });

  it("온보딩 미완료 + 쿠키 없음: /app/onboarding으로 리다이렉트하고 쿠키를 굽지 않는다", async () => {
    singleMock.mockResolvedValue({ data: { onboarding_completed: false } });
    const res = await updateSession(makeRequest("/app"));
    expect(fromMock).toHaveBeenCalledTimes(1);
    expect(res.status).toBe(307);
    expect(res.headers.get("location")).toContain("/app/onboarding");
    expect(res.cookies.get("q_onboarded")?.value).toBeUndefined();
  });

  it("profiles 조회 실패(profile=null): 리다이렉트·쿠키 발급 없이 통과", async () => {
    singleMock.mockResolvedValue({ data: null });
    const res = await updateSession(makeRequest("/app"));
    expect(fromMock).toHaveBeenCalledTimes(1);
    expect(res.headers.get("location")).toBeNull();
    expect(res.cookies.get("q_onboarded")?.value).toBeUndefined();
  });
});

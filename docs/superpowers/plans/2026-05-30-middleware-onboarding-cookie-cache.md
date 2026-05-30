# 미들웨어 온보딩 read-through 쿠키 캐시 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/app/*` 매 요청마다 발생하는 `profiles.onboarding_completed` 조회(N+1)를 read-through 쿠키 캐시로 제거한다.

**Architecture:** 미들웨어가 `onboarding_completed=true`를 한 번 확인하면 `q_onboarded=1` 쿠키를 응답에 굽는다. 이후 요청은 쿠키가 있으면 `profiles` 조회를 건너뛴다. `auth.getUser()` 세션 검증은 그대로 유지(쿠키는 프로필 조회만 단축). `onboarding_completed`는 단조(true로만 전이)라 stale/위조 쿠키도 안전 방향으로만 작용한다(위조 시 자기 온보딩 화면만 스킵 = 무해).

**Tech Stack:** Next.js(커스텀 빌드) middleware, `@supabase/ssr` `createServerClient`, vitest ^4.1.7.

---

## File Structure

- **Modify:** `apps/web/lib/supabase/middleware.ts` — `updateSession()`의 온보딩 블록에 쿠키 read gate + write 추가. 다른 분기(미인증 → /login, 로그인 사용자 → /app)는 변경 없음.
- **Create:** `apps/web/lib/supabase/middleware.test.ts` — `@supabase/ssr` 목 + `NextRequest` 빌더로 `updateSession`을 단위 테스트. (기존 테스트 중 `@supabase/ssr`를 목하는 것이 없어 새 패턴을 확립한다.)

**테스트 환경 주의:** 미들웨어는 DOM과 무관하고 `next/server`의 `Request`/`Response` 전역이 필요하므로, 테스트 파일 첫 줄에 `// @vitest-environment node`를 두어 전역 jsdom 대신 node 환경을 쓴다.

---

## Task 1: read-through write — 온보딩 확인 시 q_onboarded 쿠키 굽기

**Files:**
- Create: `apps/web/lib/supabase/middleware.test.ts`
- Modify: `apps/web/lib/supabase/middleware.ts:36-48`

- [ ] **Step 1: Write the failing test (scaffold + Test A)**

Create `apps/web/lib/supabase/middleware.test.ts`:

```ts
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npx vitest run lib/supabase/middleware.test.ts`
Expected: FAIL — `fromMock` 호출(1회)은 통과하나, `res.cookies.get("q_onboarded")`가 `undefined`라 `.value` 단언에서 실패(아직 쿠키를 굽지 않음).

- [ ] **Step 3: Write minimal implementation**

In `apps/web/lib/supabase/middleware.ts`, replace the onboarding block (lines 36-46) with (쿠키 set만 추가, gate는 Task 2):

```ts
  // 온보딩 미완료 사용자가 /app/* 접근 시 /app/onboarding으로 (단, /app/onboarding 자체는 통과)
  if (user && request.nextUrl.pathname.startsWith("/app") && request.nextUrl.pathname !== "/app/onboarding") {
    const { data: profile } = await supabase
      .from("profiles")
      .select("onboarding_completed")
      .eq("id", user.id)
      .single();
    if (profile && !profile.onboarding_completed) {
      return NextResponse.redirect(new URL("/app/onboarding", request.url));
    }
    if (profile?.onboarding_completed) {
      response.cookies.set("q_onboarded", "1", {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        sameSite: "lax",
        path: "/",
        maxAge: 60 * 60 * 24 * 365,
      });
    }
  }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/web && npx vitest run lib/supabase/middleware.test.ts`
Expected: PASS (1 test).

- [ ] **Step 5: Commit**

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/web/lib/supabase/middleware.ts apps/web/lib/supabase/middleware.test.ts
git commit -m "feat(web): 온보딩 확인 시 q_onboarded 쿠키 발급 (read-through write)"
```

---

## Task 2: read gate — 쿠키 존재 시 profiles 조회 스킵

**Files:**
- Modify: `apps/web/lib/supabase/middleware.ts:37` (온보딩 블록의 `if` 조건)
- Modify: `apps/web/lib/supabase/middleware.test.ts` (Test B + Test C 추가)

- [ ] **Step 1: Write the failing test (Test B) + regression guard (Test C)**

Append inside the `describe(...)` block in `apps/web/lib/supabase/middleware.test.ts`:

```ts
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
```

- [ ] **Step 2: Run tests to verify Test B fails**

Run: `cd apps/web && npx vitest run lib/supabase/middleware.test.ts`
Expected: FAIL on Test B — gate가 아직 없어 쿠키가 있어도 `fromMock`이 호출됨 → `expect(fromMock).not.toHaveBeenCalled()` 실패. (Test C는 기존 리다이렉트 동작이라 이미 통과 = 회귀 가드.)

- [ ] **Step 3: Add the read gate to the onboarding block condition**

In `apps/web/lib/supabase/middleware.ts`, change the onboarding block's `if` condition to add the cookie gate as a 4th term:

```ts
  if (
    user &&
    request.nextUrl.pathname.startsWith("/app") &&
    request.nextUrl.pathname !== "/app/onboarding" &&
    request.cookies.get("q_onboarded")?.value !== "1"
  ) {
```

(The block body — query, redirect, cookie set — stays exactly as left by Task 1.)

- [ ] **Step 4: Run tests to verify all pass**

Run: `cd apps/web && npx vitest run lib/supabase/middleware.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/web/lib/supabase/middleware.ts apps/web/lib/supabase/middleware.test.ts
git commit -m "feat(web): q_onboarded 쿠키 존재 시 온보딩 조회 스킵 (N+1 제거)"
```

---

## Task 3: 전체 스위트 + 타입체크 검증

**Files:** (변경 없음 — 검증만)

- [ ] **Step 1: Run the full web test suite**

Run: `cd apps/web && npx vitest run`
Expected: PASS — 신규 3 테스트 포함 전체 그린, 기존 테스트(`app/page.test.tsx`, `components/home/AlphaCard.test.tsx`, `components/backtest/BacktestPage.test.tsx`) 회귀 없음.

- [ ] **Step 2: Typecheck the web app**

Run: `cd apps/web && npx tsc --noEmit`
Expected: 오류 없음(exit 0).

- [ ] **Step 3: Lint the changed files**

Run: `cd apps/web && npx eslint lib/supabase/middleware.ts lib/supabase/middleware.test.ts`
Expected: 오류 없음(exit 0).

- [ ] **Step 4: Commit (only if Step 1-3 produced any fixups; otherwise skip)**

검증만으로 변경이 없으면 커밋 생략. 수정이 발생하면:

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/web/lib/supabase/middleware.ts apps/web/lib/supabase/middleware.test.ts
git commit -m "fix(web): 미들웨어 쿠키 캐시 타입·린트 정리"
```

---

## Post-implementation (controller, 구현·리뷰 후)

코드 리뷰 통과 후 컨트롤러가 문서를 갱신한다(구현 Task에 포함하지 않음 — SHA 등 런타임 값 의존):

1. `docs/STATUS.md` — "알려진 결함"의 미들웨어 N+1 항목을 해결 처리(strikethrough + "2026-05-30 해결" + 구현 커밋 SHA), "최근 변경 이력" 맨 위 한 줄 추가, "마지막 업데이트" 갱신.
2. `docs/ARCHITECTURE.md` — "핵심 설계 결정"에 read-through 쿠키 캐시 결정 추가(Why: 매 `/app/*` 요청 N+1 제거; How: 단조 플래그 → stale/위조 안전; secure는 prod에서만).
3. `docs/ROADMAP.md` — "현재 추천 다음 작업"에서 "미들웨어 N+1" 제거, "운영 자동화"를 다음으로 승격.
4. `docs/USER_ACTIONS.md` — 신규 사용자 액션 없음(쿠키는 자동 발급). 변경 불필요.

---

## Self-Review

**1. Spec coverage:**
- N+1 제거(쿠키 존재 시 조회 스킵) → Task 2 / Test B. ✓
- read-through write(확인 시 쿠키 발급) → Task 1 / Test A. ✓
- 기존 온보딩 리다이렉트 가드 유지 → Task 2 / Test C(회귀 가드). ✓
- `auth.getUser()` 세션 검증 유지(쿠키가 인증을 스킵하지 않음) → Test B의 `getUserMock` 단언. ✓
- 단조 플래그 안전성(stale/위조 무해) → ARCHITECTURE 결정 노트(Post-implementation). ✓

**2. Placeholder scan:** TBD/TODO/"적절히 처리" 없음. 모든 스텝에 실제 코드·명령·기대출력 포함. ✓

**3. Type consistency:** 목 체인 `from().select().eq().single()`이 구현의 `supabase.from("profiles").select("onboarding_completed").eq("id", user.id).single()`와 정확히 일치. 쿠키 키 `q_onboarded`·값 `"1"`이 read(`request.cookies.get("q_onboarded")?.value !== "1"`)/write(`response.cookies.set("q_onboarded", "1", ...)`) 양쪽 동일. `updateSession` 시그니처(`(request: NextRequest)`)와 테스트 호출 일치. ✓

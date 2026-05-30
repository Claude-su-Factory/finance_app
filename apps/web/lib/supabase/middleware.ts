import { createServerClient } from "@supabase/ssr";
import { NextResponse, type NextRequest } from "next/server";

export async function updateSession(request: NextRequest) {
  let response = NextResponse.next({ request });

  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll: () => request.cookies.getAll(),
        setAll: (list) => {
          list.forEach(({ name, value }) => request.cookies.set(name, value));
          response = NextResponse.next({ request });
          list.forEach(({ name, value, options }) =>
            response.cookies.set(name, value, options)
          );
        },
      },
    }
  );

  const { data: { user } } = await supabase.auth.getUser();

  // /app/* 는 인증 필수
  if (!user && request.nextUrl.pathname.startsWith("/app")) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  // 로그인 사용자가 /login·/signup 접근 시 /app으로
  if (user && ["/login", "/signup"].includes(request.nextUrl.pathname)) {
    return NextResponse.redirect(new URL("/app", request.url));
  }

  // 온보딩 미완료 사용자가 /app/* 접근 시 /app/onboarding으로 (단, /app/onboarding 자체는 통과)
  // read-through 쿠키 캐시: onboarding_completed은 단조(false→true 전이만)라 q_onboarded=1 쿠키는 무조건 신뢰 가능
  // → 매 /app/* 요청의 profiles 조회(N+1)를 제거. profile=null(조회 실패)이면 쿠키 미발급 → 다음 요청에서 재조회(안전한 degrade).
  if (
    user &&
    request.nextUrl.pathname.startsWith("/app") &&
    request.nextUrl.pathname !== "/app/onboarding" &&
    request.cookies.get("q_onboarded")?.value !== "1"
  ) {
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

  return response;
}

import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = { title: "개인정보 처리방침 — Quotient" };

export default function Privacy() {
  return (
    <main className="min-h-screen p-12 max-w-3xl mx-auto">
      <h1 className="font-mono text-3xl mb-2">개인정보 처리방침</h1>
      <p className="text-fg-muted text-sm mb-8">최종 갱신: 2026-05-22</p>

      <h2 className="font-mono text-xl mt-8 mb-2">수집 항목</h2>
      <ul className="space-y-1 text-fg">
        <li>· 이메일, 비밀번호 (해시), 이름</li>
        <li>· 보유 자산 정보</li>
        <li>· 분석 채팅 기록</li>
      </ul>

      <h2 className="font-mono text-xl mt-8 mb-2">처리 위탁</h2>
      <ul className="space-y-1 text-fg">
        <li>· Supabase (인증·DB 호스팅)</li>
        <li>· Anthropic (분석 엔진)</li>
        <li>· Resend (이메일 발송)</li>
        <li>· Sentry (에러 추적)</li>
        <li>· PostHog (활동 분석)</li>
      </ul>

      <h2 className="font-mono text-xl mt-8 mb-2">보유 기간</h2>
      <p className="text-fg">회원 탈퇴 시 즉시 파기. 결제 기록은 한국 세법에 따라 7년 보관 (익명화).</p>

      <footer className="border-t border-line mt-12 pt-4 text-fg-muted text-xs flex items-center justify-between">
        <span>투자 자문이 아닙니다. 모든 의사결정은 본인 책임입니다.</span>
        <Link href="/" className="underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent">홈으로</Link>
      </footer>
    </main>
  );
}

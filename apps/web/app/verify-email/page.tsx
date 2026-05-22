import type { Metadata } from "next";

export const metadata: Metadata = { title: "이메일 인증 — Quotient" };

// Next.js 15+: searchParams는 Promise — async 컴포넌트 + await 필수
export default async function VerifyEmailPage({
  searchParams,
}: {
  searchParams: Promise<{ email?: string }>;
}) {
  const { email } = await searchParams;
  return (
    <main className="min-h-screen flex items-center justify-center px-6">
      <div className="max-w-md text-center space-y-4">
        <h1 className="font-mono text-2xl">이메일을 확인해주세요</h1>
        <p className="text-fg-muted">
          {email ? <span className="font-mono">{email}</span> : "가입한 이메일"}로 인증 메일을 보냈습니다.
        </p>
        <p className="text-fg-muted text-sm">
          메일 내 링크를 클릭하면 가입이 완료됩니다.
        </p>
      </div>
    </main>
  );
}

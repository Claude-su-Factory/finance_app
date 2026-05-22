import Link from "next/link";

export default function LandingPage() {
  return (
    <main className="min-h-screen flex flex-col">
      <section className="flex-1 flex flex-col items-center justify-center gap-8 px-6">
        <h1 className="font-mono text-5xl md:text-7xl tracking-tight text-center">
          Portfolio Intelligence Terminal
        </h1>
        <p className="text-fg-muted text-center max-w-xl">
          한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.
        </p>
        <div className="flex gap-3">
          <Link href="/signup" className="px-6 py-2 bg-bb-accent text-bg font-mono">
            가입하기
          </Link>
          <Link href="/login" className="px-6 py-2 border border-line font-mono">
            로그인
          </Link>
        </div>
      </section>
      <footer className="border-t border-line py-4 px-6 text-fg-muted text-xs">
        투자 자문이 아닙니다. 모든 의사결정은 본인 책임입니다.
      </footer>
    </main>
  );
}

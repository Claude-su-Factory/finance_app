"use client";
import { useEffect } from "react";

export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    // TODO(Task 13): Sentry 통합 후 Sentry.captureException(error) 추가
    console.error("page error:", error);
  }, [error]);

  // 사용자에게는 내부 정보 노출 금지 — generic 메시지만
  return (
    <main className="min-h-screen flex items-center justify-center px-6 bg-bg text-fg">
      <div className="max-w-md text-center space-y-4">
        <h1 className="font-mono text-3xl text-bb-down">500</h1>
        <p className="text-fg-muted text-sm">예상치 못한 오류가 발생했습니다. 잠시 후 다시 시도해주세요.</p>
        {error.digest && <p className="text-fg-subtle text-[10px] font-mono">{error.digest}</p>}
        <button
          onClick={reset}
          className="border border-line px-4 py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent"
        >
          다시 시도
        </button>
      </div>
    </main>
  );
}

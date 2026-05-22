"use client";
export function DemoOrStartStep({
  onDemo, onStart, loading,
}: { onDemo: () => void; onStart: () => void; loading: boolean }) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="font-mono text-2xl">시작 방법</h2>
        <p className="text-fg-muted text-sm mt-1">데모 데이터로 둘러보거나, 빈 포트폴리오로 시작합니다.</p>
      </div>
      <div className="space-y-3">
        <button
          type="button"
          onClick={onDemo}
          disabled={loading}
          className="w-full border border-line p-4 text-left disabled:opacity-50 hover:border-bb-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent transition-colors"
        >
          <div className="font-mono">데모 포트폴리오로 시작</div>
          <div className="text-fg-muted text-xs mt-1">샘플 종목 5개가 자동 입력됩니다 (W3에서 활성).</div>
        </button>
        <button
          type="button"
          onClick={onStart}
          disabled={loading}
          className="w-full border border-line p-4 text-left disabled:opacity-50 hover:border-bb-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent transition-colors"
        >
          <div className="font-mono">빈 상태로 시작</div>
          <div className="text-fg-muted text-xs mt-1">직접 자산을 추가합니다.</div>
        </button>
      </div>
    </div>
  );
}

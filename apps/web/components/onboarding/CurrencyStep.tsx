"use client";
export function CurrencyStep({
  value, onChange, onNext,
}: { value: "KRW" | "USD"; onChange: (v: "KRW" | "USD") => void; onNext: () => void }) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="font-mono text-2xl">기본 통화</h2>
        <p className="text-fg-muted text-sm mt-1">자산 평가액·수익률 표시 통화입니다. 추후 설정에서 변경 가능합니다.</p>
      </div>
      <div className="grid grid-cols-2 gap-3">
        {(["KRW", "USD"] as const).map((c) => (
          <button
            key={c}
            type="button"
            onClick={() => onChange(c)}
            className={`border p-6 font-mono text-2xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent ${value === c ? "border-bb-accent text-bb-accent" : "border-line text-fg"}`}
            aria-pressed={value === c}
          >
            {c}
          </button>
        ))}
      </div>
      <button
        type="button"
        onClick={onNext}
        className="w-full bg-bb-accent text-bg font-mono py-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
      >
        다음
      </button>
    </div>
  );
}

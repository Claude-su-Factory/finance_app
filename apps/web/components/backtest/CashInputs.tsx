"use client";

export type ContributionMode = "lump" | "dca";

export function CashInputs({
  initialCash,
  mode,
  monthly,
  onInitialCash,
  onMode,
  onMonthly,
}: {
  initialCash: number;
  mode: ContributionMode;
  monthly: number;
  onInitialCash: (v: number) => void;
  onMode: (m: ContributionMode) => void;
  onMonthly: (v: number) => void;
}) {
  return (
    <div className="flex gap-3">
      <label className="flex-1">
        <div className="text-xs text-fg-muted mb-1">초기 자금 (₩)</div>
        <input
          type="number"
          min={0}
          value={initialCash}
          onChange={(e) => onInitialCash(Number(e.target.value) || 0)}
          className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full tabular-nums"
        />
      </label>
      <label className="flex-1">
        <div className="text-xs text-fg-muted mb-1">투입 방식</div>
        <select
          value={mode}
          onChange={(e) => onMode(e.target.value as ContributionMode)}
          className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full"
        >
          <option value="lump">일시불</option>
          <option value="dca">월 적립</option>
        </select>
      </label>
      <label className="flex-1">
        <div className="text-xs text-fg-muted mb-1">월 적립금 (₩)</div>
        <input
          type="number"
          min={0}
          value={monthly}
          disabled={mode === "lump"}
          onChange={(e) => onMonthly(Number(e.target.value) || 0)}
          className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full tabular-nums disabled:opacity-40"
        />
      </label>
    </div>
  );
}

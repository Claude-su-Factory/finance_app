"use client";
import type { CoverageWarning } from "@/lib/api/backtest";

export function CoverageNotice({
  warnings,
  clampedStart,
}: {
  warnings: CoverageWarning[];
  clampedStart: string;
}) {
  if (warnings.length === 0) return null;
  return (
    <div className="border border-bb-warn/40 bg-bb-warn/5 p-3 text-xs font-mono space-y-1">
      <div className="text-bb-warn">
        일부 종목의 데이터가 짧아 시작일이 {clampedStart}로 조정되었습니다.
      </div>
      {warnings.map((w) => (
        <div key={w.symbol} className="text-fg-muted">
          · {w.symbol}: {w.message} (최초 {w.first_available})
        </div>
      ))}
    </div>
  );
}

"use client";
import type { BacktestResult } from "@/lib/api/backtest";

function pct(v: number | null): string {
  if (v === null || Number.isNaN(v)) return "—";
  return `${v.toFixed(2)}%`;
}

export function CompareTable({ result }: { result: BacktestResult }) {
  const rows = [
    { label: "내 전략", tr: result.metrics.total_return_pct, cagr: result.metrics.cagr_pct, mdd: result.metrics.mdd_pct, bold: true },
    { label: "KOSPI", tr: result.benchmarks.kospi.metrics.total_return_pct, cagr: result.benchmarks.kospi.metrics.cagr_pct, mdd: result.benchmarks.kospi.metrics.mdd_pct },
    { label: "S&P 500", tr: result.benchmarks.spx.metrics.total_return_pct, cagr: result.benchmarks.spx.metrics.cagr_pct, mdd: result.benchmarks.spx.metrics.mdd_pct },
    { label: "한미 60/40", tr: result.benchmarks.sixty_forty.metrics.total_return_pct, cagr: result.benchmarks.sixty_forty.metrics.cagr_pct, mdd: result.benchmarks.sixty_forty.metrics.mdd_pct },
  ];
  return (
    <div className="border border-line">
      <table className="w-full text-sm font-mono">
        <thead>
          <tr className="text-fg-muted text-xs border-b border-line">
            <th className="text-left px-3 py-2 font-normal">전략·벤치마크</th>
            <th className="text-right px-3 py-2 font-normal">총수익률</th>
            <th className="text-right px-3 py-2 font-normal">CAGR</th>
            <th className="text-right px-3 py-2 font-normal">MDD</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.label} className={`border-b border-line/40 last:border-b-0 ${r.bold ? "text-bb-accent" : ""}`}>
              <td className="text-left px-3 py-2">{r.label}</td>
              <td className="text-right px-3 py-2 tabular-nums">{pct(r.tr)}</td>
              <td className="text-right px-3 py-2 tabular-nums">{pct(r.cagr)}</td>
              <td className="text-right px-3 py-2 tabular-nums">{pct(r.mdd)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

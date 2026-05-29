"use client";
import type { StrategyMetrics } from "@/lib/api/backtest";

function pct(v: number | null, signed = false): string {
  if (v === null || Number.isNaN(v)) return "—";
  const s = v.toFixed(2);
  return signed && v >= 0 ? `+${s}%` : `${s}%`;
}

function won(v: number): string {
  return `₩${Math.round(v).toLocaleString()}`;
}

export function MetricCards({ m }: { m: StrategyMetrics }) {
  const cards: { label: string; value: string; tone?: "up" | "down" }[] = [
    { label: "총수익률", value: pct(m.total_return_pct, true), tone: m.total_return_pct >= 0 ? "up" : "down" },
    { label: "CAGR", value: pct(m.cagr_pct, true), tone: (m.cagr_pct ?? 0) >= 0 ? "up" : "down" },
    { label: "MDD", value: pct(m.mdd_pct), tone: "down" },
    { label: "변동성", value: pct(m.volatility_pct) },
    { label: "초과수익 vs 60/40", value: pct(m.excess_vs_6040_pct, true), tone: m.excess_vs_6040_pct >= 0 ? "up" : "down" },
  ];
  return (
    <div className="grid grid-cols-2 sm:grid-cols-5 gap-2">
      {cards.map((c) => (
        <div key={c.label} className="border border-line p-3">
          <div className="text-[10px] text-fg-muted font-mono">{c.label}</div>
          <div
            className={`font-mono text-base tabular-nums ${
              c.tone === "up" ? "text-bb-up" : c.tone === "down" ? "text-bb-down" : ""
            }`}
          >
            {c.value}
          </div>
        </div>
      ))}
      <div className="col-span-2 sm:col-span-5 flex justify-between text-xs text-fg-muted font-mono pt-1">
        <span>누적 투입 {won(m.total_contributed)}</span>
        <span>최종 평가액 {won(m.final_equity)}</span>
      </div>
    </div>
  );
}

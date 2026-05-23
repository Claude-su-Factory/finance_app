"use client";
import type { Holding } from "@/lib/api/holdings";

export function TopHoldingsCard({ holdings }: { holdings: Holding[] }) {
  const top = [...holdings]
    .sort((a, b) => b.market_value_krw - a.market_value_krw)
    .slice(0, 5);
  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">보유 상위 5</div>
      {top.length === 0 ? (
        <div className="text-fg-muted text-sm font-mono">보유 자산 없음</div>
      ) : (
        <ul className="space-y-1 font-mono text-sm">
          {top.map((h) => (
            <li key={h.id} className="flex items-baseline gap-2">
              <span className="flex-1 truncate">{h.symbol}</span>
              <span className="tabular-nums text-xs text-fg-muted">
                {h.weight_pct.toFixed(1)}%
              </span>
              <span
                className={`tabular-nums text-xs ${
                  h.pnl_pct >= 0 ? "text-bb-up" : "text-bb-down"
                }`}
              >
                {h.pnl_pct >= 0 ? "+" : ""}
                {h.pnl_pct.toFixed(2)}%
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

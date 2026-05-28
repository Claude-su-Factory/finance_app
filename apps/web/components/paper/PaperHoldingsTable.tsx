"use client";

import type { PaperHolding } from "@/lib/api/paper";

export function PaperHoldingsTable({ holdings }: { holdings: PaperHolding[] }) {
  if (holdings.length === 0) {
    return <div className="text-fg-muted text-sm">보유 자산 없음. &quot;+ 매매&quot;로 시작.</div>;
  }
  const fmt = (n: number) => Math.round(n).toLocaleString("ko-KR");
  return (
    <table className="w-full font-mono text-xs">
      <thead className="text-fg-muted border-b border-line">
        <tr>
          <th className="text-left p-2">종목</th>
          <th className="text-right p-2">수량</th>
          <th className="text-right p-2">평단</th>
          <th className="text-right p-2">현재가</th>
          <th className="text-right p-2">평가액 (KRW)</th>
          <th className="text-right p-2">손익</th>
        </tr>
      </thead>
      <tbody>
        {holdings.map((h) => (
          <tr key={h.id} className="border-b border-line/50">
            <td className="p-2">
              <span className="text-bb-accent">{h.symbol}</span> {h.name}
            </td>
            <td className="text-right p-2">{h.quantity}</td>
            <td className="text-right p-2">{fmt(h.avg_cost)} {h.currency}</td>
            <td className="text-right p-2">{fmt(h.current_price)} {h.currency}</td>
            <td className="text-right p-2">{fmt(h.market_value_krw)}</td>
            <td className={`text-right p-2 ${h.pnl_krw >= 0 ? "text-bb-up" : "text-bb-down"}`}>
              {h.pnl_pct >= 0 ? "+" : ""}{h.pnl_pct.toFixed(2)}%
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

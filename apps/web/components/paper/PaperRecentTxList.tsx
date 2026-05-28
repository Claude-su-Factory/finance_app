"use client";

import type { PaperTransaction } from "@/lib/api/paper";

export function PaperRecentTxList({ transactions }: { transactions: PaperTransaction[] }) {
  if (transactions.length === 0) {
    return <div className="text-fg-muted text-sm">매매 이력 없음.</div>;
  }
  const fmt = (n: number) => Math.round(n).toLocaleString("ko-KR");
  return (
    <ul className="space-y-1 font-mono text-xs">
      {transactions.slice(0, 5).map((t) => (
        <li key={t.id} className="flex justify-between py-1 border-b border-line/40">
          <span>
            {t.created_at.slice(0, 10)} ·{" "}
            <span className={t.action === "buy" ? "text-bb-up" : "text-bb-down"}>
              {t.action === "buy" ? "매수" : "매도"}
            </span>{" "}
            <span className="text-bb-accent">{t.symbol}</span>{" "}
            {t.quantity}@{fmt(t.price)} {t.currency}
          </span>
          <span className="text-fg-muted">{fmt(t.total_krw)} KRW</span>
        </li>
      ))}
    </ul>
  );
}

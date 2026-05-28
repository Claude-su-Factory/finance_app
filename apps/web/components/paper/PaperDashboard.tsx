"use client";

import type { PaperAccount } from "@/lib/api/paper";

export function PaperDashboard({
  account, equity, pnlKrw, pnlPct,
}: {
  account: PaperAccount;
  equity: number;
  pnlKrw: number;
  pnlPct: number;
}) {
  const sign = pnlKrw >= 0 ? "text-bb-up" : "text-bb-down";
  const fmt = (n: number) => Math.round(n).toLocaleString("ko-KR");
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
      <Card label="가상 현금" value={`${fmt(account.cash_balance)} KRW`} />
      <Card label="평가액" value={`${fmt(equity)} KRW`} />
      <Card
        label="손익 vs 초기"
        value={`${pnlKrw >= 0 ? "+" : ""}${fmt(pnlKrw)} KRW`}
        sub={`${pnlPct >= 0 ? "+" : ""}${pnlPct.toFixed(2)}%`}
        signClass={sign}
      />
    </div>
  );
}

function Card({ label, value, sub, signClass }: { label: string; value: string; sub?: string; signClass?: string }) {
  return (
    <div className="border border-line bg-bg-subtle p-4">
      <div className="font-mono text-[10px] text-fg-muted tracking-widest mb-1">{label}</div>
      <div className={`font-mono text-xl ${signClass ?? "text-fg"}`}>{value}</div>
      {sub && <div className={`font-mono text-xs mt-1 ${signClass ?? "text-fg-muted"}`}>{sub}</div>}
    </div>
  );
}

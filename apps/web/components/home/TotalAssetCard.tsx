"use client";

import { useEffect, useState } from "react";
import type { Holding } from "@/lib/api/holdings";

export function TotalAssetCard({ holdings }: { holdings: Holding[] }) {
  const total = holdings.reduce((s, h) => s + h.market_value_krw, 0);
  const pnl = holdings.reduce((s, h) => s + h.pnl_krw, 0);
  const cost = holdings.reduce((s, h) => s + h.cost_basis_krw, 0);
  const pnlPct = cost > 0 ? (pnl / cost) * 100 : 0;

  const [displayed, setDisplayed] = useState(0);
  useEffect(() => {
    let raf = 0;
    const start = performance.now();
    const dur = 600;
    const from = displayed;
    function tick(now: number) {
      const t = Math.min(1, (now - start) / dur);
      const eased = 1 - Math.pow(1 - t, 3);
      setDisplayed(from + (total - from) * eased);
      if (t < 1) raf = requestAnimationFrame(tick);
    }
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [total]);

  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-1">총 자산 (KRW)</div>
      <div className="font-mono text-3xl tabular-nums">
        ₩{Math.round(displayed).toLocaleString()}
      </div>
      <div className={`text-sm font-mono mt-1 ${pnl >= 0 ? "text-bb-up" : "text-bb-down"}`}>
        {pnl >= 0 ? "+" : ""}{Math.round(pnl).toLocaleString()} ({pnlPct.toFixed(2)}%)
      </div>
    </div>
  );
}

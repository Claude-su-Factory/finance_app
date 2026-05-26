"use client";

import { useEffect, useState } from "react";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { fetchPriceHistory } from "@/lib/api/history";
import { fetchTicker, type Ticker } from "@/lib/api/market";

export function USIndicesCard() {
  const [spx, setSpx] = useState<ChartPoint[]>([]);
  const [ndx, setNdx] = useState<ChartPoint[]>([]);
  const [tickers, setTickers] = useState<Record<string, Ticker>>({});

  useEffect(() => {
    (async () => {
      const [a, b, ts] = await Promise.all([
        fetchPriceHistory("SPX", "1mo").catch(() => []),
        fetchPriceHistory("NDX", "1mo").catch(() => []),
        fetchTicker().catch(() => [] as Ticker[]),
      ]);
      setSpx(a.map((p) => ({ x: p.date, value: p.close })));
      setNdx(b.map((p) => ({ x: p.date, value: p.close })));
      const m: Record<string, Ticker> = {};
      for (const t of ts) m[t.symbol] = t;
      setTickers(m);
    })();
  }, []);

  return (
    <div className="grid grid-cols-1 gap-4">
      <LineChartCard
        title="S&P 500"
        subtitle="NYSE · 1mo"
        current={tickers.SPX?.price}
        changePct={tickers.SPX?.change_pct}
        points={spx}
      />
      <LineChartCard
        title="NASDAQ 100"
        subtitle="NDX · 1mo"
        current={tickers.NDX?.price}
        changePct={tickers.NDX?.change_pct}
        points={ndx}
      />
    </div>
  );
}

"use client";

import { useEffect, useState } from "react";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { fetchPriceHistory } from "@/lib/api/history";
import { fetchTicker, type Ticker } from "@/lib/api/market";

export function KRIndicesCard() {
  const [kospi, setKospi] = useState<ChartPoint[]>([]);
  const [kosdaq, setKosdaq] = useState<ChartPoint[]>([]);
  const [tickers, setTickers] = useState<Record<string, Ticker>>({});

  useEffect(() => {
    (async () => {
      const [k1, k2, ts] = await Promise.all([
        fetchPriceHistory("KOSPI", "1mo").catch(() => []),
        fetchPriceHistory("KOSDAQ", "1mo").catch(() => []),
        fetchTicker().catch(() => [] as Ticker[]),
      ]);
      setKospi(k1.map((p) => ({ x: p.date, value: p.close })));
      setKosdaq(k2.map((p) => ({ x: p.date, value: p.close })));
      const m: Record<string, Ticker> = {};
      for (const t of ts) m[t.symbol] = t;
      setTickers(m);
    })();
  }, []);

  return (
    <div className="grid grid-cols-1 gap-4">
      <LineChartCard
        title="KOSPI"
        subtitle="KRX 종합지수 · 1mo"
        current={tickers.KOSPI?.price}
        changePct={tickers.KOSPI?.change_pct}
        points={kospi}
      />
      <LineChartCard
        title="KOSDAQ"
        subtitle="코스닥 종합 · 1mo"
        current={tickers.KOSDAQ?.price}
        changePct={tickers.KOSDAQ?.change_pct}
        points={kosdaq}
      />
    </div>
  );
}

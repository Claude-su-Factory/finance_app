"use client";

import { useEffect, useState } from "react";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { fetchFxHistory } from "@/lib/api/history";

export function FxCard({ pair, title }: { pair: [string, string]; title: string }) {
  const [points, setPoints] = useState<ChartPoint[]>([]);
  const [current, setCurrent] = useState<number | undefined>();
  const [changePct, setChangePct] = useState<number | undefined>();

  useEffect(() => {
    (async () => {
      const data = await fetchFxHistory(pair[0], pair[1], 30).catch(() => []);
      const mapped = data.map((p) => ({ x: p.observed_at, value: p.rate }));
      setPoints(mapped);
      if (mapped.length > 0) {
        const last = mapped[mapped.length - 1].value;
        const first = mapped[0].value;
        setCurrent(last);
        if (first > 0) setChangePct(((last - first) / first) * 100);
      }
    })();
  }, [pair]);

  return (
    <LineChartCard
      title={title}
      subtitle={`${pair[0]}→${pair[1]} · 30d 누적`}
      current={current}
      changePct={changePct}
      points={points}
    />
  );
}

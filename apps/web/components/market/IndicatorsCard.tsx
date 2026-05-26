"use client";

import { useEffect, useState } from "react";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { fetchIndicatorHistory } from "@/lib/api/history";

export function IndicatorsCard({
  code,
  title,
  unit,
}: {
  code: string;
  title: string;
  unit?: string;
}) {
  const [points, setPoints] = useState<ChartPoint[]>([]);
  const [current, setCurrent] = useState<number | undefined>();
  const [changePct, setChangePct] = useState<number | undefined>();

  useEffect(() => {
    (async () => {
      const data = await fetchIndicatorHistory(code, 90).catch(() => []);
      const mapped = data.map((p) => ({ x: p.observed_at, value: p.value }));
      setPoints(mapped);
      if (mapped.length > 0) {
        const last = mapped[mapped.length - 1].value;
        const first = mapped[0].value;
        setCurrent(last);
        if (first > 0) setChangePct(((last - first) / first) * 100);
      }
    })();
  }, [code]);

  return (
    <LineChartCard
      title={title}
      subtitle={`${code} · 90d 누적`}
      current={current}
      changePct={changePct}
      points={points}
      unit={unit}
    />
  );
}

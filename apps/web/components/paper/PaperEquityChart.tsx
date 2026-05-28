"use client";

import type { EquityPoint } from "@/lib/api/paper";

export function PaperEquityChart({ series }: { series: EquityPoint[] }) {
  if (series.length < 2) {
    return <div className="text-fg-muted text-sm font-mono">시계열 데이터 부족</div>;
  }
  const values = series.map((p) => p.equity_krw);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const width = 720;
  const height = 80;
  const points = series.map((p, i) => {
    const x = (i / (series.length - 1)) * width;
    const y = height - ((p.equity_krw - min) / range) * height;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  return (
    <div>
      <div className="font-mono text-[10px] text-fg-muted tracking-widest mb-2">평가액 추이 ({series.length}일)</div>
      <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-20">
        <polyline points={points.join(" ")} fill="none" stroke="#FFD500" strokeWidth="1.5" />
      </svg>
      <div className="flex justify-between font-mono text-[10px] text-fg-muted mt-1">
        <span>{series[0].date}</span>
        <span>{series[series.length - 1].date}</span>
      </div>
    </div>
  );
}

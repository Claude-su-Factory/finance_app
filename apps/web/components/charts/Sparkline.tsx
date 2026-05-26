"use client";

import dynamic from "next/dynamic";
import { trendColor } from "./chart-tokens";

// recharts 컴포넌트의 defaultProps 타입이 Next.js dynamic과 충돌 — as any로 회피
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const ResponsiveContainer = dynamic<any>(
  () => import("recharts").then((m) => m.ResponsiveContainer),
  { ssr: false },
);
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const LineChart = dynamic<any>(() => import("recharts").then((m) => m.LineChart), { ssr: false });
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const Line = dynamic<any>(() => import("recharts").then((m) => m.Line), { ssr: false });

export function Sparkline({
  points,
  width = 80,
  height = 24,
}: {
  points: { value: number }[];
  width?: number;
  height?: number;
}) {
  if (points.length < 2) {
    return <span className="inline-block text-fg-muted text-xs" style={{ width, height }}>—</span>;
  }
  const color = trendColor(points);
  return (
    <span style={{ display: "inline-block", width, height }}>
      <ResponsiveContainer>
        <LineChart data={points} margin={{ top: 2, right: 2, bottom: 2, left: 2 }}>
          <Line type="monotone" dataKey="value" stroke={color} strokeWidth={1} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </span>
  );
}

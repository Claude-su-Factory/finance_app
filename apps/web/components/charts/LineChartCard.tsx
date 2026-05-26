"use client";

import dynamic from "next/dynamic";
import { CHART_COLORS, trendColor } from "./chart-tokens";

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
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const XAxis = dynamic<any>(() => import("recharts").then((m) => m.XAxis), { ssr: false });
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const YAxis = dynamic<any>(() => import("recharts").then((m) => m.YAxis), { ssr: false });
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const Tooltip = dynamic<any>(() => import("recharts").then((m) => m.Tooltip), { ssr: false });

export type ChartPoint = { x: string; value: number };

export function LineChartCard({
  title,
  subtitle,
  current,
  changePct,
  points,
  height = 160,
  unit,
}: {
  title: string;
  subtitle?: string;
  current?: number;
  changePct?: number;
  points: ChartPoint[];
  height?: number;
  unit?: string;
}) {
  const color = trendColor(points);
  const positive = (changePct ?? 0) >= 0;
  return (
    <div className="border border-line p-4">
      <div className="flex items-baseline justify-between mb-2">
        <div>
          <div className="font-mono text-sm">{title}</div>
          {subtitle && <div className="text-xs text-fg-muted font-mono">{subtitle}</div>}
        </div>
        {current !== undefined && (
          <div className="text-right">
            <div className="font-mono text-base tabular-nums">
              {current.toLocaleString()}
              {unit ? ` ${unit}` : ""}
            </div>
            {changePct !== undefined && (
              <div className={`text-xs font-mono tabular-nums ${positive ? "text-bb-up" : "text-bb-down"}`}>
                {positive ? "+" : ""}{changePct.toFixed(2)}%
              </div>
            )}
          </div>
        )}
      </div>
      <div style={{ width: "100%", height }}>
        {points.length === 0 ? (
          <div className="h-full flex items-center justify-center text-xs text-fg-muted font-mono">
            데이터 없음
          </div>
        ) : (
          <ResponsiveContainer>
            <LineChart data={points} margin={{ top: 5, right: 8, bottom: 0, left: 0 }}>
              <XAxis dataKey="x" tick={{ fontSize: 10, fill: CHART_COLORS.muted }} hide />
              <YAxis domain={["auto", "auto"]} tick={{ fontSize: 10, fill: CHART_COLORS.muted }} hide />
              <Tooltip
                contentStyle={{ background: "#0a0a0a", border: `1px solid ${CHART_COLORS.line}`, fontSize: 11, fontFamily: "monospace" }}
                labelStyle={{ color: CHART_COLORS.muted }}
                itemStyle={{ color }}
                formatter={(v: unknown) => (typeof v === "number" ? v.toLocaleString() : String(v))}
              />
              <Line type="monotone" dataKey="value" stroke={color} strokeWidth={1.5} dot={false} isAnimationActive={false} />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}

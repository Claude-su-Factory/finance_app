"use client";
import dynamic from "next/dynamic";
import { CHART_COLORS } from "@/components/charts/chart-tokens";
import type { BacktestResult } from "@/lib/api/backtest";

/* eslint-disable @typescript-eslint/no-explicit-any */
const ResponsiveContainer = dynamic<any>(() => import("recharts").then((m) => m.ResponsiveContainer), { ssr: false });
const LineChart = dynamic<any>(() => import("recharts").then((m) => m.LineChart), { ssr: false });
const Line = dynamic<any>(() => import("recharts").then((m) => m.Line), { ssr: false });
const XAxis = dynamic<any>(() => import("recharts").then((m) => m.XAxis), { ssr: false });
const YAxis = dynamic<any>(() => import("recharts").then((m) => m.YAxis), { ssr: false });
const Tooltip = dynamic<any>(() => import("recharts").then((m) => m.Tooltip), { ssr: false });
const Legend = dynamic<any>(() => import("recharts").then((m) => m.Legend), { ssr: false });
/* eslint-enable @typescript-eslint/no-explicit-any */

const SIXTY_FORTY_COLOR = "#A78BFA"; // 보라 — 토큰 외 4번째 라인 구분용

type ChartRow = {
  x: string;
  strategy?: number;
  kospi?: number;
  spx?: number;
  sixtyForty?: number;
  contributed?: number;
};

function buildRows(res: BacktestResult): ChartRow[] {
  const byDate = new Map<string, ChartRow>();
  const ensure = (d: string): ChartRow => {
    let row = byDate.get(d);
    if (!row) {
      row = { x: d };
      byDate.set(d, row);
    }
    return row;
  };
  for (const p of res.equity_series) ensure(p.date).strategy = p.value;
  for (const p of res.benchmarks.kospi.equity_series) ensure(p.date).kospi = p.value;
  for (const p of res.benchmarks.spx.equity_series) ensure(p.date).spx = p.value;
  for (const p of res.benchmarks.sixty_forty.equity_series) ensure(p.date).sixtyForty = p.value;
  for (const p of res.contributed_series) ensure(p.date).contributed = p.value;
  return [...byDate.values()].sort((a, b) => a.x.localeCompare(b.x));
}

export function BacktestEquityChart({ result }: { result: BacktestResult }) {
  const rows = buildRows(result);
  return (
    <div className="border border-line p-4">
      <div className="font-mono text-sm mb-2">평가액 추이 (₩)</div>
      <div style={{ width: "100%", height: 320 }}>
        <ResponsiveContainer>
          <LineChart data={rows} margin={{ top: 5, right: 12, bottom: 0, left: 0 }}>
            <XAxis dataKey="x" tick={{ fontSize: 10, fill: CHART_COLORS.muted }} minTickGap={40} />
            <YAxis
              tick={{ fontSize: 10, fill: CHART_COLORS.muted }}
              tickFormatter={(v: number) => `${Math.round(v / 1_000_000)}M`}
              width={40}
            />
            <Tooltip
              contentStyle={{ background: "#0a0a0a", border: `1px solid ${CHART_COLORS.line}`, fontSize: 11, fontFamily: "monospace" }}
              labelStyle={{ color: CHART_COLORS.muted }}
              formatter={(v: unknown) => (typeof v === "number" ? `₩${v.toLocaleString()}` : String(v))}
            />
            <Legend wrapperStyle={{ fontSize: 11, fontFamily: "monospace" }} />
            <Line type="monotone" dataKey="strategy" name="내 전략" stroke={CHART_COLORS.accent} strokeWidth={2} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="kospi" name="KOSPI" stroke={CHART_COLORS.up} strokeWidth={1} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="spx" name="S&P 500" stroke={CHART_COLORS.warn} strokeWidth={1} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="sixtyForty" name="한미 60/40" stroke={SIXTY_FORTY_COLOR} strokeWidth={1} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="contributed" name="투입 원금" stroke={CHART_COLORS.muted} strokeWidth={1} strokeDasharray="4 3" dot={false} isAnimationActive={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

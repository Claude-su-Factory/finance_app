"use client";
import type { BacktestResult } from "@/lib/api/backtest";
import { MetricCards } from "./MetricCards";
import { BacktestEquityChart } from "./BacktestEquityChart";
import { CompareTable } from "./CompareTable";
import { CoverageNotice } from "./CoverageNotice";

export function BacktestResults({ result }: { result: BacktestResult }) {
  return (
    <div className="space-y-3">
      <CoverageNotice warnings={result.coverage_warnings} clampedStart={result.clamped_start} />
      <MetricCards m={result.metrics} />
      <BacktestEquityChart result={result} />
      <CompareTable result={result} />
    </div>
  );
}

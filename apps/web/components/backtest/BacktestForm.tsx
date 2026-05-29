"use client";
import { useState } from "react";
import { PeriodPicker } from "./PeriodPicker";
import { CashInputs, type ContributionMode } from "./CashInputs";
import { RebalanceSelect } from "./RebalanceSelect";
import { BasketBuilder, type BasketRow } from "./BasketBuilder";
import type {
  BacktestPeriod,
  BacktestReq,
  RebalanceFreq,
} from "@/lib/api/backtest";

export function BacktestForm({
  onRun,
  running,
}: {
  onRun: (req: BacktestReq) => void;
  running: boolean;
}) {
  const [period, setPeriod] = useState<BacktestPeriod>("3Y");
  const [initialCash, setInitialCash] = useState(10_000_000);
  const [mode, setMode] = useState<ContributionMode>("lump");
  const [monthly, setMonthly] = useState(500_000);
  const [rebalance, setRebalance] = useState<RebalanceFreq>("quarterly");
  const [rows, setRows] = useState<BasketRow[]>([]);

  const canRun =
    rows.length > 0 &&
    rows.every((r) => r.weight > 0) &&
    initialCash > 0 &&
    !running;

  function submit() {
    if (!canRun) return;
    onRun({
      period,
      initial_cash: initialCash,
      monthly_contribution: mode === "dca" ? monthly : 0,
      rebalance,
      basket: rows.map((r) => ({ instrument_id: r.inst.id, weight: r.weight })),
    });
  }

  return (
    <div className="border border-line p-4 space-y-4">
      <div>
        <div className="text-xs text-fg-muted mb-1">기간 (종료일 = 오늘)</div>
        <PeriodPicker value={period} onChange={setPeriod} />
      </div>
      <CashInputs
        initialCash={initialCash}
        mode={mode}
        monthly={monthly}
        onInitialCash={setInitialCash}
        onMode={setMode}
        onMonthly={setMonthly}
      />
      <BasketBuilder rows={rows} onChange={setRows} />
      <div className="flex gap-3 items-end">
        <RebalanceSelect value={rebalance} onChange={setRebalance} />
        <button
          type="button"
          onClick={submit}
          disabled={!canRun}
          className="flex-1 bg-bb-accent text-bg font-mono text-sm py-2 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {running ? "백테스트 중…" : "백테스트 실행"}
        </button>
      </div>
    </div>
  );
}

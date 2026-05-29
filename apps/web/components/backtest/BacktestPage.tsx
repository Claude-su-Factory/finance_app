"use client";
import { useState } from "react";
import { BacktestForm } from "./BacktestForm";
import { BacktestResults } from "./BacktestResults";
import {
  runBacktest,
  isBacktestError,
  type BacktestReq,
  type BacktestResult,
} from "@/lib/api/backtest";

export function BacktestPage() {
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<BacktestResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleRun(req: BacktestReq) {
    setRunning(true);
    setError(null);
    try {
      const res = await runBacktest(req);
      if (isBacktestError(res)) {
        setError(res.error.message);
        setResult(null);
      } else {
        setResult(res);
      }
    } catch {
      setError("백테스트 실행 중 오류가 발생했습니다.");
      setResult(null);
    } finally {
      setRunning(false);
    }
  }

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-4">
      <div>
        <h1 className="font-mono text-lg text-bb-accent">백테스트</h1>
        <p className="text-xs text-fg-muted font-mono mt-1">
          과거 데이터로 바스켓 + 적립·리밸런싱 전략을 시뮬레이션하고 KOSPI·S&P·한미 60/40과 비교합니다. (최근 5년·종료일 오늘)
        </p>
      </div>
      <BacktestForm onRun={handleRun} running={running} />
      {error && (
        <div className="border border-bb-down/40 bg-bb-down/5 p-3 text-xs font-mono text-bb-down">
          {error}
        </div>
      )}
      {result ? (
        <BacktestResults result={result} />
      ) : (
        !error && (
          <div className="border border-line border-dashed p-8 text-center text-sm text-fg-muted font-mono">
            바스켓과 전략을 설정하고 실행하세요.
          </div>
        )
      )}
    </div>
  );
}

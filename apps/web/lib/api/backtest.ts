import { authFetch } from "./auth-fetch";

export type BacktestPeriod = "1Y" | "3Y" | "5Y" | "all";
export type RebalanceFreq = "none" | "quarterly" | "semiannual" | "annual";

export type BasketInput = { instrument_id: string; weight: number };

export type BacktestReq = {
  period: BacktestPeriod;
  initial_cash: number;
  monthly_contribution: number;
  basket: BasketInput[];
  rebalance: RebalanceFreq;
};

export type ValuePoint = { date: string; value: number };

export type NormalizedLeg = {
  instrument_id: string;
  symbol: string;
  name: string;
  weight: number; // 정규화된 비중 0..1
};

// 벤치마크 metrics (Go SeriesMetrics) — twr_pct 포함
export type SeriesMetrics = {
  total_return_pct: number;
  cagr_pct: number | null;
  mdd_pct: number;
  volatility_pct: number;
  twr_pct: number;
};

export type BenchmarkResult = {
  equity_series: ValuePoint[];
  metrics: SeriesMetrics;
};

// 전략 metrics (Go StrategyMetrics) — excess·투입·최종 포함, twr 제외
export type StrategyMetrics = {
  total_return_pct: number;
  cagr_pct: number | null;
  mdd_pct: number;
  volatility_pct: number;
  excess_vs_6040_pct: number;
  total_contributed: number;
  final_equity: number;
};

export type CoverageWarning = {
  symbol: string;
  first_available: string;
  message: string;
};

export type BacktestResult = {
  clamped_start: string;
  end: string;
  normalized_basket: NormalizedLeg[];
  equity_series: ValuePoint[];
  contributed_series: ValuePoint[];
  benchmarks: {
    kospi: BenchmarkResult;
    spx: BenchmarkResult;
    sixty_forty: BenchmarkResult;
  };
  metrics: StrategyMetrics;
  coverage_warnings: CoverageWarning[];
};

export type BacktestErrorBody = {
  error: {
    code: "VALIDATION" | "ASSET_NOT_SUPPORTED" | "INSUFFICIENT_DATA";
    message: string;
    min_days?: number;
    current_days?: number;
  };
};

// 422(검증·자산미지원·데이터부족)는 throw 대신 union 반환 — 알파 카드(getAlpha)와 동일 패턴.
export async function runBacktest(
  req: BacktestReq,
): Promise<BacktestResult | BacktestErrorBody> {
  const res = await authFetch("/v1/backtest/run", {
    method: "POST",
    body: JSON.stringify(req),
  });
  if (res.status === 422 || res.status === 400) {
    return (await res.json()) as BacktestErrorBody;
  }
  if (!res.ok) {
    throw new Error(`backtest failed: ${res.status}`);
  }
  return (await res.json()) as BacktestResult;
}

export function isBacktestError(
  r: BacktestResult | BacktestErrorBody,
): r is BacktestErrorBody {
  return (r as BacktestErrorBody).error !== undefined;
}

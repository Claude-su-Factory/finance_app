import { authFetch } from "./auth-fetch";

export type AlphaPeriod = "1m" | "90d" | "1y" | "all";

export type AlphaSeriesPoint = { date: string; value_pct: number };

export type AlphaBenchmark = {
  key: "kospi" | "sp500" | "kr_us_6040";
  label: string;
  total_return_pct: number;
  alpha_pp: number;
  series: AlphaSeriesPoint[] | null;
};

export type AlphaResult = {
  period: AlphaPeriod;
  days_requested: number;
  days_used: number;
  since: string;
  fx_mode: "spot";
  model: "current_holdings_backward_simulation";
  portfolio: {
    total_return_pct: number;
    series: AlphaSeriesPoint[];
    data_gaps?: { symbol: string; first_price_date: string }[];
  };
  benchmarks: AlphaBenchmark[];
};

export type AlphaInsufficient = {
  error: {
    code: "INSUFFICIENT_DATA";
    reason: "account_too_young" | "no_holdings";
    message: string;
    min_days: number;
    current_days: number;
  };
};

// HTTP 422 응답을 throw로 처리하면 사용 측이 try/catch + status 분기 부담.
// 정상 결과 또는 부족 상태를 union으로 반환.
export async function getAlpha(period: AlphaPeriod): Promise<AlphaResult | AlphaInsufficient> {
  const res = await authFetch(`/v1/portfolio/alpha?period=${period}`);
  if (res.status === 422) {
    return (await res.json()) as AlphaInsufficient;
  }
  if (!res.ok) {
    throw new Error(`alpha fetch failed: ${res.status}`);
  }
  return (await res.json()) as AlphaResult;
}

export function isInsufficient(r: AlphaResult | AlphaInsufficient): r is AlphaInsufficient {
  return (r as AlphaInsufficient).error !== undefined;
}

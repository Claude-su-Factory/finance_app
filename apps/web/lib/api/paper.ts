import { authFetch } from "./auth-fetch";

export type PaperAccount = {
  user_id: string;
  initial_cash: number;
  cash_balance: number;
  base_currency: "KRW";
  created_at: string;
  updated_at: string;
};

export type PaperHolding = {
  id: string;
  user_id: string;
  instrument_id: string;
  symbol: string;
  name: string;
  currency: string;
  quantity: number;
  avg_cost: number;
  current_price: number;
  market_value: number;
  market_value_krw: number;
  pnl_krw: number;
  pnl_pct: number;
  created_at: string;
  updated_at: string;
};

export type PaperTransaction = {
  id: string;
  user_id: string;
  instrument_id: string;
  symbol: string;
  action: "buy" | "sell";
  quantity: number;
  price: number;
  currency: string;
  fx_to_krw: number;
  total_krw: number;
  active: boolean;
  created_at: string;
};

export type EquityPoint = { date: string; equity_krw: number };

export type PaperPortfolioResponse = {
  account: PaperAccount;
  holdings: PaperHolding[];
  summary: {
    total_equity_krw: number;
    total_pnl_krw: number;
    total_pnl_pct: number;
  };
  equity_series: EquityPoint[];
};

export type TradeError = {
  code: string;
  message: string;
  need_krw?: number;
  have_krw?: number;
  need_qty?: number;
  have_qty?: number;
};

export type TradeResult =
  | {
      transaction: PaperTransaction;
      new_cash_balance: number;
      holding: { instrument_id: string; quantity: number; avg_cost: number } | null;
    }
  | { error: TradeError };

export async function getPortfolio(period: "1m" | "90d" | "1y" | "all" = "90d"): Promise<PaperPortfolioResponse> {
  const res = await authFetch(`/v1/paper/portfolio?period=${period}`);
  if (!res.ok) throw new Error(`portfolio failed: ${res.status}`);
  return res.json();
}

export async function listTransactions(
  limit = 50,
  before?: string,
): Promise<{ transactions: PaperTransaction[]; has_more: boolean }> {
  const q = new URLSearchParams({ limit: String(limit) });
  if (before) q.set("before", before);
  const res = await authFetch(`/v1/paper/transactions?${q}`);
  if (!res.ok) throw new Error(`list tx failed: ${res.status}`);
  return res.json();
}

export async function trade(body: {
  instrument_id: string;
  action: "buy" | "sell";
  quantity: number;
  reason?: string;
}): Promise<TradeResult> {
  const res = await authFetch("/v1/paper/transactions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (res.status === 422) return res.json();
  if (!res.ok) throw new Error(`trade failed: ${res.status}`);
  return res.json();
}

export async function resetPortfolio(initialCash?: number): Promise<PaperAccount> {
  const res = await authFetch("/v1/paper/reset", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(initialCash != null ? { initial_cash: initialCash } : {}),
  });
  if (!res.ok) throw new Error(`reset failed: ${res.status}`);
  return res.json();
}

export function isTradeError(r: TradeResult): r is { error: TradeError } {
  return (r as { error?: unknown }).error !== undefined;
}

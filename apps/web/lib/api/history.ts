import { authFetch, readError } from "./auth-fetch";

export type PricePoint = { date: string; close: number };
export type IndicatorPoint = { observed_at: string; value: number };
export type FxPoint = { observed_at: string; rate: number };

export type Range = "1w" | "1mo" | "6mo" | "1y" | "5y";

export async function fetchPriceHistory(symbol: string, range: Range): Promise<PricePoint[]> {
  const res = await authFetch(`/v1/prices/history?symbol=${encodeURIComponent(symbol)}&range=${range}`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.points ?? [];
}

export async function fetchPriceHistoryBatch(
  ids: string[],
  range: Range = "1w",
): Promise<Record<string, PricePoint[]>> {
  if (ids.length === 0) return {};
  const res = await authFetch(`/v1/prices/history?ids=${ids.join(",")}&range=${range}`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.items ?? {};
}

export async function fetchIndicatorHistory(code: string, days = 90): Promise<IndicatorPoint[]> {
  const res = await authFetch(`/v1/indicators/history?code=${encodeURIComponent(code)}&days=${days}`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.points ?? [];
}

export async function fetchFxHistory(base: string, quote: string, days = 30): Promise<FxPoint[]> {
  const res = await authFetch(`/v1/fx/history?base=${base}&quote=${quote}&days=${days}`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.points ?? [];
}

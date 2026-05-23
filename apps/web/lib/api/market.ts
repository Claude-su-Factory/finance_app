import { authFetch } from "./auth-fetch";

export type Ticker = {
  symbol: string;
  name: string;
  price: number;
  change_pct: number;
};

export async function fetchTicker(): Promise<Ticker[]> {
  const res = await authFetch("/v1/market/ticker");
  if (!res.ok) return [];
  return res.json() as Promise<Ticker[]>;
}

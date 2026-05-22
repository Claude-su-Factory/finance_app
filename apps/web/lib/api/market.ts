export type Ticker = {
  symbol: string;
  name: string;
  price: number;
  change_pct: number;
};

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function fetchTicker(accessToken: string): Promise<Ticker[]> {
  const res = await fetch(`${API_BASE}/v1/market/ticker`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    cache: "no-store",
  });
  if (!res.ok) return [];
  return res.json() as Promise<Ticker[]>;
}

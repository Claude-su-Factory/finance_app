import { authFetch, readError } from "./auth-fetch";

export type WatchlistItem = {
  instrument_id: string;
  added_at: string;
  symbol: string;
  exchange: string;
  name: string;
  asset_class: string;
  currency: string;
  price: number;
  change_pct: number;
};

export async function listWatchlist(): Promise<WatchlistItem[]> {
  const res = await authFetch("/v1/watchlist");
  if (!res.ok) throw await readError(res);
  return res.json() as Promise<WatchlistItem[]>;
}

export async function addWatchlist(instrument_id: string): Promise<void> {
  const res = await authFetch("/v1/watchlist", {
    method: "POST",
    body: JSON.stringify({ instrument_id }),
  });
  if (!res.ok && res.status !== 201) throw await readError(res);
}

export async function removeWatchlist(instrument_id: string): Promise<void> {
  const res = await authFetch(`/v1/watchlist/${instrument_id}`, {
    method: "DELETE",
  });
  if (!res.ok && res.status !== 204) throw await readError(res);
}

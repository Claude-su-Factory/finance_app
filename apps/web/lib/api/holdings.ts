import { authFetch, readError } from "./auth-fetch";

export type Holding = {
  id: string;
  instrument_id: string;
  quantity: number;
  avg_cost: number;
  opened_at?: string | null;
  note?: string | null;
  created_at: string;
  updated_at: string;
  symbol: string;
  exchange: string;
  name: string;
  asset_class: string;
  currency: string;
  current_price: number;
  market_value: number;
  market_value_krw: number;
  cost_basis_krw: number;
  pnl_krw: number;
  pnl_pct: number;
  weight_pct: number;
};

export async function listHoldings(): Promise<Holding[]> {
  const res = await authFetch("/v1/holdings");
  if (!res.ok) throw await readError(res);
  return res.json() as Promise<Holding[]>;
}

export async function createHolding(input: {
  instrument_id: string;
  quantity: number;
  avg_cost: number;
  opened_at?: string;
  note?: string;
  reason?: string;
}): Promise<Holding> {
  const res = await authFetch("/v1/holdings", {
    method: "POST",
    body: JSON.stringify(input),
  });
  if (!res.ok) throw await readError(res);
  return res.json() as Promise<Holding>;
}

export async function updateHolding(
  id: string,
  patch: Partial<Pick<Holding, "quantity" | "avg_cost" | "note" | "opened_at">> & { reason?: string },
): Promise<Holding> {
  const res = await authFetch(`/v1/holdings/${id}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw await readError(res);
  return res.json() as Promise<Holding>;
}

export async function deleteHolding(id: string): Promise<void> {
  const res = await authFetch(`/v1/holdings/${id}`, { method: "DELETE" });
  if (!res.ok && res.status !== 204) throw await readError(res);
}

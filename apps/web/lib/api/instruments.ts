import { authFetch, readError } from "./auth-fetch";

export type InstrumentResult = {
  id: string;
  symbol: string;
  exchange: string;
  name: string;
  currency: string; // "KRW" | "USD"
  asset_class: string; // "KR_STOCK" | "US_STOCK" | "ETF" | "INDEX" | "FX" | "CASH"
};

export async function searchInstruments(
  query: string,
): Promise<InstrumentResult[]> {
  if (!query.trim()) return [];
  const res = await authFetch(
    `/v1/instruments/search?q=${encodeURIComponent(query)}`,
  );
  if (!res.ok) throw await readError(res);
  return res.json() as Promise<InstrumentResult[]>;
}

export async function selectInstrument(
  query: string,
  instrument_id: string,
): Promise<void> {
  await authFetch("/v1/instruments/select", {
    method: "POST",
    body: JSON.stringify({ query, instrument_id }),
  });
}

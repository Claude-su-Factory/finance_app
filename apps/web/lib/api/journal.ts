import { authFetch } from "./auth-fetch";

export type JournalAction = "buy" | "sell" | "observation" | "other";

export type JournalEntry = {
  id: string;
  user_id: string;
  entry_type: "auto" | "manual";
  action?: JournalAction;
  related_holding_id?: string;
  related_holding?: { symbol: string; name: string };
  related_symbols: string[];
  title?: string;
  content: string;
  created_at: string;
  updated_at: string;
};

export type AnalysisRun = {
  id: string;
  user_id: string;
  run_type: "auto_monthly" | "on_demand";
  period_start: string;
  period_end: string;
  entries_count: number;
  content_md: string;
  model: string;
  created_at: string;
};

export type JournalListResult = { entries: JournalEntry[]; has_more: boolean };

export async function listEntries(limit = 50, before?: string): Promise<JournalListResult> {
  const q = new URLSearchParams();
  q.set("limit", String(limit));
  if (before) q.set("before", before);
  const res = await authFetch(`/v1/journal/entries?${q}`);
  if (!res.ok) throw new Error(`list failed: ${res.status}`);
  return res.json();
}

export async function createEntry(body: {
  action?: JournalAction;
  related_symbols?: string[];
  title?: string;
  content: string;
}): Promise<JournalEntry> {
  const res = await authFetch("/v1/journal/entries", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err?.error?.message ?? `create failed: ${res.status}`);
  }
  return res.json();
}

export async function patchEntry(
  id: string,
  body: Partial<{
    action: JournalAction;
    related_symbols: string[];
    title: string;
    content: string;
  }>
): Promise<JournalEntry | { error: { code: string; message: string } }> {
  const res = await authFetch(`/v1/journal/entries/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (res.status === 422) return res.json();
  if (!res.ok) throw new Error(`patch failed: ${res.status}`);
  return res.json();
}

export async function deleteEntry(id: string): Promise<void> {
  const res = await authFetch(`/v1/journal/entries/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error(`delete failed: ${res.status}`);
}

export type AnalyzeResult =
  | AnalysisRun
  | { error: { code: string; reason: string; message: string } };

export async function analyzeNow(periodDays = 90): Promise<AnalyzeResult> {
  const res = await authFetch("/v1/journal/analyze", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ period_days: periodDays }),
  });
  if (res.status === 422 || res.status === 429) return res.json();
  if (!res.ok) throw new Error(`analyze failed: ${res.status}`);
  return res.json();
}

export async function listAnalyses(limit = 20): Promise<{ analyses: AnalysisRun[] }> {
  const res = await authFetch(`/v1/journal/analyses?limit=${limit}`);
  if (!res.ok) throw new Error(`analyses failed: ${res.status}`);
  return res.json();
}

export function isAnalyzeError(
  r: AnalyzeResult
): r is { error: { code: string; reason: string; message: string } } {
  return (r as { error?: unknown }).error !== undefined;
}

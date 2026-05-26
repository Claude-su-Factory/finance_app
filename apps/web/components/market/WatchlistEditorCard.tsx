"use client";

import { useEffect, useState } from "react";
import { InstrumentSearchInput } from "@/components/portfolio/InstrumentSearchInput";
import { listWatchlist, addWatchlist, removeWatchlist, type WatchlistItem } from "@/lib/api/watchlist";
import type { InstrumentResult } from "@/lib/api/instruments";
import { Sparkline } from "@/components/charts/Sparkline";
import { fetchPriceHistoryBatch } from "@/lib/api/history";

export function WatchlistEditorCard() {
  const [items, setItems] = useState<WatchlistItem[]>([]);
  const [sparks, setSparks] = useState<Record<string, { value: number }[]>>({});
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      const data = await listWatchlist();
      setItems(data);
      const ids = data.map((w) => w.instrument_id);
      if (ids.length > 0) {
        const batch = await fetchPriceHistoryBatch(ids, "1w").catch(() => ({}));
        const sp: Record<string, { value: number }[]> = {};
        for (const [iid, points] of Object.entries(batch)) {
          sp[iid] = points.map((p) => ({ value: p.close }));
        }
        setSparks(sp);
      }
    } catch (e: unknown) {
      setErr((e as { message?: string })?.message ?? "로드 실패");
    }
  }

  useEffect(() => { load(); }, []);

  async function handleAdd(inst: InstrumentResult) {
    if (inst.asset_class !== "KR_STOCK" && inst.asset_class !== "US_STOCK" && inst.asset_class !== "ETF") {
      setErr("관심 종목은 주식·ETF만 추가할 수 있습니다.");
      return;
    }
    try {
      await addWatchlist(inst.id);
      await load();
      setErr(null);
    } catch (e: unknown) {
      const code = (e as { code?: string })?.code;
      setErr(code === "CONFLICT" ? "이미 추가됨" : "추가 실패");
    }
  }

  async function handleRemove(iid: string) {
    try {
      await removeWatchlist(iid);
      setItems((prev) => prev.filter((x) => x.instrument_id !== iid));
    } catch {
      setErr("삭제 실패");
    }
  }

  return (
    <div className="border border-line p-4 md:col-span-2 lg:col-span-3">
      <div className="font-mono text-sm mb-3">관심 종목</div>
      <div className="mb-3">
        <InstrumentSearchInput onSelect={handleAdd} placeholder="종목 검색하여 추가" />
      </div>
      {err && <p className="text-bb-down text-xs font-mono mb-2">{err}</p>}
      {items.length === 0 ? (
        <p className="text-fg-muted text-xs font-mono">관심 종목이 없습니다. 위에서 검색하여 추가하세요.</p>
      ) : (
        <ul className="divide-y divide-line/50">
          {items.map((w) => (
            <li key={w.instrument_id} className="flex items-center gap-3 py-2 font-mono text-sm">
              <div className="flex-1 min-w-0">
                <div className="truncate">{w.symbol}</div>
                <div className="text-xs text-fg-muted truncate">{w.name}</div>
              </div>
              <Sparkline points={sparks[w.instrument_id] ?? []} width={80} height={24} />
              <div className="text-right tabular-nums w-24">
                {w.price > 0 ? w.price.toLocaleString() : "—"}
              </div>
              <div className={`text-xs tabular-nums w-16 text-right ${w.change_pct >= 0 ? "text-bb-up" : "text-bb-down"}`}>
                {w.change_pct >= 0 ? "+" : ""}{w.change_pct.toFixed(2)}%
              </div>
              <button
                onClick={() => handleRemove(w.instrument_id)}
                className="text-xs text-fg-muted hover:text-bb-down px-2"
                title="삭제"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

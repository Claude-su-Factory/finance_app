"use client";

import { useEffect, useState } from "react";
import { listHoldings, type Holding } from "@/lib/api/holdings";
import { Sparkline } from "@/components/charts/Sparkline";
import { fetchPriceHistoryBatch } from "@/lib/api/history";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { AddHoldingDialog } from "./AddHoldingDialog";
import { EditHoldingDialog } from "./EditHoldingDialog";
import { DeleteConfirmDialog } from "./DeleteConfirmDialog";
import { HoldingDetailPanel } from "./HoldingDetailPanel";

type SortKey = "weight_pct" | "market_value_krw" | "pnl_pct" | "symbol";

export function HoldingsTable() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("weight_pct");
  const [addOpen, setAddOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<Holding | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Holding | null>(null);
  const [sparks, setSparks] = useState<Record<string, { value: number }[]>>({});
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailTarget, setDetailTarget] = useState<Holding | null>(null);

  async function load() {
    try {
      const data = await listHoldings();
      setHoldings(data);
      setErr(null);
      // 스파크라인 batch (7일)
      const ids = data.map((h) => h.instrument_id);
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
      setHoldings([]);
    }
  }

  useEffect(() => {
    load();
  }, []);

  if (holdings === null) {
    return (
      <div className="space-y-2">
        {[1, 2, 3].map((i) => <Skeleton key={i} className="h-10 w-full" />)}
      </div>
    );
  }

  const filtered = holdings.filter((h) =>
    !query || h.symbol.toLowerCase().includes(query.toLowerCase()) || h.name.toLowerCase().includes(query.toLowerCase())
  );
  const sorted = [...filtered].sort((a, b) => {
    if (sortKey === "symbol") return a.symbol.localeCompare(b.symbol);
    return (b[sortKey] as number) - (a[sortKey] as number);
  });

  return (
    <>
      {holdings.length === 0 ? (
        <div className="border border-line p-12 text-center text-fg-muted">
          <p className="font-mono">보유 자산이 없습니다.</p>
          <p className="text-xs mt-2">첫 종목을 등록하세요.</p>
          <Button className="mt-4" onClick={() => setAddOpen(true)}>+ 첫 종목 추가</Button>
        </div>
      ) : (
        <div>
          <div className="flex items-center gap-2 mb-4">
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="종목·코드 검색"
              className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-64"
            />
            <select
              value={sortKey}
              onChange={(e) => setSortKey(e.target.value as SortKey)}
              className="bg-bg-deep border border-line px-2 py-1.5 text-sm font-mono"
            >
              <option value="weight_pct">비중 ↓</option>
              <option value="market_value_krw">평가액 ↓</option>
              <option value="pnl_pct">수익률 ↓</option>
              <option value="symbol">종목 A→Z</option>
            </select>
            {err && <span className="text-bb-down text-xs">{err}</span>}
            <Button onClick={() => setAddOpen(true)} className="ml-auto">+ 추가</Button>
          </div>

          <div className="border border-line overflow-x-auto">
            <table className="w-full text-sm font-mono">
              <thead className="border-b border-line bg-bg-deep text-fg-muted text-xs">
                <tr>
                  <th className="text-left px-3 py-2">종목</th>
                  <th className="text-right px-3 py-2">수량</th>
                  <th className="text-right px-3 py-2">평단가</th>
                  <th className="text-right px-3 py-2">현재가</th>
                  <th className="text-right px-3 py-2">평가액 (KRW)</th>
                  <th className="text-right px-3 py-2">손익 (KRW)</th>
                  <th className="text-right px-3 py-2">수익률</th>
                  <th className="text-right px-3 py-2">비중</th>
                  <th className="text-right px-3 py-2">7일</th>
                  <th className="px-3 py-2"></th>
                </tr>
              </thead>
              <tbody>
                {sorted.map((h) => {
                  const selected = detailOpen && detailTarget?.id === h.id;
                  return (
                  <tr
                    key={h.id}
                    onClick={() => { setDetailTarget(h); setDetailOpen(true); }}
                    className={`border-b border-line/50 cursor-pointer ${
                      selected ? "bg-bb-accent/10" : "hover:bg-bg-deep/50"
                    }`}
                  >
                    <td className="px-3 py-2">
                      <div>{h.symbol}</div>
                      <div className="text-xs text-fg-muted">{h.name}</div>
                    </td>
                    <td className="text-right px-3 py-2">{h.quantity}</td>
                    <td className="text-right px-3 py-2">{h.avg_cost.toLocaleString()}</td>
                    <td className="text-right px-3 py-2">{h.current_price > 0 ? h.current_price.toLocaleString() : "—"}</td>
                    <td className="text-right px-3 py-2">{Math.round(h.market_value_krw).toLocaleString()}</td>
                    <td className={`text-right px-3 py-2 ${h.pnl_krw >= 0 ? "text-bb-up" : "text-bb-down"}`}>
                      {Math.round(h.pnl_krw).toLocaleString()}
                    </td>
                    <td className={`text-right px-3 py-2 ${h.pnl_pct >= 0 ? "text-bb-up" : "text-bb-down"}`}>
                      {h.current_price > 0 ? `${h.pnl_pct.toFixed(2)}%` : "—"}
                    </td>
                    <td className="text-right px-3 py-2">{h.weight_pct.toFixed(1)}%</td>
                    <td className="px-3 py-2">
                      <Sparkline points={sparks[h.instrument_id] ?? []} width={70} height={20} />
                    </td>
                    <td className="px-3 py-2 text-right">
                      <button
                        onClick={(e) => { e.stopPropagation(); setEditTarget(h); setEditOpen(true); }}
                        className="text-xs text-fg-muted hover:text-fg mr-2"
                      >수정</button>
                      <button
                        onClick={(e) => { e.stopPropagation(); setDeleteTarget(h); setDeleteOpen(true); }}
                        className="text-xs text-fg-muted hover:text-bb-down"
                      >삭제</button>
                    </td>
                  </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
      <AddHoldingDialog open={addOpen} onOpenChange={setAddOpen} onAdded={() => void load()} />
      <EditHoldingDialog holding={editTarget} open={editOpen} onOpenChange={setEditOpen} onSaved={() => void load()} />
      <DeleteConfirmDialog holding={deleteTarget} open={deleteOpen} onOpenChange={setDeleteOpen} onDeleted={() => void load()} />
      <HoldingDetailPanel
        holding={detailTarget}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        onEdit={(h) => { setEditTarget(h); setEditOpen(true); }}
        onDelete={(h) => { setDeleteTarget(h); setDeleteOpen(true); }}
      />
    </>
  );
}

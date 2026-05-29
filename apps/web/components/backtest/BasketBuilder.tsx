"use client";
import { InstrumentSearchInput } from "@/components/portfolio/InstrumentSearchInput";
import type { InstrumentResult } from "@/lib/api/instruments";
import { X } from "lucide-react";

export type BasketRow = { inst: InstrumentResult; weight: number };

export const MAX_LEGS = 10;

export function BasketBuilder({
  rows,
  onChange,
}: {
  rows: BasketRow[];
  onChange: (rows: BasketRow[]) => void;
}) {
  const sum = rows.reduce((a, r) => a + (r.weight || 0), 0);

  function add(inst: InstrumentResult) {
    if (rows.length >= MAX_LEGS) return;
    if (rows.some((r) => r.inst.id === inst.id)) return; // 중복 차단
    onChange([...rows, { inst, weight: 0 }]);
  }
  function setWeight(id: string, w: number) {
    onChange(rows.map((r) => (r.inst.id === id ? { ...r, weight: w } : r)));
  }
  function remove(id: string) {
    onChange(rows.filter((r) => r.inst.id !== id));
  }

  return (
    <div>
      <div className="text-xs text-fg-muted mb-1">
        바스켓 (목표 비중 · 실행 시 자동 정규화)
      </div>
      <div className="border border-line divide-y divide-line/50">
        {rows.length === 0 && (
          <div className="px-3 py-2 text-xs text-fg-muted font-mono">
            종목을 추가하세요 (최대 {MAX_LEGS}개)
          </div>
        )}
        {rows.map((r) => (
          <div key={r.inst.id} className="flex items-center gap-2 px-3 py-2">
            <div className="flex-1 font-mono text-sm">
              {r.inst.symbol}
              <span className="text-fg-muted text-xs ml-2">{r.inst.name}</span>
            </div>
            <input
              type="number"
              min={0}
              value={r.weight}
              onChange={(e) => setWeight(r.inst.id, Number(e.target.value) || 0)}
              className="w-16 bg-bg-deep border border-line px-2 py-1 text-sm font-mono text-right tabular-nums"
              aria-label={`${r.inst.symbol} 비중`}
            />
            <span className="text-xs text-fg-muted">%</span>
            <button
              type="button"
              onClick={() => remove(r.inst.id)}
              className="text-fg-muted hover:text-bb-down"
              aria-label={`${r.inst.symbol} 삭제`}
            >
              <X size={14} />
            </button>
          </div>
        ))}
      </div>
      {rows.length < MAX_LEGS && (
        <div className="mt-2">
          <InstrumentSearchInput onSelect={add} placeholder="＋ 종목 추가 (검색)" />
        </div>
      )}
      <div className="text-xs text-fg-muted mt-1 text-right tabular-nums">
        합계 {Math.round(sum * 100) / 100}%
      </div>
    </div>
  );
}

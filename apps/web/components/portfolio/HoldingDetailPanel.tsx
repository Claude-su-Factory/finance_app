"use client";

import { useEffect, useState } from "react";
import type { Holding } from "@/lib/api/holdings";
import { fetchPriceHistory } from "@/lib/api/history";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { Button } from "@/components/ui/button";

export function HoldingDetailPanel({
  holding,
  open,
  onClose,
  onEdit,
  onDelete,
}: {
  holding: Holding | null;
  open: boolean;
  onClose: () => void;
  onEdit: (h: Holding) => void;
  onDelete: (h: Holding) => void;
}) {
  const [points, setPoints] = useState<ChartPoint[]>([]);

  // 패널 열릴 때마다 30일 가격 fetch
  useEffect(() => {
    if (!open || !holding) {
      setPoints([]);
      return;
    }
    let cancelled = false;
    fetchPriceHistory(holding.symbol, "1mo")
      .then((data) => {
        if (cancelled) return;
        setPoints(data.map((p) => ({ x: p.date, value: p.close })));
      })
      .catch(() => {
        if (!cancelled) setPoints([]);
      });
    return () => { cancelled = true; };
  }, [open, holding]);

  // ESC 닫기
  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!holding) return null;

  const positive = holding.pnl_krw >= 0;
  const positivePct = holding.pnl_pct >= 0;

  return (
    <>
      {/* 반투명 backdrop — 클릭 시 닫기. 모바일·태블릿에서만 표시 (lg 이상은 background 클릭 비활성) */}
      <div
        onClick={onClose}
        className={`fixed inset-0 z-40 bg-black/40 transition-opacity lg:hidden ${
          open ? "opacity-100 pointer-events-auto" : "opacity-0 pointer-events-none"
        }`}
      />

      {/* 패널 */}
      <aside
        className={`fixed top-9 bottom-6 right-0 z-50 w-[420px] max-w-[90vw] border-l border-line bg-bg shadow-2xl overflow-y-auto transition-transform duration-200 ${
          open ? "translate-x-0" : "translate-x-full"
        }`}
        aria-hidden={!open}
      >
        {/* 헤더 */}
        <div className="sticky top-0 bg-bg border-b border-line px-4 py-3 flex items-baseline justify-between z-10">
          <div className="min-w-0">
            <div className="font-mono text-base tabular-nums truncate">{holding.symbol}</div>
            <div className="text-xs text-fg-muted truncate">
              {holding.name} · {holding.exchange} · {holding.currency}
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-fg-muted hover:text-fg text-xl font-mono leading-none ml-2"
            title="닫기 (ESC)"
            aria-label="닫기"
          >
            ×
          </button>
        </div>

        <div className="p-4 space-y-4">
          {/* 현재가 + 손익 */}
          <div>
            <div className="text-xs text-fg-muted font-mono mb-1">현재가</div>
            <div className="font-mono text-2xl tabular-nums">
              {holding.current_price > 0
                ? `${holding.current_price.toLocaleString()} ${holding.currency}`
                : "—"}
            </div>
            {holding.current_price > 0 && (
              <div className={`text-sm font-mono tabular-nums mt-1 ${positivePct ? "text-bb-up" : "text-bb-down"}`}>
                {positivePct ? "+" : ""}{holding.pnl_pct.toFixed(2)}% (vs 평단가)
              </div>
            )}
          </div>

          {/* 30일 차트 */}
          <LineChartCard
            title="30일 가격 추이"
            subtitle={`${holding.symbol} · 1mo`}
            points={points}
            height={140}
          />

          {/* 보유 상세 */}
          <div className="border border-line p-4">
            <div className="text-xs text-fg-muted font-mono mb-3">보유 상세</div>
            <dl className="space-y-2 font-mono text-sm">
              <Row label="수량" value={holding.quantity.toLocaleString()} />
              <Row label={`평단가 (${holding.currency})`} value={holding.avg_cost.toLocaleString()} />
              <Row label="평가액 (원본 통화)" value={
                holding.current_price > 0 ? holding.market_value.toLocaleString() : "—"
              } />
              <Row label="평가액 (KRW)" value={`₩${Math.round(holding.market_value_krw).toLocaleString()}`} />
              <Row label="투자원금 (KRW)" value={`₩${Math.round(holding.cost_basis_krw).toLocaleString()}`} />
              <Row
                label="손익 (KRW)"
                value={`${positive ? "+" : ""}${Math.round(holding.pnl_krw).toLocaleString()}`}
                valueClass={positive ? "text-bb-up" : "text-bb-down"}
              />
              <Row label="비중" value={`${holding.weight_pct.toFixed(1)}%`} />
              <Row label="자산군" value={holding.asset_class} />
              {holding.opened_at && (
                <Row label="매수일" value={String(holding.opened_at).slice(0, 10)} />
              )}
            </dl>
          </div>

          {/* 메모 */}
          {holding.note && (
            <div className="border border-line p-4">
              <div className="text-xs text-fg-muted font-mono mb-2">메모</div>
              <p className="font-mono text-sm whitespace-pre-wrap">{holding.note}</p>
            </div>
          )}

          {/* 액션 */}
          <div className="flex gap-2 pt-2">
            <Button variant="ghost" onClick={() => onEdit(holding)} className="flex-1">
              수정
            </Button>
            <Button
              onClick={() => onDelete(holding)}
              className="flex-1 bg-bb-down text-bg hover:bg-bb-down/80"
            >
              삭제
            </Button>
          </div>
        </div>
      </aside>
    </>
  );
}

function Row({
  label,
  value,
  valueClass,
}: {
  label: string;
  value: string;
  valueClass?: string;
}) {
  return (
    <div className="flex justify-between items-baseline gap-3">
      <dt className="text-fg-muted text-xs">{label}</dt>
      <dd className={`tabular-nums ${valueClass ?? ""}`}>{value}</dd>
    </div>
  );
}

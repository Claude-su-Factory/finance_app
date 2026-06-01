"use client";
import { use } from "react";
import { TradeDialog } from "@/components/paper/TradeDialog";
import { ResetDialog } from "@/components/paper/ResetDialog";
import { NewEntryDialog } from "@/components/journal/NewEntryDialog";
import { AddHoldingDialog } from "@/components/portfolio/AddHoldingDialog";
import { EditHoldingDialog } from "@/components/portfolio/EditHoldingDialog";
import { DeleteConfirmDialog } from "@/components/portfolio/DeleteConfirmDialog";
import type { PaperHolding } from "@/lib/api/paper";
import type { Holding } from "@/lib/api/holdings";

const noop = () => {};

// TradeDialog.holdings 는 PaperHolding[] (lib/api/paper.ts). /v1/paper/portfolio 픽스처와 동일 형태.
const samplePaperHoldings: PaperHolding[] = [
  {
    id: "ph1", user_id: "preview-user", instrument_id: "i-samsung", symbol: "005930",
    name: "삼성전자", currency: "KRW", quantity: 40, avg_cost: 71000, current_price: 79800,
    market_value: 3192000, market_value_krw: 3192000, pnl_krw: 352000, pnl_pct: 12.39,
    created_at: "2026-03-02T00:00:00Z", updated_at: "2026-05-31T00:00:00Z",
  },
];

// EditHoldingDialog/DeleteConfirmDialog.holding 은 Holding (lib/api/holdings.ts) — 다른 타입.
// Holding 전체 필수 필드를 채운다(weight_pct/cost_basis_krw/exchange 포함). holding 누락 시 빈 화면.
const sampleHolding: Holding = {
  id: "h1", instrument_id: "i-samsung", quantity: 50, avg_cost: 68000,
  opened_at: "2026-03-02", note: null,
  created_at: "2026-03-02T00:00:00Z", updated_at: "2026-05-31T00:00:00Z",
  symbol: "005930", exchange: "KRX", name: "삼성전자", asset_class: "KR_STOCK",
  currency: "KRW", current_price: 79800, market_value: 3990000, market_value_krw: 3990000,
  cost_basis_krw: 3400000, pnl_krw: 590000, pnl_pct: 17.35, weight_pct: 53.1,
};

// 각 모달을 open=true 로 렌더. 콜백명은 실측표대로(delete=onDeleted, NOT onConfirm).
// ResetDialog는 currentInitial(필수)이 계획표에 누락돼 있어 10000000 더미를 전달.
const REGISTRY: Record<string, React.ReactNode> = {
  trade: <TradeDialog open onOpenChange={noop} onTraded={noop} cashBalance={6029504} holdings={samplePaperHoldings} />,
  reset: <ResetDialog open onOpenChange={noop} onReset={noop} currentInitial={10000000} />,
  "journal-new": <NewEntryDialog open onOpenChange={noop} onCreated={noop} />,
  "holding-add": <AddHoldingDialog open onOpenChange={noop} onAdded={noop} />,
  "holding-edit": <EditHoldingDialog open onOpenChange={noop} onSaved={noop} holding={sampleHolding} />,
  "holding-delete": <DeleteConfirmDialog open onOpenChange={noop} onDeleted={noop} holding={sampleHolding} />,
};

export default function PreviewModal({ params }: { params: Promise<{ name: string }> }) {
  const { name } = use(params);
  return (
    <div className="min-h-[60vh] flex items-center justify-center p-8">
      {REGISTRY[name] ?? <div className="text-fg-muted">unknown modal: {name}</div>}
    </div>
  );
}

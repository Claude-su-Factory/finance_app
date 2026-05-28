"use client";

import { useState, useMemo } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { InstrumentSearchInput } from "@/components/portfolio/InstrumentSearchInput";
import type { InstrumentResult } from "@/lib/api/instruments";
import { trade, isTradeError, type PaperHolding } from "@/lib/api/paper";

export function TradeDialog({
  open,
  onOpenChange,
  onTraded,
  cashBalance,
  holdings,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onTraded: () => void;
  cashBalance: number;
  holdings: PaperHolding[];
}) {
  const [action, setAction] = useState<"buy" | "sell">("buy");
  const [selectedInst, setSelectedInst] = useState<InstrumentResult | null>(null);
  const [selectedHoldingId, setSelectedHoldingId] = useState<string>("");
  const [quantity, setQuantity] = useState("");
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const selectedHolding = useMemo(
    () => holdings.find((h) => h.id === selectedHoldingId) ?? null,
    [holdings, selectedHoldingId],
  );

  function reset() {
    setSelectedInst(null);
    setSelectedHoldingId("");
    setQuantity("");
    setReason("");
    setErr(null);
  }

  async function submit() {
    setErr(null);
    const q = parseFloat(quantity);
    if (!(q > 0)) {
      setErr("수량은 0보다 커야 합니다");
      return;
    }
    let instrumentId: string;
    if (action === "buy") {
      if (!selectedInst) {
        setErr("종목을 선택해주세요");
        return;
      }
      instrumentId = selectedInst.id;
    } else {
      if (!selectedHolding) {
        setErr("매도할 종목을 선택해주세요");
        return;
      }
      instrumentId = selectedHolding.instrument_id;
    }

    setSubmitting(true);
    try {
      const result = await trade({
        instrument_id: instrumentId,
        action,
        quantity: q,
        reason: reason || undefined,
      });
      if (isTradeError(result)) {
        if (result.error.code === "INSUFFICIENT_CASH") {
          setErr(
            `잔액 부족 (필요: ${result.error.need_krw?.toLocaleString()} KRW, 보유: ${result.error.have_krw?.toLocaleString()} KRW)`,
          );
        } else if (result.error.code === "INSUFFICIENT_HOLDING") {
          setErr(`보유 수량 부족 (필요: ${result.error.need_qty}, 보유: ${result.error.have_qty})`);
        } else {
          setErr(result.error.message);
        }
        return;
      }
      onTraded();
      reset();
      onOpenChange(false);
    } catch (e: unknown) {
      setErr((e as Error).message ?? "체결 실패");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) reset();
        onOpenChange(v);
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">가상 매매</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div className="flex gap-2">
            <button
              onClick={() => {
                setAction("buy");
                reset();
              }}
              className={`flex-1 font-mono text-xs px-3 py-2 border ${action === "buy" ? "border-bb-up text-bb-up" : "border-line text-fg-muted"}`}
            >
              매수
            </button>
            <button
              onClick={() => {
                setAction("sell");
                reset();
              }}
              className={`flex-1 font-mono text-xs px-3 py-2 border ${action === "sell" ? "border-bb-down text-bb-down" : "border-line text-fg-muted"}`}
            >
              매도
            </button>
          </div>

          {action === "buy" ? (
            <div>
              <Label className="text-xs font-mono">종목</Label>
              <InstrumentSearchInput onSelect={setSelectedInst} />
            </div>
          ) : (
            <div>
              <Label className="text-xs font-mono">보유 종목</Label>
              <select
                value={selectedHoldingId}
                onChange={(e) => setSelectedHoldingId(e.target.value)}
                className="w-full bg-bg-card border border-line px-3 py-1.5 text-sm font-mono"
              >
                <option value="">선택…</option>
                {holdings.map((h) => (
                  <option key={h.id} value={h.id}>
                    {h.symbol} — {h.name} ({h.quantity})
                  </option>
                ))}
              </select>
            </div>
          )}

          <div>
            <Label className="text-xs font-mono">수량</Label>
            <Input
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              type="number"
              step="any"
            />
          </div>

          <div className="font-mono text-[10px] text-fg-muted">
            현재 잔여 현금: {Math.round(cashBalance).toLocaleString("ko-KR")} KRW
          </div>

          <div>
            <Label className="text-xs font-mono">💭 매매 이유 (선택, 200자)</Label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              maxLength={200}
              rows={2}
              className="w-full bg-bg-card border border-line p-2 text-sm font-mono"
            />
            <p className="text-[10px] text-fg-muted font-mono mt-1">* 작성 시 매매 일기 자동 기록</p>
          </div>

          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            취소
          </Button>
          <Button onClick={() => void submit()} disabled={submitting}>
            {submitting ? "체결 중…" : action === "buy" ? "매수 확정" : "매도 확정"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

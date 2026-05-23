"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { updateHolding, type Holding } from "@/lib/api/holdings";

export function EditHoldingDialog({
  holding,
  open,
  onOpenChange,
  onSaved,
}: {
  holding: Holding | null;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSaved: () => void;
}) {
  const [quantity, setQuantity] = useState("");
  const [avgCost, setAvgCost] = useState("");
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (holding) {
      setQuantity(String(holding.quantity));
      setAvgCost(String(holding.avg_cost));
      setNote(holding.note ?? "");
      setErr(null);
    }
  }, [holding]);

  if (!holding) return null;

  async function submit() {
    const h = holding;
    if (!h) return;
    const q = parseFloat(quantity);
    const c = parseFloat(avgCost);
    if (!(q > 0)) { setErr("수량은 0보다 커야 합니다"); return; }
    if (!(c >= 0)) { setErr("평단가는 0 이상이어야 합니다"); return; }
    setSubmitting(true);
    try {
      await updateHolding(h.id, { quantity: q, avg_cost: c, note: note || null });
      onSaved();
      onOpenChange(false);
    } catch (e: unknown) {
      setErr((e as { message?: string })?.message ?? "수정 실패");
    } finally {
      setSubmitting(false);
    }
  }

  // base-ui onOpenChange: (open: boolean, eventDetails) => void — 래퍼로 시그니처 맞춤
  function handleOpenChange(v: boolean) {
    if (!v) setErr(null);
    onOpenChange(v);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">{holding.symbol} 수정</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label className="text-xs font-mono">수량</Label>
              <Input
                value={quantity}
                onChange={(e) => setQuantity(e.target.value)}
                type="number"
                step="any"
              />
            </div>
            <div>
              <Label className="text-xs font-mono">평단가 ({holding.currency})</Label>
              <Input
                value={avgCost}
                onChange={(e) => setAvgCost(e.target.value)}
                type="number"
                step="any"
              />
            </div>
          </div>
          <div>
            <Label className="text-xs font-mono">메모</Label>
            <Input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              maxLength={200}
            />
          </div>
          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>취소</Button>
          <Button onClick={() => void submit()} disabled={submitting}>
            {submitting ? "저장 중…" : "저장"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

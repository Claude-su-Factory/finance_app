"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { InstrumentSearchInput } from "./InstrumentSearchInput";
import { createHolding } from "@/lib/api/holdings";
import type { InstrumentResult } from "@/lib/api/instruments";

export function AddHoldingDialog({
  open,
  onOpenChange,
  onAdded,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onAdded: () => void;
}) {
  const [inst, setInst] = useState<InstrumentResult | null>(null);
  const [quantity, setQuantity] = useState("");
  const [avgCost, setAvgCost] = useState("");
  const [openedAt, setOpenedAt] = useState("");
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function reset() {
    setInst(null);
    setQuantity("");
    setAvgCost("");
    setOpenedAt("");
    setNote("");
    setErr(null);
  }

  async function submit() {
    if (!inst) { setErr("종목을 선택해주세요"); return; }
    const q = parseFloat(quantity);
    const c = parseFloat(avgCost);
    if (!(q > 0)) { setErr("수량은 0보다 커야 합니다"); return; }
    if (!(c >= 0)) { setErr("평단가는 0 이상이어야 합니다"); return; }
    setSubmitting(true);
    setErr(null);
    try {
      await createHolding({
        instrument_id: inst.id,
        quantity: q,
        avg_cost: c,
        opened_at: openedAt || undefined,
        note: note || undefined,
      });
      onAdded();
      reset();
      onOpenChange(false);
    } catch (e: unknown) {
      const code = (e as { code?: string })?.code;
      const msg = (e as { message?: string })?.message;
      if (code === "CONFLICT") {
        setErr("이미 등록된 종목입니다. 수정으로 진행해주세요.");
      } else {
        setErr(msg ?? "추가 실패");
      }
    } finally {
      setSubmitting(false);
    }
  }

  // base-ui onOpenChange: (open: boolean, eventDetails: ...) => void
  function handleOpenChange(v: boolean) {
    if (!v) reset();
    onOpenChange(v);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">보유 자산 추가</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="border-l-2 border-bb-warn/60 bg-bb-warn/5 px-3 py-2 text-xs text-fg-muted leading-relaxed">
            <span className="font-mono text-bb-warn">i</span>{" "}
            입력 데이터는 본인이 직접 기록하며 실제 보유 여부는 검증하지 않습니다.
            분석 정확성은 입력 정확성에 따릅니다.
          </div>
          <div>
            <Label className="text-xs font-mono">종목</Label>
            <InstrumentSearchInput onSelect={setInst} />
          </div>
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
              <Label className="text-xs font-mono">
                평단가 ({inst?.currency ?? "통화"})
              </Label>
              <Input
                value={avgCost}
                onChange={(e) => setAvgCost(e.target.value)}
                type="number"
                step="any"
              />
            </div>
          </div>
          <div>
            <Label className="text-xs font-mono">매수일 (선택)</Label>
            <Input
              value={openedAt}
              onChange={(e) => setOpenedAt(e.target.value)}
              type="date"
            />
          </div>
          <div>
            <Label className="text-xs font-mono">메모 (선택)</Label>
            <Input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              maxLength={200}
            />
          </div>
          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            취소
          </Button>
          <Button onClick={() => void submit()} disabled={submitting}>
            {submitting ? "추가 중…" : "추가"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

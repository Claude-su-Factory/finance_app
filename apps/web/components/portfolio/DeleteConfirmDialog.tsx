"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { deleteHolding, type Holding } from "@/lib/api/holdings";

export function DeleteConfirmDialog({
  holding,
  open,
  onOpenChange,
  onDeleted,
}: {
  holding: Holding | null;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onDeleted: () => void;
}) {
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  if (!holding) return null;

  async function submit() {
    const h = holding;
    if (!h) return;
    setSubmitting(true);
    try {
      await deleteHolding(h.id);
      onDeleted();
      onOpenChange(false);
    } catch (e: unknown) {
      setErr((e as { message?: string })?.message ?? "삭제 실패");
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
          <DialogTitle className="font-mono">{holding.symbol} 삭제</DialogTitle>
        </DialogHeader>
        <p className="text-sm font-mono">이 보유 자산을 삭제합니다. 되돌릴 수 없습니다.</p>
        {err && <p className="text-bb-down text-xs font-mono mt-2">{err}</p>}
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>취소</Button>
          <Button
            onClick={() => void submit()}
            disabled={submitting}
            className="bg-bb-down text-bg hover:bg-bb-down/80"
          >
            {submitting ? "삭제 중…" : "삭제"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

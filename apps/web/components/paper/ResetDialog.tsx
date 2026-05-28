"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { resetPortfolio } from "@/lib/api/paper";

export function ResetDialog({
  open,
  onOpenChange,
  onReset,
  currentInitial,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onReset: () => void;
  currentInitial: number;
}) {
  const [initial, setInitial] = useState(String(currentInitial));
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    try {
      const n = parseFloat(initial);
      await resetPortfolio(isNaN(n) ? undefined : n);
      onReset();
      onOpenChange(false);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">⚠ Paper Portfolio 리셋</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <p className="text-sm text-fg-muted">
            전체 보유 자산을 삭제하고 현금을 초기화합니다.
            <br />
            매매 이력은 보존됩니다.
          </p>
          <div>
            <Label className="text-xs font-mono">새 초기 자금 (KRW)</Label>
            <Input
              value={initial}
              onChange={(e) => setInitial(e.target.value)}
              type="number"
              placeholder="10000000"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            취소
          </Button>
          <Button onClick={() => void submit()} disabled={submitting}>
            {submitting ? "리셋 중…" : "리셋 확정"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

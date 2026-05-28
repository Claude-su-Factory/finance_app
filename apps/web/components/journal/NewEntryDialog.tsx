"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { createEntry, type JournalAction } from "@/lib/api/journal";

export function NewEntryDialog({
  open, onOpenChange, onCreated,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onCreated: () => void;
}) {
  const [action, setAction] = useState<JournalAction>("observation");
  const [title, setTitle] = useState("");
  const [symbols, setSymbols] = useState("");
  const [content, setContent] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit() {
    if (content.length < 1) { setErr("내용을 입력해주세요"); return; }
    if (content.length > 2000) { setErr("2000자 이내"); return; }
    setSubmitting(true);
    setErr(null);
    try {
      const symbolList = symbols.split(",").map((s) => s.trim()).filter(Boolean).slice(0, 10);
      await createEntry({
        action,
        related_symbols: symbolList,
        title: title || undefined,
        content,
      });
      setAction("observation"); setTitle(""); setSymbols(""); setContent("");
      onCreated();
      onOpenChange(false);
    } catch (e: unknown) {
      setErr((e as Error).message ?? "생성 실패");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">📓 새 일기 entry</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div>
            <Label className="text-xs font-mono">제목 (선택, 100자)</Label>
            <Input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={100} />
          </div>
          <div>
            <Label className="text-xs font-mono">종류</Label>
            <select
              value={action}
              onChange={(e) => setAction(e.target.value as JournalAction)}
              className="w-full bg-bg-card border border-line px-3 py-1.5 text-sm font-mono"
            >
              <option value="observation">관찰</option>
              <option value="buy">매수</option>
              <option value="sell">매도</option>
              <option value="other">기타</option>
            </select>
          </div>
          <div>
            <Label className="text-xs font-mono">관련 종목 (콤마 구분, 최대 10개)</Label>
            <Input value={symbols} onChange={(e) => setSymbols(e.target.value)} placeholder="005930, NVDA" />
          </div>
          <div>
            <Label className="text-xs font-mono">내용 (1~2000자)</Label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              maxLength={2000}
              rows={6}
              className="w-full bg-bg-card border border-line p-2 text-sm font-mono"
            />
            <div className="text-right text-[10px] text-fg-muted font-mono">{content.length}/2000</div>
          </div>
          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>취소</Button>
          <Button onClick={submit} disabled={submitting}>
            {submitting ? "저장 중…" : "저장"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

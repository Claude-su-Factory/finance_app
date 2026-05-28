"use client";

import { useState } from "react";
import type { JournalEntry } from "@/lib/api/journal";
import { deleteEntry, patchEntry } from "@/lib/api/journal";

export function EntryItem({ entry, onChanged }: { entry: JournalEntry; onChanged: () => void }) {
  const [editing, setEditing] = useState(false);
  const [content, setContent] = useState(entry.content);
  const border = entry.entry_type === "auto" ? "border-bb-accent" : "border-bb-info";
  const actionLabel = labelFor(entry.action);
  const date = entry.created_at.slice(0, 10);

  async function save() {
    await patchEntry(entry.id, { content });
    setEditing(false);
    onChanged();
  }

  async function remove() {
    if (!confirm("삭제하시겠습니까?")) return;
    await deleteEntry(entry.id);
    onChanged();
  }

  return (
    <div className={`border-l-2 ${border} pl-3 py-2`}>
      <div className="font-mono text-[10px] text-fg-muted">
        {date} · <span className={signColor(entry.action)}>{actionLabel}</span>
        {entry.related_holding && <> · {entry.related_holding.symbol} {entry.related_holding.name}</>}
        {entry.related_symbols.length > 0 && <> · {entry.related_symbols.join(", ")}</>}
        · {entry.entry_type === "auto" ? "자동" : "수동"}
      </div>
      {entry.title && <div className="text-sm font-medium mt-1">{entry.title}</div>}
      {editing ? (
        <div className="mt-1">
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            maxLength={2000}
            className="w-full bg-bg-card border border-line p-2 text-sm font-mono"
            rows={4}
          />
          <div className="flex gap-2 mt-1">
            <button onClick={save} className="text-xs font-mono text-bb-accent">저장</button>
            <button onClick={() => setEditing(false)} className="text-xs font-mono text-fg-muted">취소</button>
          </div>
        </div>
      ) : (
        <div className="text-sm mt-1 whitespace-pre-wrap">{entry.content}</div>
      )}
      {entry.entry_type === "manual" && !editing && (
        <div className="flex gap-2 mt-2 text-xs font-mono">
          <button onClick={() => setEditing(true)} className="text-fg-muted hover:text-fg">수정</button>
          <button onClick={remove} className="text-bb-down/70 hover:text-bb-down">삭제</button>
        </div>
      )}
    </div>
  );
}

function labelFor(a?: string) {
  switch (a) {
    case "buy": return "매수";
    case "sell": return "매도";
    case "observation": return "관찰";
    case "other": return "기타";
    default: return "";
  }
}

function signColor(a?: string) {
  if (a === "buy") return "text-bb-up";
  if (a === "sell") return "text-bb-down";
  return "text-fg-muted";
}

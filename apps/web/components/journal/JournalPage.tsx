"use client";

import { useEffect, useState } from "react";
import {
  listEntries, listAnalyses, analyzeNow, isAnalyzeError,
  type JournalEntry, type AnalysisRun, type AnalyzeResult,
} from "@/lib/api/journal";
import { EntryItem } from "./EntryItem";
import { AnalysisCard } from "./AnalysisCard";
import { NewEntryDialog } from "./NewEntryDialog";

export function JournalPage() {
  const [entries, setEntries] = useState<JournalEntry[] | null>(null);
  const [analyses, setAnalyses] = useState<AnalysisRun[]>([]);
  const [newOpen, setNewOpen] = useState(false);
  const [analyzing, setAnalyzing] = useState(false);
  const [analyzeMsg, setAnalyzeMsg] = useState<string | null>(null);

  async function refresh() {
    const [e, a] = await Promise.all([listEntries(50), listAnalyses(20)]);
    setEntries(e.entries);
    setAnalyses(a.analyses);
  }

  useEffect(() => {
    refresh().catch(() => {
      setEntries([]);
      setAnalyses([]);
    });
  }, []);

  async function onAnalyze() {
    setAnalyzing(true);
    setAnalyzeMsg(null);
    const result: AnalyzeResult = await analyzeNow(90);
    setAnalyzing(false);
    if (isAnalyzeError(result)) {
      setAnalyzeMsg(result.error.message);
      return;
    }
    setAnalyses([result, ...analyses]);
  }

  return (
    <div className="p-6 md:p-8 max-w-3xl mx-auto space-y-6">
      <header className="flex items-baseline justify-between">
        <div>
          <h1 className="font-mono text-2xl">📓 매매 일기</h1>
          <p className="text-fg-muted text-sm mt-1">매매 결정과 시장 관찰을 기록합니다. 월 1회 자동 회고 + 필요 시 분석 요청.</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setNewOpen(true)}
            className="font-mono text-xs px-3 py-1.5 border border-line hover:border-fg-muted"
          >
            + 새 entry
          </button>
          <button
            onClick={onAnalyze}
            disabled={analyzing}
            className="font-mono text-xs px-3 py-1.5 border border-bb-accent text-bb-accent hover:bg-bb-accent/10 disabled:opacity-50"
          >
            {analyzing ? "분석 중…" : "⚡ 분석 요청"}
          </button>
        </div>
      </header>

      {analyzeMsg && (
        <div className="border-l-2 border-bb-down bg-bg-card p-3 text-sm font-mono">
          {analyzeMsg}
        </div>
      )}

      {analyses.length > 0 && (
        <section>
          <h2 className="font-mono text-xs text-fg-muted tracking-widest mb-2">AI 분석</h2>
          {analyses.map((a) => <AnalysisCard key={a.id} run={a} />)}
        </section>
      )}

      <section>
        <h2 className="font-mono text-xs text-fg-muted tracking-widest mb-2">일기 entries</h2>
        {entries === null ? (
          <div className="text-fg-muted text-sm font-mono">로딩…</div>
        ) : entries.length === 0 ? (
          <div className="text-fg-muted text-sm">아직 일기가 없습니다. 새 entry 작성으로 시작하세요.</div>
        ) : (
          <div className="space-y-3">
            {entries.map((e) => <EntryItem key={e.id} entry={e} onChanged={refresh} />)}
          </div>
        )}
      </section>

      <NewEntryDialog open={newOpen} onOpenChange={setNewOpen} onCreated={refresh} />
    </div>
  );
}

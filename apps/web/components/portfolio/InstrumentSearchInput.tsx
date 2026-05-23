"use client";

import { useEffect, useState } from "react";
import { searchInstruments, selectInstrument, type InstrumentResult } from "@/lib/api/instruments";

export function InstrumentSearchInput({
  onSelect,
  placeholder = "종목 검색 (예: 삼성전자, AAPL)",
}: {
  onSelect: (inst: InstrumentResult) => void;
  placeholder?: string;
}) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<InstrumentResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!query.trim()) {
      setResults([]);
      return;
    }
    const t = setTimeout(async () => {
      setLoading(true);
      try {
        const r = await searchInstruments(query);
        setResults(r);
        setOpen(true);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 300);
    return () => clearTimeout(t);
  }, [query]);

  function handlePick(inst: InstrumentResult) {
    // holdings 대상이 아닌 자산군은 선택 차단 (백엔드 422와 정합)
    if (inst.asset_class === "INDEX" || inst.asset_class === "FX" || inst.asset_class === "CASH") {
      return;
    }
    void selectInstrument(query, inst.id); // 학습은 fire-and-forget
    setOpen(false);
    setQuery(`${inst.symbol} — ${inst.name}`);
    onSelect(inst);
  }

  return (
    <div className="relative">
      <input
        value={query}
        onChange={(e) => { setQuery(e.target.value); setOpen(true); }}
        placeholder={placeholder}
        className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full"
        onFocus={() => { if (results.length) setOpen(true); }}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
      />
      {open && (results.length > 0 || loading) && (
        <div className="absolute z-10 mt-1 w-full border border-line bg-bg-deep max-h-64 overflow-auto">
          {loading && <div className="px-3 py-2 text-xs text-fg-muted">검색 중…</div>}
          {results.map((r) => {
            const disabled = r.asset_class === "INDEX" || r.asset_class === "FX" || r.asset_class === "CASH";
            return (
              <button
                key={r.id}
                type="button"
                onMouseDown={(e) => e.preventDefault()} // onBlur보다 앞서 실행되도록
                onClick={() => handlePick(r)}
                disabled={disabled}
                className={`w-full text-left px-3 py-2 font-mono text-sm border-b border-line/50 last:border-b-0 ${disabled ? "opacity-40 cursor-not-allowed" : "hover:bg-line/30"}`}
                title={disabled ? `${r.asset_class} 자산은 보유에 추가할 수 없습니다` : undefined}
              >
                <span>{r.symbol}</span>
                <span className="text-fg-muted text-xs ml-2">{r.exchange}</span>
                <div className="text-xs text-fg-muted">{r.name}</div>
              </button>
            );
          })}
          {!loading && results.length === 0 && query && (
            <div className="px-3 py-2 text-xs text-fg-muted">결과 없음</div>
          )}
        </div>
      )}
    </div>
  );
}

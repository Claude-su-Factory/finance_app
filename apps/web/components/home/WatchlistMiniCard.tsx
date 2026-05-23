"use client";
import { useEffect, useState } from "react";
import { listWatchlist, type WatchlistItem } from "@/lib/api/watchlist";

export function WatchlistMiniCard() {
  const [items, setItems] = useState<WatchlistItem[] | null>(null);
  useEffect(() => {
    listWatchlist()
      .then(setItems)
      .catch(() => setItems([]));
  }, []);

  if (items === null) {
    return (
      <div className="border border-line p-4 font-mono text-xs text-fg-muted">
        로드 중…
      </div>
    );
  }

  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">관심 종목</div>
      {items.length === 0 ? (
        <div className="text-fg-muted text-sm font-mono">
          아직 없음. 마켓 탭 (W5)에서 추가 예정
        </div>
      ) : (
        <ul className="space-y-1 font-mono text-sm">
          {items.slice(0, 5).map((w) => (
            <li key={w.instrument_id} className="flex items-baseline gap-2">
              <span className="flex-1 truncate">{w.symbol}</span>
              <span className="tabular-nums">
                {w.price > 0 ? w.price.toLocaleString() : "—"}
              </span>
              <span
                className={`tabular-nums text-xs ${
                  w.change_pct >= 0 ? "text-bb-up" : "text-bb-down"
                }`}
              >
                {w.change_pct >= 0 ? "+" : ""}
                {w.change_pct.toFixed(2)}%
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

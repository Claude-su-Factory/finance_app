"use client";
import { useEffect, useState } from "react";
import { fetchTicker, type Ticker } from "@/lib/api/market";

export function MarketWidgetsCard() {
  const [items, setItems] = useState<Ticker[]>([]);
  useEffect(() => {
    fetchTicker().then(setItems).catch(() => {});
  }, []);
  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">마켓</div>
      <ul className="space-y-1 font-mono text-sm">
        {items.map((t) => (
          <li key={t.symbol} className="flex items-baseline gap-2">
            <span className="flex-1">{t.name}</span>
            <span className="tabular-nums">
              {t.price > 0 ? t.price.toLocaleString() : "—"}
            </span>
            <span
              className={`tabular-nums text-xs ${
                t.change_pct >= 0 ? "text-bb-up" : "text-bb-down"
              }`}
            >
              {t.change_pct >= 0 ? "+" : ""}
              {t.change_pct.toFixed(2)}%
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

"use client";

import { useEffect, useState } from "react";
import { fetchTicker, type Ticker } from "@/lib/api/market";

const SEED_DISPLAY = [
  { symbol: "KOSPI", label: "KOSPI" },
  { symbol: "SPX", label: "S&P 500" },
  { symbol: "USD_KRW", label: "USD/KRW" },
];

function formatPrice(symbol: string, price: number): string {
  if (price === 0) return "—";
  if (symbol === "USD_KRW") return price.toFixed(2);
  return price.toLocaleString("ko-KR", {
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
  });
}

function changeClass(pct: number): string {
  if (pct > 0) return "text-bb-up";
  if (pct < 0) return "text-bb-down";
  return "text-fg";
}

export function TopTicker() {
  const [items, setItems] = useState<Ticker[]>([]);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setInterval> | undefined;

    async function load() {
      // 탭이 백그라운드면 호출 skip (배터리·API quota 절약)
      if (typeof document !== "undefined" && document.visibilityState === "hidden") return;
      const data = await fetchTicker();
      if (!cancelled) setItems(data);
    }

    void load();
    timer = setInterval(() => void load(), 60_000);
    return () => {
      cancelled = true;
      if (timer) clearInterval(timer);
    };
  }, []);

  const merged = SEED_DISPLAY.map((seed) => {
    const found = items.find((it) => it.symbol === seed.symbol);
    return {
      label: seed.label,
      price: found?.price ?? 0,
      changePct: found?.change_pct ?? 0,
      symbol: seed.symbol,
    };
  });

  return (
    <header className="h-9 border-b border-line bg-bg flex items-center px-4 gap-6 text-xs">
      <span className="font-mono text-bb-accent">QUOTIENT</span>
      {merged.map((it) => (
        <span key={it.symbol} className="font-mono text-fg-muted">
          {it.label}{" "}
          <span className={changeClass(it.changePct)}>
            {formatPrice(it.symbol, it.price)}
          </span>
          {it.price > 0 && (
            <span className={`${changeClass(it.changePct)} ml-1`}>
              ({it.changePct >= 0 ? "+" : ""}
              {it.changePct.toFixed(2)}%)
            </span>
          )}
        </span>
      ))}
      <span className="ml-auto font-mono text-fg-muted text-[10px]">
        시세 지연 15분
      </span>
    </header>
  );
}

"use client";

import type { Holding } from "@/lib/api/holdings";

const PALETTE = ["#00FFFF", "#FFD500", "#00FF7F", "#FF3344", "#A78BFA", "#FF9F1C"];

export function AllocationDonut({ holdings }: { holdings: Holding[] }) {
  const byClass = new Map<string, number>();
  for (const h of holdings) {
    byClass.set(h.asset_class, (byClass.get(h.asset_class) ?? 0) + h.market_value_krw);
  }
  const total = Array.from(byClass.values()).reduce((s, v) => s + v, 0);

  if (total === 0) {
    return (
      <div className="border border-line p-4">
        <div className="text-xs text-fg-muted font-mono mb-1">자산 분포</div>
        <div className="text-fg-muted text-sm font-mono">데이터 없음</div>
      </div>
    );
  }

  const slices = Array.from(byClass.entries()).map(([k, v], i) => ({
    key: k,
    value: v,
    pct: (v / total) * 100,
    color: PALETTE[i % PALETTE.length],
  }));

  const R = 50;
  const C = 2 * Math.PI * R;
  let offset = 0;

  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">자산 분포 (자산군)</div>
      <div className="flex items-center gap-4">
        <svg viewBox="0 0 120 120" className="w-32 h-32 -rotate-90">
          <circle cx="60" cy="60" r={R} fill="none" stroke="#1a1a1a" strokeWidth="14" />
          {slices.map((s) => {
            const dash = (s.pct / 100) * C;
            const seg = (
              <circle
                key={s.key}
                cx="60" cy="60" r={R}
                fill="none"
                stroke={s.color}
                strokeWidth="14"
                strokeDasharray={`${dash} ${C - dash}`}
                strokeDashoffset={-offset}
              />
            );
            offset += dash;
            return seg;
          })}
        </svg>
        <ul className="flex-1 space-y-1 font-mono text-xs">
          {slices.map((s) => (
            <li key={s.key} className="flex items-center gap-2">
              <span className="inline-block w-2 h-2" style={{ background: s.color }} />
              <span className="flex-1">{s.key}</span>
              <span className="tabular-nums">{s.pct.toFixed(1)}%</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

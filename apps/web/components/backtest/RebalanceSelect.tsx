"use client";
import type { RebalanceFreq } from "@/lib/api/backtest";

const OPTS: { value: RebalanceFreq; label: string }[] = [
  { value: "none", label: "없음" },
  { value: "quarterly", label: "분기" },
  { value: "semiannual", label: "반기" },
  { value: "annual", label: "연" },
];

export function RebalanceSelect({
  value,
  onChange,
}: {
  value: RebalanceFreq;
  onChange: (r: RebalanceFreq) => void;
}) {
  return (
    <label className="flex-1">
      <div className="text-xs text-fg-muted mb-1">리밸런싱</div>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value as RebalanceFreq)}
        className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full"
      >
        {OPTS.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  );
}

"use client";
import { clsx } from "clsx";
import type { BacktestPeriod } from "@/lib/api/backtest";

const OPTS: { value: BacktestPeriod; label: string }[] = [
  { value: "1Y", label: "1Y" },
  { value: "3Y", label: "3Y" },
  { value: "5Y", label: "5Y" },
  { value: "all", label: "전체" },
];

export function PeriodPicker({
  value,
  onChange,
}: {
  value: BacktestPeriod;
  onChange: (p: BacktestPeriod) => void;
}) {
  return (
    <div className="flex gap-1.5">
      {OPTS.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(o.value)}
          className={clsx(
            "px-3 py-1 text-xs font-mono border transition-colors",
            value === o.value
              ? "border-bb-accent text-bb-accent"
              : "border-line text-fg-muted hover:text-fg",
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

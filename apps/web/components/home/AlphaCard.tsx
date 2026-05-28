"use client";

import { useEffect, useState } from "react";
import { getAlpha, isInsufficient, type AlphaPeriod, type AlphaResult, type AlphaInsufficient } from "@/lib/api/portfolio";

const PERIODS: AlphaPeriod[] = ["1m", "90d", "1y", "all"];
const LABELS: Record<AlphaPeriod, string> = { "1m": "1M", "90d": "90D", "1y": "1Y", "all": "All" };

export function AlphaCard() {
  const [period, setPeriod] = useState<AlphaPeriod>("90d");
  const [data, setData] = useState<AlphaResult | AlphaInsufficient | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    getAlpha(period)
      .then((r) => setData(r))
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [period]);

  return (
    <div className="border border-line bg-bg-subtle p-5">
      <Header period={period} onChange={setPeriod} />
      <Body data={data} loading={loading} />
    </div>
  );
}

function Header({ period, onChange }: { period: AlphaPeriod; onChange: (p: AlphaPeriod) => void }) {
  return (
    <div className="flex items-center justify-between mb-3">
      <div className="font-mono text-[10px] text-fg-muted tracking-widest">ALPHA</div>
      <div className="flex gap-1">
        {PERIODS.map((p) => (
          <button
            key={p}
            onClick={() => onChange(p)}
            className={`font-mono text-[10px] px-2 py-0.5 border ${
              period === p ? "border-bb-accent text-bb-accent" : "border-line text-fg-muted hover:text-fg"
            }`}
          >
            {LABELS[p]}
          </button>
        ))}
      </div>
    </div>
  );
}

function Body({ data, loading }: { data: AlphaResult | AlphaInsufficient | null; loading: boolean }) {
  if (data === null && !loading) {
    return <div className="font-mono text-xs text-fg-muted">로드 실패</div>;
  }
  if (data === null) {
    return <div className="font-mono text-xs text-fg-muted">로딩…</div>;
  }
  if (isInsufficient(data)) {
    return <Empty err={data.error} />;
  }
  return <Filled data={data} dim={loading} />;
}

function Empty({ err }: { err: AlphaInsufficient["error"] }) {
  return (
    <div className="space-y-2 py-2">
      <p className="text-sm">{err.message}</p>
      {err.reason === "account_too_young" && (
        <p className="font-mono text-[10px] text-fg-muted">
          가입 {err.current_days}일째 — {err.min_days - err.current_days}일 후부터 비교 가능
        </p>
      )}
      {err.reason === "no_holdings" && (
        <a href="/app/portfolio" className="font-mono text-[10px] text-bb-accent hover:text-bb-warn">
          포트폴리오로 이동 →
        </a>
      )}
    </div>
  );
}

function Filled({ data, dim }: { data: AlphaResult; dim: boolean }) {
  const fmt = (v: number) => `${v >= 0 ? "+" : ""}${v.toFixed(2)}%p`;
  const sign = (v: number) => (v >= 0 ? "text-bb-up" : "text-bb-down");
  return (
    <div className={`space-y-2 transition-opacity ${dim ? "opacity-60" : "opacity-100"}`}>
      {data.benchmarks.map((b) => (
        <div key={b.key} className="flex justify-between font-mono text-xs">
          <span className="text-fg-muted">vs {b.label}</span>
          <span className={sign(b.alpha_pp)}>{fmt(b.alpha_pp)}</span>
        </div>
      ))}
      <Chart data={data} />
      <div className="font-mono text-[10px] text-fg-muted pt-1">
        {data.days_used < data.days_requested && data.days_requested > 0 && (
          <>가입 {data.days_used}일 · </>
        )}
        환율 변동 포함 · 현재 보유 기준 시뮬레이션
        {data.portfolio.data_gaps && data.portfolio.data_gaps.length > 0 && (
          <> · {data.portfolio.data_gaps.length}개 종목 데이터 부족</>
        )}
      </div>
    </div>
  );
}

function Chart({ data }: { data: AlphaResult }) {
  const lines = [
    { series: data.portfolio.series, color: "#FFD500" },
    { series: data.benchmarks[0].series ?? [], color: "#00FFFF" },
    { series: data.benchmarks[1].series ?? [], color: "#FF9900" },
  ];
  const all = lines.flatMap((l) => l.series.map((p) => p.value_pct));
  if (all.length === 0) return null;
  const min = Math.min(...all, 0);
  const max = Math.max(...all, 0);
  const range = max - min || 1;
  const width = 240;
  const height = 36;
  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-9 mt-2">
      {lines.map((l, idx) => (
        <polyline
          key={idx}
          fill="none"
          stroke={l.color}
          strokeWidth="1.2"
          points={l.series
            .map((p, i) => {
              const x = (i / Math.max(l.series.length - 1, 1)) * width;
              const y = height - ((p.value_pct - min) / range) * height;
              return `${x.toFixed(1)},${y.toFixed(1)}`;
            })
            .join(" ")}
        />
      ))}
    </svg>
  );
}

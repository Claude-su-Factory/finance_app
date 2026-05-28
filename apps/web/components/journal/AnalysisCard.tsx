import type { AnalysisRun } from "@/lib/api/journal";

export function AnalysisCard({ run }: { run: AnalysisRun }) {
  const border = run.run_type === "auto_monthly" ? "border-bb-warn" : "border-bb-info";
  const label = run.run_type === "auto_monthly" ? "월간 자동 회고" : "사용자 요청 분석";
  const icon = run.run_type === "auto_monthly" ? "📅" : "💡";
  return (
    <div className={`border-l-2 ${border} bg-bg-card p-4 mb-3`}>
      <div className="font-mono text-[10px] text-fg-muted mb-2">
        {icon} {label} · {run.period_start} ~ {run.period_end} · {run.entries_count}개 entries
      </div>
      <div className="text-sm whitespace-pre-wrap">{run.content_md}</div>
    </div>
  );
}

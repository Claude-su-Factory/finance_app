"use client";

export function ResumePrompt({
  partial,
  onResume,
  onDismiss,
}: {
  partial: string;
  onResume: () => void;
  onDismiss: () => void;
}) {
  return (
    <div className="m-4 p-3 border border-bb-down/50 bg-bb-down/10 font-mono text-xs">
      <p className="mb-2">이전 응답이 중단되었습니다.</p>
      {partial && <p className="text-fg-muted mb-2 line-clamp-3">{partial}</p>}
      <div className="flex gap-2">
        <button onClick={onResume} className="px-3 py-1 border border-bb-accent text-bb-accent">
          이어서 받기
        </button>
        <button onClick={onDismiss} className="px-3 py-1 border border-line text-fg-muted">
          무시
        </button>
      </div>
    </div>
  );
}

// apps/web/components/preview/PreviewSwitcher.tsx
// /preview 전용 화면 스위처. 실제 사이드바·⌘K는 /app(게이팅)을 가리키므로 미리보기 내 이동은 이걸로.
import { PREVIEW_SCREENS } from "@/lib/preview/screens";

export function PreviewSwitcher() {
  return (
    <details className="fixed bottom-4 right-4 z-[9999] font-mono text-xs">
      <summary className="cursor-pointer list-none rounded-md border border-line bg-bg px-3 py-2 shadow-lg select-none">
        ◧ 미리보기 화면
      </summary>
      <nav className="mt-2 max-h-[70vh] w-64 overflow-auto rounded-md border border-line bg-bg p-2 shadow-xl">
        {PREVIEW_SCREENS.map((g) => (
          <div key={g.group} className="mb-2">
            <div className="px-2 py-1 text-[10px] uppercase tracking-wider text-fg-muted">{g.group}</div>
            {g.screens.map((s) => (
              <a key={s.path} href={s.path} className="block rounded px-2 py-1 hover:bg-bg-subtle">
                {s.label}
              </a>
            ))}
          </div>
        ))}
        <div className="border-t border-line px-2 pt-2 text-[10px] text-fg-muted">
          ⌘K: 아무 화면에서나 눌러 커맨드 팰릿
        </div>
      </nav>
    </details>
  );
}

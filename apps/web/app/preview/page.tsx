import { PREVIEW_SCREENS } from "@/lib/preview/screens";

export default function PreviewHub() {
  return (
    <div className="p-6 md:p-8">
      <header className="mb-6">
        <h1 className="font-mono text-2xl">UI 미리보기</h1>
        <p className="mt-1 text-sm text-fg-muted">목 데이터로 렌더되는 개발 전용 화면. 클릭해 들어가 검수한다.</p>
      </header>
      <div className="space-y-6">
        {PREVIEW_SCREENS.map((g) => (
          <section key={g.group}>
            <h2 className="mb-2 font-mono text-xs uppercase tracking-wider text-fg-muted">{g.group}</h2>
            <div className="grid grid-cols-2 gap-2 md:grid-cols-3 lg:grid-cols-4">
              {g.screens.map((s) => (
                <a key={s.path} href={s.path} className="rounded-md border border-line px-3 py-2 font-mono text-sm hover:bg-bg-subtle">
                  {s.label}
                  <span className="block text-[11px] text-fg-muted">{s.path}</span>
                </a>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

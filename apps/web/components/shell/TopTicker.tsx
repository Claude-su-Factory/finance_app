// W2 (데이터 수집 파이프라인) 완료 후 실시간 시세로 교체
export function TopTicker() {
  const items = [
    { sym: "KOSPI",   val: "—", chg: null },
    { sym: "S&P 500", val: "—", chg: null },
    { sym: "USD/KRW", val: "—", chg: null },
  ];
  return (
    <header className="h-9 border-b border-line bg-bg flex items-center px-4 gap-6 text-xs">
      <span className="font-mono text-bb-accent">QUOTIENT</span>
      {items.map((it) => (
        <span key={it.sym} className="font-mono text-fg-muted">
          {it.sym} <span className="text-fg">{it.val}</span>
        </span>
      ))}
      <span className="ml-auto font-mono text-fg-muted text-[10px]">시세 지연 15분</span>
    </header>
  );
}

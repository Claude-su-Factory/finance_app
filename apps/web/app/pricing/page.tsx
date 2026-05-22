export default function Pricing() {
  return (
    <main className="min-h-screen p-12">
      <h1 className="font-mono text-3xl mb-6">PRICING</h1>
      <div className="border border-line p-6">
        <h2 className="font-mono text-xl">Free</h2>
        <p className="text-fg-muted mt-2">현재 모든 기능 무료 (베타).</p>
        <ul className="mt-4 space-y-1 text-fg">
          <li>· 포트폴리오 관리</li>
          <li>· 분석가 채팅 (월 30회)</li>
          <li>· 일일 브리핑</li>
          <li>· 한국·미국 시세 (15분 지연)</li>
        </ul>
      </div>
    </main>
  );
}

import Link from "next/link";

export default function LandingPage() {
  return (
    <main className="min-h-screen bg-bg text-fg">
      <LiveTickerBar />
      <Hero />
      <DashboardPreview />
      <ChatPreview />
      <FeatureGrid />
      <PaperTradingTeaser />
      <TrustCards />
      <FAQ />
      <BottomCTA />
      <Footer />
    </main>
  );
}

/* ──────────────── 상단 라이브 ticker (mock) ──────────────── */
function LiveTickerBar() {
  const items = [
    { sym: "KOSPI", val: "2,580.45", pct: "+0.72%", up: true },
    { sym: "KOSDAQ", val: "735.12", pct: "-0.34%", up: false },
    { sym: "S&P 500", val: "5,847.20", pct: "+0.45%", up: true },
    { sym: "NASDAQ", val: "20,415.80", pct: "+0.61%", up: true },
    { sym: "DJI", val: "42,318.50", pct: "+0.28%", up: true },
    { sym: "USD/KRW", val: "1,378.50", pct: "-0.12%", up: false },
    { sym: "EUR/KRW", val: "1,492.30", pct: "+0.21%", up: true },
    { sym: "US 10Y", val: "4.28%", pct: "+0.03", up: true },
  ];
  return (
    <div className="border-b border-line bg-bg-subtle overflow-hidden">
      <div className="flex gap-8 px-6 py-2 font-mono text-xs whitespace-nowrap">
        {items.map((it) => (
          <span key={it.sym} className="flex items-center gap-2">
            <span className="text-fg-muted">{it.sym}</span>
            <span>{it.val}</span>
            <span className={it.up ? "text-bb-up" : "text-bb-down"}>{it.pct}</span>
          </span>
        ))}
      </div>
    </div>
  );
}

/* ──────────────── Hero ──────────────── */
function Hero() {
  return (
    <section className="border-b border-line">
      <div className="max-w-6xl mx-auto px-6 py-20 md:py-28">
        <p className="font-mono text-xs text-fg-muted mb-6 tracking-widest">
          PORTFOLIO · INTELLIGENCE · TERMINAL
        </p>
        <h1 className="text-4xl md:text-6xl leading-tight tracking-tight mb-6">
          한국·미국 자산을 한 화면에.
          <br />
          <span className="text-bb-accent">자연어로 묻고, 즉시 분석.</span>
        </h1>
        <p className="text-fg-muted text-lg max-w-2xl mb-10 leading-relaxed">
          블룸버그 터미널의 정보 밀도·미감을 개인 가격대로.
          실 자산 분석 · Paper Trading · AI 학습을 한 도구에서.
        </p>

        {/* 3축 가치 */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-px bg-line mb-10 border border-line">
          {[
            { tag: "01", color: "text-bb-accent", title: "실 자산 분석", desc: "직접 입력한 보유 자산을 KRW 환산·손익·비중·시계열로 종합 분석" },
            { tag: "02", color: "text-bb-info", title: "Paper Trading", desc: "가상 자금으로 매매 시뮬레이션·백테스트. 실 매매 전 전략 검증 (Phase 2)" },
            { tag: "03", color: "text-bb-up", title: "AI 분석가 + 학습", desc: "Claude가 도구를 호출해 실데이터로 답변. 개념 질문도 친절히 설명" },
          ].map((v) => (
            <div key={v.tag} className="bg-bg p-6">
              <div className={`font-mono text-xs mb-2 ${v.color}`}>{v.tag}</div>
              <div className="text-lg mb-1">{v.title}</div>
              <div className="text-fg-muted text-sm leading-relaxed">{v.desc}</div>
            </div>
          ))}
        </div>

        <div className="flex flex-wrap gap-3">
          <Link href="/signup" className="px-6 py-3 bg-bb-accent text-bg font-mono text-sm hover:bg-bb-warn focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg">
            무료로 시작 →
          </Link>
          <Link href="/login" className="px-6 py-3 border border-line font-mono text-sm hover:border-fg-muted">
            로그인
          </Link>
          <span className="self-center font-mono text-xs text-fg-muted ml-2">
            MVP는 100% 무료 · 신용카드 불필요
          </span>
        </div>
      </div>
    </section>
  );
}

/* ──────────────── Dashboard preview ──────────────── */
function DashboardPreview() {
  return (
    <section className="border-b border-line">
      <div className="max-w-6xl mx-auto px-6 py-20">
        <p className="font-mono text-xs text-fg-muted tracking-widest mb-3">PREVIEW · /app/home</p>
        <h2 className="text-3xl mb-10">이런 화면을 매일 봅니다.</h2>

        <div className="border border-line bg-bg-subtle p-6">
          {/* 미니 사이드바 + 메인 */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
            <PreviewCard label="총자산 (KRW)">
              <div className="font-mono text-3xl text-bb-accent">42,182,500</div>
              <div className="font-mono text-sm text-bb-up mt-1">+1,245,300 (+3.04%)</div>
            </PreviewCard>

            <PreviewCard label="자산 분포">
              <div className="flex items-center gap-4">
                <svg viewBox="0 0 36 36" className="w-20 h-20">
                  <circle cx="18" cy="18" r="15.91" fill="transparent" stroke="#262626" strokeWidth="3" />
                  <circle cx="18" cy="18" r="15.91" fill="transparent" stroke="#FFD500" strokeWidth="3" strokeDasharray="55 100" strokeDashoffset="25" />
                  <circle cx="18" cy="18" r="15.91" fill="transparent" stroke="#00FFFF" strokeWidth="3" strokeDasharray="30 100" strokeDashoffset="-30" />
                  <circle cx="18" cy="18" r="15.91" fill="transparent" stroke="#00FF7F" strokeWidth="3" strokeDasharray="15 100" strokeDashoffset="-60" />
                </svg>
                <div className="font-mono text-xs space-y-1">
                  <div><span className="text-bb-accent">●</span> KR 55%</div>
                  <div><span className="text-bb-info">●</span> US 30%</div>
                  <div><span className="text-bb-up">●</span> CASH 15%</div>
                </div>
              </div>
            </PreviewCard>

            <PreviewCard label="알파 vs KOSPI · 90D">
              <div className="font-mono text-3xl text-bb-up">+8.42<span className="text-base text-fg-muted ml-1">%p</span></div>
              <div className="font-mono text-xs text-fg-muted mt-1">내 포트폴리오가 KOSPI 대비 우위</div>
              <svg viewBox="0 0 200 30" className="w-full h-8 mt-2">
                <polyline points="0,22 20,20 40,18 60,15 80,12 100,10 120,8 140,7 160,5 180,4 200,3" fill="none" stroke="#00FF7F" strokeWidth="1.5" />
              </svg>
            </PreviewCard>

            <PreviewCard label="상위 5 보유" className="lg:col-span-2">
              <table className="w-full font-mono text-xs">
                <tbody className="text-sm">
                  <tr><td className="py-1">삼성전자</td><td className="text-right text-fg-muted">35%</td><td className="text-right text-bb-up w-20">+8.3%</td></tr>
                  <tr><td className="py-1">AAPL</td><td className="text-right text-fg-muted">19%</td><td className="text-right text-bb-up w-20">+9.8%</td></tr>
                  <tr><td className="py-1">SK하이닉스</td><td className="text-right text-fg-muted">18%</td><td className="text-right text-bb-down w-20">-2.3%</td></tr>
                  <tr><td className="py-1">NVDA</td><td className="text-right text-fg-muted">16%</td><td className="text-right text-bb-up w-20">+15.4%</td></tr>
                  <tr><td className="py-1">현대차</td><td className="text-right text-fg-muted">10%</td><td className="text-right text-bb-up w-20">+13.0%</td></tr>
                </tbody>
              </table>
            </PreviewCard>

            <PreviewCard label="오늘의 브리핑 · 07:00 KST">
              <p className="text-sm leading-relaxed">
                반도체 비중이 50%를 넘어 섹터 집중도가 높습니다. USD/KRW 소폭 하락으로 US 주식 KRW 환산액이…
              </p>
            </PreviewCard>
          </div>
          <p className="font-mono text-[10px] text-fg-muted text-right mt-3">
            * 데모 데이터입니다. 실제 화면은 본인 보유 자산 기준으로 표시됩니다.
          </p>
        </div>
      </div>
    </section>
  );
}

function PreviewCard({ label, children, className = "" }: { label: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={`border border-line bg-bg p-5 ${className}`}>
      <div className="font-mono text-[10px] text-fg-muted tracking-widest mb-3">{label}</div>
      {children}
    </div>
  );
}

/* ──────────────── AI Chat preview ──────────────── */
function ChatPreview() {
  return (
    <section className="border-b border-line bg-bg-subtle">
      <div className="max-w-6xl mx-auto px-6 py-20 grid md:grid-cols-2 gap-10 items-start">
        <div>
          <p className="font-mono text-xs text-fg-muted tracking-widest mb-3">AI ANALYST</p>
          <h2 className="text-3xl mb-4">분석가에게 물어보듯.</h2>
          <p className="text-fg-muted leading-relaxed mb-6">
            Claude가 실데이터를 도구로 호출해 답합니다.
            "내 통화 분산 어때?", "삼성전자 30일 추이는?", "PER이 뭐야?" — 분석도, 개념 설명도 모두.
          </p>
          <ul className="space-y-2 font-mono text-xs">
            <li><span className="text-bb-info mr-2">▸</span>9개 도구 — 포트폴리오·시세·차트·관심 종목·경제 지표</li>
            <li><span className="text-bb-info mr-2">▸</span>SSE 스트리밍 — 토큰 단위 즉시 표시</li>
            <li><span className="text-bb-info mr-2">▸</span>Sonnet 4.6 기본 / Opus 4.7 심층 / Haiku 내부 요약</li>
            <li><span className="text-bb-info mr-2">▸</span>월 30회 무료</li>
          </ul>
        </div>

        <div className="border border-line bg-bg p-5">
          <div className="font-mono text-[10px] text-fg-muted mb-3">USER · 14:21</div>
          <div className="text-sm mb-5">내 포트폴리오 통화 분산 어때?</div>

          <div className="font-mono text-[10px] text-bb-accent mb-2">QUOTIENT · 14:21 · Sonnet 4.6</div>
          <div className="border-l-2 border-bb-info pl-3 mb-2 font-mono text-[11px] text-fg-muted">
            🔧 <span className="text-bb-info">get_portfolio</span> · 5개 보유 조회 · 287ms
          </div>
          <div className="border-l-2 border-bb-info pl-3 mb-3 font-mono text-[11px] text-fg-muted">
            🔧 <span className="text-bb-info">calc_portfolio_metrics</span> · 통화 분산 계산 · 142ms
          </div>
          <div className="text-sm leading-relaxed space-y-2">
            <p>현재 포트폴리오의 통화 분산은:</p>
            <ul className="text-fg-muted space-y-1 list-disc list-inside ml-2">
              <li><span className="text-fg">KRW 약 63%</span> (삼성전자·SK하이닉스·현대차 합산)</li>
              <li><span className="text-fg">USD 약 37%</span> (AAPL·NVDA)</li>
            </ul>
            <p>분석 관점에서 USD 비중이 한국 개인 평균(15~20%) 대비 높아…</p>
          </div>
          <div className="font-mono text-[10px] text-fg-muted mt-3">
            (데이터 기준: 2026-05-27 14:21 KST · 시세 지연 15분)
          </div>
        </div>
      </div>
    </section>
  );
}

/* ──────────────── Feature grid ──────────────── */
function FeatureGrid() {
  const features = [
    {
      tag: "01 · INTEGRATE",
      title: "통합 포트폴리오",
      desc: "KOSPI·KOSDAQ·NASDAQ·NYSE 자산을 한 곳에. KRW 환산·손익·비중·7일 스파크라인 자동 계산.",
      color: "text-bb-accent",
    },
    {
      tag: "02 · ANALYZE",
      title: "AI 분석가",
      desc: "Claude 기반 자연어 인터페이스. 9개 도구로 실데이터 분석. 도구 호출이 화면에 모두 노출 — 블랙박스 아님.",
      color: "text-bb-info",
    },
    {
      tag: "03 · MONITOR",
      title: "마켓 모니터",
      desc: "KOSPI·S&P·환율·금리·국채를 한 눈에. 5년 일봉 차트 · 매분 갱신 · 15분 지연.",
      color: "text-bb-up",
    },
    {
      tag: "04 · BRIEF",
      title: "매일 아침 브리핑",
      desc: "07:00 KST 자동 발송. 보유 자산·관심 종목·시장 변화 3~5문장 요약. AI가 매일 다른 관점으로 짚어줌.",
      color: "text-bb-warn",
    },
  ];
  return (
    <section className="border-b border-line">
      <div className="max-w-6xl mx-auto px-6 py-20">
        <p className="font-mono text-xs text-fg-muted tracking-widest mb-3">FEATURES</p>
        <h2 className="text-3xl mb-10">한 도구로 다 끝.</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-px bg-line border border-line">
          {features.map((f) => (
            <div key={f.tag} className="bg-bg p-8">
              <div className={`font-mono text-xs mb-3 ${f.color} tracking-widest`}>{f.tag}</div>
              <h3 className="text-2xl mb-3">{f.title}</h3>
              <p className="text-fg-muted leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ──────────────── Paper trading teaser ──────────────── */
function PaperTradingTeaser() {
  return (
    <section className="border-b border-line">
      <div className="max-w-6xl mx-auto px-6 py-20">
        <div className="flex items-baseline gap-3 mb-3">
          <p className="font-mono text-xs text-fg-muted tracking-widest">COMING · PHASE 2</p>
          <span className="font-mono text-[10px] px-2 py-0.5 border border-bb-warn/40 text-bb-warn">곧 출시</span>
        </div>
        <h2 className="text-3xl mb-6">Paper Trading + 백테스트.</h2>
        <p className="text-fg-muted leading-relaxed max-w-3xl mb-8">
          실제 자금을 쓰기 전에 가상 자금으로 매매 전략을 검증합니다.
          5년 전 시점부터 NVDA를 매월 적립했다면? 환율 헷지 비중을 30% 두었다면?
          AI 매매 일기가 가상 매매 결정의 패턴까지 사후 분석합니다.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-px bg-line border border-line">
          <div className="bg-bg p-6">
            <div className="font-mono text-xs text-bb-info mb-2">VIRTUAL FUNDS</div>
            <p className="text-sm text-fg-muted">가상 자금 1,000만원으로 시작. 매매·평단·손익 모두 추적.</p>
          </div>
          <div className="bg-bg p-6">
            <div className="font-mono text-xs text-bb-info mb-2">BACKTEST</div>
            <p className="text-sm text-fg-muted">"5년 전부터 이렇게 굴렸다면?" 과거 시점 시뮬레이션.</p>
          </div>
          <div className="bg-bg p-6">
            <div className="font-mono text-xs text-bb-info mb-2">AI JOURNAL</div>
            <p className="text-sm text-fg-muted">매매 결정 이유 기록 → AI가 사후 패턴 분석.</p>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ──────────────── Trust cards (안 하는 것) ──────────────── */
function TrustCards() {
  const items = [
    { title: "자금 보관·이체 안 함", desc: "결제는 PG에 완전 위임. 우리는 사용자 자금을 보관·이체하지 않습니다." },
    { title: "마이데이터 자동 연동 안 함", desc: "보유 자산은 사용자가 직접 입력. 실제 보유 여부는 검증하지 않으며 분석 도구로만 제공." },
    { title: "직접적인 매수/매도 추천 안 함", desc: "AI는 \"분석 관점에서 ~를 점검해볼 수 있습니다\" 형태로만 답합니다." },
    { title: "세금·법무 자문 안 함", desc: "관련 질문은 \"전문가에게 문의하세요\"로 안내합니다." },
  ];
  return (
    <section className="border-b border-line bg-bg-subtle">
      <div className="max-w-6xl mx-auto px-6 py-20">
        <p className="font-mono text-xs text-fg-muted tracking-widest mb-3">PROMISE</p>
        <h2 className="text-3xl mb-3">안 하는 것을 분명히.</h2>
        <p className="text-fg-muted mb-10 max-w-2xl">
          한국 금융 규제와 개인 운영의 한계를 정확히 인식하고 만듭니다.
          시작 전 무엇을 기대해선 안 되는지 알려드립니다.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {items.map((it) => (
            <div key={it.title} className="border border-line bg-bg p-5">
              <div className="flex items-baseline gap-3 mb-2">
                <span className="font-mono text-bb-down">✕</span>
                <h3 className="text-base">{it.title}</h3>
              </div>
              <p className="text-fg-muted text-sm leading-relaxed pl-6">{it.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ──────────────── FAQ ──────────────── */
function FAQ() {
  const qs = [
    {
      q: "보유 자산은 어떻게 검증하나요?",
      a: "검증하지 않습니다. 사용자가 직접 입력하며 본인 분석 도구로 제공합니다. 다른 사용자에게 노출되지 않으므로 본인 정확도가 곧 분석 정확도입니다.",
    },
    {
      q: "다른 핀테크(토스·뱅크샐러드)와 차이는?",
      a: "그쪽은 마이데이터로 통합 자산 관리. 우리는 라이선스 없이 분석에만 집중 — 정보 밀도가 높은 모노스페이스 UI, 자연어 AI 분석가, 한국+미국 자산 한 화면. 개발자·파워유저 타겟.",
    },
    {
      q: "AI는 어떤 모델인가요? 비용은?",
      a: "Anthropic Claude (Sonnet 4.6 기본, Opus 4.7 심층, Haiku 내부 요약). MVP는 무료, 월 30회 한도. Phase 2에 ₩14,900 Pro 플랜으로 한도 확장 예정.",
    },
    {
      q: "Paper Trading은 언제 나오나요?",
      a: "Phase 2 (MVP 출시 후 가입자 100명 또는 3개월 경과 시점). 가상 자금 + 백테스트 + AI 매매 일기를 한 번에.",
    },
  ];
  return (
    <section className="border-b border-line">
      <div className="max-w-3xl mx-auto px-6 py-20">
        <p className="font-mono text-xs text-fg-muted tracking-widest mb-3">FAQ</p>
        <h2 className="text-3xl mb-10">자주 묻는 질문.</h2>
        <div className="space-y-px bg-line border border-line">
          {qs.map((it) => (
            <details key={it.q} className="bg-bg group">
              <summary className="px-5 py-4 cursor-pointer flex items-baseline gap-3 list-none">
                <span className="font-mono text-bb-accent text-xs">Q.</span>
                <span className="flex-1">{it.q}</span>
                <span className="font-mono text-fg-muted text-xs group-open:rotate-180 transition-transform">▾</span>
              </summary>
              <div className="px-5 pb-5 pl-12 text-fg-muted text-sm leading-relaxed">{it.a}</div>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ──────────────── Bottom CTA ──────────────── */
function BottomCTA() {
  return (
    <section className="border-b border-line">
      <div className="max-w-4xl mx-auto px-6 py-20 text-center">
        <p className="font-mono text-xs text-fg-muted tracking-widest mb-3">READY?</p>
        <h2 className="text-4xl md:text-5xl mb-6 leading-tight">
          분석가가 옆에 있는 포트폴리오를<br />
          <span className="text-bb-accent">지금 무료로.</span>
        </h2>
        <p className="text-fg-muted mb-8 max-w-xl mx-auto">
          1분 가입 → 보유 자산 입력 → AI에게 첫 질문.<br />
          이메일 또는 Google 로그인.
        </p>
        <div className="flex flex-wrap justify-center gap-3">
          <Link href="/signup" className="px-6 py-3 bg-bb-accent text-bg font-mono text-sm hover:bg-bb-warn">
            무료로 시작 →
          </Link>
          <Link href="/login" className="px-6 py-3 border border-line font-mono text-sm hover:border-fg-muted">
            이미 계정이 있어요
          </Link>
        </div>
      </div>
    </section>
  );
}

/* ──────────────── Footer ──────────────── */
function Footer() {
  return (
    <footer className="bg-bg-subtle">
      <div className="max-w-6xl mx-auto px-6 py-10 grid grid-cols-1 md:grid-cols-4 gap-8 text-sm">
        <div>
          <div className="font-mono text-bb-accent mb-3">QUOTIENT</div>
          <p className="text-fg-muted text-xs leading-relaxed">
            Portfolio Intelligence Terminal.<br />
            블룸버그 정보 밀도를 개인 가격대로.
          </p>
        </div>
        <div>
          <div className="font-mono text-xs text-fg-muted tracking-widest mb-3">LEGAL</div>
          <ul className="space-y-2">
            <li><Link href="/terms" className="hover:text-fg text-fg-muted">이용약관</Link></li>
            <li><Link href="/privacy" className="hover:text-fg text-fg-muted">개인정보 처리방침</Link></li>
          </ul>
        </div>
        <div>
          <div className="font-mono text-xs text-fg-muted tracking-widest mb-3">CONTACT</div>
          <ul className="space-y-2 text-fg-muted">
            <li>sdl182975@gmail.com</li>
          </ul>
        </div>
        <div>
          <div className="font-mono text-xs text-fg-muted tracking-widest mb-3">STATUS</div>
          <p className="text-fg-muted text-xs leading-relaxed">
            MVP · 무료 · 광고는 마켓 페이지 하단에만.
            Phase 2부터 Pro 플랜(₩14,900) 예정.
          </p>
        </div>
      </div>
      <div className="border-t border-line">
        <div className="max-w-6xl mx-auto px-6 py-4 text-fg-muted text-xs font-mono flex flex-wrap gap-x-6 gap-y-2 justify-between">
          <span>© 2026 Quotient · 본 서비스는 투자 자문업이 아닙니다.</span>
          <span>시세 데이터 15분 지연 · 모든 의사결정은 본인 책임</span>
        </div>
      </div>
    </footer>
  );
}

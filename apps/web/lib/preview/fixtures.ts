// apps/web/lib/preview/fixtures.ts
// 키: API pathname. 값: 200 응답 바디. 한국어 더미. (Task 2~3에서 항목 추가)
export const MOCKS: Record<string, unknown> = {
  // ── 포트폴리오 ──────────────────────────────────────────────────────────────
  "/v1/holdings": [
    {
      id: "h1", instrument_id: "i-samsung", quantity: 50, avg_cost: 68000,
      opened_at: "2026-03-02", note: null,
      created_at: "2026-03-02T00:00:00Z", updated_at: "2026-05-31T00:00:00Z",
      symbol: "005930", exchange: "KRX", name: "삼성전자", asset_class: "KR_STOCK",
      currency: "KRW", current_price: 79800, market_value: 3990000, market_value_krw: 3990000,
      cost_basis_krw: 3400000, pnl_krw: 590000, pnl_pct: 17.35, weight_pct: 53.1,
    },
    {
      id: "h2", instrument_id: "i-aapl", quantity: 12, avg_cost: 180.2,
      opened_at: "2026-04-10", note: null,
      created_at: "2026-04-10T00:00:00Z", updated_at: "2026-05-31T00:00:00Z",
      symbol: "AAPL", exchange: "NASDAQ", name: "Apple Inc.", asset_class: "US_STOCK",
      currency: "USD", current_price: 212.5, market_value: 2550, market_value_krw: 3519000,
      cost_basis_krw: 3073000, pnl_krw: 446000, pnl_pct: 14.5, weight_pct: 46.9,
    },
  ],

  // ── 알파 카드 (홈 AlphaCard) ────────────────────────────────────────────────
  // AlphaResult (lib/api/portfolio.ts). 쿼리(?period=)는 catch-all이 떼어내 키는 경로만.
  // Chart가 benchmarks[0]·[1].series를 직접 참조 → 벤치마크 최소 2개 필수(3개 채움).
  // alpha_pp = portfolio.total_return_pct − benchmark.total_return_pct (정합).
  "/v1/portfolio/alpha": {
    period: "90d", days_requested: 90, days_used: 90, since: "2026-03-03",
    fx_mode: "spot", model: "current_holdings_backward_simulation",
    portfolio: {
      total_return_pct: 7.53,
      series: [
        { date: "2026-03-03", value_pct: 0 }, { date: "2026-03-24", value_pct: 1.8 },
        { date: "2026-04-14", value_pct: 3.5 }, { date: "2026-05-05", value_pct: 5.9 },
        { date: "2026-05-31", value_pct: 7.53 },
      ],
    },
    benchmarks: [
      {
        key: "kospi", label: "KOSPI", total_return_pct: 3.2, alpha_pp: 4.33,
        series: [
          { date: "2026-03-03", value_pct: 0 }, { date: "2026-03-24", value_pct: 1.1 },
          { date: "2026-04-14", value_pct: 0.8 }, { date: "2026-05-05", value_pct: 2.4 },
          { date: "2026-05-31", value_pct: 3.2 },
        ],
      },
      {
        key: "sp500", label: "S&P 500", total_return_pct: 5.1, alpha_pp: 2.43,
        series: [
          { date: "2026-03-03", value_pct: 0 }, { date: "2026-03-24", value_pct: 1.5 },
          { date: "2026-04-14", value_pct: 2.9 }, { date: "2026-05-05", value_pct: 4.2 },
          { date: "2026-05-31", value_pct: 5.1 },
        ],
      },
      {
        key: "kr_us_6040", label: "한·미 60/40", total_return_pct: 4.05, alpha_pp: 3.48,
        series: [
          { date: "2026-03-03", value_pct: 0 }, { date: "2026-03-24", value_pct: 1.3 },
          { date: "2026-04-14", value_pct: 2.0 }, { date: "2026-05-05", value_pct: 3.3 },
          { date: "2026-05-31", value_pct: 4.05 },
        ],
      },
    ],
  },

  // ── 마켓 (AppShell TopTicker + KRIndicesCard + USIndicesCard) ──────────────
  // TopTicker(AppShell). 실제 ticker 응답 타입에 맞춰 보정. 틀려도 TopTicker는 빈 상태로 degrade.
  "/v1/market/ticker": [
    { symbol: "KOSPI", name: "코스피", price: 2712.34, change_pct: 0.82 },
    { symbol: "KOSDAQ", name: "코스닥", price: 871.2, change_pct: -0.31 },
    { symbol: "USDKRW", name: "원/달러", price: 1378.5, change_pct: 0.14 },
    { symbol: "SPX", name: "S&P 500", price: 5482.1, change_pct: 0.41 },
    { symbol: "NDX", name: "NASDAQ 100", price: 19823.5, change_pct: 0.67 },
  ],

  // ── Paper Trading ──────────────────────────────────────────────────────────
  "/v1/paper/portfolio": {
    account: {
      user_id: "preview-user", initial_cash: 10000000, cash_balance: 6029504,
      base_currency: "KRW", created_at: "2026-03-01T00:00:00Z", updated_at: "2026-05-31T00:00:00Z",
    },
    holdings: [
      {
        id: "ph1", user_id: "preview-user", instrument_id: "i-samsung", symbol: "005930",
        name: "삼성전자", currency: "KRW", quantity: 40, avg_cost: 71000, current_price: 79800,
        market_value: 3192000, market_value_krw: 3192000, pnl_krw: 352000, pnl_pct: 12.39,
        created_at: "2026-03-02T00:00:00Z", updated_at: "2026-05-31T00:00:00Z",
      },
      {
        id: "ph2", user_id: "preview-user", instrument_id: "i-nvda", symbol: "NVDA",
        name: "NVIDIA", currency: "USD", quantity: 8, avg_cost: 102.4, current_price: 138.7,
        market_value: 1109.6, market_value_krw: 1531248, pnl_krw: 400752, pnl_pct: 35.45,
        created_at: "2026-04-10T00:00:00Z", updated_at: "2026-05-31T00:00:00Z",
      },
    ],
    summary: { total_equity_krw: 10752752, total_pnl_krw: 752752, total_pnl_pct: 7.53 },
    equity_series: [
      { date: "2026-03-01", equity_krw: 10000000 },
      { date: "2026-04-01", equity_krw: 10210000 },
      { date: "2026-05-01", equity_krw: 10480000 },
      { date: "2026-05-31", equity_krw: 10752752 },
    ],
  },
  "/v1/paper/transactions": {
    transactions: [
      {
        id: "tx1", user_id: "preview-user", instrument_id: "i-samsung", symbol: "005930",
        action: "buy", quantity: 40, price: 71000, currency: "KRW", fx_to_krw: 1,
        total_krw: 2840000, active: true, created_at: "2026-03-02T01:00:00Z",
      },
      {
        id: "tx2", user_id: "preview-user", instrument_id: "i-nvda", symbol: "NVDA",
        action: "buy", quantity: 8, price: 102.4, currency: "USD", fx_to_krw: 1380,
        total_krw: 1130496, active: true, created_at: "2026-04-10T01:00:00Z",
      },
    ],
    has_more: false,
  },

  // ── 매매 일기 ───────────────────────────────────────────────────────────────
  "/v1/journal/entries": {
    entries: [
      {
        id: "je1", user_id: "preview-user", entry_type: "auto",
        action: "buy", related_holding_id: "h1",
        related_holding: { symbol: "005930", name: "삼성전자" },
        related_symbols: ["005930"],
        title: "삼성전자 매수",
        content: "반도체 업황 회복 기대. HBM 수요 증가 국면에서 저점 분할 매수 진입.",
        created_at: "2026-03-02T01:00:00Z", updated_at: "2026-03-02T01:00:00Z",
      },
      {
        id: "je2", user_id: "preview-user", entry_type: "manual",
        action: "observation", related_holding_id: null,
        related_symbols: ["SPX", "NDX"],
        title: "미국 시장 관찰",
        content: "Fed 금리 동결 기조 확인. 나스닥 기술주 모멘텀 지속 중. 포트 비중 점검 필요.",
        created_at: "2026-04-15T09:30:00Z", updated_at: "2026-04-15T09:30:00Z",
      },
      {
        id: "je3", user_id: "preview-user", entry_type: "auto",
        action: "buy", related_holding_id: "h2",
        related_holding: { symbol: "AAPL", name: "Apple Inc." },
        related_symbols: ["AAPL"],
        title: "Apple 매수",
        content: "서비스 매출 비중 확대와 견조한 아이폰 수요. 환율 부담에도 분할 매수로 접근.",
        created_at: "2026-04-10T01:00:00Z", updated_at: "2026-04-10T01:00:00Z",
      },
    ],
    has_more: false,
  },
  "/v1/journal/analyses": {
    analyses: [
      {
        id: "an1", user_id: "preview-user", run_type: "auto_monthly",
        period_start: "2026-05-01", period_end: "2026-05-31",
        entries_count: 5, model: "claude-sonnet-4-6",
        content_md: "## 5월 매매 회고\n\n삼성전자·NVDA 중심의 반도체 포지션이 월간 수익률을 견인했습니다. 전반적으로 매수 타이밍 판단이 적절했으며, 분할 매수 전략이 변동성 리스크를 낮추는 데 효과적이었습니다.",
        created_at: "2026-06-01T00:00:00Z",
      },
    ],
  },

  // ── 백테스트 (BacktestPage "실행" 시 POST /v1/backtest/run) ──────────────────
  // BacktestResult (lib/api/backtest.ts). 결과 뷰가 참조하는 전 필드를 채움:
  // metrics(7) · benchmarks{kospi,spx,sixty_forty}.{equity_series,metrics} · equity_series.
  // 날짜는 4개 시계열(전략+벤치 3종) 동일하게 맞춰 차트 병합 시 빈 칸 없음.
  // 정합: 전략 +85%(10M→18.5M), excess_vs_6040 = 85 − 55 = 30.
  "/v1/backtest/run": {
    clamped_start: "2021-06-01", end: "2026-05-31",
    normalized_basket: [
      { instrument_id: "i-samsung", symbol: "005930", name: "삼성전자", weight: 0.6 },
      { instrument_id: "i-aapl", symbol: "AAPL", name: "Apple Inc.", weight: 0.4 },
    ],
    equity_series: [
      { date: "2021-06-01", value: 10000000 }, { date: "2022-06-01", value: 11800000 },
      { date: "2023-06-01", value: 13200000 }, { date: "2024-06-01", value: 15100000 },
      { date: "2025-06-01", value: 16900000 }, { date: "2026-05-31", value: 18500000 },
    ],
    contributed_series: [
      { date: "2021-06-01", value: 10000000 }, { date: "2022-06-01", value: 10000000 },
      { date: "2023-06-01", value: 10000000 }, { date: "2024-06-01", value: 10000000 },
      { date: "2025-06-01", value: 10000000 }, { date: "2026-05-31", value: 10000000 },
    ],
    benchmarks: {
      kospi: {
        equity_series: [
          { date: "2021-06-01", value: 10000000 }, { date: "2022-06-01", value: 10900000 },
          { date: "2023-06-01", value: 11600000 }, { date: "2024-06-01", value: 12700000 },
          { date: "2025-06-01", value: 13500000 }, { date: "2026-05-31", value: 14200000 },
        ],
        metrics: { total_return_pct: 42.0, cagr_pct: 7.2, mdd_pct: -28.5, volatility_pct: 20.1, twr_pct: 42.0 },
      },
      spx: {
        equity_series: [
          { date: "2021-06-01", value: 10000000 }, { date: "2022-06-01", value: 11500000 },
          { date: "2023-06-01", value: 13000000 }, { date: "2024-06-01", value: 14600000 },
          { date: "2025-06-01", value: 15800000 }, { date: "2026-05-31", value: 16800000 },
        ],
        metrics: { total_return_pct: 68.0, cagr_pct: 11.0, mdd_pct: -19.8, volatility_pct: 16.4, twr_pct: 68.0 },
      },
      sixty_forty: {
        equity_series: [
          { date: "2021-06-01", value: 10000000 }, { date: "2022-06-01", value: 11200000 },
          { date: "2023-06-01", value: 12400000 }, { date: "2024-06-01", value: 13700000 },
          { date: "2025-06-01", value: 14700000 }, { date: "2026-05-31", value: 15500000 },
        ],
        metrics: { total_return_pct: 55.0, cagr_pct: 9.2, mdd_pct: -15.2, volatility_pct: 12.1, twr_pct: 55.0 },
      },
    },
    metrics: {
      total_return_pct: 85.0, cagr_pct: 13.1, mdd_pct: -22.4, volatility_pct: 18.6,
      excess_vs_6040_pct: 30.0, total_contributed: 10000000, final_equity: 18500000,
    },
    coverage_warnings: [],
  },

  // ── 채팅 ────────────────────────────────────────────────────────────────────
  "/v1/chat/sessions": [
    {
      id: "cs1", user_id: "preview-user",
      title: "포트폴리오 리밸런싱 전략",
      created_at: "2026-05-28T10:00:00Z", updated_at: "2026-05-28T10:30:00Z",
    },
    {
      id: "cs2", user_id: "preview-user",
      title: "NVDA 매수 근거 분석",
      created_at: "2026-05-20T14:00:00Z", updated_at: "2026-05-20T14:45:00Z",
    },
  ],
  "/v1/chat/usage": {
    usage: {
      user_id: "preview-user", year_month: "2026-06",
      chat_count: 12, input_tokens: 48200, output_tokens: 21300, opus_count: 2,
    },
    limits: { chat: 100, input_tokens: 500000, output_tokens: 200000, opus: 10 },
  },

  // ── 워치리스트 (마켓 WatchlistEditorCard) ──────────────────────────────────
  "/v1/watchlist": [
    {
      instrument_id: "i-hynix", symbol: "000660", exchange: "KRX",
      name: "SK하이닉스", asset_class: "KR_STOCK", currency: "KRW",
      price: 178500, change_pct: 1.42, added_at: "2026-05-01T00:00:00Z",
    },
    {
      instrument_id: "i-tsmc", symbol: "TSM", exchange: "NYSE",
      name: "Taiwan Semiconductor", asset_class: "US_STOCK", currency: "USD",
      price: 182.3, change_pct: -0.58, added_at: "2026-05-10T00:00:00Z",
    },
  ],

  // ── 종목 검색 (InstrumentSearchInput: 백테스트 바스켓·종목 추가 모달 공용) ──────
  // InstrumentResult[] (lib/api/instruments.ts). ?q= 는 catch-all이 제거 → 검색어 무관 동일 목록.
  // INDEX/FX/CASH는 handlePick에서 선택 차단 → 보유·바스켓 가능한 자산만 둔다.
  "/v1/instruments/search": [
    { id: "i-samsung", symbol: "005930", exchange: "KRX", name: "삼성전자", currency: "KRW", asset_class: "KR_STOCK" },
    { id: "i-hynix", symbol: "000660", exchange: "KRX", name: "SK하이닉스", currency: "KRW", asset_class: "KR_STOCK" },
    { id: "i-aapl", symbol: "AAPL", exchange: "NASDAQ", name: "Apple Inc.", currency: "USD", asset_class: "US_STOCK" },
    { id: "i-nvda", symbol: "NVDA", exchange: "NASDAQ", name: "NVIDIA", currency: "USD", asset_class: "US_STOCK" },
    { id: "i-spy", symbol: "SPY", exchange: "NYSE", name: "SPDR S&P 500 ETF", currency: "USD", asset_class: "ETF" },
  ],
};

// 명시적 목이 없는 /v1/* 는 빈 객체로 degrade. 대부분 컴포넌트가 .catch(()=>setX([]))라 안전.
// Object.hasOwn — `in`은 프로토타입 체인까지 봐서 "toString" 같은 키가 오탐된다.
export function lookupFixture(pathname: string): unknown {
  return Object.hasOwn(MOCKS, pathname) ? MOCKS[pathname] : {};
}

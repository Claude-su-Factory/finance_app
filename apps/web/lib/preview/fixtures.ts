// apps/web/lib/preview/fixtures.ts
// 키: API pathname. 값: 200 응답 바디. 한국어 더미. (Task 2~3에서 항목 추가)
export const MOCKS: Record<string, unknown> = {
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
  // TopTicker(AppShell). 실제 ticker 응답 타입에 맞춰 보정. 틀려도 TopTicker는 빈 상태로 degrade.
  "/v1/market/ticker": [
    { symbol: "KOSPI", name: "코스피", price: 2712.34, change_pct: 0.82 },
    { symbol: "KOSDAQ", name: "코스닥", price: 871.2, change_pct: -0.31 },
    { symbol: "USDKRW", name: "원/달러", price: 1378.5, change_pct: 0.14 },
  ],
};

// 명시적 목이 없는 /v1/* 는 빈 객체로 degrade. 대부분 컴포넌트가 .catch(()=>setX([]))라 안전.
// Object.hasOwn — `in`은 프로토타입 체인까지 봐서 "toString" 같은 키가 오탐된다.
export function lookupFixture(pathname: string): unknown {
  return Object.hasOwn(MOCKS, pathname) ? MOCKS[pathname] : {};
}

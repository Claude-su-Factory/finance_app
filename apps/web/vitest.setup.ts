import "@testing-library/jest-dom/vitest";

// recharts ResponsiveContainer는 jsdom에 없는 ResizeObserver를 생성자에서 요구한다.
// no-op 스텁 → 크래시 방지 + 컨테이너 크기 0 유지(차트 비렌더, Legend 텍스트가 표·카드와 충돌 안 함).
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver;

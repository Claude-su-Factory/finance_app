// 차트 색상 토큰 — Tailwind 토큰과 정합.
export const CHART_COLORS = {
  up: "#00FF7F",
  down: "#FF3344",
  accent: "#00FFFF",
  warn: "#FFD500",
  muted: "#666666",
  line: "#1a1a1a",
} as const;

export function trendColor(points: { value: number }[]): string {
  if (points.length < 2) return CHART_COLORS.muted;
  return points[points.length - 1].value >= points[0].value
    ? CHART_COLORS.up
    : CHART_COLORS.down;
}

import { BRAND } from "./seo";

export function SquareMark({ size }: { size: number }) {
  return (
    <div
      style={{
        width: `${size}px`,
        height: `${size}px`,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: BRAND.bg,
        color: BRAND.accent,
        fontFamily: "monospace",
        fontWeight: 700,
        fontSize: Math.round(size * 0.62),
        letterSpacing: "-0.04em",
      }}
    >
      Q
    </div>
  );
}

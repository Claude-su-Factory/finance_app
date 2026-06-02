import { ImageResponse } from "next/og";
import { BRAND, OG_EYEBROW, OG_TAGLINE_EN } from "@/lib/seo";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "Quotient — Portfolio Intelligence Terminal";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "1200px",
          height: "630px",
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          background: BRAND.bg,
          padding: "0 96px",
          fontFamily: "monospace",
        }}
      >
        <div
          style={{
            display: "flex",
            fontSize: 24,
            letterSpacing: "0.28em",
            color: BRAND.muted,
          }}
        >
          {OG_EYEBROW}
        </div>
        <div
          style={{
            display: "flex",
            fontSize: 140,
            fontWeight: 700,
            color: BRAND.accent,
            letterSpacing: "-0.02em",
            marginTop: 12,
          }}
        >
          QUOTIENT
        </div>
        <div
          style={{
            display: "flex",
            fontSize: 30,
            color: BRAND.fg,
            marginTop: 20,
          }}
        >
          {OG_TAGLINE_EN}
        </div>
        <div style={{ display: "flex", gap: 14, marginTop: 40 }}>
          <div
            style={{ width: 72, height: 10, background: BRAND.accent }}
          />
          <div style={{ width: 72, height: 10, background: BRAND.info }} />
          <div style={{ width: 72, height: 10, background: BRAND.up }} />
        </div>
      </div>
    ),
    { ...size },
  );
}

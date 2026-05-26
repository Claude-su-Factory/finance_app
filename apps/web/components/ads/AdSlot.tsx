"use client";

const ADS_ENABLED = process.env.NEXT_PUBLIC_ENABLE_ADS === "true";

// AdSlot은 의미 슬롯명을 받고, 비활성 상태면 자체 placeholder를 표시.
// Phase 2에서 AdSense 가입 후 slot → data-ad-slot 매핑 추가.
export function AdSlot({
  slot,
  height = 90,
  label,
}: {
  slot: string;
  height?: number;
  label?: string;
}) {
  if (!ADS_ENABLED) {
    return (
      <div
        className="border border-dashed border-line/50 flex items-center justify-center text-fg-muted/60 font-mono text-xs"
        style={{ height }}
        data-ad-slot={slot}
      >
        ADS_DISABLED · {label ?? slot}
      </div>
    );
  }
  return (
    <div className="border border-dashed border-line/50" style={{ height }} data-ad-slot={slot} />
  );
}

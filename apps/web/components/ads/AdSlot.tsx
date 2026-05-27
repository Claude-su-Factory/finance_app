"use client";

import { useEffect } from "react";

const ADS_ENABLED = process.env.NEXT_PUBLIC_ENABLE_ADS === "true";
const ADSENSE_CLIENT = process.env.NEXT_PUBLIC_ADSENSE_CLIENT; // 예: ca-pub-1234567890123456

// AdSense slot 매핑 — slot 이름을 env로 매핑.
// 가입 후 본인 slot ID를 env에 채우면 활성, 미설정이면 placeholder.
const SLOT_IDS: Record<string, string | undefined> = {
  market_bottom: process.env.NEXT_PUBLIC_ADSENSE_SLOT_MARKET_BOTTOM,
};

declare global {
  interface Window {
    adsbygoogle?: unknown[];
  }
}

export function AdSlot({
  slot,
  height = 90,
  label,
}: {
  slot: string;
  height?: number;
  label?: string;
}) {
  const slotId = SLOT_IDS[slot];
  const live = ADS_ENABLED && ADSENSE_CLIENT && slotId;

  useEffect(() => {
    if (!live) return;
    try {
      (window.adsbygoogle = window.adsbygoogle || []).push({});
    } catch {
      // adsbygoogle 미로드 — script가 아직 안 붙음. 다음 mount에서 재시도.
    }
  }, [live]);

  if (!live) {
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
    <ins
      className="adsbygoogle block"
      style={{ display: "block", height }}
      data-ad-client={ADSENSE_CLIENT}
      data-ad-slot={slotId}
      data-ad-format="auto"
      data-full-width-responsive="true"
    />
  );
}

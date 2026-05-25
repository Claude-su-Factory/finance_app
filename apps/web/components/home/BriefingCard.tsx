"use client";

import { useEffect, useState } from "react";
import { getTodayBriefing, type Briefing } from "@/lib/api/briefing";

export function BriefingCard() {
  const [briefing, setBriefing] = useState<Briefing | null | undefined>(undefined);
  useEffect(() => { getTodayBriefing().then(setBriefing); }, []);

  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">오늘 브리핑</div>
      {briefing === undefined ? (
        <div className="text-fg-muted text-sm font-mono">로드 중…</div>
      ) : briefing === null ? (
        <div className="text-fg-muted text-sm font-mono">
          오늘 브리핑은 아직 준비 중입니다. 매일 07:00 KST 무렵 생성됩니다.
        </div>
      ) : (
        <div className="font-mono text-sm whitespace-pre-wrap leading-relaxed">
          {briefing.content_md}
        </div>
      )}
    </div>
  );
}

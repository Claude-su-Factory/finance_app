"use client";

import { useEffect, useState } from "react";
import { getUsage, type Usage } from "@/lib/api/usage";

export function UsageBadge({ refreshKey }: { refreshKey: number }) {
  const [usage, setUsage] = useState<Usage | null>(null);
  useEffect(() => { getUsage().then(setUsage).catch(() => {}); }, [refreshKey]);
  if (!usage) return null;
  const { chat_count, opus_count } = usage.usage;
  return (
    <div className="font-mono text-xs text-fg-muted">
      <span>{chat_count}/{usage.limits.chat}회</span>
      <span className="mx-2">·</span>
      <span>Opus {opus_count}/{usage.limits.opus}</span>
    </div>
  );
}

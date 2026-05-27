"use client";

import { usePathname } from "next/navigation";
import { useEffect } from "react";
import { capture, initPostHog } from "@/lib/analytics/posthog";

// 클라이언트 부팅 시 1회 init + 라우트 변경마다 $pageview emit.
// DSN/Key 미설정이면 모든 호출 no-op.
export function PostHogProvider({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  useEffect(() => {
    initPostHog();
  }, []);

  useEffect(() => {
    if (!pathname) return;
    capture("$pageview", { $current_url: window.location.href });
  }, [pathname]);

  return <>{children}</>;
}

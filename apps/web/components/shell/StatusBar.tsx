"use client";
import { useEffect, useState } from "react";

export function StatusBar() {
  const [now, setNow] = useState<Date | null>(null);
  useEffect(() => {
    setNow(new Date());
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);
  // SSR 시 빈 자리 차지하고 클라이언트에서 채움 (hydration mismatch 방지)
  const time = now ? now.toLocaleTimeString("ko-KR", { hour12: false, timeZone: "Asia/Seoul" }) : "--:--:--";
  return (
    <footer className="h-6 border-t border-line bg-bg flex items-center px-4 gap-4 text-[10px] font-mono text-fg-muted">
      <span>↑ {time} KST</span>
      <span className="text-bb-up">● API</span>
      <span className="ml-auto">⌘K 명령 팔레트</span>
    </footer>
  );
}

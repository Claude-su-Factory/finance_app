"use client";

import { useState } from "react";
import { SessionList } from "./SessionList";

export function ChatPage({ sessionId }: { sessionId: string | null }) {
  const [refreshKey, setRefreshKey] = useState(0);

  return (
    <div className="flex h-[calc(100vh-2.25rem-1.5rem)]">
      <aside className="w-64 border-r border-line overflow-y-auto">
        <SessionList currentId={sessionId} refreshKey={refreshKey} />
      </aside>
      <main className="flex-1 flex items-center justify-center text-fg-muted font-mono text-sm">
        {sessionId ? `세션 ${sessionId} (메시지·입력 UI는 W4-T14)` : "새 대화를 시작하세요"}
        <button
          onClick={() => setRefreshKey((k) => k + 1)}
          className="ml-4 underline text-xs"
        >
          세션 새로고침
        </button>
      </main>
    </div>
  );
}

"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { listSessions, deleteSession, type ChatSession } from "@/lib/api/chat-sessions";

export function SessionList({
  currentId,
  refreshKey,
}: {
  currentId: string | null;
  refreshKey: number;
}) {
  const [sessions, setSessions] = useState<ChatSession[] | null>(null);
  const router = useRouter();

  useEffect(() => {
    listSessions().then(setSessions).catch(() => setSessions([]));
  }, [refreshKey]);

  if (sessions === null) {
    return <div className="p-3 text-xs text-fg-muted font-mono">로드 중…</div>;
  }

  return (
    <div className="p-2">
      <Link
        href="/app/chat"
        className="block px-3 py-2 mb-2 text-sm font-mono border border-line hover:bg-bg-deep text-center"
      >
        + 새 대화
      </Link>
      {sessions.length === 0 ? (
        <p className="px-3 py-2 text-xs text-fg-muted font-mono">아직 대화 없음</p>
      ) : (
        <ul className="space-y-1">
          {sessions.map((s) => (
            <li key={s.id} className="group flex items-center">
              <Link
                href={`/app/chat/${s.id}`}
                className={`flex-1 px-3 py-2 text-sm font-mono truncate ${
                  s.id === currentId ? "bg-bg-deep text-bb-accent" : "hover:bg-bg-deep/50"
                }`}
              >
                {s.title || "(제목 없음)"}
              </Link>
              <button
                onClick={async () => {
                  if (confirm("삭제하시겠습니까?")) {
                    await deleteSession(s.id);
                    if (s.id === currentId) router.push("/app/chat");
                    else setSessions((prev) => prev?.filter((x) => x.id !== s.id) ?? null);
                  }
                }}
                className="opacity-0 group-hover:opacity-100 px-2 text-xs text-bb-down hover:opacity-100"
                title="세션 삭제"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

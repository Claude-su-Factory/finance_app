"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { SessionList } from "./SessionList";
import { MessageList } from "./MessageList";
import { InputBox } from "./InputBox";
import { UsageBadge } from "./UsageBadge";
import { streamChat } from "@/lib/api/chat";
import { listMessages, getUnfinished, type ChatMessage } from "@/lib/api/chat-sessions";
import { ResumePrompt } from "./ResumePrompt";
import type { ToolEvent } from "./StreamingMessage";

export function ChatPage({ sessionId }: { sessionId: string | null }) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [streamingText, setStreamingText] = useState("");
  const [toolEvents, setToolEvents] = useState<ToolEvent[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [sessionsKey, setSessionsKey] = useState(0);
  const [usageKey, setUsageKey] = useState(0);
  const [unfinished, setUnfinished] = useState<ChatMessage | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const router = useRouter();

  useEffect(() => {
    if (sessionId) {
      listMessages(sessionId).then(setMessages).catch(() => setMessages([]));
      getUnfinished(sessionId).then(setUnfinished).catch(() => setUnfinished(null));
    } else {
      setMessages([]);
      setUnfinished(null);
    }
    setStreamingText("");
    setToolEvents([]);
  }, [sessionId]);

  useEffect(() => () => abortRef.current?.abort(), []);

  async function handleSend(message: string, useOpus: boolean) {
    setIsStreaming(true);
    setStreamingText("");
    setToolEvents([]);
    abortRef.current = new AbortController();

    // 임시 user 메시지 표시
    const tempUser: ChatMessage = {
      id: `tmp-${Date.now()}`,
      session_id: sessionId ?? "",
      role: "user",
      content: message,
      input_tokens: 0,
      output_tokens: 0,
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, tempUser]);

    try {
      let returnedSessionId: string | undefined;
      for await (const ev of streamChat(
        { session_id: sessionId ?? undefined, message, use_opus: useOpus },
        abortRef.current.signal,
      )) {
        if (ev.type === "token") {
          setStreamingText((p) => p + ev.data.text);
        } else if (ev.type === "tool_call") {
          setToolEvents((prev) => [...prev, { id: ev.data.id, name: ev.data.name, status: "running" }]);
        } else if (ev.type === "tool_result") {
          setToolEvents((prev) => prev.map((t) => t.id === ev.data.id ? { ...t, status: "done" } : t));
        } else if (ev.type === "done") {
          returnedSessionId = ev.data.session_id;
          if (sessionId) {
            const fresh = await listMessages(sessionId);
            setMessages(fresh);
          }
        } else if (ev.type === "error") {
          alert(`에러: ${ev.data.message}`);
        }
      }
      // 새 세션이면 backend가 반환한 ID로 직접 라우팅 (race-free)
      if (!sessionId && returnedSessionId) {
        setSessionsKey((k) => k + 1);
        router.push(`/app/chat/${returnedSessionId}`);
      }
    } catch (e: unknown) {
      if ((e as { name?: string })?.name === "AbortError") {
        // 사용자 cancel
      } else {
        console.error("chat error", e);
      }
    } finally {
      setIsStreaming(false);
      setStreamingText("");
      setToolEvents([]);
      setUsageKey((k) => k + 1);
    }
  }

  return (
    <div className="flex h-[calc(100vh-2.25rem-1.5rem)]">
      <aside className="w-64 border-r border-line overflow-y-auto">
        <SessionList currentId={sessionId} refreshKey={sessionsKey} />
      </aside>
      <main className="flex-1 flex flex-col">
        <header className="border-b border-line px-4 py-2 flex items-center justify-between">
          <h1 className="font-mono text-sm">AI 분석가</h1>
          <UsageBadge refreshKey={usageKey} />
        </header>
        {unfinished && (
          <ResumePrompt
            partial={unfinished.content}
            onResume={() => {
              const lastUser = [...messages].reverse().find((m) => m.role === "user");
              if (lastUser) handleSend(lastUser.content, false);
              setUnfinished(null);
            }}
            onDismiss={() => setUnfinished(null)}
          />
        )}
        <MessageList
          messages={messages}
          streamingText={streamingText}
          toolEvents={toolEvents}
          isStreaming={isStreaming}
        />
        <InputBox onSend={handleSend} disabled={isStreaming} />
      </main>
    </div>
  );
}

"use client";

import { useEffect, useRef } from "react";
import { MessageItem } from "./MessageItem";
import { StreamingMessage, type ToolEvent } from "./StreamingMessage";
import type { ChatMessage } from "@/lib/api/chat-sessions";

export function MessageList({
  messages,
  streamingText,
  toolEvents,
  isStreaming,
}: {
  messages: ChatMessage[];
  streamingText: string;
  toolEvents: ToolEvent[];
  isStreaming: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    ref.current?.scrollTo({ top: ref.current.scrollHeight });
  }, [messages.length, streamingText, toolEvents.length]);

  return (
    <div ref={ref} className="flex-1 overflow-y-auto p-4">
      {messages.map((m) => <MessageItem key={m.id} message={m} />)}
      {isStreaming && <StreamingMessage text={streamingText} toolEvents={toolEvents} />}
    </div>
  );
}

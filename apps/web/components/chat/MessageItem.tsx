"use client";

import { ToolStep } from "./ToolStep";
import type { ChatMessage } from "@/lib/api/chat-sessions";

export function MessageItem({ message }: { message: ChatMessage }) {
  const isUser = message.role === "user";
  return (
    <div className={`flex ${isUser ? "justify-end" : "justify-start"} mb-3`}>
      <div className={`max-w-[80%] px-4 py-2 font-mono text-sm whitespace-pre-wrap ${isUser ? "bg-bb-accent/10 border border-bb-accent/30" : "border border-line"}`}>
        {message.content}
        {message.tool_calls != null && (
          <div className="mt-2 pt-2 border-t border-line/50 space-y-1">
            <ToolStep name="(이전 도구 호출)" status="done" />
          </div>
        )}
      </div>
    </div>
  );
}

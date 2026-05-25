"use client";

import { ToolStep } from "./ToolStep";

export type ToolEvent = { id: string; name: string; status: "running" | "done" | "error" };

export function StreamingMessage({
  text,
  toolEvents,
}: {
  text: string;
  toolEvents: ToolEvent[];
}) {
  return (
    <div className="flex justify-start mb-3">
      <div className="max-w-[80%] px-4 py-2 border border-line font-mono text-sm whitespace-pre-wrap">
        {toolEvents.length > 0 && (
          <div className="mb-2 space-y-1">
            {toolEvents.map((t) => (
              <ToolStep key={t.id} name={t.name} status={t.status} />
            ))}
          </div>
        )}
        {text}
        <span className="inline-block w-2 h-4 ml-1 bg-bb-accent align-middle animate-pulse" />
      </div>
    </div>
  );
}

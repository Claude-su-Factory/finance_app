"use client";

import { useEffect, useState } from "react";

export function ToolStep({
  name,
  status,
}: {
  name: string;
  status: "running" | "done" | "error";
}) {
  const [elapsed, setElapsed] = useState(0);
  useEffect(() => {
    if (status !== "running") return;
    const start = performance.now();
    const t = setInterval(() => setElapsed((performance.now() - start) / 1000), 100);
    return () => clearInterval(t);
  }, [status]);

  const color =
    status === "running" ? "text-bb-accent" : status === "done" ? "text-bb-up" : "text-bb-down";
  const label = status === "running" ? "RUNNING" : status === "done" ? "DONE" : "ERROR";
  return (
    <div className={`font-mono text-xs ${color}`}>
      ▸ {name}() {label} {elapsed > 0 ? `${elapsed.toFixed(2)}s` : ""}
    </div>
  );
}

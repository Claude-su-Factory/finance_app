"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";

export function InputBox({
  onSend,
  disabled,
}: {
  onSend: (message: string, useOpus: boolean) => void;
  disabled: boolean;
}) {
  const [text, setText] = useState("");
  const [opus, setOpus] = useState(false);

  function submit() {
    if (!text.trim() || disabled) return;
    onSend(text.trim(), opus);
    setText("");
  }

  return (
    <div className="border-t border-line p-3">
      <div className="flex gap-2">
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              submit();
            }
          }}
          placeholder={disabled ? "응답 받는 중…" : "메시지 (Shift+Enter 줄바꿈)"}
          className="flex-1 bg-bg-deep border border-line px-3 py-2 font-mono text-sm resize-none"
          rows={2}
          disabled={disabled}
        />
        <div className="flex flex-col gap-2">
          <label className="flex items-center gap-1 text-xs font-mono cursor-pointer">
            <input
              type="checkbox"
              checked={opus}
              onChange={(e) => setOpus(e.target.checked)}
              disabled={disabled}
            />
            심층 (Opus)
          </label>
          <Button onClick={submit} disabled={disabled || !text.trim()}>
            보내기
          </Button>
        </div>
      </div>
    </div>
  );
}

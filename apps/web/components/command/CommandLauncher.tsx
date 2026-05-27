"use client";

import { useState } from "react";
import { CommandPalette } from "./CommandPalette";
import { useKeyboardShortcuts } from "./useKeyboardShortcuts";

// 전역 명령 팔레트 + 단축키 호스트.
// AppShell 안 한 번만 마운트되어 ⌘K · / · 1~5 · g h/p/c/m/s 처리.
export function CommandLauncher() {
  const [open, setOpen] = useState(false);
  useKeyboardShortcuts({ onOpenPalette: () => setOpen(true) });
  return <CommandPalette open={open} onOpenChange={setOpen} />;
}

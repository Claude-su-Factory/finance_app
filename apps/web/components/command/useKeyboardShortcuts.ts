"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

const NAV_BY_KEY: Record<string, string> = {
  "1": "/app",
  "2": "/app/portfolio",
  "3": "/app/chat",
  "4": "/app/market",
  "5": "/app/settings",
};

// vim-like g-prefix chord: g → h(ome), p(ortfolio), c(hat), m(arket), s(ettings)
const CHORD_NAV: Record<string, string> = {
  h: "/app",
  p: "/app/portfolio",
  c: "/app/chat",
  m: "/app/market",
  s: "/app/settings",
};

// 입력 중에는 단축키 무시 — 텍스트 필드/contentEditable 안에서는 비활성
function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return true;
  if (target.isContentEditable) return true;
  return false;
}

export function useKeyboardShortcuts({
  onOpenPalette,
}: {
  onOpenPalette: () => void;
}) {
  const router = useRouter();

  useEffect(() => {
    let chordPending: string | null = null;
    let chordTimer: ReturnType<typeof setTimeout> | null = null;

    function clearChord() {
      chordPending = null;
      if (chordTimer) {
        clearTimeout(chordTimer);
        chordTimer = null;
      }
    }

    function handler(e: KeyboardEvent) {
      // ⌘K / Ctrl+K — 어디서나 작동 (입력 중에도)
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        clearChord();
        onOpenPalette();
        return;
      }

      // / — 명령 팔레트 열기 (입력 중이 아닐 때만)
      if (e.key === "/" && !e.metaKey && !e.ctrlKey && !isEditableTarget(e.target)) {
        e.preventDefault();
        clearChord();
        onOpenPalette();
        return;
      }

      // 입력 중일 때는 chord/숫자 단축키 비활성
      if (isEditableTarget(e.target) || e.metaKey || e.ctrlKey || e.altKey) return;

      // 숫자 1~5 — 탭 이동
      if (NAV_BY_KEY[e.key]) {
        e.preventDefault();
        clearChord();
        router.push(NAV_BY_KEY[e.key]);
        return;
      }

      // g chord (vim-like)
      if (e.key === "g") {
        chordPending = "g";
        clearTimeout(chordTimer!);
        chordTimer = setTimeout(() => { chordPending = null; }, 1000);
        return;
      }
      if (chordPending === "g" && CHORD_NAV[e.key]) {
        e.preventDefault();
        router.push(CHORD_NAV[e.key]);
        clearChord();
        return;
      }
      // chord 외 키 입력 → chord 취소
      if (chordPending) clearChord();
    }

    window.addEventListener("keydown", handler);
    return () => {
      window.removeEventListener("keydown", handler);
      clearChord();
    };
  }, [router, onOpenPalette]);
}

"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Command } from "cmdk";
import { searchInstruments, type InstrumentResult } from "@/lib/api/instruments";

const NAV_ITEMS = [
  { key: "1", label: "홈", href: "/app" },
  { key: "2", label: "포트폴리오", href: "/app/portfolio" },
  { key: "3", label: "채팅 (AI 분석가)", href: "/app/chat" },
  { key: "4", label: "마켓", href: "/app/market" },
  { key: "5", label: "설정", href: "/app/settings" },
] as const;

export function CommandPalette({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [instruments, setInstruments] = useState<InstrumentResult[]>([]);

  // 검색 디바운스
  useEffect(() => {
    if (!open) return;
    if (!query.trim()) {
      setInstruments([]);
      return;
    }
    const t = setTimeout(async () => {
      try {
        const r = await searchInstruments(query);
        setInstruments(r);
      } catch {
        setInstruments([]);
      }
    }, 200);
    return () => clearTimeout(t);
  }, [query, open]);

  // 모달 닫힐 때 입력 초기화
  useEffect(() => {
    if (!open) {
      setQuery("");
      setInstruments([]);
    }
  }, [open]);

  const go = useCallback((href: string) => {
    onOpenChange(false);
    router.push(href);
  }, [onOpenChange, router]);

  const askAI = useCallback(() => {
    onOpenChange(false);
    // 입력 prefill을 sessionStorage로 전달 — 채팅 페이지가 mount 시 읽어서 input box에 채움 (W4 후속 backlog)
    if (query.trim()) {
      sessionStorage.setItem("chat_prefill", query.trim());
    }
    router.push("/app/chat");
  }, [onOpenChange, query, router]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/40"
      onClick={() => onOpenChange(false)}
    >
      <div onClick={(e) => e.stopPropagation()}>
        <Command
          className="w-[640px] max-w-[90vw] border border-line bg-bg-deep shadow-lg font-mono text-sm"
          shouldFilter={false}
          loop
        >
          <Command.Input
            placeholder="명령·종목 검색 또는 'AI에게 묻기'..."
            value={query}
            onValueChange={setQuery}
            className="w-full px-4 py-3 bg-transparent border-b border-line outline-none placeholder:text-fg-muted/60"
            autoFocus
          />
          <Command.List className="max-h-[400px] overflow-y-auto p-2">
            <Command.Empty className="px-3 py-4 text-xs text-fg-muted">결과 없음</Command.Empty>

            {/* AI 질문 — 입력이 있을 때만 노출 */}
            {query.trim() && (
              <Command.Group heading="AI 분석가" className="text-[10px] text-fg-muted px-2 pt-1 pb-1 uppercase">
                <Command.Item
                  value={`ai-ask-${query}`}
                  onSelect={askAI}
                  className="flex items-center gap-2 px-3 py-2 cursor-pointer rounded data-[selected=true]:bg-bb-accent/10 data-[selected=true]:text-bb-accent"
                >
                  <span className="text-bb-accent">▸</span>
                  <span className="truncate">"{query}" — AI에게 묻기</span>
                </Command.Item>
              </Command.Group>
            )}

            {/* 종목 검색 결과 */}
            {instruments.length > 0 && (
              <Command.Group heading="종목" className="text-[10px] text-fg-muted px-2 pt-2 pb-1 uppercase">
                {instruments.map((inst) => (
                  <Command.Item
                    key={inst.id}
                    value={`inst-${inst.id}`}
                    onSelect={() => {
                      // 종목 선택 시 마켓 탭으로 이동 (관심 종목 추가는 사용자가 직접)
                      go("/app/market");
                    }}
                    className="flex items-center gap-2 px-3 py-2 cursor-pointer rounded data-[selected=true]:bg-bb-accent/10 data-[selected=true]:text-bb-accent"
                  >
                    <span className="tabular-nums">{inst.symbol}</span>
                    <span className="text-fg-muted text-xs">{inst.exchange}</span>
                    <span className="text-fg-muted text-xs ml-auto truncate">{inst.name}</span>
                  </Command.Item>
                ))}
              </Command.Group>
            )}

            {/* 탭 이동 (항상 노출) */}
            <Command.Group heading="이동" className="text-[10px] text-fg-muted px-2 pt-2 pb-1 uppercase">
              {NAV_ITEMS.map((item) => (
                <Command.Item
                  key={item.href}
                  value={`nav-${item.key}-${item.label}`}
                  onSelect={() => go(item.href)}
                  className="flex items-center gap-3 px-3 py-2 cursor-pointer rounded data-[selected=true]:bg-bb-accent/10 data-[selected=true]:text-bb-accent"
                >
                  <kbd className="px-1.5 py-0.5 text-[10px] border border-line text-fg-muted">{item.key}</kbd>
                  <span>{item.label}</span>
                </Command.Item>
              ))}
            </Command.Group>
          </Command.List>

          <div className="border-t border-line px-3 py-1.5 flex items-center justify-between text-[10px] text-fg-muted">
            <span>↑↓ 탐색 · ⏎ 선택 · ESC 닫기</span>
            <span>1~5 탭 이동 · / 검색 · c 채팅</span>
          </div>
        </Command>
      </div>
    </div>
  );
}

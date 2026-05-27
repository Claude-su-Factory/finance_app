"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Home, Wallet, MessageSquare, BarChart3, Settings, Heart } from "lucide-react";
import { clsx } from "clsx";

const items = [
  { href: "/app", icon: Home, label: "홈" },
  { href: "/app/portfolio", icon: Wallet, label: "포트폴리오" },
  { href: "/app/chat", icon: MessageSquare, label: "채팅" },
  { href: "/app/market", icon: BarChart3, label: "마켓" },
  { href: "/app/settings", icon: Settings, label: "설정" },
];

// Toss 송금 링크 — 미설정 시 후원 아이콘 표시 안 함.
// 예: NEXT_PUBLIC_TOSS_DONATION_URL=https://toss.me/your-nickname
const TOSS_URL = process.env.NEXT_PUBLIC_TOSS_DONATION_URL;

export function Sidebar() {
  const path = usePathname();
  return (
    <aside className="w-14 border-r border-line bg-bg flex flex-col items-center py-3 gap-1">
      <div className="font-mono text-bb-accent text-[10px] mb-3">Q</div>
      {items.map(({ href, icon: Icon, label }) => {
        const active = path === href;
        return (
          <Link
            key={href}
            href={href}
            className={clsx(
              "w-10 h-10 flex items-center justify-center transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent",
              active ? "text-bb-accent" : "text-fg-muted hover:text-fg"
            )}
            title={label}
            aria-label={label}
          >
            <Icon size={18} strokeWidth={1.5} />
          </Link>
        );
      })}
      {TOSS_URL ? (
        <a
          href={TOSS_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-auto w-10 h-10 flex items-center justify-center text-fg-muted/60 hover:text-bb-warn transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent"
          title="후원하기 (Toss)"
          aria-label="후원하기"
        >
          <Heart size={16} strokeWidth={1.5} />
        </a>
      ) : null}
    </aside>
  );
}

// 인증·온보딩 redirect는 proxy.ts(미들웨어)가 담당 (deep link 우회 방지)
// 여기서는 셸만 렌더. user 확인은 미들웨어가 이미 수행.
import { AppShell } from "@/components/shell/AppShell";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return <AppShell>{children}</AppShell>;
}

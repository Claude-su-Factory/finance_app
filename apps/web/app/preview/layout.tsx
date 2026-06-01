import { connection } from "next/server";
import { notFound } from "next/navigation";
import { AppShell } from "@/components/shell/AppShell";
import { PreviewSwitcher } from "@/components/preview/PreviewSwitcher";

// ENABLE_PREVIEW=1 일 때만 노출되는 개발 전용 미리보기. 프로덕션엔 env 없어 404.
// connection()으로 요청 시점 평가 강제 — 빌드 타임 프리렌더로 가드가 박제되는 것 방지.
export default async function PreviewLayout({ children }: { children: React.ReactNode }) {
  await connection();
  if (process.env.ENABLE_PREVIEW !== "1") notFound();
  return (
    <AppShell>
      {children}
      <PreviewSwitcher />
    </AppShell>
  );
}

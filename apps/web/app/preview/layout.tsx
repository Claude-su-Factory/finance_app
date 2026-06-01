import { notFound } from "next/navigation";
import { AppShell } from "@/components/shell/AppShell";
import { PreviewSwitcher } from "@/components/preview/PreviewSwitcher";

// ENABLE_PREVIEW=1 일 때만 노출되는 개발 전용 미리보기. 프로덕션엔 env 없어 404.
export default function PreviewLayout({ children }: { children: React.ReactNode }) {
  if (process.env.ENABLE_PREVIEW !== "1") notFound();
  return (
    <AppShell>
      {children}
      <PreviewSwitcher />
    </AppShell>
  );
}

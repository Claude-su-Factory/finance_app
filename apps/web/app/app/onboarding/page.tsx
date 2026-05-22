// 인증·redirect는 proxy.ts가 처리. 여기서는 wizard만 렌더.
// 단, 이미 온보딩 완료한 사용자가 직접 URL 접근하면 /app으로 보내야 함 — proxy는 /app/onboarding 자체는 통과시키므로 여기서 처리.
import { redirect } from "next/navigation";
import { createSupabaseServer } from "@/lib/supabase/server";
import { Wizard } from "@/components/onboarding/Wizard";

export const metadata = { title: "온보딩 — Quotient" };

export default async function OnboardingPage() {
  const supabase = await createSupabaseServer();
  const { data: { user } } = await supabase.auth.getUser();
  if (!user) redirect("/login");

  const { data: profile } = await supabase
    .from("profiles")
    .select("onboarding_completed")
    .eq("id", user.id)
    .single();

  if (profile?.onboarding_completed) redirect("/app");

  return <Wizard />;
}

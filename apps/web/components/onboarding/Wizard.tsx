"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { createSupabaseBrowser } from "@/lib/supabase/client";
import { StepIndicator } from "./StepIndicator";
import { CurrencyStep } from "./CurrencyStep";
import { DemoOrStartStep } from "./DemoOrStartStep";

export function Wizard() {
  const [step, setStep] = useState(1);
  const [currency, setCurrency] = useState<"KRW" | "USD">("KRW");
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const router = useRouter();

  async function complete(demo: boolean) {
    setLoading(true);
    setErr(null);
    const supabase = createSupabaseBrowser();
    const { data: { user } } = await supabase.auth.getUser();
    if (!user) { router.push("/login"); return; }

    const { error } = await supabase
      .from("profiles")
      .update({ base_currency: currency, onboarding_completed: true })
      .eq("id", user.id);

    if (error) {
      setErr("프로필 저장에 실패했습니다. 잠시 후 다시 시도해주세요.");
      setLoading(false);
      return;
    }

    // demo seeding은 W3에서 holdings 구현 후 추가. 현재는 flag만 통과.
    void demo;

    router.push("/app");
    router.refresh();
  }

  return (
    <main className="min-h-screen flex flex-col">
      <div className="border-b border-line p-4 font-mono text-xs text-fg-muted">
        ONBOARDING — {step}/2
      </div>
      <StepIndicator current={step} total={2} />
      <div className="flex-1 flex items-center justify-center px-6">
        <div className="w-full max-w-md">
          {step === 1 && (
            <CurrencyStep
              value={currency}
              onChange={setCurrency}
              onNext={() => setStep(2)}
            />
          )}
          {step === 2 && (
            <DemoOrStartStep
              onDemo={() => complete(true)}
              onStart={() => complete(false)}
              loading={loading}
            />
          )}
          {err && <p className="text-bb-down text-xs mt-4 font-mono">{err}</p>}
        </div>
      </div>
    </main>
  );
}

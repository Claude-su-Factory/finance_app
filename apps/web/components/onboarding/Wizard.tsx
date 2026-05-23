"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { createSupabaseBrowser } from "@/lib/supabase/client";
import { createHolding } from "@/lib/api/holdings";
import { StepIndicator } from "./StepIndicator";
import { CurrencyStep } from "./CurrencyStep";
import { HoldingsStep, type DraftHolding } from "./HoldingsStep";
import { DemoOrStartStep } from "./DemoOrStartStep";

export function Wizard() {
  const [step, setStep] = useState(1);
  const [currency, setCurrency] = useState<"KRW" | "USD">("KRW");
  const [drafts, setDrafts] = useState<DraftHolding[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const router = useRouter();

  async function complete(demo: boolean) {
    setLoading(true);
    setErr(null);
    const supabase = createSupabaseBrowser();
    // 세션 + access_token 존재 보장 (Critical 6 — drafts authFetch 401 방지)
    const { data: { session } } = await supabase.auth.getSession();
    if (!session?.user || !session.access_token) {
      router.push("/login");
      return;
    }
    const user = session.user;

    const { error } = await supabase
      .from("profiles")
      .update({ base_currency: currency, onboarding_completed: true })
      .eq("id", user.id);

    if (error) {
      setErr("프로필 저장에 실패했습니다. 잠시 후 다시 시도해주세요.");
      setLoading(false);
      return;
    }

    // drafts 일괄 적재 (부분 실패 허용 + toast 안내)
    const failed: string[] = [];
    for (const d of drafts) {
      try {
        await createHolding({
          instrument_id: d.instrument.id,
          quantity: d.quantity,
          avg_cost: d.avg_cost,
        });
      } catch (e) {
        console.warn("draft holding failed", d.instrument.symbol, e);
        failed.push(d.instrument.symbol);
      }
    }
    if (failed.length > 0) {
      const { toast } = await import("sonner");
      toast.error(`다음 종목 추가에 실패했습니다: ${failed.join(", ")}. 포트폴리오에서 재시도해주세요.`);
    }

    void demo; // demo seeding은 Phase 2

    router.push("/app");
    router.refresh();
  }

  return (
    <main className="min-h-screen flex flex-col">
      <div className="border-b border-line p-4 font-mono text-xs text-fg-muted">
        ONBOARDING — {step}/3
      </div>
      <StepIndicator current={step} total={3} />
      <div className="flex-1 flex items-center justify-center px-6">
        <div className="w-full max-w-lg">
          {step === 1 && (
            <CurrencyStep value={currency} onChange={setCurrency} onNext={() => setStep(2)} />
          )}
          {step === 2 && (
            <HoldingsStep
              value={drafts}
              onChange={setDrafts}
              onNext={() => setStep(3)}
              onSkip={() => { setDrafts([]); setStep(3); }}
            />
          )}
          {step === 3 && (
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

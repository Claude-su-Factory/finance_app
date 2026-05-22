"use client";
import { useState } from "react";
import { AuthCard } from "@/components/auth/AuthCard";
import { createSupabaseBrowser } from "@/lib/supabase/client";
import { useRouter } from "next/navigation";

export default function ResetPasswordPage() {
  const [pw, setPw] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const router = useRouter();

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    if (pw.length < 8) { setErr("최소 8자"); return; }
    setBusy(true);
    const supabase = createSupabaseBrowser();
    const { error } = await supabase.auth.updateUser({ password: pw });
    setBusy(false);
    if (error) { setErr(error.message); return; }
    router.push("/login");
  }

  return (
    <AuthCard title="새 비밀번호 설정">
      <form onSubmit={onSubmit} className="space-y-4">
        <input
          type="password"
          value={pw}
          onChange={(e) => setPw(e.target.value)}
          className="w-full bg-bg border border-line px-3 py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent"
          placeholder="8자 이상"
          autoComplete="new-password"
        />
        {err && <p className="text-bb-down text-xs">{err}</p>}
        <button
          type="submit"
          disabled={busy}
          className="w-full bg-bb-accent text-bg font-mono py-2 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
        >
          {busy ? "변경 중…" : "변경하기"}
        </button>
      </form>
    </AuthCard>
  );
}

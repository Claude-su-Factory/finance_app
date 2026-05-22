"use client";
import { useState } from "react";
import { AuthCard } from "@/components/auth/AuthCard";
import { createSupabaseBrowser } from "@/lib/supabase/client";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    const supabase = createSupabaseBrowser();
    const { error } = await supabase.auth.resetPasswordForEmail(email, {
      redirectTo: `${window.location.origin}/reset-password`,
    });
    setBusy(false);
    if (error) { setErr(error.message); return; }
    setSent(true);
  }

  if (sent) {
    return (
      <AuthCard title="이메일 발송 완료">
        <p className="text-fg-muted text-sm"><span className="font-mono">{email}</span>로 비밀번호 재설정 링크를 보냈습니다.</p>
        <p className="text-fg-muted text-xs">메일이 보이지 않으면 스팸함을 확인해주세요.</p>
      </AuthCard>
    );
  }

  return (
    <AuthCard title="비밀번호 재설정" subtitle="가입한 이메일을 입력해주세요">
      <form onSubmit={onSubmit} className="space-y-4">
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          className="w-full bg-bg border border-line px-3 py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent"
          placeholder="email@example.com"
          autoComplete="email"
        />
        {err && <p className="text-bb-down text-xs">{err}</p>}
        <button
          type="submit"
          disabled={busy}
          className="w-full bg-bb-accent text-bg font-mono py-2 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
        >
          {busy ? "전송 중…" : "재설정 링크 받기"}
        </button>
      </form>
    </AuthCard>
  );
}

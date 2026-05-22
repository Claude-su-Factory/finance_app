"use client";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { createSupabaseBrowser } from "@/lib/supabase/client";

const schema = z.object({
  email: z.string().email("올바른 이메일을 입력해주세요"),
  password: z.string().min(1, "비밀번호를 입력해주세요"),
});
type FormData = z.infer<typeof schema>;

export function LoginForm() {
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({ resolver: zodResolver(schema) });
  const [serverError, setServerError] = useState<string | null>(null);
  const router = useRouter();

  async function onSubmit(data: FormData) {
    setServerError(null);
    const supabase = createSupabaseBrowser();
    const { error } = await supabase.auth.signInWithPassword({ email: data.email, password: data.password });
    // 보안: 어떤 필드가 틀렸는지 노출하지 않음 (계정 열거 공격 방지)
    if (error) { setServerError("이메일 또는 비밀번호가 올바르지 않습니다."); return; }
    router.push("/app");
    router.refresh();
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label className="text-xs text-fg-muted">이메일</label>
        <input {...register("email")} type="email" className="w-full bg-bg border border-line px-3 py-2 mt-1 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent" autoComplete="email" />
        {errors.email && <p className="text-bb-down text-xs mt-1">{errors.email.message}</p>}
      </div>
      <div>
        <label className="text-xs text-fg-muted">비밀번호</label>
        <input {...register("password")} type="password" className="w-full bg-bg border border-line px-3 py-2 mt-1 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent" autoComplete="current-password" />
        {errors.password && <p className="text-bb-down text-xs mt-1">{errors.password.message}</p>}
      </div>
      <Link href="/forgot-password" className="text-xs text-fg-muted underline">비밀번호를 잊으셨나요?</Link>
      {serverError && <p className="text-bb-down text-xs">{serverError}</p>}
      <button type="submit" disabled={isSubmitting} className="w-full bg-bb-accent text-bg font-mono py-2 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg">
        {isSubmitting ? "로그인 중…" : "로그인"}
      </button>
    </form>
  );
}

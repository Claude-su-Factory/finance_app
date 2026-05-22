"use client";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { createSupabaseBrowser } from "@/lib/supabase/client";

// PIPA: 약관·개인정보 처리방침 분리 동의 + 만 14세 이상 확인
const schema = z.object({
  email: z.string().email("올바른 이메일을 입력해주세요"),
  password: z.string().min(8, "비밀번호는 최소 8자입니다"),
  agree_terms: z.literal(true, { message: "서비스 약관에 동의해주세요" }),
  agree_privacy: z.literal(true, { message: "개인정보 처리방침에 동의해주세요" }),
  age_14: z.literal(true, { message: "만 14세 이상이어야 가입할 수 있습니다" }),
});
type FormData = z.infer<typeof schema>;

export function SignupForm() {
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({
    resolver: zodResolver(schema),
  });
  const [serverError, setServerError] = useState<string | null>(null);
  const router = useRouter();

  async function onSubmit(data: FormData) {
    setServerError(null);
    const supabase = createSupabaseBrowser();
    const { error } = await supabase.auth.signUp({
      email: data.email,
      password: data.password,
      options: { emailRedirectTo: `${window.location.origin}/auth/callback` },
    });
    if (error) { setServerError(error.message); return; }
    router.push(`/verify-email?email=${encodeURIComponent(data.email)}`);
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label className="text-xs text-fg-muted">이메일</label>
        <input
          {...register("email")}
          type="email"
          className="w-full bg-bg border border-line px-3 py-2 mt-1 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent"
          autoComplete="email"
        />
        {errors.email && <p className="text-bb-down text-xs mt-1">{errors.email.message}</p>}
      </div>
      <div>
        <label className="text-xs text-fg-muted">비밀번호 (8자 이상)</label>
        <input
          {...register("password")}
          type="password"
          className="w-full bg-bg border border-line px-3 py-2 mt-1 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent"
          autoComplete="new-password"
        />
        {errors.password && <p className="text-bb-down text-xs mt-1">{errors.password.message}</p>}
      </div>

      <div className="space-y-2">
        <label className="flex items-start gap-2 text-xs text-fg-muted">
          <input type="checkbox" {...register("agree_terms")} className="mt-1" />
          <span><Link href="/terms" className="underline">서비스 약관</Link>에 동의합니다.</span>
        </label>
        {errors.agree_terms && <p className="text-bb-down text-xs">{errors.agree_terms.message}</p>}

        <label className="flex items-start gap-2 text-xs text-fg-muted">
          <input type="checkbox" {...register("agree_privacy")} className="mt-1" />
          <span><Link href="/privacy" className="underline">개인정보 처리방침</Link>에 동의합니다.</span>
        </label>
        {errors.agree_privacy && <p className="text-bb-down text-xs">{errors.agree_privacy.message}</p>}

        <label className="flex items-start gap-2 text-xs text-fg-muted">
          <input type="checkbox" {...register("age_14")} className="mt-1" />
          <span>만 14세 이상입니다.</span>
        </label>
        {errors.age_14 && <p className="text-bb-down text-xs">{errors.age_14.message}</p>}
      </div>

      {serverError && <p className="text-bb-down text-xs">{serverError}</p>}
      <button
        type="submit"
        disabled={isSubmitting}
        className="w-full bg-bb-accent text-bg font-mono py-2 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
      >
        {isSubmitting ? "가입 처리 중…" : "가입하기"}
      </button>
    </form>
  );
}

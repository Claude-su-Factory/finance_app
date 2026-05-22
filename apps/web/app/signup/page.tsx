import type { Metadata } from "next";
import Link from "next/link";
import { AuthCard } from "@/components/auth/AuthCard";
import { SignupForm } from "@/components/auth/SignupForm";
import { GoogleButton } from "@/components/auth/GoogleButton";

export const metadata: Metadata = { title: "가입 — Quotient" };

export default function SignupPage() {
  return (
    <AuthCard title="가입" subtitle="Portfolio Intelligence Terminal">
      <SignupForm />
      <div className="text-center text-fg-muted text-xs">또는</div>
      <GoogleButton />
      <p className="text-center text-xs text-fg-muted">
        이미 계정이 있으세요? <Link href="/login" className="underline">로그인</Link>
      </p>
    </AuthCard>
  );
}

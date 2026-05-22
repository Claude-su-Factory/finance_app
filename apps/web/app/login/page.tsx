import type { Metadata } from "next";
import Link from "next/link";
import { AuthCard } from "@/components/auth/AuthCard";
import { LoginForm } from "@/components/auth/LoginForm";
import { GoogleButton } from "@/components/auth/GoogleButton";

export const metadata: Metadata = { title: "로그인 — Quotient" };

export default function LoginPage() {
  return (
    <AuthCard title="로그인">
      <LoginForm />
      <div className="text-center text-fg-muted text-xs">또는</div>
      <GoogleButton />
      <p className="text-center text-xs text-fg-muted">
        계정이 없으세요? <Link href="/signup" className="underline">가입</Link>
      </p>
    </AuthCard>
  );
}

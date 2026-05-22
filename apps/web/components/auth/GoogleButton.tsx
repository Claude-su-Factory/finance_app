"use client";
import { createSupabaseBrowser } from "@/lib/supabase/client";

export function GoogleButton() {
  async function onClick() {
    const supabase = createSupabaseBrowser();
    await supabase.auth.signInWithOAuth({
      provider: "google",
      options: { redirectTo: `${window.location.origin}/auth/callback` },
    });
  }
  return (
    <button onClick={onClick} className="w-full border border-line py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-bb-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg">
      Google로 계속하기
    </button>
  );
}

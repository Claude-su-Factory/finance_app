import { createSupabaseServer } from "@/lib/supabase/server";

export default async function HomePage() {
  const supabase = await createSupabaseServer();
  const { data: { user } } = await supabase.auth.getUser();

  const { data: profile } = await supabase
    .from("profiles")
    .select("display_name")
    .eq("id", user!.id)
    .single();

  return (
    <div className="p-8">
      <h1 className="font-mono text-2xl mb-2">홈</h1>
      <p className="text-fg-muted">환영합니다{profile?.display_name ? `, ${profile.display_name}` : ""}. 대시보드는 W3에서 구현 예정.</p>
    </div>
  );
}

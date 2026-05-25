import { authFetch } from "./auth-fetch";

export type Briefing = {
  user_id: string;
  date: string;
  content_md: string;
  model: string;
  created_at: string;
};

export async function getTodayBriefing(): Promise<Briefing | null> {
  const res = await authFetch("/v1/briefings/today");
  if (res.status === 404) return null;
  if (!res.ok) return null;
  return res.json();
}

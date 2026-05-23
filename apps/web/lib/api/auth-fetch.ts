import { createSupabaseBrowser } from "@/lib/supabase/client";

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function getToken(): Promise<string | null> {
  const supabase = createSupabaseBrowser();
  const {
    data: { session },
  } = await supabase.auth.getSession();
  return session?.access_token ?? null;
}

export async function authFetch(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const token = await getToken();
  const headers = new Headers(init.headers);
  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(`${API_BASE}${path}`, { ...init, headers, cache: "no-store" });
}

export type ApiError = { code: string; message: string };

export async function readError(res: Response): Promise<ApiError> {
  try {
    const body = await res.json();
    if (body?.error) return body.error as ApiError;
    return { code: `HTTP_${res.status}`, message: res.statusText };
  } catch {
    return { code: `HTTP_${res.status}`, message: res.statusText };
  }
}

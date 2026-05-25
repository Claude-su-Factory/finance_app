import { readError } from "./auth-fetch";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type ChatEvent =
  | { type: "token"; data: { text: string } }
  | { type: "tool_call"; data: { id: string; name: string; input: Record<string, unknown> } }
  | { type: "tool_result"; data: { id: string; name: string; result: string } }
  | { type: "done"; data: { session_id: string; message_id: string; input_tokens: number; output_tokens: number } }
  | { type: "error"; data: { message: string } };

export type ChatRequest = {
  session_id?: string;
  message: string;
  use_opus?: boolean;
};

// streamChat는 async generator로 SSE 이벤트를 yield. AbortController로 cancel 가능.
export async function* streamChat(req: ChatRequest, signal?: AbortSignal): AsyncGenerator<ChatEvent> {
  // SSE는 fetch reader 기반이라 authFetch 대신 직접 supabase 토큰 인라인 첨부.
  const supabaseMod = await import("@/lib/supabase/client");
  const supabase = supabaseMod.createSupabaseBrowser();
  const { data: { session } } = await supabase.auth.getSession();
  const token = session?.access_token;

  const res = await fetch(`${API_BASE}/v1/chat`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(req),
    signal,
  });
  if (!res.ok || !res.body) {
    throw await readError(res);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });

    let idx: number;
    while ((idx = buf.indexOf("\n\n")) !== -1) {
      const raw = buf.slice(0, idx);
      buf = buf.slice(idx + 2);
      const ev = parseSSE(raw);
      if (ev) yield ev;
    }
  }
}

function parseSSE(raw: string): ChatEvent | null {
  const lines = raw.split("\n");
  let event = "";
  let dataStr = "";
  for (const line of lines) {
    if (line.startsWith("event: ")) event = line.slice(7);
    if (line.startsWith("data: ")) dataStr += line.slice(6);
  }
  if (!event || !dataStr) return null;
  try {
    return { type: event as ChatEvent["type"], data: JSON.parse(dataStr) } as ChatEvent;
  } catch {
    return null;
  }
}

import { authFetch, readError } from "./auth-fetch";

export type ChatSession = {
  id: string;
  user_id: string;
  title: string;
  created_at: string;
  updated_at: string;
};

export type ChatMessage = {
  id: string;
  session_id: string;
  role: "user" | "assistant" | "tool";
  content: string;
  tool_calls?: unknown;
  input_tokens: number;
  output_tokens: number;
  model?: string | null;
  finished_at?: string | null;
  created_at: string;
};

export async function listSessions(): Promise<ChatSession[]> {
  const res = await authFetch("/v1/chat/sessions");
  if (!res.ok) throw await readError(res);
  return res.json();
}

export async function deleteSession(id: string): Promise<void> {
  const res = await authFetch(`/v1/chat/sessions/${id}`, { method: "DELETE" });
  if (!res.ok && res.status !== 204) throw await readError(res);
}

export async function listMessages(sessionId: string): Promise<ChatMessage[]> {
  const res = await authFetch(`/v1/chat/sessions/${sessionId}/messages`);
  if (!res.ok) throw await readError(res);
  return res.json();
}

export async function getUnfinished(sessionId: string): Promise<ChatMessage | null> {
  const res = await authFetch(`/v1/chat/sessions/${sessionId}/unfinished`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.unfinished ?? null;
}

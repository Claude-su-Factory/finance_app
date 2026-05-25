import { authFetch, readError } from "./auth-fetch";

export type Usage = {
  usage: {
    user_id: string;
    year_month: string;
    chat_count: number;
    input_tokens: number;
    output_tokens: number;
    opus_count: number;
  };
  limits: {
    chat: number;
    input_tokens: number;
    output_tokens: number;
    opus: number;
  };
};

export async function getUsage(): Promise<Usage> {
  const res = await authFetch("/v1/chat/usage");
  if (!res.ok) throw await readError(res);
  return res.json();
}

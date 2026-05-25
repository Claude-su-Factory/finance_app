import { ChatPage } from "@/components/chat/ChatPage";

export default async function ChatSessionPage({
  params,
}: {
  params: Promise<{ sessionId: string }>;
}) {
  const { sessionId } = await params;
  return <ChatPage sessionId={sessionId} />;
}

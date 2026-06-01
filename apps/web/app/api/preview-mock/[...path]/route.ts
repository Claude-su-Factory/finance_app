// apps/web/app/api/preview-mock/[...path]/route.ts
// /preview 전용 목 API. ENABLE_PREVIEW=1 일 때만 동작(프로덕션 404).
// authFetch가 NEXT_PUBLIC_API_URL=http://localhost:3000/api/preview-mock 로 보낸 /v1/* 를 받는다.
import { NextResponse } from "next/server";
import { lookupFixture } from "@/lib/preview/fixtures";

function enabled() {
  return process.env.ENABLE_PREVIEW === "1";
}

export async function GET(_req: Request, ctx: { params: Promise<{ path: string[] }> }) {
  if (!enabled()) return new NextResponse(null, { status: 404 });
  const { path } = await ctx.params;
  const pathname = "/" + (path ?? []).join("/"); // ["v1","holdings"] → "/v1/holdings"
  return NextResponse.json(lookupFixture(pathname));
}

// 쓰기 액션: happy-path 성공만(상태유지 X). 컴포넌트의 토스트·낙관적 UI를 위해 200.
async function writeOk() {
  if (!enabled()) return new NextResponse(null, { status: 404 });
  return NextResponse.json({ ok: true });
}
export const POST = writeOk;
export const PUT = writeOk;
export const PATCH = writeOk;
export const DELETE = writeOk;

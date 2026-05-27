// Next.js 16 instrumentation — Sentry 서버/엣지 초기화.
// 환경변수 NEXT_PUBLIC_SENTRY_DSN_WEB가 없으면 init이 no-op.
// docs: https://docs.sentry.io/platforms/javascript/guides/nextjs/

export async function register() {
  if (!process.env.NEXT_PUBLIC_SENTRY_DSN_WEB) {
    return;
  }
  if (process.env.NEXT_RUNTIME === "nodejs") {
    await import("./sentry.server.config");
  }
  if (process.env.NEXT_RUNTIME === "edge") {
    await import("./sentry.edge.config");
  }
}

export async function onRequestError(
  err: unknown,
  request: {
    path: string;
    method: string;
    headers: { [key: string]: string };
  },
  context: {
    routerKind: "Pages Router" | "App Router";
    routePath: string;
    routeType: "render" | "route" | "action" | "middleware";
    renderSource?:
      | "react-server-components"
      | "react-server-components-payload"
      | "server-rendering";
    revalidateReason: "on-demand" | "stale" | undefined;
    renderType: "dynamic" | "dynamic-resume";
  }
) {
  if (!process.env.NEXT_PUBLIC_SENTRY_DSN_WEB) {
    return;
  }
  const Sentry = await import("@sentry/nextjs");
  Sentry.captureRequestError(err, request, context);
}

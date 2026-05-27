import * as Sentry from "@sentry/nextjs";

// 클라이언트 init은 instrumentation-client.ts에서 다시 호출된다.
// DSN 미설정 시 모든 함수가 no-op.
Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN_WEB,
  environment: process.env.NEXT_PUBLIC_ENV ?? "development",
  tracesSampleRate: 0.1,
  replaysSessionSampleRate: 0, // 세션 리플레이 비활성 (PII)
  replaysOnErrorSampleRate: 0,
  sendDefaultPii: false,
});

// PostHog 초기화 + 이벤트 wrapper.
// NEXT_PUBLIC_POSTHOG_KEY 미설정 시 모든 함수가 no-op.
"use client";

import posthog from "posthog-js";

let initialized = false;

export function initPostHog() {
  if (initialized || typeof window === "undefined") return;
  const key = process.env.NEXT_PUBLIC_POSTHOG_KEY;
  if (!key) return;
  posthog.init(key, {
    api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST ?? "https://app.posthog.com",
    capture_pageview: false, // App Router에서 직접 처리
    capture_pageleave: true,
    person_profiles: "identified_only",
  });
  initialized = true;
}

export function capture(event: string, props?: Record<string, unknown>) {
  if (!initialized) return;
  posthog.capture(event, props);
}

export function identify(userId: string, traits?: Record<string, unknown>) {
  if (!initialized) return;
  posthog.identify(userId, traits);
}

export function reset() {
  if (!initialized) return;
  posthog.reset();
}

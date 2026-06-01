# UI 인터랙티브 미리보기

백엔드·DB·인증 없이 실제 페이지 컴포넌트를 가짜 데이터로 띄워, 브라우저에서 클릭하며 검수하는 개발 도구.

## 사용법

```bash
cd apps/web
npm run preview        # ENABLE_PREVIEW=1 + 더미 env로 next dev -p 3000
```

브라우저로 http://localhost:3000/preview 를 연다. 우하단 스위처 또는 허브에서 화면을 클릭해 순회한다.

- 신규 런타임 의존성 없음. 3000 포트가 점유돼 있으면 비우거나 `package.json`의 `preview` 스크립트에서 포트와 `NEXT_PUBLIC_API_URL`을 함께 바꾼다.
- 쓰기 액션은 성공 응답만(상태유지 X). 사이드바·⌘K 결과는 실제 `/app`을 가리켜 클릭 시 `/login`으로 튄다 — 이동은 스위처로.

## 구조

- `app/preview/*` — 미리보기 라우트(실제 컴포넌트 렌더)
- `app/api/preview-mock/[...path]/route.ts` — `/v1/*` 목 API(catch-all)
- `lib/preview/fixtures.ts` — 엔드포인트별 한국어 목 + `lookupFixture`
- `lib/preview/screens.ts` — 화면 목록(허브·스위처 공유)
- `components/preview/PreviewSwitcher.tsx` — 고정 위치 스위처

## 새 화면/모달 추가

1. `app/preview/...`에 라우트 추가(실제 컴포넌트 렌더)
2. 필요한 엔드포인트 목을 `lib/preview/fixtures.ts MOCKS`에 추가
3. `lib/preview/screens.ts`에 링크 추가

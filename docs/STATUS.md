# Quotient — 구현 상태

마지막 업데이트: 2026-05-22

## 현재 Phase

**Phase 1 — MVP W1 실행 (T1~T12 완료, T13~T15 외부 셋업 차단)**

## 진행 중

- [ ] W1-T13 Sentry + PostHog (외부 DSN 필요)
- [ ] W1-T14 Fly + Vercel 배포 (외부 계정 필요)
- [ ] W1-T15 GitHub Actions CI/CD (GitHub repo + FLY_API_TOKEN 필요)
- [ ] W1-T16 통합 동작 검증 (T13~T15 후)
  - ✅ 섹션 1: 정체성·카피
  - ⚠️ 섹션 2: 정보 구조 (잠정 합의, 섹션 6에서 락인됨 — 본 항목 보존용)
  - ✅ 섹션 3: 데이터 모델
  - ✅ 섹션 4: 데이터 수집 파이프라인
  - ✅ 섹션 5: AI 채팅 흐름
  - ✅ 섹션 6: 정보 구조 락인 (+ Long-running 작업 진행 표시 정책 포함)
  - ✅ 섹션 7: 인증·결제·구독
  - ✅ 섹션 8: 에러·관측·보안
  - ✅ 섹션 9: 테스트·배포·MVP 일정
  - ✅ 섹션 10: 보강 명세 (자체 검토 반영) + 검토 이력

## 완료

- ✅ Phase 0 (브레인스토밍·스펙·plan 작성·subagent 검토)
- ✅ W1-T1 모노레포 구조 + 기본 설정 (`6f35a7c`, `fda393a`)
- ✅ W1-T2 Supabase 로컬 + profiles 마이그레이션 (`2268f2a`)
- ✅ W1-T3 profiles RLS 정책 (`352b3fd`)
- ✅ W1-T4 Go 백엔드 스캐폴딩 + healthz/readyz (`f70d754`, `8700245`)
- ✅ W1-T5 JWT 검증 미들웨어 + CORS (`58bf1e8`, `bc7b483`)
- ✅ W1-T6 profiles GET·PATCH 엔드포인트 (`92063c2`, `b3ea3d7`)
- ✅ W1-T7 Next.js 스캐폴딩 + Tailwind v4 블룸버그 토큰 + shadcn v4 + 랜딩/법적 페이지 (`193cb0f`, `0939b4a`)
- ✅ W1-T8 Supabase 클라이언트 + SSR + proxy 미들웨어 (`a7ab721`)
- ✅ W1-T9 가입·로그인·이메일 인증·Google OAuth (PIPA 분리 동의 + 14세 차단) (`2d55518`, `ebde451`)
- ✅ W1-T10 비밀번호 재설정 (`c0459f0`)
- ✅ W1-T11 앱 셸 (사이드바·티커 placeholder·상태바) + 500 페이지 (`b6b93e7`)
- ✅ W1-T12 온보딩 wizard 2단계 (`f23e86b`)

## 알려진 결함 / 백로그

- **Go API**: `pgx` 풀이 postgres 슈퍼유저로 연결 — RLS 우회. 현재는 핸들러 `WHERE id = uid` 필터로 격리. W3 이전에 사용자 JWT 기반 쿼리로 전환 결정 필요 (spec §10-1, profile_repo_pg.go TODO)
- **Next.js 미들웨어**: profile fetch가 모든 `/app/*` 요청마다 발생 (N+1). 사용자 100명까지는 무시 가능. 트래픽 증가 시 JWT custom claim 캐싱 검토
- **Pretendard 폰트**: pretendard CSS만 import, next/font/local로 최적화 미적용
- **Profile handler 통합 테스트**: fake repo만 있고 실제 Postgres 통합 테스트는 W2 testcontainers-go 도입 후
- **stack 버전 변경**: Next.js 16.2.6 + Tailwind v4 (스펙은 15 + v3) — 최신 GA 수용, 스펙 문서 업데이트 필요
- **Go 1.25 강제**: pgx/v5 v5.9.2가 Go 1.25 요구. Task 14 Dockerfile · Task 15 CI 모두 `golang:1.25-alpine` / `go-version: "1.25"` 사용 필요
- **Supabase Auth JWT secret**: CLI v2.98이 legacy 키 노출 안 함. 사용자가 dashboard에서 "Legacy JWT Secret" 활성화 필요. JWKS 마이그레이션 백로그
- **온보딩 단계 수**: 스펙 §6은 3단계, W1 구현은 2단계 (holdings API 미구현). W3에서 3단계로 복원 예정

## 최근 변경 이력

- 2026-05-22 W1-T11·T12 완료. 앱 셸 + 온보딩 wizard 2단계.
- 2026-05-22 W1-T8·T9·T10 완료. Supabase SSR/proxy 미들웨어 + 가입/로그인/OAuth/비밀번호 재설정 (PIPA 준수).
- 2026-05-22 W1-T7 완료. Next.js 16 + Tailwind v4 + shadcn v4 (스펙 15/v3에서 최신 GA로 이전).
- 2026-05-22 W1-T4·T5·T6 완료. Go API healthz/readyz + JWT/CORS + profiles 엔드포인트.
- 2026-05-22 W1-T2·T3 완료. Supabase 로컬 + profiles 테이블·RLS.
- 2026-05-22 W1-T1 완료. 모노레포 구조 + 기본 설정.
- 2026-05-22 W1 plan 작성 완료 + subagent 검토 (general-purpose) 1차 사이클. 9 Critical + 12 Important + 6 Minor 패치 적용.
- 2026-05-22 1차 자체 검토 완료. Critical 6건 + Important 4건 식별·패치 (섹션 10). 사용자 검토 단계로 이행.
- 2026-05-22 섹션 9 (테스트·배포·MVP 일정) 확정. 일정 6주, 모두 무료 출시, 광고 슬롯 추상화, 결제 비활성.
- 2026-05-22 수익화 경로 1 선택: MVP 완전 무료 + 광고 슬롯 토글 + 결제는 사업자 등록 시점 활성화.
- 2026-05-22 섹션 8 (에러·관측·보안) 확정. Sentry + PostHog + Fly logs + Resend, 3 레이어 에러 처리, RLS 이중 방어, rehype-sanitize 의무, PIPA 준수.
- 2026-05-22 섹션 7 (인증·결제·구독) 확정. Supabase Auth + Google OAuth, Toss 빌링키 정기결제, 7일 이내 환불, 부가세 포함 ₩14,900, webhook 멱등성 의무.
- 2026-05-22 섹션 6 (정보 구조 락인) 확정. 경로 맵 12개, 온보딩 wizard, 전역 컴포넌트, Long-running 작업 progress UI 정책(블룸버그 풍 1px 라인 + monospace 진행률 + sonner toast).
- 2026-05-22 섹션 5 (AI 채팅 흐름) 확정. 모델 라우팅(Sonnet 4.6 기본 + Opus 4.7 심층 + Haiku 4.5 내부), tool use 9개, Freemium 10회/월·Pro ₩14,900, 일일 브리핑 07:00 KST.
- 2026-05-22 섹션 4 (데이터 수집 파이프라인) 확정. KRX·Yahoo·FRED·ECOS·exchangerate.host. 시세 15분 지연. KIS는 Phase 3.
- 2026-05-22 하네스 골격 생성 + 브레인스토밍 섹션 1·2·3 결정 사항 스펙 파일에 이관.
- 2026-05-22 프로젝트 시작. MVP 스코프·아키텍처·데이터 모델 확정.

---
업데이트 규칙: 기능 완료 시 즉시 갱신. 알려진 결함은 발견 즉시 등재.

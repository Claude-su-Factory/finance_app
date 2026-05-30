# Quotient — 로드맵

## 현재 추천 다음 작업

모든 코드 작업 완료. 남은 작업은 외부 계정·키 발급(사용자 액션)뿐이다.

(CSV import는 드롭(2026-05-29). 사유: 유일 가치가 "실 자산 초기 입력 절감"뿐인데 증권사별 포맷 파싱 유지보수 비용 과다. 실 자산 입력은 수동 UX + Phase 3 KIS Open API 자동 동기화로 대체.)
(AI 교육자 역할 완료(2026-05-29): 개념 질문 친절 답변 + footer 게이팅. STATUS 참조.)
(백테스트 완료(2026-05-29): NAV/유닛 통합 바스켓 시뮬레이터 + KOSPI·S&P·60/40 비교. STATUS 참조.)
(백테스트 커버리지 경고 결함 fix 완료(2026-05-30): 경고 기준선을 `naturalStart`로 교정 + 벤치마크·fx 사유 경고 추가. STATUS 참조.)
(미들웨어 N+1 제거 완료(2026-05-30): read-through 쿠키 캐시(`q_onboarded`)로 매 `/app/*` profiles 조회 제거. 단조 플래그 → 캐시 안전. STATUS 참조.)
(운영 자동화 완료(2026-05-30): 부팅 시 지수·NASDAQ 자동 백필(`SeedIfEmpty`, 비동기·멱등) + Fly `release_command` Go 마이그레이터(이력 테이블 공유). 사용자 수동 ops 0. STATUS 참조.)

**현재 추천 다음 작업**: 아래 외부 계정·키 발급을 순서대로 완료하면 production 배포가 가능하다.

외부 계정·키 발급 (사용자 액션):

1. **Supabase 프로덕션 프로젝트** — region `ap-northeast-2` 생성 + `supabase db push` → DATABASE_URL·SUPABASE_JWT_SECRET 확보 (`docs/DEPLOY.md` §2)
2. **Fly.io API 배포** — 계정 + `flyctl secrets set` + `flyctl deploy` (`docs/DEPLOY.md` §3)
3. **Vercel Next.js 배포** — git integration 연결 + env 등록 (`docs/DEPLOY.md` §4)
4. **Sentry + PostHog 가입** — DSN/Key 발급 후 환경변수 주입 (`docs/DEPLOY.md` §5·§6)
5. **GitHub Secrets 등록** — `FLY_API_TOKEN` 등 (`docs/DEPLOY.md` §7)
6. **E2E 스모크 검증** — 배포 직후 `docs/E2E_SMOKE.md` 시나리오 수동 통과
7. **Anthropic API 키 발급** — production 키 → Fly secrets에 주입

## Phase 0 — 스펙 (현재)

| Tier | 작업 |
|---|---|
| 1 | MVP 디자인 스펙 9개 섹션 완료 |
| 1 | 스펙 자체 검토 (Critical/Important/Minor) 및 패치 |
| 2 | 사용자 스펙 승인 |
| 2 | 구현 계획 작성 (`docs/superpowers/plans/`) — subagent 검토 필수 |

## Phase 1 — MVP (목표 **6주**, 무료 출시)

타겟: 5개 탭 (홈·포트폴리오·AI 채팅·마켓·설정). 수동 입력 + 공개 시세. **모두 무료 + 광고 슬롯 토글**. 결제는 비활성 (Phase 2 사업자 등록 시 활성화).

완료(W1·W2a·W2b·W3·W4·W5 결과)는 [`STATUS.md`](STATUS.md)에서 확인. 남은 작업:

| Tier | 작업 | 예정 |
|---|---|---|
| — | (모든 코드 작업 완료) | — |
| 1 | W1 외부 계정 셋업 (사용자 액션) | 사용자 |

## 수익화·차별화 후속 (Phase 2~3)

본 서비스는 개인 자격(사업자 등록 X) + 마이데이터·자동매매 X + 직접 자문 X 제약. 그 안에서 다음 옵션:

| 시점 | 항목 | 비고 |
|---|---|---|
| 즉시(코드 준비됨) | AdSense 활성 | 사용자가 가입 후 env 주입 → 마켓 페이지 하단 노출 |
| 즉시(코드 준비됨) | Toss 개인 후원 | `NEXT_PUBLIC_TOSS_DONATION_URL` 설정 시 사이드바 footer |
| ~~Phase 2 핵심 차별화~~ | ~~Paper Trading 백테스트(서브시스템 B)~~ | **완료(2026-05-29)** — NAV/유닛 통합 바스켓 시뮬레이터(전략 + KOSPI·S&P·60/40 동일 캐시플로우) + 초과수익. STATUS 참조 |
| ~~Phase 2 차별화~~ | ~~AI 분석가 = 학습 도우미~~ | **완료(2026-05-29)** — 시스템 프롬프트 교육자 역할 + footer 게이팅 |
| Phase 2 | 세금 계산기 (양도·배당세) | "조언"이 아닌 단순 시뮬레이터 |
| Phase 2 | 공시 알림 (DART) | 공개 API, 규제 무관 |
| Phase 3 (사업자 등록 후) | Toss Payments 구독료(₩14,900/월) | 사업자 등록 + 통신판매업 신고 후 활성 |
| Phase 3 | 증권사 affiliate (가입 보상) | 통신판매업 신고 필요. 단순 광고 형태로만 — "추천" 표현 금지 |

⛔ **전체 사용자 수익률 랭킹·리더보드는 영구 불가**: 자기 신고 데이터의 검증 수단 부재로 신뢰성 붕괴 + "상위 포트폴리오 노출" = 투자권유 영역(라이선스 필요) + 위험 매매 유도. 후킹은 알파 카드(외부 지수 비교)로 대체.

⛔ 규제로 영구 불가: 마이데이터 연동, 자동매매·거래중개, 개별 종목 매수/매도 추천, 자금 보관·이체.

## Phase 2 — 확장 (MVP 출시 후)

전환 조건: 가입자 100명 또는 일평균 PV 500 또는 MVP 출시 후 3개월 경과.

| Tier | 작업 |
|---|---|
| 1 | **Paper Trading 백테스트(서브시스템 B)** — 과거 시점 + 전략(정액 적립, 리밸런싱) → 시뮬레이션 결과 비교 |
| 1 | AI 시스템 프롬프트에 교육자 역할 추가 — 개념 질문에 친절 답변 |
| 1 | ~~AI 시스템 프롬프트 교육자 역할~~ — **완료(2026-05-29)** |
| 1 | (선택) 사업자 등록 + 통신판매업 신고 + Toss 가맹 |
| 1 | (사업자 등록 시) `PAYMENTS_ENABLED=true` + Toss 위젯·webhook·정기 결제 cron 구현·활성화 |
| 1 | AdSense 가입 + `ENABLE_ADS=true` (가입자 100명 + 일평균 PV 500 도달 시) |
| 2 | 조건 알림 (가격·지표, 이메일/디스코드) |
| 2 | 주간·월간 자동 리포트 |
| 3 | 종목 심층 분석 페이지 |

## Phase 3 — 자동화

| Tier | 작업 |
|---|---|
| 1 | KIS Open API 연동 (본인 계좌 자동 동기화) |
| 2 | 백테스팅·시뮬레이션 엔진 |
| 3 | 코인·DeFi 지원 검토 |

---
업데이트 규칙: 작업 완료 시 항목 제거. "현재 추천 다음 작업" 즉시 재설정.

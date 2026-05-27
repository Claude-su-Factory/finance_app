# Quotient — 로드맵

## 현재 추천 다음 작업

Phase 1 핵심 + 모든 코드 작업 완료. **남은 항목은 사용자가 직접 셋팅해야 하는 외부 계정·키 발급뿐**:

1. **Supabase 프로덕션 프로젝트** — region `ap-northeast-2` 생성 + `supabase db push` → DATABASE_URL·SUPABASE_JWT_SECRET 확보 (`docs/DEPLOY.md` §2)
2. **Fly.io API 배포** — 계정 + `flyctl secrets set` + `flyctl deploy` (`docs/DEPLOY.md` §3)
3. **Vercel Next.js 배포** — git integration 연결 + env 등록 (`docs/DEPLOY.md` §4)
4. **Sentry + PostHog 가입** — DSN/Key 발급 후 환경변수 주입 (`docs/DEPLOY.md` §5·§6)
5. **GitHub Secrets 등록** — `FLY_API_TOKEN` 등 (`docs/DEPLOY.md` §7)
6. **E2E 스모크 검증** — 배포 직후 `docs/E2E_SMOKE.md` 시나리오 수동 통과
7. **Anthropic API 키 발급** — production 키 → Fly secrets에 주입

코드 후속 작업은 Phase 1에 없음. Phase 2 항목은 아래 표 참조.

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
| Phase 2 | **AI 매매 일기** | 사용자가 매매 이유 기록 → Claude가 사후 패턴 분석("8월에 감정적 매도 패턴"). 규제 무관, 본 서비스의 큰 차별화 카드 |
| Phase 2 | CSV import (증권사 거래내역 업로드) | 마이데이터 대안, 진입 장벽 ↓ |
| Phase 2 | 세금 계산기 (양도·배당세) | "조언"이 아닌 단순 시뮬레이터 |
| Phase 2 | 공시 알림 (DART) | 공개 API, 규제 무관 |
| Phase 3 (사업자 등록 후) | Toss Payments 구독료(₩14,900/월) | 사업자 등록 + 통신판매업 신고 후 활성 |
| Phase 3 | 증권사 affiliate (가입 보상) | 통신판매업 신고 필요. 단순 광고 형태로만 — "추천" 표현 금지 |

⛔ 규제로 영구 불가: 마이데이터 연동, 자동매매·거래중개, 개별 종목 매수/매도 추천, 자금 보관·이체.

## Phase 2 — 확장 (MVP 출시 후)

전환 조건: 가입자 100명 또는 일평균 PV 500 또는 MVP 출시 후 3개월 경과.

| Tier | 작업 |
|---|---|
| 1 | (선택) 사업자 등록 + 통신판매업 신고 + Toss 가맹 |
| 1 | (사업자 등록 시) `PAYMENTS_ENABLED=true` + Toss 위젯·webhook·정기 결제 cron 구현·활성화 |
| 1 | AdSense 가입 + `ENABLE_ADS=true` (가입자 100명 + 일평균 PV 500 도달 시) |
| 1 | CSV 업로드 + LLM 파싱 (증권사 거래내역) |
| 1 | `transactions`·`cash_transactions` 테이블 도입 |
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

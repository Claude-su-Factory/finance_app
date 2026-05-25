# Quotient — 로드맵

## 현재 추천 다음 작업

W5 plan 작성 — 마켓 탭(`/app/market`). KR/US 지수·환율·경제 지표 카드 + 라인 차트(recharts 도입) + 관심 종목 추가/제거 UI. 광고 슬롯(`<AdSlot>`) 추상화 함께 검토.

후속 backlog: AI RealClient 실 구현(`claude-api` 스킬), 포트폴리오 미니 스파크라인·sliding panel.

## Phase 0 — 스펙 (현재)

| Tier | 작업 |
|---|---|
| 1 | MVP 디자인 스펙 9개 섹션 완료 |
| 1 | 스펙 자체 검토 (Critical/Important/Minor) 및 패치 |
| 2 | 사용자 스펙 승인 |
| 2 | 구현 계획 작성 (`docs/superpowers/plans/`) — subagent 검토 필수 |

## Phase 1 — MVP (목표 **6주**, 무료 출시)

타겟: 5개 탭 (홈·포트폴리오·AI 채팅·마켓·설정). 수동 입력 + 공개 시세. **모두 무료 + 광고 슬롯 토글**. 결제는 비활성 (Phase 2 사업자 등록 시 활성화).

완료(W1·W2a·W2b·W3·W4 결과)는 [`STATUS.md`](STATUS.md)에서 확인. 남은 작업:

| Tier | 작업 | 예정 |
|---|---|---|
| 1 | 마켓 탭 (지수·환율·관심종목 페이지) | W5 |
| 1 | `<AdSlot>` 컴포넌트 추상화 (`ENABLE_ADS=false` 기본) | W5 |
| 1 | AI RealClient 실 구현 (anthropic-sdk-go) | W5·후속 |
| 2 | 명령 팔레트 (⌘K) | Phase 1 후반 |
| 3 | 키보드 단축키 풀세트 | Phase 1 후반 |

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

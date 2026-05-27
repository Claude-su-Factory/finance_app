# E2E 스모크 체크리스트

배포 직후 또는 주요 변경 후 골든패스가 깨지지 않았는지 확인하는 수동 시나리오.
전체 소요 ~15분. 자동화는 Phase 2(Playwright 도입 시).

**전제**: production 환경에서 새 브라우저(시크릿 창)로 진행. dev 데이터 잔존 영향 없음.

---

## 1. 인증 (3분)

### 1-1. 이메일 가입 → 인증 메일 → 로그인
- [ ] 랜딩(`/`) → "시작하기" 클릭 → `/auth/signup`
- [ ] PIPA 동의 체크박스(이용약관·개인정보처리방침·14세 이상) 표시 확인
- [ ] 14세 미만 체크 시 가입 차단 검증
- [ ] 이메일 + 비밀번호 + 동의 → 가입 버튼 → "인증 메일 전송" 안내
- [ ] 이메일 inbox에서 verification link 클릭 → 자동 로그인 → `/onboarding` 리다이렉트
- [ ] Supabase Dashboard → Authentication → Users에 가입 행 1개 추가 확인

### 1-2. 로그아웃 → 재로그인
- [ ] 사이드바 또는 설정에서 로그아웃 → `/`
- [ ] `/auth/signin` → 같은 이메일/비번 → `/app/*` 진입

### 1-3. 비밀번호 재설정
- [ ] `/auth/signin` → "비밀번호 잊으셨나요?" → 이메일 입력
- [ ] reset 메일 수신 → link 클릭 → 새 비밀번호 입력 → 로그인

### 1-4. Google OAuth
- [ ] `/auth/signin` → "Google로 계속" → 구글 동의 → `/app/*` 진입
- [ ] 신규 사용자라면 PIPA 동의 단계 거치는지 확인

---

## 2. 온보딩 (2분)

### 2-1. 3단계 wizard
- [ ] Step 1: 기본 통화(KRW/USD) 선택 → 다음
- [ ] Step 2: 일일 브리핑 on/off → 다음
- [ ] Step 3: 최초 보유 자산 추가 (선택 사항) — 종목 검색 + 수량 + 평단가
  - [ ] 검색 디바운스 동작 (300ms)
  - [ ] 한글 검색("삼성전자") + 영문("AAPL") 양쪽 결과 표시
  - [ ] INDEX/FX/CASH는 검색 결과에 안 나옴 (asset_class 가드)
- [ ] 완료 → `/app/home` 도착, sidebar에 사용자 이름 표시

### 2-2. 온보딩 재진입 방지
- [ ] `/onboarding` 직접 접근 → 이미 완료면 `/app/home` 리다이렉트

---

## 3. 홈 대시보드 (1분)

- [ ] 6 카드(총자산·자산 분포·상위5·마켓·관심종목·브리핑) 렌더링
- [ ] 총자산 카운트업 애니메이션 동작
- [ ] 자산 분포 도넛 차트 (recharts)
- [ ] 상단 TopTicker 5종(KOSPI/KOSDAQ/S&P500/NASDAQ/USDKRW) 표시
- [ ] 60초마다 ticker 갱신 (네트워크 탭에서 `/v1/market/ticker` 요청 주기 확인)

---

## 4. 포트폴리오 (3분)

### 4-1. 보유 자산 CRUD
- [ ] `/app/portfolio` → 테이블 렌더링
- [ ] "추가" 버튼 → 모달 → 종목 검색 → 수량/평단가 입력 → 저장 → 행 추가
- [ ] 행 클릭 → 우측 sliding panel 열림
  - [ ] 30일 line chart 렌더링
  - [ ] 보유 상세 8필드 표시
  - [ ] ESC + backdrop 클릭으로 닫힘
- [ ] 행의 "수정" 버튼 → 모달 → 수량 변경 → 저장 → 테이블 반영 (panel과 stopPropagation 분리)
- [ ] "삭제" 버튼 → confirm → 행 제거

### 4-2. FX 환산
- [ ] USD 종목 추가 후 KRW 환산 평가액 검증 (대략 가격 × USDKRW)
- [ ] PnL %·비중 % 모두 합리적 값

### 4-3. 7일 스파크라인
- [ ] 각 행에 7일 미니 차트 표시
- [ ] batch fetch — 네트워크 탭에서 모든 holdings 한 번에 요청

---

## 5. 관심 종목 (1분)

- [ ] `/app/market`의 "관심 종목" editor → 종목 검색 → 추가
- [ ] watchlist에 INDEX/FX 추가 시도 → 거부 (asset_class 가드)
- [ ] 홈 대시보드의 관심 종목 카드에 반영
- [ ] 삭제 → 행 제거

---

## 6. AI 채팅 (3분)

### 6-1. 첫 메시지
- [ ] `/app/chat` → 입력창에 "내 포트폴리오 분석해줘" → Enter
- [ ] SSE 스트리밍 토큰이 끊김 없이 화면에 표시
- [ ] tool indicator(`🔧 get_portfolio` 등) 표시 후 결과 도착
- [ ] 마지막에 disclaimer 자동 부착: `(데이터 기준: YYYY-MM-DD HH:MM KST, 시세 지연 15분)`
- [ ] 사이드바에 새 세션 자동 추가

### 6-2. 도구 라우팅
- [ ] "삼성전자 현재가 알려줘" → `get_quote` 호출 → 시세 출력
- [ ] "관심 종목 보여줘" → `get_watchlist` → 목록 출력
- [ ] "Fed 기준금리 추이" → `get_economic_indicator` (DFF) → 시계열

### 6-3. 세션 관리
- [ ] 좌측 세션 리스트에서 이전 세션 클릭 → 메시지 히스토리 복원
- [ ] 세션 삭제 → 목록에서 제거

### 6-4. 사용량 한도
- [ ] 우측 상단 배지에 `n/30`, `tokens/50K` 표시
- [ ] (한도 초과 시 429 응답 확인은 dev에서만)

### 6-5. 끊김 처리
- [ ] SSE 진행 중 새로고침 → 같은 세션 재진입 시 부분 응답이 남아있는지(unfinished endpoint)

---

## 7. 일일 브리핑 (1분)

- [ ] 홈의 브리핑 카드 — 오늘 07:00 KST 이후 접속이면 콘텐츠 표시
- [ ] 콘텐츠 한국어 3~5문장, 보유 자산·시장·관심 종목 언급
- [ ] 자정 직후 접속 시 "오늘 브리핑 아직 없음" 메시지

---

## 8. 관측 (1분)

- [ ] Sentry → Issues — production에 에러 0개(또는 알려진 것만)
- [ ] PostHog → Live events — `$pageview` 이벤트 도착 확인
- [ ] Fly logs — error 레벨 없음, "cron started" 로그 부팅 시 1회

---

## 9. RLS 격리 (1분)

다른 사용자 계정(또는 incognito 창)으로 가입 → 본인 holdings/watchlist/chat 세션이 보이지 않아야 함.

---

## 알려진 비-스모크 항목 (수동 검증 외)

- 결제 흐름 — Phase 2 사업자 등록 후
- 광고 슬롯 — `NEXT_PUBLIC_ENABLE_ADS=false` 기본
- KIS Open API 연동 — Phase 3
- CSV 업로드 — Phase 2

---

체크리스트 완료 후 `docs/STATUS.md` "최근 변경 이력"에 "Production 스모크 PASS YYYY-MM-DD" 한 줄 추가.

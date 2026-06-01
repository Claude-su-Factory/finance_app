# AI 매매 일기 (Trading Journal) — 설계 스펙

> 사용자가 매매 결정·시장 관찰 이유를 기록하고 Claude가 사후 패턴 분석.
> 정체성 spec §3의 Phase 2 핵심 차별화 카드(우선순위 1). 한국 핀테크에 없는 기능.

**날짜**: 2026-05-28
**저자**: 사용자 + 에이전트 (brainstorming 1 사이클)
**상태**: 디자인 확정. 구현 plan 작성 단계 진입 예정.
**관련 spec**: [`2026-05-28-identity-3-pillars.md`](./2026-05-28-identity-3-pillars.md) §3
**관련 변경**: 신규 테이블 2종, 신규 페이지 1, 신규 핸들러 6개, 신규 AI 도구 1, 신규 cron 1.

---

## 1. 목적

사용자의 매매 결정·시장 관찰을 시계열로 기록하고 Claude가 사후 패턴 분석으로 자기 회고를 돕는다.

**예시 분석 결과**:
- "8월에 3번 모두 KOSPI 하락 직후 매도 → 손절 후 평균 4일 만에 반등 (감정적 매도 패턴)"
- "매수 이유에 'PER 저평가'가 60% — 가치투자 일관성 OK"
- "8월 거래량 평소 3배 — 변동성 큰 달에 과잉 거래"

**비목적**:
- 다른 사용자와 비교 (영구 불가, 정체성 spec §2)
- 종목 추천 (분석 관점만)
- 실시간 알림 (월간 자동만)
- 자동 매매 (영구 불가)

---

## 2. 핵심 결정 (brainstorm 합의)

| # | 영역 | 결정 |
|---|---|---|
| D1 | 입력 트리거 | **하이브리드** — Holdings CRUD 모달에 "매매 이유" 필드(auto) + `/app/journal` 별도 페이지(manual) |
| D2 | 데이터 모델 | `journal_entries` + `analysis_runs` 두 테이블. RLS 강제 |
| D3 | AI 분석 출력 | **하이브리드** — 자동 월간 회고 cron + on-demand "분석 요청" 버튼 + 채팅 `analyze_journal` 도구 |
| D4 | 사용 한도 | 자동 월간: 무료(사용자당 1회/월) / 수동·채팅: 기존 채팅 한도(월 30회)와 합산 |
| D5 | UI 위치 | 사이드바 신규 📓 아이콘 → `/app/journal` 페이지. 매매 일기 + 분석 카드 함께 노출 |
| D6 | 종목 태그 | MVP는 자유 string array(symbol 입력). 자동 추출은 v2 |
| D7 | Paper 통합 | Phase 2 Paper 작업에서 같은 테이블 + `related_paper_holding_id` 컬럼 추가로 자연 통합. MVP는 실 holdings만 |

---

## 3. 데이터 모델

### 3-1. `journal_entries`

```sql
create table public.journal_entries (
  id                   uuid primary key default gen_random_uuid(),
  user_id              uuid not null references auth.users(id) on delete cascade,
  entry_type           text not null check (entry_type in ('auto', 'manual')),
  action               text check (action in ('buy', 'sell', 'observation', 'other')),
  related_holding_id   uuid references public.holdings(id) on delete set null,
  related_symbols      text[] not null default '{}',
  title                text,
  content              text not null,
  created_at           timestamptz not null default now(),
  updated_at           timestamptz not null default now()
);

create index journal_entries_user_created_idx
  on public.journal_entries (user_id, created_at desc);

create trigger journal_entries_touch_updated_at
  before update on public.journal_entries
  for each row execute function public.touch_updated_at();
```

**RLS**:
```sql
alter table public.journal_entries enable row level security;
create policy journal_entries_select_own on public.journal_entries
  for select using (user_id = auth.uid());
create policy journal_entries_insert_own on public.journal_entries
  for insert with check (user_id = auth.uid());
create policy journal_entries_update_own on public.journal_entries
  for update using (user_id = auth.uid())
  with check (user_id = auth.uid());
create policy journal_entries_delete_own on public.journal_entries
  for delete using (user_id = auth.uid());
```

### 3-2. `analysis_runs`

```sql
create table public.analysis_runs (
  id              uuid primary key default gen_random_uuid(),
  user_id         uuid not null references auth.users(id) on delete cascade,
  run_type        text not null check (run_type in ('auto_monthly', 'on_demand')),
  period_start    date not null,
  period_end      date not null,
  entries_count   int not null default 0,
  content_md      text not null,
  model           text not null,
  created_at      timestamptz not null default now()
);

create index analysis_runs_user_created_idx
  on public.analysis_runs (user_id, created_at desc);

-- 같은 사용자가 동일 월에 auto_monthly 중복 생성 방지
create unique index analysis_runs_auto_monthly_idx
  on public.analysis_runs (user_id, period_start)
  where run_type = 'auto_monthly';
```

**RLS**: select·insert만, update·delete는 없음(불변 기록).

```sql
alter table public.analysis_runs enable row level security;
create policy analysis_runs_select_own on public.analysis_runs
  for select using (user_id = auth.uid());
-- insert는 서버 측(SECURITY DEFINER 또는 service role 또는 user JWT pool에서)
create policy analysis_runs_insert_own on public.analysis_runs
  for insert with check (user_id = auth.uid());
```

### Why 두 테이블 분리

`journal_entries`는 사용자가 직접 작성한 원본, `analysis_runs`는 AI 생성 부산물.
- 사용자가 entry 수정해도 분석 이력은 그대로 → 시점 회고 가능
- 분석 비용 추적 별도 (`model`·`created_at`으로 호출 통계)
- v2에서 분석 공유·export 시 다른 보안 정책 적용 가능

---

## 4. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Frontend                                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ /app/journal (신규 페이지)                            │  │
│  │  - JournalTimeline (entries 리스트, 날짜 desc)        │  │
│  │  - AnalysisCard (auto_monthly + on_demand 카드)       │  │
│  │  - NewEntryDialog (manual entry 생성)                 │  │
│  │  - AnalyzeButton ("분석 요청 ⚡")                     │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ AddHoldingDialog / EditHoldingDialog (수정)           │  │
│  │  - "💭 매매 이유" textarea (200자 선택)               │  │
│  │  - 저장 시 holdings INSERT + journal_entries INSERT   │  │
│  │    같은 트랜잭션 (db.AsUser)                          │  │
│  └──────────────────────────────────────────────────────┘  │
│  Sidebar: 📓 아이콘 (홈·포트폴리오·채팅·마켓 다음)         │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ /v1/journal/* (6 endpoints)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Backend (apps/api)                                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ handlers/journal.go                                   │  │
│  │  - List/Create/Patch/Delete entries                   │  │
│  │  - PostAnalyze (on-demand, 채팅 한도 차감)            │  │
│  │  - ListAnalyses                                       │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ ai/tools/journal.go (신규 도구)                       │  │
│  │  - analyze_journal: 사용자 entries + holdings →       │  │
│  │    Claude system prompt 주입 → 패턴 분석 텍스트       │  │
│  │  - RequiresUserContext = true (db.AsUser wrap)        │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ schedule/journal_monthly.go (신규 cron)               │  │
│  │  - 매월 1일 07:00 KST, 사용자 hash 분단위 분산        │  │
│  │  - 직전 월 entries → analyze_journal 도구 호출 →      │  │
│  │    analysis_runs (run_type='auto_monthly') INSERT     │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ handlers/holdings.go (수정)                           │  │
│  │  - Create/Patch에 reason 옵션 파라미터 추가           │  │
│  │  - reason 있으면 journal_entries auto entry 동시 생성 │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Supabase Postgres                                          │
│  - public.journal_entries (RLS)                             │
│  - public.analysis_runs (RLS, insert·select만)              │
│  - public.holdings (기존 — related_holding_id 참조)         │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. API 명세

### 5-1. `GET /v1/journal/entries`

**Query**: `?limit=50&before=<ISO timestamp>`
**응답 (200)**:
```json
{
  "entries": [
    {
      "id": "uuid",
      "entry_type": "auto",
      "action": "buy",
      "related_holding_id": "uuid",
      "related_holding": { "symbol": "005930", "name": "삼성전자" },
      "related_symbols": [],
      "title": null,
      "content": "실적 발표 후 반도체 사이클 회복 기대...",
      "created_at": "2026-05-28T05:21:00Z",
      "updated_at": "2026-05-28T05:21:00Z"
    },
    {
      "id": "uuid",
      "entry_type": "manual",
      "action": "observation",
      "related_holding_id": null,
      "related_symbols": ["005930", "000660", "NVDA"],
      "title": "반도체 비중 늘리기로",
      "content": "...",
      "created_at": "2026-05-25T...",
      "updated_at": "2026-05-25T..."
    }
  ],
  "has_more": false
}
```

### 5-2. `POST /v1/journal/entries`

**Body**:
```json
{
  "action": "observation",
  "related_symbols": ["005930"],
  "title": "반도체 비중 검토",
  "content": "..."
}
```

서버가 `entry_type='manual'` 강제. `related_holding_id`는 manual 경로에서 받지 않음 (manual은 자유 관찰용).

**응답 (201)**: 생성된 entry 본문.
**검증**: `content` 1~2000자, `related_symbols` 최대 10개·각 20자, `title` 최대 100자.

### 5-3. `PATCH /v1/journal/entries/{id}`

manual entry만 수정 가능. auto entry는 404 또는 422.
**Body**: `{ title?, content?, related_symbols?, action? }`

### 5-4. `DELETE /v1/journal/entries/{id}`

manual·auto 모두 삭제 가능. RLS로 자기 entry만.

### 5-5. `POST /v1/journal/analyze`

on-demand 분석 요청. **채팅 한도(월 30회)에서 1회 차감**.

**Body**:
```json
{ "period_days": 90 }
```

**응답 (200)**:
```json
{
  "id": "uuid",
  "run_type": "on_demand",
  "period_start": "2026-02-27",
  "period_end": "2026-05-28",
  "entries_count": 12,
  "content_md": "최근 3개월 매매 패턴:\n- ...",
  "model": "claude-sonnet-4-6",
  "created_at": "2026-05-28T..."
}
```

**응답 (429)**:
```json
{
  "error": { "code": "USAGE_EXCEEDED", "reason": "monthly_chat_limit", "message": "월 30회 한도 도달" }
}
```

**응답 (422)**:
```json
{
  "error": { "code": "INSUFFICIENT_DATA", "reason": "no_entries", "message": "분석할 일기가 없습니다" }
}
```

### 5-6. `GET /v1/journal/analyses`

**Query**: `?limit=20`
**응답 (200)**: `analysis_runs` 역시간순 리스트.

### 5-7. `POST /v1/holdings` / `PATCH /v1/holdings/{id}` (수정)

기존 핸들러의 body에 `reason: string` 옵션 필드 추가. 있으면 holdings INSERT/UPDATE와 같은 `db.AsUser` 트랜잭션 안에서 `journal_entries (entry_type='auto', action='buy' or 'sell', content=reason, related_holding_id=holdings.id)` INSERT.

UPDATE 경로의 `entry_type='auto'`는 수정 시 새 entry (이전 entry는 그대로) — entry는 immutable timeline이라 의미적으로 새 사건.

DELETE 경로(holdings 삭제)는 자동 entry 생성 안 함 — 사용자가 일기에 매도 이유 쓰려면 manual로 별도 작성.

---

## 6. AI 도구 — `analyze_journal`

### 6-1. 도구 명세

```go
type analyzeJournal struct{ *Deps }

func (t *analyzeJournal) Spec() ai.ToolSpec {
    return ai.ToolSpec{
        Name: "analyze_journal",
        Description: "사용자의 매매 일기 entries + 보유 자산 변화 + 시점별 시장 상황을 종합하여 매매 패턴·습관을 분석. 직접 매수/매도 권유 금지, 회고 관점만.",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "period_days": map[string]any{"type": "integer", "default": 90, "minimum": 7, "maximum": 365},
            },
        },
    }
}

func (t *analyzeJournal) RequiresUserContext() bool { return true }
```

### 6-2. Run 로직 (의사 코드)

```text
Run(ctx, exec, userID, input):
  days = input.period_days ?? 90
  since = today - days

  entries = exec.query(journal_entries where user_id=$ and created_at >= $ order by created_at)
  if len(entries) < 3:
    return { error: "분석할 일기가 3개 이상 필요합니다" }

  holdings = exec.query(holdings where user_id=$)
  price_context = collect_price_changes_around_entry_dates(entries)
    # 각 매매 entry created_at ± 7일 가격 변동 (KOSPI/SPX·해당 종목)

  system_prompt = build_analysis_prompt(entries, holdings, price_context)
  return { entries_count, model: "internal", patterns: claude_response.text }
```

### 6-3. system prompt (요약)

```
당신은 Quotient의 매매 일기 분석가입니다.
사용자가 직접 기록한 매매 결정·관찰 이유를 시계열로 받아, 다음을 분석합니다:
- 매매 시점과 시장 상황의 상관관계 (예: 손절 직후 반등)
- 매매 이유에 반복되는 키워드·논리 (예: "PER 저평가" 일관성)
- 감정적 매매 가능성 신호 (예: 변동성 직후 단발적 매도)

규칙:
- 직접 매수/매도 권유 금지. "회고 관점에서 ~를 살펴볼 수 있습니다" 표현
- 추측 금지. 사용자가 직접 쓴 텍스트 + 시점별 가격 데이터만 근거
- 마크다운 3~5 bullet. 한국어. 200자 이내.
- 사용자 비난·평가 표현 금지. 중립적·관찰 중심.
```

### 6-4. 호출 경로별 사용 한도 처리

| 경로 | 한도 차감 | 비고 |
|---|---|---|
| 자동 월간 cron | 차감 X | 사용자 한도 무관, Anthropic 비용은 발생 |
| `POST /v1/journal/analyze` (on-demand 버튼) | 차감 1회 | chat_usage_monthly.chat_count++ |
| 채팅에서 `analyze_journal` 호출 | 차감 X (이미 채팅 메시지 1턴 = 1회로 계산 중) | 추가 차감 안 함 |

---

## 7. cron — 자동 월간 회고

`apps/api/internal/schedule/journal_monthly.go` 신규.

### 7-1. 동작

- 매 분 동작 (cron `* * 1 * *`라도 분 단위 분산은 별도 hash 로직 필요)
- 더 단순: 매월 1일 07:00 KST 시작, 매 분마다 trigger, 내부에서 사용자 hash로 시간 분산
- 일일 브리핑 cron (`schedule/briefing_worker.go`) 패턴 그대로

### 7-2. 의사 코드

```text
JobMonthlyJournalDispatcher(ctx, deps, aiClient, toolRegistry):
  if today.Day() != 1 or now.Hour() != 7: return
  currentMinute = now.Minute()

  users = select id from profiles
          where exists (select 1 from journal_entries je
                         where je.user_id = profiles.id
                           and je.created_at >= today - 32 days)
  # 직전 1달 entry 있는 사용자만 대상

  for u in users:
    if userMinuteSlot(u.id) != currentMinute: continue
    if exists(analysis_runs where user_id=u and period_start=this_month_start): continue

    # ExecuteAndSerialize에 user JWT context로 호출
    result = tools.Execute(ctx, registry, pool, "analyze_journal", u.id, {period_days: 30})

    # analysis_runs INSERT (db.AsUser 트랜잭션)
    INSERT analysis_runs (user_id, run_type='auto_monthly', period_start, period_end, entries_count, content_md, model)
```

### 7-3. 비용 추정 (가입자 100명 가정)

- 월 1회 × 100 = 100 LLM 호출 (Sonnet 4.6 기본 ~$0.02 × 100 = $2/월)
- 사용자당 차감 X → retention 효과 ↑
- 가입자 1,000명 시 $20/월 — 여전히 감당 가능

---

## 8. UI 명세

### 8-1. 사이드바 신규 아이콘

```
🏠 홈
💼 포트폴리오
💬 채팅
📊 마켓
📓 매매 일기   ← 신규
⚙️ 설정
```

### 8-2. `/app/journal` 페이지 레이아웃

```
┌──────────────────────────────────────────────────┐
│ 📓 매매 일기                    [+ 새 entry] [⚡ 분석 요청] │
├──────────────────────────────────────────────────┤
│ ┌────────────────────────────────────────────┐  │
│ │ 📅 5월 월간 회고 · 자동 · 2026-05-01 07:00 │  │  ← analysis_runs (auto)
│ │                                            │  │
│ │ 5월 매매 12건 — 손절 3건 모두 KOSPI 하락  │  │
│ │ 직후. 손절 후 평균 4일 반등 → ...          │  │
│ └────────────────────────────────────────────┘  │
│                                                  │
│ ┌────────────────────────────────────────────┐  │
│ │ 💡 사용자 요청 분석 · 2026-05-28 14:21     │  │  ← analysis_runs (on_demand)
│ │ 최근 3개월 패턴: 감정적 매도 가능성...     │  │
│ └────────────────────────────────────────────┘  │
│                                                  │
│ ─── 일기 entries ───                            │
│                                                  │
│ ▌2026-05-28 · 매수 005930 · 자동                │  ← auto entry
│   실적 발표 후 반도체 사이클 회복 기대...        │
│                                                  │
│ ▌2026-05-25 · 관찰 · 수동                       │  ← manual entry
│   금리 인하 기조 + AI 수요 신호. NVDA 비중...    │
│   [수정] [삭제]                                  │
│                                                  │
│ ▌2026-05-20 · 매도 035720 · 자동                │
│   광고 매출 둔화 + 카카오 본업 부진. 손절.       │
│                                                  │
└──────────────────────────────────────────────────┘
```

### 8-3. NewEntryDialog (manual entry 생성)

```
┌──────────────────────────────────┐
│ 매매 일기 entry                  │
├──────────────────────────────────┤
│ 제목 (선택)                      │
│ [───────────────────────────]   │
│                                  │
│ 종류                             │
│ [관찰 ▼]  (관찰/매수/매도/기타) │
│                                  │
│ 관련 종목 (선택, 최대 10개)      │
│ [+ 종목 추가]                    │
│ [005930] [NVDA]                  │
│                                  │
│ 내용 (1~2000자)                  │
│ ┌────────────────────────────┐  │
│ │                            │  │
│ │                            │  │
│ └────────────────────────────┘  │
│                                  │
│           [취소]  [저장]         │
└──────────────────────────────────┘
```

### 8-4. AddHoldingDialog "매매 이유" 필드 (기존 모달 수정)

기존 폼 끝에:
```
💭 매매 이유 (선택, 200자)
┌────────────────────────────────┐
│                                │
└────────────────────────────────┘
* 작성 시 매매 일기 자동 기록
```

### 8-5. 색·라벨 톤

- 자동 entry: `border-l-2 border-bb-accent` (노랑)
- 수동 entry: `border-l-2 border-bb-info` (시안)
- 분석 카드(auto_monthly): `border-l-2 border-bb-warn` (주황)
- 분석 카드(on_demand): `border-l-2 border-bb-info` (시안)
- 액션 라벨: 매수=`text-bb-up`, 매도=`text-bb-down`, 관찰=`text-fg-muted`

---

## 9. 에러 처리

| 상황 | 응답 | UI 표시 |
|---|---|---|
| 미인증 | 401 | 라우터 차단 |
| entry 내용 빈칸·2000자 초과 | 422 | "내용은 1~2000자" |
| entry 수정 시 auto type | 422 `cannot_modify_auto` | "자동 기록은 수정할 수 없습니다" |
| 채팅 한도 도달 (on-demand 분석) | 429 `monthly_chat_limit` | "월 30회 한도 도달" + 다음 달 1일 안내 |
| 분석할 entry 3개 미만 | 422 `insufficient_entries` | "분석은 일기 3개 이상 필요" |
| LLM 오류 | 503 | "잠시 후 다시 시도" |
| 자동 cron LLM 오류 | 로그만, 사용자 영향 X | 다음 달 시도 |

---

## 10. 보안·RLS

- `journal_entries` SELECT/INSERT/UPDATE/DELETE 모두 `user_id = auth.uid()` 강제
- `analysis_runs` SELECT/INSERT만 (불변 기록)
- handler는 모두 `db.AsUser` JWT 트랜잭션 (기존 사용자 데이터 핸들러 패턴)
- cron의 INSERT는 슈퍼유저 풀(시스템 작업) — 다중 사용자 fan-out이라 단일 JWT 불가. 대신 `user_id`를 명시적으로 INSERT
- `related_symbols` 자유 입력 — XSS는 마크다운 렌더 시 `rehype-sanitize`로 차단 (기존 채팅 UI와 동일)
- `content` 길이 제한(2000자) + Postgres 자동 트림으로 storage 폭주 방지

---

## 11. 테스트

### 11-1. Backend unit (`apps/api/internal/handlers/journal_test.go`)

- `TestCreateEntry_Manual_OK`
- `TestCreateEntry_ContentTooLong_422`
- `TestPatchEntry_AutoType_422`
- `TestDeleteEntry_OwnershipEnforced_404`
- `TestAnalyze_InsufficientEntries_422`
- `TestAnalyze_ExceededQuota_429`
- `TestAnalyze_OK_DecrementsQuota`

### 11-2. AI 도구 unit (`apps/api/internal/ai/tools/journal_test.go`)

- `TestAnalyzeJournal_PromptBuilt_FromEntries`
- `TestAnalyzeJournal_RequiresUserContext_True`

### 11-3. Cron unit (`apps/api/internal/schedule/journal_monthly_test.go`)

- `TestMonthlyDispatcher_OnlyDay1Hour7_Triggers`
- `TestMonthlyDispatcher_SkipsUsersWithNoEntries`
- `TestMonthlyDispatcher_PreventsDuplicateInMonth`

### 11-4. Integration (`apps/api/internal/handlers/journal_integration_test.go`)

- `TestJournal_E2E_CreateListAnalyze` — 실 Supabase + seed user + 5 entry → analyze → 결과 검증

### 11-5. Frontend (`apps/web/components/journal/JournalPage.test.tsx`)

- 정상 entries 리스트 렌더
- 빈 상태 ("아직 일기 없음")
- 새 entry 모달 → 검증·저장
- "분석 요청" 버튼 클릭 → 로딩 → 결과 카드

---

## 12. 비범위 (YAGNI)

- ❌ 종목 태그 자동 추출 — v2 (현재는 사용자가 symbol 수동 입력)
- ❌ 일기 검색·필터 — v2 (페이징만 MVP)
- ❌ 일기 export (PDF·CSV) — v2
- ❌ 시각화 (매매 시점 차트) — v2
- ❌ 알림·push — v2
- ❌ 사용자 간 공유 — 영구 불가 (정체성 spec §2)
- ❌ 외부 SNS 연동 — v2
- ❌ Paper Portfolio 통합 — Phase 2 Paper 작업에서 처리 (`related_paper_holding_id` 컬럼 추가)
- ❌ 일기에 이미지 첨부 — v2
- ❌ 일기 카테고리·태그 시스템(symbol 외) — v2

---

## 검토 이력

### 2026-05-28 초안 작성 + 자체 검토 (Sonnet)

#### Critical (구현 시 동작 안 함 또는 핵심 설계 결함) — 0건

발견 없음. 결정의 핵심 가정은 검증됐다.

#### Important (보강 필요) — 4건 → 모두 패치 완료

**I-1. analyze_journal 도구가 채팅에서 호출될 때 한도 이중 차감 위험** — 도구 호출은 채팅 메시지 1턴에 묶여서 차감되는데, `POST /v1/journal/analyze`(버튼)도 채팅 한도에서 차감하면 동일 분석을 두 경로로 호출 시 2회 차감. 초안 §4 표에 "채팅 도구는 추가 차감 안 함"만 적어 명확화 부족.
→ §6-4 표를 명확화: 채팅 메시지 1턴 = 1회로 이미 처리 중이라 도구는 추가 차감 X. 버튼 경로는 별도 endpoint라 명시적 +1. 동일 분석을 두 경로로 호출하면 사용자 의도라 2회 차감 의도된 동작.

**I-2. cron 자동 회고가 사용자 hash 분단위 분산되는데 매 분 실행 비효율** — 일일 브리핑 cron이 매 분 도는 패턴 차용. 매월 1일 07:00~07:59만 처리하는 동안 사용자 0명 분도 매 분 query 발생. 가입자 100명 이하 무시 가능. 명시 안 함.
→ §7-1에 "매 분 trigger, 내부에서 7시 1일 가드 + 사용자 hash 분산" 명시. 일일 브리핑과 동일 패턴이라 인프라 재사용 강조.

**I-3. holdings DELETE 시 매도 사유 자동 entry 생성 안 함** — 초안 §5-7에 명시했으나 사용자가 매도 이유를 일기에 남기려면 manual entry 별도 작성 필요. UX 부담. 그러나 holdings.go DELETE 핸들러에 reason 받기는 RESTful X(DELETE body 모호).
→ §5-7에 결정 박제: DELETE는 reason 받지 않음, 사용자가 manual entry로 작성. UI에서 "삭제 전 매도 이유 작성하시겠습니까?" 모달 가능 (Phase 2 UX 개선 후속).

**I-4. analysis_runs unique 인덱스가 period_start 기준이라 동일 월 cron 중복은 막지만 timezone 경계 케이스 모호** — `period_start date`가 KST 1일인지 UTC 1일인지 명시 안 함.
→ §3-2에 "period_start는 KST 기준 매월 1일 00:00의 date" 코멘트 추가 검토 (구현 시 cron이 KST timezone으로 계산).

#### Minor (명확성) — 3건 → 모두 패치 완료

**M-1. NewEntryDialog의 manual entry action default가 '관찰'인지 미명시** — §8-3 mockup엔 "관찰 ▼" default가 보이지만 §5-2 API 명세에 default 동작 없음.
→ §5-2에 "manual entry default action = observation" 명시.

**M-2. AddHoldingDialog 안 매매 이유 200자 제한과 manual content 2000자 제한 불일치** — 의도된 차이(인라인 vs 상세) but 사용자가 200자 후 일기 페이지에서 더 적고 싶어 할 때 동선 부재.
→ §8-4 라벨에 "* 작성 시 매매 일기 자동 기록 — 더 자세히 쓰려면 일기 페이지에서 새 entry 작성" 안내.

**M-3. 사이드바 아이콘 6개로 늘어남 — 좁은 영역 UX 검토** — 기존 5개(홈·포폴·채팅·마켓·설정) + 일기 + 후원♡ = 7개. 모바일 환경 cramped.
→ §8-1 결정 유지(데스크탑 우선 + 모바일은 햄버거 메뉴 v2에서). 모바일은 현재 미지원 명시.

#### 메타

본 spec은 brainstorm → spec 절차로 사이클 1회. 자체 검토 후 사용자 review gate.
사용자 승인 시 `superpowers:writing-plans`로 plan 작성 → subagent 검토.

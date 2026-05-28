-- 매매 일기 entries
create table public.journal_entries (
  id                   uuid primary key default gen_random_uuid(),
  user_id              uuid not null references auth.users(id) on delete cascade,
  entry_type           text not null check (entry_type in ('auto', 'manual')),
  action               text check (action in ('buy', 'sell', 'observation', 'other')),
  related_holding_id   uuid references public.holdings(id) on delete set null,
  related_symbols      text[] not null default '{}',
  title                text,
  content              text not null check (length(content) between 1 and 2000),
  created_at           timestamptz not null default now(),
  updated_at           timestamptz not null default now()
);

create index journal_entries_user_created_idx
  on public.journal_entries (user_id, created_at desc);

create trigger journal_entries_touch_updated_at
  before update on public.journal_entries
  for each row execute function public.touch_updated_at();

-- AI 분석 결과 (불변)
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

-- 동일 월 auto_monthly 중복 방지
create unique index analysis_runs_auto_monthly_idx
  on public.analysis_runs (user_id, period_start)
  where run_type = 'auto_monthly';

-- RLS
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

alter table public.analysis_runs enable row level security;
create policy analysis_runs_select_own on public.analysis_runs
  for select using (user_id = auth.uid());
create policy analysis_runs_insert_own on public.analysis_runs
  for insert with check (user_id = auth.uid());

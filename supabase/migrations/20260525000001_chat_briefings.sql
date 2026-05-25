-- 채팅 세션
create table public.chat_sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users(id) on delete cascade,
  title text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index chat_sessions_user_updated_idx on public.chat_sessions (user_id, updated_at desc);

create trigger chat_sessions_touch
  before update on public.chat_sessions
  for each row execute function public.touch_updated_at();

-- 채팅 메시지
create table public.chat_messages (
  id uuid primary key default gen_random_uuid(),
  session_id uuid not null references public.chat_sessions(id) on delete cascade,
  role text not null check (role in ('user', 'assistant', 'tool')),
  content text not null default '',
  tool_calls jsonb,
  input_tokens int not null default 0,
  output_tokens int not null default 0,
  model text,
  finished_at timestamptz,
  created_at timestamptz not null default now()
);
create index chat_messages_session_created_idx on public.chat_messages (session_id, created_at);

-- 사용량 캐시 (월별)
create table public.chat_usage_monthly (
  user_id uuid not null references auth.users(id) on delete cascade,
  year_month text not null,
  chat_count int not null default 0,
  input_tokens int not null default 0,
  output_tokens int not null default 0,
  opus_count int not null default 0,
  primary key (user_id, year_month)
);

-- 일일 브리핑
create table public.ai_briefings (
  user_id uuid not null references auth.users(id) on delete cascade,
  date date not null,
  content_md text not null,
  model text not null,
  created_at timestamptz not null default now(),
  primary key (user_id, date)
);

-- RLS
alter table public.chat_sessions       enable row level security;
alter table public.chat_messages       enable row level security;
alter table public.chat_usage_monthly  enable row level security;
alter table public.ai_briefings        enable row level security;

create policy "chat_sessions_select_own" on public.chat_sessions
  for select using (auth.uid() = user_id);
create policy "chat_sessions_insert_own" on public.chat_sessions
  for insert with check (auth.uid() = user_id);
create policy "chat_sessions_update_own" on public.chat_sessions
  for update using (auth.uid() = user_id) with check (auth.uid() = user_id);
create policy "chat_sessions_delete_own" on public.chat_sessions
  for delete using (auth.uid() = user_id);

create policy "chat_messages_select_own" on public.chat_messages
  for select using (
    exists (select 1 from public.chat_sessions s
            where s.id = session_id and s.user_id = auth.uid())
  );
create policy "chat_messages_insert_own" on public.chat_messages
  for insert with check (
    exists (select 1 from public.chat_sessions s
            where s.id = session_id and s.user_id = auth.uid())
  );

create policy "chat_usage_monthly_select_own" on public.chat_usage_monthly
  for select using (auth.uid() = user_id);

create policy "ai_briefings_select_own" on public.ai_briefings
  for select using (auth.uid() = user_id);

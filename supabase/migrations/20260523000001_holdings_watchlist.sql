-- holdings: 사용자 보유 자산
create table public.holdings (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users(id) on delete cascade,
  instrument_id uuid not null references public.instruments(id) on delete cascade,
  quantity numeric(20, 8) not null check (quantity > 0),
  avg_cost numeric(20, 4) not null check (avg_cost >= 0),
  opened_at date,
  note text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint holdings_user_instrument_unique unique (user_id, instrument_id)
);

create index holdings_user_created_idx on public.holdings (user_id, created_at desc);

create trigger holdings_touch
  before update on public.holdings
  for each row execute function public.touch_updated_at();

-- watchlist: 관심 종목
create table public.watchlist (
  user_id uuid not null references auth.users(id) on delete cascade,
  instrument_id uuid not null references public.instruments(id) on delete cascade,
  added_at timestamptz not null default now(),
  primary key (user_id, instrument_id)
);

create index watchlist_user_idx on public.watchlist (user_id, instrument_id);

-- RLS
alter table public.holdings  enable row level security;
alter table public.watchlist enable row level security;

create policy "holdings_select_own" on public.holdings
  for select using (auth.uid() = user_id);
create policy "holdings_insert_own" on public.holdings
  for insert with check (auth.uid() = user_id);
create policy "holdings_update_own" on public.holdings
  for update using (auth.uid() = user_id) with check (auth.uid() = user_id);
create policy "holdings_delete_own" on public.holdings
  for delete using (auth.uid() = user_id);

create policy "watchlist_select_own" on public.watchlist
  for select using (auth.uid() = user_id);
create policy "watchlist_insert_own" on public.watchlist
  for insert with check (auth.uid() = user_id);
create policy "watchlist_delete_own" on public.watchlist
  for delete using (auth.uid() = user_id);

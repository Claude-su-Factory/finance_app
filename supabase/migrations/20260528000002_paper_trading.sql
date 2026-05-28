-- 사용자당 단일 paper account
create table public.paper_account (
  user_id        uuid primary key references auth.users(id) on delete cascade,
  initial_cash   numeric(20,2) not null default 10000000,
  cash_balance   numeric(20,2) not null default 10000000,
  base_currency  text not null default 'KRW' check (base_currency in ('KRW')),
  created_at     timestamptz not null default now(),
  updated_at     timestamptz not null default now()
);

create trigger paper_account_touch_updated_at
  before update on public.paper_account
  for each row execute function public.touch_updated_at();

-- 가상 보유
create table public.paper_holdings (
  id            uuid primary key default gen_random_uuid(),
  user_id       uuid not null references auth.users(id) on delete cascade,
  instrument_id uuid not null references public.instruments(id),
  quantity      numeric(20,6) not null check (quantity > 0),
  avg_cost      numeric(20,6) not null check (avg_cost >= 0),
  created_at    timestamptz not null default now(),
  updated_at    timestamptz not null default now(),
  unique (user_id, instrument_id)
);

create index paper_holdings_user_idx on public.paper_holdings (user_id);

create trigger paper_holdings_touch_updated_at
  before update on public.paper_holdings
  for each row execute function public.touch_updated_at();

-- 가상 매매 이력 (불변; 리셋 시 active=false)
create table public.paper_transactions (
  id            uuid primary key default gen_random_uuid(),
  user_id       uuid not null references auth.users(id) on delete cascade,
  instrument_id uuid not null references public.instruments(id),
  action        text not null check (action in ('buy', 'sell')),
  quantity      numeric(20,6) not null check (quantity > 0),
  price         numeric(20,6) not null check (price >= 0),
  currency      text not null,
  fx_to_krw     numeric(20,6) not null check (fx_to_krw > 0),
  total_krw     numeric(20,2) not null,
  active        boolean not null default true,
  created_at    timestamptz not null default now()
);

-- 동일 timestamp 결정성 보장 위해 id 포함
create index paper_transactions_user_created_idx
  on public.paper_transactions (user_id, created_at desc, id desc);

-- RLS
alter table public.paper_account enable row level security;
create policy paper_account_select_own on public.paper_account
  for select using (user_id = auth.uid());
create policy paper_account_insert_own on public.paper_account
  for insert with check (user_id = auth.uid());
create policy paper_account_update_own on public.paper_account
  for update using (user_id = auth.uid())
  with check (user_id = auth.uid());

alter table public.paper_holdings enable row level security;
create policy paper_holdings_select_own on public.paper_holdings
  for select using (user_id = auth.uid());
create policy paper_holdings_insert_own on public.paper_holdings
  for insert with check (user_id = auth.uid());
create policy paper_holdings_update_own on public.paper_holdings
  for update using (user_id = auth.uid())
  with check (user_id = auth.uid());
create policy paper_holdings_delete_own on public.paper_holdings
  for delete using (user_id = auth.uid());

alter table public.paper_transactions enable row level security;
create policy paper_transactions_select_own on public.paper_transactions
  for select using (user_id = auth.uid());
create policy paper_transactions_insert_own on public.paper_transactions
  for insert with check (user_id = auth.uid());
create policy paper_transactions_update_own on public.paper_transactions
  for update using (user_id = auth.uid())
  with check (user_id = auth.uid());

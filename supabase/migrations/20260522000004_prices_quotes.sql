-- 일봉 시계열 (PriceBar)
create table public.prices (
  instrument_id uuid not null references public.instruments(id) on delete cascade,
  date date not null,
  open numeric(20,6) not null,
  high numeric(20,6) not null,
  low  numeric(20,6) not null,
  close numeric(20,6) not null,
  volume bigint not null default 0,
  primary key (instrument_id, date)
);
create index prices_date_idx on public.prices (instrument_id, date desc);

-- 직전 시세 캐시 (분 단위 폴링)
create table public.quotes (
  instrument_id uuid primary key references public.instruments(id) on delete cascade,
  price numeric(20,6) not null,
  change_abs numeric(20,6) not null default 0,
  change_pct numeric(8,4) not null default 0,
  updated_at timestamptz not null default now()
);
create index quotes_updated_idx on public.quotes (updated_at desc);

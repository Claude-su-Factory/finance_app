-- 종목 마스터 (KR_STOCK, US_STOCK, ETF, CASH, INDEX, FX)
create table public.instruments (
  id uuid primary key default gen_random_uuid(),
  symbol text not null,
  exchange text not null,
  isin text,                                       -- KR 종목 ISIN (선택). KIND는 노출 안 함 → NULL
  name text not null,
  asset_class text not null check (asset_class in ('KR_STOCK','US_STOCK','ETF','CASH','INDEX','FX')),
  currency text not null check (currency in ('KRW','USD')),
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint instruments_symbol_exchange_unique unique (symbol, exchange)
);
create index instruments_active_class_idx on public.instruments (asset_class) where is_active = true;

-- 종목 별칭 (한글·영문·티커 매핑)
create table public.instrument_aliases (
  alias text primary key,
  instrument_id uuid not null references public.instruments(id) on delete cascade,
  source text not null default 'seed' check (source in ('seed','learned')),
  created_at timestamptz not null default now()
);
create index instrument_aliases_inst_idx on public.instrument_aliases (instrument_id);

-- updated_at 자동 갱신 (W1에서 정의된 함수 재사용)
create trigger instruments_touch
  before update on public.instruments
  for each row execute function public.touch_updated_at();

-- 시드: 핵심 지수·환율 (TopTicker용)
insert into public.instruments (symbol, exchange, name, asset_class, currency) values
  ('KOSPI',   'KRX-IDX',   'KOSPI 종합',   'INDEX', 'KRW'),
  ('KOSDAQ',  'KRX-IDX',   'KOSDAQ 종합',  'INDEX', 'KRW'),
  ('SPX',     'NYSE-IDX',  'S&P 500',      'INDEX', 'USD'),
  ('NDX',     'NASDAQ-IDX','NASDAQ 100',   'INDEX', 'USD'),
  ('USD_KRW', 'FX',        'USD/KRW',      'FX',    'KRW'),
  ('EUR_KRW', 'FX',        'EUR/KRW',      'FX',    'KRW'),
  ('JPY_KRW', 'FX',        'JPY/KRW',      'FX',    'KRW');

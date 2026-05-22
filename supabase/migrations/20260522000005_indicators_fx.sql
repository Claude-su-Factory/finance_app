-- 경제 지표 (금리·실업률·CPI 등)
create table public.economic_indicators (
  code text not null,
  observed_at timestamptz not null,
  name text not null,
  value numeric(20,6) not null,
  unit text,
  primary key (code, observed_at)
);
create index indicators_code_obs_idx on public.economic_indicators (code, observed_at desc);

-- 환율 시계열 (별도 테이블)
create table public.fx_rates (
  base text not null,
  quote text not null,
  observed_at timestamptz not null,
  rate numeric(20,8) not null,
  primary key (base, quote, observed_at)
);
create index fx_rates_pair_obs_idx on public.fx_rates (base, quote, observed_at desc);

alter table public.instruments         enable row level security;
alter table public.instrument_aliases  enable row level security;
alter table public.prices              enable row level security;
alter table public.quotes              enable row level security;
alter table public.economic_indicators enable row level security;
alter table public.fx_rates            enable row level security;

-- 인증 사용자: 읽기 허용
create policy market_read_inst on public.instruments         for select to authenticated using (true);
create policy market_read_alia on public.instrument_aliases  for select to authenticated using (true);
create policy market_read_prc  on public.prices              for select to authenticated using (true);
create policy market_read_qte  on public.quotes              for select to authenticated using (true);
create policy market_read_ind  on public.economic_indicators for select to authenticated using (true);
create policy market_read_fx   on public.fx_rates            for select to authenticated using (true);

-- service_role: 전체 쓰기 명시 (W3 RLS 강화 시에도 안전)
create policy market_write_inst on public.instruments         for all to service_role using (true) with check (true);
create policy market_write_alia on public.instrument_aliases  for all to service_role using (true) with check (true);
create policy market_write_prc  on public.prices              for all to service_role using (true) with check (true);
create policy market_write_qte  on public.quotes              for all to service_role using (true) with check (true);
create policy market_write_ind  on public.economic_indicators for all to service_role using (true) with check (true);
create policy market_write_fx   on public.fx_rates            for all to service_role using (true) with check (true);

-- RLS 활성화
alter table public.profiles enable row level security;

-- 본인 row 조회
create policy "profiles_select_own"
  on public.profiles for select
  using (auth.uid() = id);

-- 본인 row 갱신
create policy "profiles_update_own"
  on public.profiles for update
  using (auth.uid() = id)
  with check (auth.uid() = id);

-- INSERT는 트리거가 처리하므로 정책 없음 (service_role만 가능)
-- DELETE도 정책 없음 (계정 삭제는 별도 Admin API 흐름)

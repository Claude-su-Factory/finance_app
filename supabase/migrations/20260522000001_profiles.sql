-- profiles 테이블: auth.users와 1:1
create table public.profiles (
  id uuid primary key references auth.users(id) on delete cascade,
  display_name text,
  base_currency text not null default 'KRW' check (base_currency in ('KRW', 'USD')),
  ui_intensity text not null default 'standard' check (ui_intensity in ('vivid', 'standard', 'subtle')),
  onboarding_completed boolean not null default false,
  daily_briefing_enabled boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- auth.users insert 시 profiles row 자동 생성
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
begin
  insert into public.profiles (id, display_name)
  values (new.id, coalesce(new.raw_user_meta_data->>'name', split_part(new.email, '@', 1)));
  return new;
end;
$$;

create trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_user();

-- updated_at 자동 갱신
create or replace function public.touch_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end $$;

create trigger profiles_touch
  before update on public.profiles
  for each row execute function public.touch_updated_at();

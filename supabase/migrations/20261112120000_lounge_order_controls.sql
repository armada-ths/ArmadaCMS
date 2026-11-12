alter table public.fair_date_configs
  add column lounge_order_open_delay_minutes integer not null default 0
    check (lounge_order_open_delay_minutes between 0 and 1440),
  add column lounge_order_force_open boolean not null default false;

create table public.lounge_order_link_settings (
  id integer primary key check (id = 1),
  token text not null default '',
  revision bigint not null default 0
);

alter table public.lounge_order_link_settings enable row level security;
revoke all on public.lounge_order_link_settings from public, anon, authenticated;
insert into public.lounge_order_link_settings (id) values (1);

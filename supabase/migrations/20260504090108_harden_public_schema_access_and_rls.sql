-- Harden ArmadaCMS application tables in the exposed `public` schema.
--
-- Rationale:
-- - Supabase exposes the `public` schema through its Data API.
-- - ArmadaCMS currently talks to Postgres directly via the Go backend rather than
--   using the Supabase Data API for these tables.
-- - We therefore want a deny-by-default posture for `anon`, `authenticated`, and
--   `service_role` on the application tables until we intentionally design and
--   document policies for any table we choose to expose later.

revoke all privileges on all tables in schema public from anon, authenticated, service_role;
revoke all privileges on all sequences in schema public from anon, authenticated, service_role;

alter default privileges for role postgres in schema public
	revoke select, insert, update, delete, truncate, references, trigger on tables
	from anon, authenticated, service_role;

alter default privileges for role postgres in schema public
	revoke usage, select, update on sequences
	from anon, authenticated, service_role;

alter table public.audit_logs enable row level security;
alter table public.blogposts enable row level security;
alter table public.employments enable row level security;
alter table public.events enable row level security;
alter table public.exhibitor_employments enable row level security;
alter table public.exhibitor_industries enable row level security;
alter table public.exhibitor_programs enable row level security;
alter table public.exhibitors enable row level security;
alter table public.fair_date_configs enable row level security;
alter table public.feature_flags enable row level security;
alter table public.highlight_cards enable row level security;
alter table public.industries enable row level security;
alter table public.profiles enable row level security;
alter table public.programs enable row level security;
alter table public.recruitment_periods enable row level security;
alter table public.recruitment_roles enable row level security;
alter table public.refresh_tokens enable row level security;
alter table public.roles enable row level security;
alter table public.teams enable row level security;
alter table public.users enable row level security;

-- ArmadaCMS local seed scaffold.
--
-- Current bootstrap responsibilities are still split:
--   1. Supabase CLI applies this file during `supabase db reset`
--   2. ArmadaCMS startup still performs idempotent seeders for roles,
--      feature flags, and the optional initial admin user.
--
-- Keep this file safe and deterministic. It should only contain local/test data
-- that is acceptable in branch databases and on developer machines.
--
-- Preview branches are intentionally deferred until a later phase, together
-- with per-PR GCP services and billing-backed Supabase branching.

insert into public.roles (name, permissions)
values
	('admin', '["*"]'),
	('member', '["profiles.list","profiles.show","profiles.create","profiles.edit","teams.list","teams.show"]')
on conflict (name) do update
set permissions = excluded.permissions;

insert into public.feature_flags (key, description, enabled)
values
	('EVENT_PAGE', 'Show the student events page', true),
	('MAP_PAGE', 'Show the fair map page', true),
	('AT_FAIR_PAGE', 'Show the at-the-fair student page', true),
	('EXHIBITOR_PACKAGES', 'Show the exhibitor packages page', true),
	('EXHIBITOR_EVENTS', 'Show the exhibitor events page', true),
	('EXHIBITOR_PAGE', 'Show the student exhibitors/companies page', true),
	('STUDENT_RECRUITMENT_PAGE', 'Show the student recruitment page', true),
	('EXHIBITOR_MAIN_PAGE', 'Show the exhibitor main/why armada page', true),
	('EXHIBITOR_TIMELINE_PAGE', 'Show the exhibitor timeline page', true),
	('EXHIBITOR_SIGNUP_PAGE', 'Show the exhibitor signup/registration page (only controls the topnav link, the page itself is controlled by the exhibitor timeline).', true),
	('ABOUT_PAGE', 'Show the about armada page', true),
	('ABOUT_TEAM_PAGE', 'Show the about team page', true),
	('ARMADA_BLOG_PAGE', 'Show the armada blog page', false)
on conflict (key) do update
set
	description = excluded.description,
	enabled = excluded.enabled;

-- The initial admin user remains in Go startup for now because it depends on
-- environment variables and should not be committed as deterministic SQL seed data.

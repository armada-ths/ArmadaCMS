-- ArmadaCMS local seed scaffold.
--
-- This file is applied by Supabase automatically on every push/merge to tracked
-- branches, on `supabase db reset`, and when creating a new Supabase branch.
-- It is safe and deterministic: it only contains local/test data acceptable in
-- branch databases and on developer machines. It never runs against an existing
-- production or staging branch.

insert into public.roles (name, permissions)
values
	('admin', '["*"]')
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

-- Initial admin user (username: admin, password: admin).
-- Uses pgcrypto bcrypt so the hash is compatible with Go\'s golang.org/x/crypto/bcrypt.
-- Only inserted if no user with username \'admin\' already exists.
insert into public.users (username, password)
select
	'admin',
	crypt('admin', gen_salt('bf', 10))
where not exists (select 1 from public.users where username = 'admin');

-- Assign the admin role to the admin user (idempotent).
insert into public.user_roles (user_id, role_id)
select u.id, r.id
from public.users u
join public.roles r on r.name = 'admin'
where u.username = 'admin'
on conflict do nothing;

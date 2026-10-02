-- Add nullable Swedish (*_sv) translation columns so public-site content can
-- be localized (armada-ths/armada.nu#295). Existing rows are unaffected;
-- English columns remain the source of truth until a Swedish value is set.

ALTER TABLE public.blogposts
  ADD COLUMN IF NOT EXISTS title_sv text,
  ADD COLUMN IF NOT EXISTS text_sv text;

ALTER TABLE public.events
  ADD COLUMN IF NOT EXISTS name_sv text,
  ADD COLUMN IF NOT EXISTS description_sv text;

ALTER TABLE public.exhibitors
  ADD COLUMN IF NOT EXISTS about_sv text;

ALTER TABLE public.teams
  ADD COLUMN IF NOT EXISTS team_name_sv text;

ALTER TABLE public.recruitment_roles
  ADD COLUMN IF NOT EXISTS name_sv text,
  ADD COLUMN IF NOT EXISTS description_sv text;

ALTER TABLE public.employments
  ADD COLUMN IF NOT EXISTS name_sv text;

ALTER TABLE public.industries
  ADD COLUMN IF NOT EXISTS name_sv text;

ALTER TABLE public.fair_date_configs
  ADD COLUMN IF NOT EXISTS description_sv text;

ALTER TABLE public.programs
  ADD COLUMN IF NOT EXISTS name_sv text;

-- Catch-up: models/highlightcard.go already defines TitleSv/SubtitleSv/
-- DescriptionSv (branch feature/highlightcard-bilingual-fields), but no
-- migration ever added these columns, so they never persisted anywhere.
ALTER TABLE public.highlight_cards
  ADD COLUMN IF NOT EXISTS title_sv text,
  ADD COLUMN IF NOT EXISTS subtitle_sv text,
  ADD COLUMN IF NOT EXISTS description_sv text;

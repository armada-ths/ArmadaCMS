ALTER TABLE public.events ADD COLUMN IF NOT EXISTS location_sv text, ADD COLUMN IF NOT EXISTS open_to_sv text, ADD COLUMN IF NOT EXISTS food_sv text, ADD COLUMN IF NOT EXISTS fee_sv text;
ALTER TABLE public.profiles ADD COLUMN IF NOT EXISTS title_sv text;
ALTER TABLE public.recruitment_periods ADD COLUMN IF NOT EXISTS name_sv text;
ALTER TABLE public.timeline_entries ADD COLUMN IF NOT EXISTS title_sv text, ADD COLUMN IF NOT EXISTS body_sv text;
ALTER TABLE public.timeline_eras ADD COLUMN IF NOT EXISTS title_sv text;

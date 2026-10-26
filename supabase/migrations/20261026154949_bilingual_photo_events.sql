ALTER TABLE public.photo_events
 ADD COLUMN IF NOT EXISTS name_sv text,
 ADD COLUMN IF NOT EXISTS description_sv text;

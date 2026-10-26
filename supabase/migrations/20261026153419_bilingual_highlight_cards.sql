-- Optional Swedish copy; existing English content remains unchanged.
ALTER TABLE public.highlight_cards
  ADD COLUMN IF NOT EXISTS title_sv text,
  ADD COLUMN IF NOT EXISTS subtitle_sv text,
  ADD COLUMN IF NOT EXISTS description_sv text,
  ADD COLUMN IF NOT EXISTS link_text_sv text;

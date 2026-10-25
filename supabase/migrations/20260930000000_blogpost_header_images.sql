-- Additional ordered header photos; the existing cover stays first.
ALTER TABLE public.blogposts
    ADD COLUMN IF NOT EXISTS image_urls jsonb NOT NULL DEFAULT '[]'::jsonb;

-- Add published flag to blogposts so posts can be saved as drafts
-- without being visible on the public site.
-- Existing posts default to published = true so nothing changes for them.

ALTER TABLE public.blogposts
  ADD COLUMN IF NOT EXISTS published boolean NOT NULL DEFAULT true;

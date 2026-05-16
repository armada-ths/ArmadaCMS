-- Add a unique constraint on public.users.username.
-- Usernames must be unique to ensure unambiguous login.
-- If duplicates exist (they shouldn't), this will fail intentionally.

CREATE UNIQUE INDEX IF NOT EXISTS users_username_key ON public.users (username);

ALTER TABLE public.users
  ADD CONSTRAINT users_username_key UNIQUE USING INDEX users_username_key;

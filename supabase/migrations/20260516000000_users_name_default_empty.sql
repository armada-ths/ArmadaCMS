-- Make users.name optional by giving it an empty-string default.
-- The column stays NOT NULL so Go code can keep using a plain string,
-- but inserts that omit name (e.g. the seed admin user) will succeed.

ALTER TABLE public.users ALTER COLUMN name SET DEFAULT '';

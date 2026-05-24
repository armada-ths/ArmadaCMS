-- Replace the single role_id FK on users with a many-to-many join table.
-- This allows each user to be assigned multiple roles whose permissions are
-- merged at login/token-refresh time.

-- 1. Create the join table that GORM will recognise via the many2many tag.
CREATE TABLE public.user_roles (
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    role_id bigint NOT NULL REFERENCES public.roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- 2. Backfill: carry every existing single-role assignment into the new table.
INSERT INTO public.user_roles (user_id, role_id)
SELECT id, role_id
FROM public.users
WHERE role_id IS NOT NULL;

-- 3. Drop the old FK constraint and column.
ALTER TABLE public.users DROP CONSTRAINT IF EXISTS fk_users_role;
ALTER TABLE public.users DROP COLUMN IF EXISTS role_id;

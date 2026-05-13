-- Fix schema gaps discovered during RDS → Supabase rehearsal restore.
-- These columns/tables exist in RDS but were missing from the snapshot migration.

-- profiles: add missing columns
ALTER TABLE public.profiles
    ADD COLUMN IF NOT EXISTS photo_url text;
ALTER TABLE public.profiles
    ADD COLUMN IF NOT EXISTS photo_file text;

-- recruitment_roles: add missing columns
ALTER TABLE public.recruitment_roles
    ADD COLUMN IF NOT EXISTS parent text;
ALTER TABLE public.recruitment_roles
    ADD COLUMN IF NOT EXISTS group_name text;

-- users: add role column (legacy field, kept alongside role_id)
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS role text DEFAULT 'admin'::text NOT NULL;

-- timeline_dates: missing table entirely
CREATE TABLE IF NOT EXISTS public.timeline_dates (
    id   integer NOT NULL,
    date text    NOT NULL,
    title text   NOT NULL,
    details text
);

CREATE SEQUENCE IF NOT EXISTS public.timeline_dates_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.timeline_dates_id_seq OWNED BY public.timeline_dates.id;

ALTER TABLE ONLY public.timeline_dates
    ALTER COLUMN id SET DEFAULT nextval('public.timeline_dates_id_seq'::regclass);

ALTER TABLE ONLY public.timeline_dates
    ADD CONSTRAINT timeline_dates_pkey PRIMARY KEY (id);

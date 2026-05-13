-- Rewrite S3 image URLs to Supabase Storage URLs.
--
-- Run ONCE per environment, after all S3 files have been copied to the
-- Supabase Storage bucket, and BEFORE flipping STORAGE_PROVIDER=supabase
-- in HCP Terraform.
--
-- Safe to re-run: REPLACE() on a string that no longer contains the old
-- prefix is a no-op.
--
-- Production S3 bucket prefix  : https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/
-- Staging S3 bucket prefix     : https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/
-- Supabase Storage public URL  : https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/

BEGIN;

-- ── blogposts.image_url ───────────────────────────────────────────────────────

UPDATE public.blogposts
SET image_url = REPLACE(
    image_url,
    'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/',
    'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'
)
WHERE image_url LIKE '%armada-cms-files-e48105192c52.s3%';

UPDATE public.blogposts
SET image_url = REPLACE(
    image_url,
    'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/',
    'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'
)
WHERE image_url LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%';

-- ── blogposts.text (inline markdown images) ───────────────────────────────────

UPDATE public.blogposts
SET text = REPLACE(
    text,
    'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/',
    'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'
)
WHERE text LIKE '%armada-cms-files-e48105192c52.s3%';

UPDATE public.blogposts
SET text = REPLACE(
    text,
    'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/',
    'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'
)
WHERE text LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%';

-- ── events.image_url ─────────────────────────────────────────────────────────

UPDATE public.events
SET image_url = REPLACE(
    image_url,
    'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/',
    'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'
)
WHERE image_url LIKE '%armada-cms-files-e48105192c52.s3%';

UPDATE public.events
SET image_url = REPLACE(
    image_url,
    'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/',
    'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'
)
WHERE image_url LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%';

-- ── exhibitors: logo_squared_url, logo_freesize_url, map_img, flyer ──────────

UPDATE public.exhibitors
SET
    logo_squared_url  = REPLACE(logo_squared_url,  'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    logo_freesize_url = REPLACE(logo_freesize_url, 'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    map_img           = REPLACE(map_img,           'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    flyer             = REPLACE(flyer,             'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/')
WHERE
    logo_squared_url  LIKE '%armada-cms-files-e48105192c52.s3%'
    OR logo_freesize_url LIKE '%armada-cms-files-e48105192c52.s3%'
    OR map_img           LIKE '%armada-cms-files-e48105192c52.s3%'
    OR flyer             LIKE '%armada-cms-files-e48105192c52.s3%';

UPDATE public.exhibitors
SET
    logo_squared_url  = REPLACE(logo_squared_url,  'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    logo_freesize_url = REPLACE(logo_freesize_url, 'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    map_img           = REPLACE(map_img,           'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    flyer             = REPLACE(flyer,             'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/')
WHERE
    logo_squared_url  LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%'
    OR logo_freesize_url LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%'
    OR map_img           LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%'
    OR flyer             LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%';

-- ── profiles: photo, photo_url, photo_file ────────────────────────────────────

UPDATE public.profiles
SET
    photo      = REPLACE(photo,      'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    photo_url  = REPLACE(photo_url,  'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    photo_file = REPLACE(photo_file, 'https://armada-cms-files-e48105192c52.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/')
WHERE
    photo      LIKE '%armada-cms-files-e48105192c52.s3%'
    OR photo_url  LIKE '%armada-cms-files-e48105192c52.s3%'
    OR photo_file LIKE '%armada-cms-files-e48105192c52.s3%';

UPDATE public.profiles
SET
    photo      = REPLACE(photo,      'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    photo_url  = REPLACE(photo_url,  'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/'),
    photo_file = REPLACE(photo_file, 'https://armada-cms-files-staging-b3f79a2e1d84.s3.eu-north-1.amazonaws.com/', 'https://rsdjnixgxqauonaofrwr.supabase.co/storage/v1/object/public/armadacms-files/')
WHERE
    photo      LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%'
    OR photo_url  LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%'
    OR photo_file LIKE '%armada-cms-files-staging-b3f79a2e1d84.s3%';

-- ── Verification query — run after COMMIT to confirm no S3 URLs remain ────────
-- SELECT 'blogposts.image_url' AS location, COUNT(*) FROM public.blogposts WHERE image_url LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'blogposts.text',          COUNT(*) FROM public.blogposts WHERE text         LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'events.image_url',         COUNT(*) FROM public.events    WHERE image_url   LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'exhibitors.logo_squared',  COUNT(*) FROM public.exhibitors WHERE logo_squared_url LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'exhibitors.logo_freesize', COUNT(*) FROM public.exhibitors WHERE logo_freesize_url LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'exhibitors.map_img',       COUNT(*) FROM public.exhibitors WHERE map_img    LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'exhibitors.flyer',         COUNT(*) FROM public.exhibitors WHERE flyer      LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'profiles.photo',           COUNT(*) FROM public.profiles  WHERE photo       LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'profiles.photo_url',       COUNT(*) FROM public.profiles  WHERE photo_url   LIKE '%amazonaws.com%'
-- UNION ALL SELECT 'profiles.photo_file',      COUNT(*) FROM public.profiles  WHERE photo_file  LIKE '%amazonaws.com%';

COMMIT;

ALTER TABLE public.exhibitors ADD COLUMN logo_thumbnail text;

CREATE TABLE public.eventro_sync_states (
    resource text NOT NULL,
    record_id bigint NOT NULL,
    PRIMARY KEY (resource, record_id),
    fair_id text NOT NULL,
    source jsonb NOT NULL DEFAULT '{}',
    overrides jsonb NOT NULL DEFAULT '{}',
    conflicts jsonb NOT NULL DEFAULT '{}',
    unmapped_products jsonb NOT NULL DEFAULT '[]'
);

-- Only the CMS backend accesses import metadata through its protected API.
ALTER TABLE public.eventro_sync_states ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON public.eventro_sync_states FROM anon, authenticated;

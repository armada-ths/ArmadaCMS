ALTER TABLE public.fair_date_configs
    ADD COLUMN IF NOT EXISTS eventro_id text;

CREATE UNIQUE INDEX IF NOT EXISTS fair_date_configs_eventro_id_key
    ON public.fair_date_configs (eventro_id);

ALTER TABLE public.fair_date_configs
    DROP COLUMN IF EXISTS ticket_end,
    DROP COLUMN IF EXISTS events_end;

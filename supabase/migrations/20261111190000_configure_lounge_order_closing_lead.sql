ALTER TABLE public.fair_date_configs
    ADD COLUMN lounge_order_close_minutes integer NOT NULL DEFAULT 30
    CHECK (lounge_order_close_minutes BETWEEN 0 AND 1440);

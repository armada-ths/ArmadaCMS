ALTER TABLE public.fair_date_configs
    ADD COLUMN day_one_opens_at text NOT NULL DEFAULT '10:00',
    ADD COLUMN day_one_closes_at text NOT NULL DEFAULT '16:00',
    ADD COLUMN day_two_opens_at text NOT NULL DEFAULT '10:00',
    ADD COLUMN day_two_closes_at text NOT NULL DEFAULT '15:00',
    ADD CONSTRAINT fair_date_configs_opening_hours_check CHECK (
        day_one_opens_at ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
        AND day_one_closes_at ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
        AND day_two_opens_at ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
        AND day_two_closes_at ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
        AND day_one_opens_at < day_one_closes_at
        AND day_two_opens_at < day_two_closes_at
    );

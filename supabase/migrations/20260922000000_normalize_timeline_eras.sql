-- Existing staging entries have no year and cannot be migrated reliably.
DELETE FROM timeline_entries;

CREATE TABLE timeline_eras (
    id         SERIAL PRIMARY KEY,
    title      TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE timeline_entries
    ADD COLUMN era_id INTEGER NOT NULL,
    ADD CONSTRAINT timeline_entries_era_id_fkey
        FOREIGN KEY (era_id) REFERENCES timeline_eras(id) ON DELETE RESTRICT,
    DROP COLUMN era,
    DROP COLUMN era_title;

CREATE INDEX timeline_entries_era_position_idx
    ON timeline_entries (era_id, sort_order, id);

INSERT INTO public.feature_flags (key, description, enabled)
VALUES (
    'ARMADA_TIMELINE_PAGE',
    'Show the Armada history timeline page',
    true
)
ON CONFLICT (key) DO NOTHING;

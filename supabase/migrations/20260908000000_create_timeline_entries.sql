CREATE TABLE timeline_entries (
    id         SERIAL PRIMARY KEY,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    era        TEXT NOT NULL,
    era_title  TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

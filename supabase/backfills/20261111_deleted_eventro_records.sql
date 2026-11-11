BEGIN;

SELECT pg_advisory_xact_lock(hashtextextended('eventro:sync-domain:events', 0));
SELECT pg_advisory_xact_lock(hashtextextended('eventro:sync-domain:exhibitors', 0));

WITH approved(resource, eventro_key, label, deleted_at) AS (
    VALUES
        ('events', 'b0192380-0fe0-4255-af5e-779832620d94', 'Valoriim After Work', TIMESTAMPTZ '2026-11-09 11:11:43.084269+00'),
        ('exhibitors', 'ff7bcb16-e226-4622-a33a-530791ec6450', 'Sigma Industry', TIMESTAMPTZ '2026-11-09 09:30:41.682086+00'),
        ('exhibitors', 'ddc621b3-929e-4f48-9ea9-3b384b4fcd81', 'Task Team Ten', TIMESTAMPTZ '2026-11-09 09:30:32.8382+00')
), checked AS (
    SELECT approved.*,
        CASE resource
            WHEN 'events' THEN EXISTS (SELECT 1 FROM public.events WHERE eventro_id = approved.eventro_key)
            WHEN 'exhibitors' THEN EXISTS (SELECT 1 FROM public.exhibitors WHERE eventro_id = approved.eventro_key)
        END AS present
    FROM approved
), inserted AS (
    INSERT INTO public.eventro_sync_blocks (resource, eventro_key, label, deleted_at)
    SELECT resource, eventro_key, label, deleted_at
    FROM checked
    WHERE NOT present
    ON CONFLICT (resource, eventro_key) DO NOTHING
    RETURNING resource, eventro_key
)
SELECT checked.resource, checked.label,
       CASE WHEN present THEN 'still in CMS'
            WHEN inserted.eventro_key IS NOT NULL THEN 'blocked'
            ELSE 'already blocked' END AS status
FROM checked LEFT JOIN inserted USING (resource, eventro_key)
ORDER BY checked.resource, checked.label;

COMMIT;

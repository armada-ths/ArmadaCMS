-- Preserve the exact access previously granted by the shared lounge permissions.
UPDATE public.roles AS role
SET permissions = (
    SELECT jsonb_agg(DISTINCT COALESCE(mapping.replacement, existing.permission))::text
    FROM jsonb_array_elements_text(role.permissions::jsonb) AS existing(permission)
    LEFT JOIN (VALUES
        ('lounge.*',      'lounge-catalogue.*'),
        ('lounge.*',      'lounge-orders.*'),
        ('lounge.view',   'lounge-catalogue.view'),
        ('lounge.view',   'lounge-orders.view'),
        ('lounge.create', 'lounge-catalogue.create'),
        ('lounge.edit',   'lounge-catalogue.edit'),
        ('lounge.edit',   'lounge-orders.edit'),
        ('lounge.delete', 'lounge-catalogue.delete')
    ) AS mapping(previous, replacement)
        ON existing.permission = mapping.previous
)
WHERE role.permissions::jsonb ?| ARRAY['lounge.*', 'lounge.view', 'lounge.create', 'lounge.edit', 'lounge.delete'];

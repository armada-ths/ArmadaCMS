-- Unify "resource.list" and "resource.show" permission strings into "resource.view".
-- Permissions are stored as a JSON text array in the roles table.
-- Duplicates (where both .list and .show existed) are collapsed to a single .view entry.
UPDATE public.roles
SET permissions = (
  SELECT json_agg(DISTINCT unified)::text
  FROM (
    SELECT
      CASE
        WHEN perm LIKE '%.list' THEN LEFT(perm, LENGTH(perm) - 4) || 'view'
        WHEN perm LIKE '%.show' THEN LEFT(perm, LENGTH(perm) - 4) || 'view'
        ELSE perm
      END AS unified
    FROM json_array_elements_text(permissions::json) AS perm
  ) sub
)
WHERE permissions IS NOT NULL AND permissions != '[]';

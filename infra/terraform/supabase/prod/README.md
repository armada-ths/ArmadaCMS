# ArmadaCMS Terraform — Supabase production

This Terraform root manages the **hosted Supabase production project** for `ArmadaCMS`.
It is designed to **import the already-created production project** and then manage a
small, explicit subset of platform settings in Git.

Today this root imports and manages:

- the Supabase project record itself via `supabase_project.production`
- exported connection metadata for future cross-workspace use

It intentionally does **not** yet manage preview branches, Edge Functions, custom API
keys, or hosted secrets. Those can be added later in small, reviewable steps.

## What it manages

| File                 | Resources / purpose                                                                                           |
| -------------------- | ------------------------------------------------------------------------------------------------------------- |
| `versions.tf`        | Terraform version and `supabase/supabase` provider                                                            |
| `variables.tf`       | Required org/password inputs plus non-secret project defaults                                                 |
| `locals.tf`          | Derived project URL, DB host, pooler user, and staging DB host                                                |
| `project.tf`         | Imports and manages `supabase_project.production`                                                             |
| `settings.tf`        | `data.supabase_pooler` — pooler URLs; `supabase_settings` intentionally **not** managed (see comments inside) |
| `outputs.tf`         | Exports project metadata and DB connection details                                                            |
| `prod.auto.tfvars`   | Committed non-secret defaults for the current production project                                              |
| `backend.tf.example` | HCP Terraform backend template                                                                                |

## Architecture notes

- This root is the Supabase equivalent of the old production database infrastructure root.
- `prevent_destroy = true` is enabled on the imported production project resource.
- ArmadaCMS connects to Postgres via its Go API using direct DB/pooler connections. It does
  not use Supabase Auth, PostgREST, Realtime, or Edge Functions.
- The provider requires `database_password` in configuration, but the Management API does
  **not** return it on import. You must provide the current password (or intentionally reset
  it in the dashboard first).
- The root exports pooler and staging DB connection details consumed by the GCP workspaces.
- Staging is a separate Supabase project. Its DB host, user, and name are stored as
  variables here and exported so `gcp/staging` can read them via `tfe_outputs` without
  hardcoding.

## Workspace dependencies

This root has no cross-workspace state dependencies.

For the full cross-workspace layout, see [`../../README.md`](../../README.md).

## HCP Terraform workspace setup

Workspace: `armadacms-supabase-prod` in the `THS-Armada` organization.

Copy `backend.tf.example` to `backend.tf`, fill in the workspace name, and run
`terraform init`.

### Environment variables

Set these in the HCP Terraform workspace:

| Variable                | Notes                                                              |
| ----------------------- | ------------------------------------------------------------------ |
| `SUPABASE_ACCESS_TOKEN` | Sensitive env var; Personal Access Token for the Supabase provider |

### Terraform variables

Set these in the HCP Terraform workspace:

| Variable            | Sensitive | Notes                                                                  |
| ------------------- | --------- | ---------------------------------------------------------------------- |
| `organization_id`   | No        | Supabase **organization slug** from Organization Settings              |
| `database_password` | Yes       | Current production DB password; required because import cannot read it |

Everything else has committed non-secret defaults in `prod.auto.tfvars`.

## Console setup before first apply

### In the Supabase dashboard

1. Open **Account preferences → Access Tokens** and create a Personal Access Token for Terraform.
2. Open **Organization Settings** and copy the **organization slug** for `organization_id`.
3. Open **Project Settings → General** and confirm the project ref is `rsdjnixgxqauonaofrwr`.
4. Confirm the project name is `ArmadaCMS` and the region is `eu-north-1`.
5. Obtain the current database password. If you no longer know it, reset it in the dashboard first, then use the new value for Terraform.
6. Optionally review the current API settings (`db_schema`, search path, max rows) so future Terraform changes do not surprise you. This root currently observes but does not manage those settings.

### In HCP Terraform

1. Create the workspace `armadacms-supabase-prod`.
2. Add the sensitive environment variable `SUPABASE_ACCESS_TOKEN`.
3. Add Terraform variable `organization_id` with the Supabase organization slug.
4. Add sensitive Terraform variable `database_password` with the current DB password.
5. Queue a run. The `import {}` block in `project.tf` will import the existing project into state on the first apply instead of trying to recreate it.

## Ongoing operations

### Importing additional managed settings later

A safe follow-up pattern is:

1. Import the relevant resource into state.
2. Start by managing only a small subset of settings in Terraform.
3. Apply and verify no unexpected drift.
4. Expand the managed surface gradually.

### If you later want REST or GraphQL access

Only opt schemas into `api.db_schema` when you intentionally want to expose them through the
Supabase Data API / GraphQL surface.

- Add the schema explicitly (for example `public` or `public,graphql_public`).
- Add explicit `GRANT`s for the roles that should reach those objects.
- Keep RLS enabled and add table-specific policies.

Do **not** expose the `storage` schema just because Storage is in use; the Storage service works
through its own API and treats the underlying schema as implementation detail / read-only metadata.

When you intentionally want Terraform to start managing project settings again, reintroduce a
`supabase_settings` resource in `settings.tf`, import it, and apply in a reviewed run.

### Rotating the database password

If you intentionally rotate the database password in Supabase:

1. Reset it in the Supabase dashboard.
2. Update the sensitive `database_password` Terraform variable in HCP Terraform.
3. Queue a new apply so Terraform state matches reality.
4. Update every runtime that uses the DB password (`gcp/prod`, local secrets, CI, etc.).

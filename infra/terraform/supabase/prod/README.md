# ArmadaCMS Terraform — Supabase production

This Terraform root manages the **hosted Supabase production project** for `ArmadaCMS`.
It is designed to **import the already-created production project** and then manage a
small, explicit subset of platform settings in Git.

Today this root imports and manages:

- the Supabase project record itself via `supabase_project.production`
- a safe subset of project settings via `supabase_settings.production`
  - `api.db_schema`
  - `api.db_extra_search_path`
  - `api.max_rows`
- exported connection metadata for future cross-workspace use

It intentionally does **not** yet manage preview branches, Edge Functions, custom API
keys, or hosted secrets. Those can be added later in small, reviewable steps.

## What it manages

| File                 | Resources / purpose                                                   |
| -------------------- | --------------------------------------------------------------------- |
| `versions.tf`        | Terraform version and `supabase/supabase` provider                    |
| `variables.tf`       | Required org/password inputs plus non-secret project defaults         |
| `locals.tf`          | Derived project URL, DB host, and managed API settings payload        |
| `project.tf`         | Imports and manages `supabase_project.production`                     |
| `settings.tf`        | Imports and manages `supabase_settings.production`; reads pooler URLs |
| `outputs.tf`         | Exports project metadata and DB connection details                    |
| `prod.auto.tfvars`   | Committed non-secret defaults for the current production project      |
| `backend.tf.example` | HCP Terraform backend template                                        |

## Architecture notes

- This root is the Supabase equivalent of the old production database infrastructure root.
- `prevent_destroy = true` is enabled on the imported production project resource.
- The Supabase provider performs **partial updates** for `supabase_settings`, so only the
  settings declared here are managed; everything else remains unchanged.
- ArmadaCMS currently talks to Postgres through its Go API and direct database connections,
  not through Supabase REST or GraphQL endpoints. This root therefore leaves
  `api.db_schema` **unset by default**, which means Terraform does not try to change
  the project's existing exposed-schema setting unless you opt in explicitly.
- The current `supabase/supabase` provider performs a REST-service health precheck before
  updating `supabase_settings`. On this project that probe can false-fail even while the
  dashboard shows the project as healthy, so `api` changes are temporarily ignored in this
  root until the provider behavior is improved or we intentionally revisit API management.
- The provider requires `database_password` in configuration, but the Management API does
  **not** return it on import. You must provide the current password (or intentionally reset
  it in the dashboard first).
- The root exports `db_host`, `db_name`, `db_user`, and `pooler_urls` so the future
  production runtime cutover can consume them through `data.tfe_outputs` instead of the
  AWS RDS workspace.

## Workspace dependencies

This root has no upstream Terraform workspace dependencies.

Right now nothing else reads this workspace yet. Once `gcp/prod` is updated to consume
Supabase DB outputs, grant `armadacms-gcp-prod` remote state read access under
**Settings → Remote state sharing** in HCP Terraform.

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
6. Optionally review the current API settings (`db_schema`, search path, max rows) so the first Terraform run does not surprise you. The checked-in posture intentionally leaves `db_schema` unset because ArmadaCMS does not currently use the Data API.

### In HCP Terraform

1. Create the workspace `armadacms-supabase-prod`.
2. Add the sensitive environment variable `SUPABASE_ACCESS_TOKEN`.
3. Add Terraform variable `organization_id` with the Supabase organization slug.
4. Add sensitive Terraform variable `database_password` with the current DB password.
5. Queue a run. The checked-in `import {}` blocks will import both the project and its settings into state on the first apply instead of trying to recreate them.

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

When you intentionally want Terraform to start managing `api` settings again, remove the
`lifecycle.ignore_changes = [api]` workaround from `settings.tf`, then apply in a reviewed run.

### Rotating the database password

If you intentionally rotate the database password in Supabase:

1. Reset it in the Supabase dashboard.
2. Update the sensitive `database_password` Terraform variable in HCP Terraform.
3. Queue a new apply so Terraform state matches reality.
4. Update every runtime that uses the DB password (`gcp/prod`, local secrets, CI, etc.).

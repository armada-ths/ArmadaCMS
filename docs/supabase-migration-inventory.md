# Supabase migration inventory

This document is the working checklist for phases 0-1 of the AWS-to-Supabase migration.
It intentionally excludes paid Supabase branching work for now.

## Live platform inventory

- AWS account / caller identity: `arn:aws:iam::593054043164:root`
- Production RDS instance identifier: `terraform-20250411194126172200000001`
- Production RDS engine/version: `postgres 17.5`
- Production RDS endpoint: `terraform-20250411194126172200000001.cg6cp6bnc2ao.eu-north-1.rds.amazonaws.com`
- Production S3 bucket name: `armada-cms-files-e48105192c52`
- Production S3 approximate object count: `126`
- Production S3 approximate total size: `253212480` bytes
- Staging S3 bucket name: `armada-cms-files-staging-b3f79a2e1d84`
- Staging S3 approximate object count: `8`
- Staging S3 approximate total size: `14592435` bytes
- Old standalone staging Supabase project ref: `yfybmnqzclpmpncyfmdc` — **to be decommissioned after GCP staging cutover is verified**
- Production Supabase project ref (`ArmadaCMS`): `rsdjnixgxqauonaofrwr` (region: North EU Stockholm) — **single parent project**
- Hosted staging branch: `dqeikqjiztvmifmnbzbf` (name: `staging`) — branch of `rsdjnixgxqauonaofrwr`; direct host `db.dqeikqjiztvmifmnbzbf.supabase.co`; pooler `aws-1-eu-north-1.pooler.supabase.com` username `postgres.dqeikqjiztvmifmnbzbf`
- Supabase plan tier / billing status: billing configured; hosted branching active

## Terraform and runtime wiring inventory

- GCP prod workspace: `armadacms-gcp-prod`
- GCP staging workspace: `armadacms-gcp-staging`
- Supabase prod workspace: `armadacms-supabase-prod`
- AWS prod workspace: `armadacms-aws-prod`
- AWS staging workspace: `armadacms-aws-staging`
- Current cross-workspace dependencies to remove later:
  - `gcp/prod` reads RDS host/DB name and S3 bucket values from `aws/prod`
  - `aws/prod` reads Cloud Run static egress IP from `gcp/prod`
  - `gcp/staging` reads S3 bucket values from `aws/staging`
- New standalone Terraform root for the hosted production Supabase project: `infra/terraform/supabase/prod`
- Current Cloud Run env vars sourced from AWS outputs:
  - production: `DB_HOST`, `DB_NAME`, `S3_BUCKET`, `AWS_REGION`
  - staging: `S3_BUCKET`, `AWS_REGION`
- Current Cloud Run secrets sourced from Secret Manager:
  - production: `DB_PASSWORD`, `jwtsecret_laganda`, `EVENTRO_*`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
  - staging: `DB_PASSWORD`, `jwtsecret_laganda`, `EVENTRO_*`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `INITIAL_ADMIN_PASSWORD`

## Local-development inventory

- Supabase CLI version: `2.98.1`
- Docker version: `4.71.0`
- Local DB runtime: Docker Compose (Postgres) — permanent, not migrating to Supabase CLI
- Local storage runtime: Docker Compose (MinIO) — permanent, not migrating to Supabase CLI
- Supabase CLI role: migration authoring (`supabase db diff`), validation (`supabase db reset`), and remote pushes (`supabase db push`) only
- Checked-in Supabase storage bucket migration: `supabase/migrations/20260504092802_create_public_storage_bucket.sql`
- Application schema inventory for the next migration phase: `docs/supabase-app-schema-inventory.md`
- Current SQL bootstrap ownership:
  - `supabase/seed.sql` seeds deterministic roles and feature flags for remote environment resets and CI
  - the optional initial admin user still comes from Go startup because it depends on environment variables
- Remaining schema migration work:
  - `DB_ENABLE_AUTOMIGRATE` needs to be set to `false` in production/staging once checked-in migration parity is confirmed (the application schema snapshot plus the gap-fix migration are now the trusted source of truth)

## DB rehearsal — completed 2026-05-13

A full end-to-end restore rehearsal was performed from the live RDS production snapshot to both Supabase environments. Both targets now contain the same production data as RDS.

**Validated restore procedure (for real cutover):**

1. `npx supabase db push --db-url <target-url>` — apply any pending migrations

2. ```sql
   TRUNCATE TABLE audit_logs, blogposts, employments, exhibitor_employments,
   exhibitor_industries, exhibitor_programs, exhibitors, events,
   fair_date_configs, feature_flags, highlight_cards, industries, profiles,
   programs, recruitment_periods, recruitment_roles, refresh_tokens, roles,
   teams, timeline_dates, users RESTART IDENTITY CASCADE;
   ```

3. `pg_restore --data-only --no-privileges --no-owner --file=restore-data.sql <dump-file>`
4. Prepend `SET session_replication_role = replica;` and append `SET session_replication_role = default;` to the SQL file
5. `psql -f restore-data-nofk.sql` — load data without FK constraint ordering failures

**Expected harmless errors:** `setval` calls for legacy RDS sequences (`goadmin_*`, `blogpost_id_seq`, `blogpost_tags_id_seq`, `refresh_token_dbs_id_seq`, `test_id_seq`, `token_id_seq`, `untitled_table_209_id_seq`, `user_id_seq`, `users_id_seq1`) — these tables do not exist in the new schema and their sequence errors are safe to ignore.

**Schema gaps discovered and fixed** — migration `supabase/migrations/20260513000000_fix_schema_gaps.sql` adds:

- `profiles.photo_url text`
- `profiles.photo_file text`
- `recruitment_roles.parent text`
- `recruitment_roles.group_name text`
- `users.role text DEFAULT 'admin' NOT NULL`
- `public.timeline_dates` table (with sequence and primary key)

**Row counts validated (both staging and production Supabase match RDS):**

| table      | count |
| ---------- | ----- |
| exhibitors | 121   |
| events     | 14    |
| industries | 19    |
| profiles   | 18    |
| roles      | 3     |
| teams      | 4     |
| users      | 5     |

## Cutover planning

- Allowed production maintenance window:
- Rollback deadline:
- Smoke-test checklist owner:
- Infra change approver:
- App deploy approver:
- Database migration approver:
- Storage migration approver:

## Migration phase status

| Phase | Description                                                       | Status                                                                                                      |
| ----- | ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| 0     | Live inventory and assumption freeze                              | ✅ Complete                                                                                                 |
| 1     | Target environment model locked                                   | ✅ Complete                                                                                                 |
| 2     | Supabase-as-code and migration discipline                         | ✅ Complete                                                                                                 |
| 3     | Application storage abstraction (provider-neutral upload service) | ✅ Complete (code ready, not yet switched in production)                                                    |
| 4     | Supabase CLI documented as migration tool only                    | ✅ Complete                                                                                                 |
| 5     | Supabase Terraform root scaffolded                                | ✅ Scaffolded (import still pending)                                                                        |
| 6     | Runtime configuration model defined                               | 🔄 In progress (staging wired, production wiring pending cutover)                                           |
| 7     | Preview-environment workflow (hosted staging branch)              | 🔄 In progress — branch created and data loaded; GCP wiring pending                                         |
| 8     | DB migration rehearsal                                            | ✅ Complete — both staging and production Supabase validated 2026-05-13                                     |
| 9     | Storage migration rehearsal                                       | ⏳ Pending                                                                                                  |
| 10    | Staging cutover                                                   | 🔄 In progress — branch ready; update HCP Terraform `armadacms-gcp-staging` workspace variables (see below) |
| 11    | Production cutover                                                | ⏳ Pending                                                                                                  |
| 12    | AWS decommission                                                  | ⏳ Pending                                                                                                  |

## Staging branch migration — 2026-05-13

The standalone Supabase staging project (`yfybmnqzclpmpncyfmdc`) is being replaced by a hosted branch on the production project.

**Completed:**

- Branch `dqeikqjiztvmifmnbzbf` (name: `staging`) created on `rsdjnixgxqauonaofrwr`
- All 5 migrations applied (branch inherits from parent)
- Production data snapshot restored and validated (row counts match)
- `infra/terraform/gcp/staging/variables.tf` default `db_host` updated to `db.dqeikqjiztvmifmnbzbf.supabase.co`

**Pending — update HCP Terraform `armadacms-gcp-staging` workspace variables:**

| Variable                       | Old value                             | New value                             |
| ------------------------------ | ------------------------------------- | ------------------------------------- |
| `db_host`                      | `db.yfybmnqzclpmpncyfmdc.supabase.co` | `db.dqeikqjiztvmifmnbzbf.supabase.co` |
| `secret_values["DB_PASSWORD"]` | `88CIA8KeW8BQmcGP`                    | `SrBXofKLsCxZEzMnpUEZtHviBnzLlXim`    |

(`db_user` and `db_name` stay `postgres`)

**After verifying GCP staging app works against the branch:**

- Delete old standalone staging project `yfybmnqzclpmpncyfmdc` via Supabase dashboard (user confirms)

## Next phase

- Ephemeral Supabase preview branches for PRs
- Per-PR GCP preview services
- Required GitHub checks for hosted Supabase previews

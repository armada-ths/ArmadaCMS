# Post-migration cleanup checklist

Items to clean up once the AWS → Supabase migration is **fully complete** (DB cutover + storage cutover + AWS decommission). Do not do these early; they serve as the migration audit trail until then.

## Files to delete

| File                                      | Why                                                                            |
| ----------------------------------------- | ------------------------------------------------------------------------------ |
| `docs/supabase-migration-inventory.md`    | Operator checklist — no longer relevant once migration is done                 |
| `docs/supabase-app-schema-inventory.md`   | Snapshot taken to plan the migration — superseded by the checked-in migrations |
| `docs/post-migration-cleanup.md`          | This file — delete it once all items are done                                  |
| `plan-awsExitToSupabase.prompt.md`        | Prompt used during planning — not a runtime artifact                           |
| `scripts/rewrite-s3-urls-to-supabase.sql` | One-time URL rewrite script — no longer needed after storage cutover           |
| `infra/terraform/aws/prod/README.md`      | Deleted with the entire `aws/prod/` root                                       |
| `infra/terraform/aws/staging/README.md`   | Deleted with the entire `aws/staging/` root                                    |

## Terraform README rewrites

These README files stay but have sections that need updating after migration:

### `infra/terraform/gcp/prod/README.md`

1. **"Storage cutover notes" section** — delete. Once `storage_provider = "supabase"` is live and AWS is decommissioned, the cutover instructions are no longer needed.
2. **Credential rotation section for S3** — remove the `storage_provider = "s3"` rotation steps; keep only the Supabase key rotation steps (and rename that section since it becomes the only case).
3. **Root layout table** — remove the `aws/prod/` row and the cross-workspace state sharing note about reading `rds_host` and `s3_bucket_name` from `armadacms-aws-prod`.

### `infra/terraform/gcp/staging/README.md`

1. **"Storage cutover notes" section** — delete (same reason as prod).
2. **Credential rotation section for S3** — remove the `storage_provider = "s3"` rotation steps.
3. **References to the old standalone staging project** — search for `yfybmnqzclpmpncyfmdc` and remove any remaining mentions.

### `infra/terraform/supabase/prod/README.md`

1. **"In HCP Terraform" step 5** — remove the note about the `import {}` block running on first apply; after the first apply the block will have fired and should be deleted from `project.tf`, so the note no longer applies.
2. **Delete `import {}` block from `infra/terraform/supabase/prod/project.tf`** after the first successful apply that imports the project into state.

### `infra/terraform/README.md`

1. **Root layout table** — remove the `aws/prod` and `aws/staging` rows once those roots are decommissioned.
2. **Cross-workspace state sharing notes** — remove references to `armadacms-aws-prod` and `armadacms-aws-staging`.

## README.md rewrites

1. **"Supabase migration groundwork" section** — delete the entire section. Replace with a short "Database migrations" section that just explains the ongoing `supabase db push` workflow for future contributors.
2. **"What is intentionally deferred" block** (inside the Supabase section) — delete once branching and preview environments are live.
3. **Infrastructure section** — change `"migrating from AWS RDS to Supabase (rehearsal complete; cutover pending)"` to just `"Supabase"`.
4. **File storage line** — change `"AWS S3 (local dev: MinIO) — storage migration pending"` to `"Supabase Storage (local dev: MinIO)"`.
5. **Step 6 "Set up file uploads"** — simplify: remove the AWS S3 subsection and the note about `S3_ENDPOINT`/`S3_PUBLIC_URL` duality; only MinIO (local) and Supabase Storage (remote) need to be documented.

## .env.example rewrites

1. **Database section comment** — remove the `DB_ENABLE_AUTOMIGRATE` block (or change it to simply `DB_ENABLE_AUTOMIGRATE=false` once parity is confirmed). Remove the "transitioning from runtime AutoMigrate" note.
2. **Storage section** — make Supabase Storage the primary documented remote config (move it before the AWS block, uncomment it with the real production project ref). Keep MinIO as the local default.
3. **AWS S3 block** — remove the staging/production AWS S3 variables (`S3_BUCKET`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` for the remote case). The MinIO block (same variable names) stays for local dev.
4. **`MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD`** — these are Docker Compose variables, not app variables. Consider moving them to a separate `# Docker Compose only` comment block or a `.env.docker-only.example` to reduce confusion.

## Terraform cleanup

### GCP roots

1. **`infra/terraform/gcp/staging/aws_state.tf`** — remove the `data "tfe_outputs" "aws_staging"` data source and any references to it in `locals.tf` once `storage_provider = "supabase"` (no more S3 bucket reads needed).
2. **`infra/terraform/gcp/prod/aws_state.tf`** (equivalent file) — same: remove once `storage_provider = "supabase"` in prod.
3. **`infra/terraform/gcp/staging/staging.auto.tfvars`** and **`infra/terraform/gcp/prod/prod.auto.tfvars`** — remove the `storage_provider = "s3"` lines (or leave as `storage_provider = "supabase"`); remove the commented-out Supabase storage block (uncomment it and make it live instead).
4. **`infra/terraform/gcp/prod/variables.tf`** — remove `db_host` default that references RDS once the prod cutover is done and prod reads from `supabase/prod` outputs instead.
5. **`infra/terraform/gcp/prod/locals.tf`** and **`infra/terraform/gcp/staging/locals.tf`** — remove the `managed_secrets` local (and revert `secrets.tf` to use `local.secret_env_vars` again) once AWS decommission is complete and rollback is no longer needed. At that point `secret_env_vars` and `managed_secrets` are identical.
6. **`infra/terraform/gcp/prod/secrets.tf`** and **`infra/terraform/gcp/staging/secrets.tf`** — revert `for_each = local.managed_secrets` back to `for_each = local.secret_env_vars` once item 5 above is done.

### AWS roots (decommission entirely)

Once S3 is drained and RDS is gone, the following can be deleted:

- `infra/terraform/aws/prod/` — the entire directory
- `infra/terraform/aws/staging/` — the entire directory
- HCP Terraform workspaces `armadacms-aws-prod` and `armadacms-aws-staging` — destroy resources, then delete workspaces

### Cross-workspace state sharing

After AWS decommission, remove the `data "tfe_outputs"` blocks in the GCP roots that read from the AWS workspaces. At that point, all DB and storage connection values come from HCP Terraform workspace variables directly (or from `supabase/prod` outputs).

## GCP Secret Manager secrets to delete

Once AWS decommission is done, delete these secrets from the GCP project (they will no longer be referenced by Cloud Run or managed by `managed_secrets`):

- `armadacms-AWS_ACCESS_KEY_ID`
- `armadacms-AWS_SECRET_ACCESS_KEY`
- `armadacms-staging-AWS_ACCESS_KEY_ID`
- `armadacms-staging-AWS_SECRET_ACCESS_KEY`

Also remove these from `managed_secrets` in both `locals.tf` files at the same time (or just collapse `managed_secrets` back into `secret_env_vars` per the Terraform cleanup note above).

### Import blocks

After the first successful apply of the `supabase/prod` root that imports the existing Supabase project into state, delete the `import {}` block from `infra/terraform/supabase/prod/project.tf`. The block is only needed for the initial import; after that it would be a no-op and could cause confusion if left in.

## Script update

**`scripts/import-remote-db.ps1`** — currently documents cloning from a reachable PostgreSQL instance. Once RDS is decommissioned, update the script's comments to point at the Supabase pooler as the source rather than RDS. The script itself (pg_dump + psql) works against any PostgreSQL; only the example env var values need updating.

## Gitignore cleanup

Remove the following entries that were added for migration purposes (after the files are removed and no longer needed for audit trail):

```env
*.dump
rehearsal-schema.sql
restore-data.sql
restore-data-nofk.sql

s3-export/
```

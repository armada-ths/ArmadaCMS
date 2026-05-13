# Post-migration cleanup checklist

Items to clean up once the AWS → Supabase migration is **fully complete** (DB cutover + storage cutover + AWS decommission). Do not do these early; they serve as the migration audit trail until then.

## Files to delete

| File                                    | Why                                                                            |
| --------------------------------------- | ------------------------------------------------------------------------------ |
| `docs/supabase-migration-inventory.md`  | Operator checklist — no longer relevant once migration is done                 |
| `docs/supabase-app-schema-inventory.md` | Snapshot taken to plan the migration — superseded by the checked-in migrations |
| `plan-awsExitToSupabase.prompt.md`      | Prompt used during planning — not a runtime artifact                           |

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
3. **`infra/terraform/gcp/staging/staging.auto.tfvars`** and **`infra/terraform/gcp/prod/prod.auto.tfvars`** — remove the `storage_provider = "s3"` lines (or leave as `storage_provider = "supabase"`).
4. **`infra/terraform/gcp/prod/variables.tf`** — remove `db_host` default that references RDS once the prod cutover is done and prod reads from `supabase/prod` outputs instead.

### AWS roots (decommission entirely)

Once S3 is drained and RDS is gone, the following can be deleted:

- `infra/terraform/aws/prod/` — the entire directory
- `infra/terraform/aws/staging/` — the entire directory
- HCP Terraform workspaces `armadacms-aws-prod` and `armadacms-aws-staging` — destroy resources, then delete workspaces

### Cross-workspace state sharing

After AWS decommission, remove the `data "tfe_outputs"` blocks in the GCP roots that read from the AWS workspaces. At that point, all DB and storage connection values come from HCP Terraform workspace variables directly (or from `supabase/prod` outputs).

## GCP Secret Manager secrets to delete

Once AWS decommission is done, delete these secrets from the GCP project (they will no longer be referenced by Cloud Run):

- `armadacms-AWS_ACCESS_KEY_ID`
- `armadacms-AWS_SECRET_ACCESS_KEY`
- `armadacms-staging-AWS_ACCESS_KEY_ID`
- `armadacms-staging-AWS_SECRET_ACCESS_KEY`

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

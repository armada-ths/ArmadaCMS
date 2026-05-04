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
- Current standalone staging Supabase project ref: `yfybmnqzclpmpncyfmdc`
- Current standalone staging Supabase region: `North EU (Stockholm)`
- Intended production Supabase project ref (`ArmadaCMS`): `rsdjnixgxqauonaofrwr`
- Intended production Supabase region: `North EU (Stockholm)`
- Supabase plan tier / billing status: billing not configured yet; avoid hosted branching and other paid rollout steps for now

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
- Current local DB workflow owner (Docker Compose / Supabase CLI): Docker Compose today, with Supabase CLI now scaffolded for migration-driven DB work
- Current local storage workflow owner (MinIO / Supabase CLI): MinIO today
- Checked-in Supabase storage bucket migration: `supabase/migrations/20260504092802_create_public_storage_bucket.sql`
- Application schema inventory for the next migration phase: `docs/supabase-app-schema-inventory.md`
- Known blockers for moving local DB work to Supabase CLI:
  - `.env` / docs still default to the Docker Compose Postgres service
  - the repository now has both the hosted Supabase baseline migration and a first generated ArmadaCMS application schema snapshot, but that snapshot still needs manual review/cleanup before it becomes the trusted long-term source of truth (see `docs/supabase-app-schema-inventory.md`)
  - the backend still needs a documented and tested path for running against the local Supabase database instead of the Docker Compose Postgres service by default
  - `DB_ENABLE_AUTOMIGRATE` now makes runtime GORM schema writes optional, but production/staging still need an explicit rollout decision for when to disable it by default
- Current SQL bootstrap ownership:
  - `supabase/seed.sql` now seeds deterministic roles and feature flags
  - the optional initial admin user still comes from Go startup because it depends on environment variables
- Known blockers for moving local storage away from MinIO:
  - provider-neutral backend upload plumbing now exists, and the default public bucket is now described in checked-in SQL, but Supabase Storage still needs generated storage access keys in each environment
  - Supabase Storage bucket policy and long-term object-path strategy still need a follow-up decision

## Cutover planning

- Allowed production maintenance window:
- Rollback deadline:
- Smoke-test checklist owner:
- Infra change approver:
- App deploy approver:
- Database migration approver:
- Storage migration approver:

## Deferred until billing setup

- Persistent hosted Supabase `staging` branch
- Ephemeral Supabase preview branches
- Per-PR GCP preview services
- Required GitHub checks for hosted Supabase previews

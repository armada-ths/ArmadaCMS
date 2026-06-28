---
description: "Use when working with Terraform files for ArmadaCMS infrastructure. Covers the GCP/Supabase layout, HCP Terraform slow-plan pitfall, cross-workspace state sharing via tfe_outputs, and workspace naming conventions."
applyTo: "infra/terraform/**"
---

# Terraform — ArmadaCMS infrastructure

Reference: [`infra/terraform/README.md`](../../infra/terraform/README.md)

## Root layout

| Root             | HCP Terraform workspace   | What it manages                                                           |
| ---------------- | ------------------------- | ------------------------------------------------------------------------- |
| `gcp/prod/`      | `armadacms-gcp-prod`      | Cloud Run, VPC egress, HTTPS LB, Cloud Build, Secret Manager             |
| `supabase/prod/` | `armadacms-supabase-prod` | Imported hosted Supabase production project and selected project settings |
| `gcp/staging/`   | `armadacms-gcp-staging`   | Cloud Run (staging), VPC egress, domain mapping, Cloud Build, Secrets    |

Workspace naming pattern: `armadacms-<provider>-<environment>`.

## GCP root — avoid local `terraform plan` (slow)

The GCP root uses CLI-driven remote execution. Uploading the plan to HCP Terraform from a local machine is noticeably slow in this repo. Prefer **queueing plans from the HCP Terraform UI** instead.

Local commands that are always fine: `terraform validate`, `terraform fmt`, `terraform import`, targeted `terraform state` operations.

## Cross-workspace state sharing (critical)

Roots share live values via `data "tfe_outputs"` — **do not hardcode outputs from one root into another**.

- `supabase/prod` reads `static_egress_ip` from `armadacms-gcp-prod` to enforce network restrictions for direct DB access.
- `supabase/prod` exports `pooler_host`, `pooler_user`, and `db_name` consumed by `gcp/prod` to populate Cloud Run DB env vars.
- `gcp/staging` reads `staging_db_host`, `staging_db_user`, `staging_db_name` from `armadacms-supabase-prod` (staging DB is a branch in the same Supabase project).

For `data "tfe_outputs"` to work, **each workspace must be granted remote state read access to the other**. Configure this in HCP Terraform under each workspace's **Settings → Remote state sharing**. This is a one-time manual step and is required after creating a new workspace — it is not expressed in Terraform config.

## Secrets — GCP Secret Manager, not Terraform state

Runtime secrets for Cloud Run (DB password, JWT secret, Eventro credentials, Supabase storage keys) are stored in **GCP Secret Manager** and injected as environment variables. They are referenced by name in `locals.secret_env_vars` in `gcp/prod/locals.tf`.

**Workflow**: Terraform creates the Secret Manager resource (the empty shell). Secret _values_ are set **directly in GCP Secret Manager** — via the GCP console or `gcloud secrets versions add <secret-id> --data-file=-`. Terraform never writes secret values in practice: the `secret_values` variable is intentionally always left `{}` and the `google_secret_manager_secret_version` resource only fires when it is non-empty.

**Never** put secret values in `.tf`, `.tfvars`, or HCP Terraform workspace variables — they would end up in Terraform state. To rotate a secret, add a new version directly in Secret Manager; Cloud Run picks it up on next deploy without a Terraform apply.

## Conventions

- One Terraform root per logical stack; do not mix unrelated providers in the same root.
- One HCP Terraform workspace per environment per root.
- `backend.tf` is gitignored. Copy `backend.tf.example` → `backend.tf` and run `terraform init` to connect to the HCP workspace.

## Adding a new environment to an existing root

1. Create a new folder following the naming pattern (e.g., `gcp/staging/`).
2. Copy the relevant root as a starting point; update `backend.tf.example` with a new workspace name following `armadacms-<provider>-<env>`.
3. Create the HCP Terraform workspace and configure remote state sharing if the new root needs outputs from another workspace.

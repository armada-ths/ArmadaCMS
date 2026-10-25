---
description: "Use when working with Terraform files for ArmadaCMS infrastructure. Covers the GCP/Supabase layout, HCP Terraform slow-plan pitfall, cross-workspace state sharing via tfe_outputs, and workspace naming conventions."
applyTo: "infra/terraform/**"
---

# Terraform — ArmadaCMS infrastructure

Reference: [`infra/terraform/README.md`](../../infra/terraform/README.md)

## Root layout

| Root             | HCP Terraform workspace   | What it manages                                                                                                             |
| ---------------- | ------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `gcp/prod/`      | `armadacms-gcp-prod`      | Cloud Run, VPC egress, HTTPS LB, Actions federation, Secret Manager                                                         |
| `supabase/prod/` | `armadacms-supabase-prod` | Imported Supabase project record and exported connection metadata                                                           |
| `gcp/staging/`   | `armadacms-gcp-staging`   | Cloud Run, domain mapping, shared Artifact Registry, Actions federation, Secrets; optional VPC egress is currently disabled |
| `gcp/local/`     | `armadacms-gcp-local`     | Local-development reCAPTCHA and Maps keys; local execution, remote state                                                    |

Workspace naming pattern: `armadacms-<provider>-<environment>`.

## GCP root — avoid local `terraform plan` (slow)

The small `gcp/local` root uses local execution with the operator's Google credentials and HCP state storage. Its plans cannot modify hosted resources because it contains only developer reCAPTCHA and Maps keys.

The GCP root uses CLI-driven remote execution. Uploading the plan to HCP Terraform from a local machine is noticeably slow in this repo. Prefer **queueing plans from the HCP Terraform UI** instead.

Use `terraform validate` and `terraform fmt` for local checks. Import, state-write and apply operations should be
left to CI, not executed through the local CLI. The `gcp/local` root is the only exception, as it is not connected to any Git branch and is not used in CI.

## Cross-workspace state sharing (critical)

Roots share live values via `data "tfe_outputs"` — **do not hardcode outputs from one root into another**.

- `gcp/prod` reads DB and Supabase Storage outputs from `armadacms-supabase-prod` to populate Cloud Run env vars.
- `gcp/prod` exports `static_egress_ip`, which is allowlisted manually in Supabase. The current `supabase/prod` root does not read GCP state or manage network restrictions.
- `gcp/staging` reads `staging_db_host`, `staging_db_user`, and `staging_db_name` for the persistent Supabase staging branch, plus the shared production Storage configuration.

For `data "tfe_outputs"` to work, each GCP workspace must be allowed to read `armadacms-supabase-prod` state. This access is a workspace prerequisite and is not expressed in Terraform config.

## Secrets — GCP Secret Manager, not Terraform state

Runtime secrets for Cloud Run (DB password, JWT secret, Eventro credentials, Supabase storage keys, and the staging Vercel bypass secret) are stored in **GCP Secret Manager** and injected as environment variables. They are mapped in `locals.secret_env_vars` in each GCP root.

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

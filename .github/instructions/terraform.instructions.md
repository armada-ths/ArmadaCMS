---
description: "Use when working with Terraform files for ArmadaCMS infrastructure. Covers the GCP/AWS split, HCP Terraform slow-plan pitfall, cross-workspace state sharing via tfe_outputs, and workspace naming conventions."
applyTo: "infra/terraform/**"
---

# Terraform — ArmadaCMS infrastructure

Reference: [`infra/terraform/README.md`](../../infra/terraform/README.md)

## Root layout

| Root          | HCP Terraform workspace    | What it manages                                                       |
|---------------|----------------------------|-----------------------------------------------------------------------|
| `gcp/prod/`   | `armadacms-gcp-prod`       | Cloud Run, VPC egress, HTTPS LB, Cloud Build, Secret Manager          |
| `aws/prod/`   | `armadacms-aws-prod`       | RDS PostgreSQL, S3 bucket, IAM upload user                            |
| `gcp/staging/`| `armadacms-gcp-staging`    | Cloud Run (staging), VPC egress, domain mapping, Cloud Build, Secrets |
| `aws/staging/`| `armadacms-aws-staging`    | Staging S3 bucket, IAM upload user (no RDS — uses Supabase)          |

Workspace naming pattern: `armadacms-<provider>-<environment>`.

## GCP root — avoid local `terraform plan` (slow)

The GCP root uses CLI-driven remote execution. Uploading the plan to HCP Terraform from a local machine is noticeably slow in this repo. Prefer **queueing plans from the HCP Terraform UI** instead.

Local commands that are always fine: `terraform validate`, `terraform fmt`, `terraform import`, targeted `terraform state` operations.

## Cross-workspace state sharing (critical)

The two roots share live values via `data "tfe_outputs"` — **do not hardcode outputs from one root into the other**.

- `gcp/prod` reads `rds_host`, `rds_db_name`, `s3_bucket_name`, `s3_bucket_region` from `armadacms-aws-prod` → populates Cloud Run env vars.
- `aws/prod` reads `static_egress_ip` from `armadacms-gcp-prod` → restricts the RDS security group and S3 IAM policy.

For `data "tfe_outputs"` to work, **each workspace must be granted remote state read access to the other**. Configure this in HCP Terraform under each workspace's **Settings → Remote state sharing**. This is a one-time manual step and is required after creating a new workspace — it is not expressed in Terraform config.

## Secrets — GCP Secret Manager, not Terraform state

Runtime secrets for Cloud Run (DB password, JWT secret, Eventro credentials, AWS keys) are stored in **GCP Secret Manager** and injected as environment variables. They are referenced by name in `locals.secret_env_vars` in `gcp/prod/locals.tf`.

**Never** put secret values in `.tf` or `.tfvars` files — they would end up in HCP Terraform state.

## Conventions

- One Terraform root per logical stack; do not mix GCP and AWS resources in the same root.
- One HCP Terraform workspace per environment per root.
- `backend.tf` is gitignored. Copy `backend.tf.example` → `backend.tf` and run `terraform init` to connect to the HCP workspace.

## Adding a new environment to an existing root

1. Create a new folder following the naming pattern (e.g., `gcp/staging/`).
2. Copy the relevant root as a starting point; update `backend.tf.example` with a new workspace name following `armadacms-<provider>-<env>`.
3. Create the HCP Terraform workspace and configure remote state sharing if the new root needs outputs from another workspace.

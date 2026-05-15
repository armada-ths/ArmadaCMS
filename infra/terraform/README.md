# Terraform layout

This directory is organized by **provider** and then by **environment/root**.

Use this README for the shared Terraform structure and conventions. For root-specific files, variables, and operational notes, follow the links to each root README.

## Current structure

```text
infra/terraform/
├── README.md
├── gcp/
│   ├── prod/      # GCP production stack — Cloud Run, networking, load balancer, secrets
│   └── staging/   # GCP staging stack — Cloud Run, domain mapping, secrets
├── supabase/
│   └── prod/      # Supabase production stack — imported hosted project + managed settings
└── aws/
    ├── prod/      # AWS production — RDS PostgreSQL, S3, IAM
    └── staging/   # AWS staging — S3, IAM (no RDS; staging uses Supabase)
```

## Active roots

| Root             | HCP Terraform workspace   | What it manages                                                   | Details                                              |
| ---------------- | ------------------------- | ----------------------------------------------------------------- | ---------------------------------------------------- |
| `gcp/prod/`      | `armadacms-gcp-prod`      | Cloud Run, VPC egress, HTTPS LB, Cloud Build, Secret Manager      | [`gcp/prod/README.md`](gcp/prod/README.md)           |
| `gcp/staging/`   | `armadacms-gcp-staging`   | Cloud Run, domain mapping, Cloud Build, Secret Manager            | [`gcp/staging/README.md`](gcp/staging/README.md)     |
| `supabase/prod/` | `armadacms-supabase-prod` | Imported hosted Supabase production project and selected settings | [`supabase/prod/README.md`](supabase/prod/README.md) |
| `aws/prod/`      | `armadacms-aws-prod`      | RDS PostgreSQL, S3 file bucket, IAM upload user                   | [`aws/prod/README.md`](aws/prod/README.md)           |
| `aws/staging/`   | `armadacms-aws-staging`   | S3 file bucket, IAM upload user (no RDS — staging uses Supabase)  | [`aws/staging/README.md`](aws/staging/README.md)     |

## Cross-workspace state sharing

Workspace pairs share live infrastructure values without hardcoding them:

**Production:**

- `aws/prod` reads the GCP NAT egress IP (`static_egress_ip`) from `armadacms-gcp-prod` to restrict the RDS security group and the S3 IAM policy to that IP.
- `supabase/prod` reads the GCP NAT egress IP (`static_egress_ip`) from `armadacms-gcp-prod` to restrict direct DB access via `supabase_network_restrictions`. It exports `pooler_host`, `pooler_user`, and `db_name` consumed by `gcp/prod`.
- `gcp/prod` reads `pooler_host`, `pooler_user`, `db_name` from `armadacms-supabase-prod` to populate Cloud Run DB environment variables, and reads `s3_bucket_name`, `s3_bucket_region` from `armadacms-aws-prod` when `storage_provider = "s3"`.

**Staging:**

- `gcp/staging` reads `staging_db_host`, `staging_db_user`, `staging_db_name` from `armadacms-supabase-prod` (staging is a branch of the same Supabase project). It reads `s3_bucket_name` and `s3_bucket_region` from `armadacms-aws-staging` only when `storage_provider = "s3"`.

All cross-workspace reads use `data "tfe_outputs"` blocks. Each consuming workspace must be granted remote state read access to the producing workspace — configure this in HCP Terraform under each workspace's **Settings → Remote state sharing**.

## Conventions

- Keep **one Terraform root per logical stack**.
- Keep **one HCP Terraform workspace per environment per root**.
- Do not mix unrelated providers in one root.

## Workspace naming pattern

```text
armadacms-<provider>-<environment>
```

Examples: `armadacms-gcp-prod`, `armadacms-supabase-prod`, `armadacms-aws-prod`, `armadacms-gcp-staging`, `armadacms-aws-staging`.

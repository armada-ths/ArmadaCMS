# Terraform layout

This directory is organized by **provider** and then by **environment/root**.

Use this README for the shared Terraform structure and conventions. For root-specific files, variables, and operational notes, follow the links to each root README.

## Current structure

```text
infra/terraform/
├── README.md
├── gcp/
│   ├── prod/      # GCP production stack — Cloud Run, networking, load balancer, secrets
│   └── staging/   # GCP staging stack — Cloud Run, domain mapping, shared images, secrets
└── supabase/
    └── prod/      # Supabase production stack — imported project record + connection metadata
```

## Active roots

| Root             | HCP Terraform workspace   | What it manages                                                   | Details                                              |
| ---------------- | ------------------------- | ----------------------------------------------------------------- | ---------------------------------------------------- |
| `gcp/prod/`      | `armadacms-gcp-prod`      | Cloud Run, VPC egress, HTTPS LB, Cloud Build, Secret Manager      | [`gcp/prod/README.md`](gcp/prod/README.md)           |
| `gcp/staging/`   | `armadacms-gcp-staging`   | Cloud Run, domain mapping, shared Artifact Registry, Cloud Build, Secret Manager | [`gcp/staging/README.md`](gcp/staging/README.md)     |
| `supabase/prod/` | `armadacms-supabase-prod` | Imported Supabase project record and exported connection metadata | [`supabase/prod/README.md`](supabase/prod/README.md) |

## Cross-workspace state sharing

Workspace pairs share live infrastructure values without hardcoding them:

**Production:**

- `gcp/prod` reads `pooler_host`, `pooler_user`, `db_name`, and Supabase Storage configuration from `armadacms-supabase-prod` to populate Cloud Run environment variables.
- `gcp/prod` exports `static_egress_ip`. That address is allowlisted manually in Supabase; the current `supabase/prod` root does not manage network restrictions or read GCP state.

**Staging:**

- `gcp/staging` reads `staging_db_host`, `staging_db_user`, and `staging_db_name` for the persistent Supabase staging branch. It also reads the production project's shared Supabase Storage endpoint and bucket.

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

# ArmadaCMS Terraform — GCP staging

This Terraform root manages the **Google Cloud staging runtime stack** for `ArmadaCMS`.

## What it manages

- Project APIs
- Secret Manager secrets and IAM bindings
- Cloud Run service
- Cloud Run custom domain mapping (`staging.cms.armada.nu`)
- Optional serverless VPC egress with Cloud NAT and a static outbound IP
- Cloud Build triggers for the GitHub → Cloud Run deploy pipeline

Staging shares the production Artifact Registry repository — Cloud Build pushes
images there and Cloud Run pulls from it. There is no separate Artifact Registry
resource here, and no external HTTPS load balancer (the domain mapping handles
TLS termination instead).

## Architecture notes

- Cloud Run runs the Go API and bundled React-Admin frontend.
- Cloud Build updates the Cloud Run image via `cloudbuild.yaml` on every push to `staging`.
- PostgreSQL is provided by a **Supabase project** (not RDS). Connection details are plain
  environment variables set in `staging.auto.tfvars`.
- File storage is Terraform-selectable: keep `storage_provider = "s3"` for the current AWS S3 backend, or switch to `storage_provider = "supabase"` to inject the Supabase Storage env contract used by the backend migration work.
- `invoker_iam_disabled = true` enables public access without an IAM binding.
- Runtime secrets always include `DB_PASSWORD`, `jwtsecret_laganda`, `EVENTRO_*`, and `INITIAL_ADMIN_PASSWORD`, plus storage-provider-specific secrets:
  - `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` when `storage_provider = "s3"`
  - `SUPABASE_STORAGE_ACCESS_KEY_ID` / `SUPABASE_STORAGE_SECRET_ACCESS_KEY` when `storage_provider = "supabase"`
- Plain env vars always include DB settings plus `STORAGE_PROVIDER`; storage-provider-specific plain env vars are injected automatically:
  - `S3_BUCKET`, `AWS_REGION` from `armadacms-aws-staging` when `storage_provider = "s3"`
  - `SUPABASE_URL`, `SUPABASE_STORAGE_S3_ENDPOINT`, `SUPABASE_STORAGE_BUCKET`, `SUPABASE_STORAGE_REGION` when `storage_provider = "supabase"`
- The Cloud Run container image is ignored by Terraform after the first deploy so
  Cloud Build can ship new revisions freely.

## Files

| File                  | Purpose                                                                            |
| --------------------- | ---------------------------------------------------------------------------------- |
| `versions.tf`         | Provider version requirements (google, tfe)                                        |
| `variables.tf`        | Configurable inputs                                                                |
| `locals.tf`           | Derived names and Cloud Run env vars, including the storage-provider switch        |
| `aws_state.tf`        | Optional `data.tfe_outputs.aws_staging` — only read when `storage_provider = "s3"` |
| `services.tf`         | GCP API enablement                                                                 |
| `iam.tf`              | Runtime service account and Cloud Build permissions                                |
| `secrets.tf`          | Secret Manager secrets                                                             |
| `networking.tf`       | Cloud NAT, Cloud Router, static egress IP, optional VPC connector                  |
| `cloud_build.tf`      | GitHub-backed Cloud Build triggers for the `staging` branch and PRs targeting it   |
| `cloud_run.tf`        | Cloud Run service                                                                  |
| `domain_mapping.tf`   | Cloud Run custom domain mapping for `staging.cms.armada.nu`                        |
| `outputs.tf`          | Useful outputs (service account emails, image URI, secret IDs)                     |
| `staging.auto.tfvars` | Committed non-secret staging defaults                                              |
| `backend.tf.example`  | HCP Terraform backend template                                                     |

## Workspace dependencies

This root reads S3 bucket name and region from `armadacms-aws-staging` only when `storage_provider = "s3"`. In that mode, the AWS staging workspace must grant this workspace read access under **Settings → Remote state sharing** (or "Share with all workspaces").

For the full cross-workspace wiring layout, see [`../../README.md`](../../README.md).

## HCP Terraform workspace setup

Workspace: `armadacms-gcp-staging` in the `THS-Armada` organization.

Copy `backend.tf.example` to `backend.tf`, fill in the workspace name, and run
`terraform init`.

**Environment variables** (set in the workspace):

| Variable                            | Notes                                                                           |
| ----------------------------------- | ------------------------------------------------------------------------------- |
| `TFE_TOKEN`                         | HCP Terraform API token — required for `data.tfe_outputs` cross-workspace reads |
| `TFC_GCP_PROVIDER_AUTH`             | `true` — enables OIDC dynamic credentials                                       |
| `TFC_GCP_WORKLOAD_PROVIDER_NAME`    | Workload identity provider resource name                                        |
| `TFC_GCP_RUN_SERVICE_ACCOUNT_EMAIL` | Service account Terraform runs as (`terraform-armadacms-staging@...`)           |

## Ongoing operations

### Rotating storage credentials

When `storage_provider = "s3"`:

1. Create a new access key for `armadacms-staging-s3` in the AWS console.
2. Update `secret_values["AWS_ACCESS_KEY_ID"]` and `secret_values["AWS_SECRET_ACCESS_KEY"]`
   in the HCP Terraform workspace variables and trigger a new run.
3. Verify uploads work, then delete the old key.

When `storage_provider = "supabase"`:

1. Generate a new S3 access key pair in the Supabase dashboard for the staging project.
2. Update `secret_values["SUPABASE_STORAGE_ACCESS_KEY_ID"]` and `secret_values["SUPABASE_STORAGE_SECRET_ACCESS_KEY"]`
   in the HCP Terraform workspace variables and trigger a new run.
3. Verify uploads work, then revoke the old Supabase storage key.

## Storage cutover notes

To prepare the staging runtime for Supabase Storage:

- Set `storage_provider = "supabase"`.
- Use these committed defaults unless you need to override them:
  - `supabase_url = "https://yfybmnqzclpmpncyfmdc.supabase.co"`
  - `supabase_storage_s3_endpoint = "https://yfybmnqzclpmpncyfmdc.storage.supabase.co/storage/v1/s3"`
  - `supabase_storage_bucket = "armadacms-files"`
  - `supabase_storage_region = "eu-north-1"`
- Populate these secret values in HCP Terraform:
  - `SUPABASE_STORAGE_ACCESS_KEY_ID`
  - `SUPABASE_STORAGE_SECRET_ACCESS_KEY`

### Rotating the JWT secret

Update `secret_values["jwtsecret_laganda"]` in the HCP Terraform workspace variables and
apply. All active sessions will be invalidated immediately.

Ensure that `jwtsecret_laganda` in staging Secret Manager is a **different value from
production** so a leaked staging token cannot authenticate against the production API.

### Deploying a new revision

Push to the `staging` branch. Cloud Build triggers automatically, builds a new image,
and updates the Cloud Run service. Terraform is not involved in normal deploys.

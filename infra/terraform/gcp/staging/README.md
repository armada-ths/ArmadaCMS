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
- AWS S3 is the file-storage backend. Bucket name and region are read automatically from
  the `armadacms-aws-staging` HCP Terraform workspace outputs.
- `invoker_iam_disabled = true` enables public access without an IAM binding.
- Runtime secrets (`DB_PASSWORD`, `jwtsecret_laganda`, `AWS_ACCESS_KEY_ID`,
  `AWS_SECRET_ACCESS_KEY`, `INITIAL_ADMIN_PASSWORD`) come from Secret Manager.
- Plain env vars (`DB_HOST`, `DB_NAME`, `S3_BUCKET`, `AWS_REGION`, etc.) are injected
  automatically from `staging.auto.tfvars` and from the `armadacms-aws-staging`
  workspace outputs.
- The Cloud Run container image is ignored by Terraform after the first deploy so
  Cloud Build can ship new revisions freely.

## Files

| File                  | Purpose                                                                          |
| --------------------- | -------------------------------------------------------------------------------- |
| `versions.tf`         | Provider version requirements (google, tfe)                                      |
| `variables.tf`        | Configurable inputs                                                              |
| `locals.tf`           | Derived names and Cloud Run env vars (including cross-workspace S3 values)       |
| `aws_state.tf`        | `data.tfe_outputs.aws_staging` — reads S3 outputs from the AWS staging workspace |
| `services.tf`         | GCP API enablement                                                               |
| `iam.tf`              | Runtime service account and Cloud Build permissions                              |
| `secrets.tf`          | Secret Manager secrets                                                           |
| `networking.tf`       | Cloud NAT, Cloud Router, static egress IP, optional VPC connector                |
| `cloud_build.tf`      | GitHub-backed Cloud Build triggers for the `staging` branch and PRs targeting it |
| `cloud_run.tf`        | Cloud Run service                                                                |
| `domain_mapping.tf`   | Cloud Run custom domain mapping for `staging.cms.armada.nu`                      |
| `outputs.tf`          | Useful outputs (service account emails, image URI, secret IDs)                   |
| `staging.auto.tfvars` | Committed non-secret staging defaults                                            |
| `backend.tf.example`  | HCP Terraform backend template                                                   |

## Workspace dependencies

This root reads S3 bucket name and region from `armadacms-aws-staging` via
`data.tfe_outputs`. The AWS staging workspace must grant this workspace read
access under **Settings → Remote state sharing** (or "Share with all workspaces").

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

### Rotating the S3 access key

1. Create a new access key for `armadacms-staging-s3` in the AWS console.
2. Update `secret_values["AWS_ACCESS_KEY_ID"]` and `secret_values["AWS_SECRET_ACCESS_KEY"]`
   in the HCP Terraform workspace variables and trigger a new run.
3. Verify uploads work, then delete the old key.

### Rotating the JWT secret

Update `secret_values["jwtsecret_laganda"]` in the HCP Terraform workspace variables and
apply. All active sessions will be invalidated immediately.

Ensure that `jwtsecret_laganda` in staging Secret Manager is a **different value from
production** so a leaked staging token cannot authenticate against the production API.

### Deploying a new revision

Push to the `staging` branch. Cloud Build triggers automatically, builds a new image,
and updates the Cloud Run service. Terraform is not involved in normal deploys.

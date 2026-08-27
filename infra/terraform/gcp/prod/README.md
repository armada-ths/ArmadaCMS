# ArmadaCMS Terraform — GCP production

This Terraform root manages the **Google Cloud production runtime stack** for `ArmadaCMS`.

## What it manages

- Project APIs
- Artifact Registry for container images
- Secret Manager secrets and IAM bindings
- Cloud Run service (optional, enabled by `deploy_cloud_run_service`)
- Serverless VPC egress with Cloud NAT and a static outbound IP
- External HTTPS load balancer
- Cloud Build triggers for the GitHub → Cloud Run deploy pipeline

## Architecture notes

- Cloud Run runs the Go API and bundled React-Admin frontend.
- Cloud Build updates the Cloud Run image via `cloudbuild.yaml`.
- PostgreSQL is provided by Supabase. Connection details are read from the `armadacms-supabase-prod` workspace outputs.
- File storage uses Supabase Storage via its S3-compatible endpoint.
- Cloud Run uses direct VPC egress on a dedicated `armadacms-serverless` VPC and subnet. Cloud NAT provides the stable outbound IP consumed by the Supabase network allowlist.
- `invoker_iam_disabled = true` enables public access without an IAM binding.
- Runtime secrets always include `DB_PASSWORD`, `jwtsecret_laganda`, `EVENTRO_*`, `REVALIDATION_SECRET`, `SUPABASE_STORAGE_ACCESS_KEY_ID`, and `SUPABASE_STORAGE_SECRET_ACCESS_KEY`.
- Plain env vars always include DB settings, `STORAGE_PROVIDER`, and Supabase Storage settings (`SUPABASE_URL`, `SUPABASE_STORAGE_S3_ENDPOINT`, `SUPABASE_STORAGE_BUCKET`, `SUPABASE_STORAGE_REGION`) — all read from the `armadacms-supabase-prod` workspace via `tfe_outputs`.
- The Cloud Run container image is ignored by Terraform after the first deploy so Cloud Build can ship new revisions freely.
- Artifact Registry cleanup policies retain the 20 most recent versions per package, delete untagged versions after 14 days, delete `pr-*` versions after 30 days, and delete other versions after 180 days.
- Cloud Run and Cloud Build use separate `armadacms-runtime` and `armadacms-deploy` service accounts. Runtime can read only its own secrets; the deployer can write images, update only the production Cloud Run service, write build logs, read the GitHub App secret, and act as the production runtime identity.

## Files

| File                   | Purpose                                                                                                               |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `versions.tf`          | Provider version requirements (google, tfe)                                                                           |
| `variables.tf`         | Configurable inputs                                                                                                   |
| `locals.tf`            | Derived names and Cloud Run env vars                                                                                  |
| `supabase_state.tf`    | `data.tfe_outputs.supabase_prod` — reads DB connection values and Supabase Storage config from the Supabase workspace |
| `services.tf`          | GCP API enablement                                                                                                    |
| `artifact_registry.tf` | Artifact Registry repository                                                                                          |
| `iam.tf`               | Runtime service account and Cloud Build permissions                                                                   |
| `secrets.tf`           | Secret Manager secrets                                                                                                |
| `networking.tf`        | Cloud NAT, Cloud Router, static egress IP, optional VPC connector                                                     |
| `cloud_build.tf`       | GitHub-backed Cloud Build triggers                                                                                    |
| `cloud_run.tf`         | Cloud Run service                                                                                                     |
| `load_balancer.tf`     | External HTTPS load balancer, serverless NEG, proxies, forwarding rules                                               |
| `outputs.tf`           | Useful outputs including `static_egress_ip`                                                                           |
| `prod.auto.tfvars`     | Committed non-secret production defaults                                                                              |
| `backend.tf.example`   | HCP Terraform backend template                                                                                        |

## Workspace dependencies

This root consumes outputs from:

- `armadacms-supabase-prod` — `pooler_host`, `pooler_user`, `db_name`

For the full cross-workspace wiring and remote state sharing setup, see [`../../README.md`](../../README.md).

## HCP Terraform workspace setup

Workspace: `armadacms-gcp-prod` in the `THS-Armada` organization.

Copy `backend.tf.example` to `backend.tf`, fill in the workspace name, and run `terraform init`.

**Environment variables** (set in the workspace):

| Variable                            | Notes                                                                           |
| ----------------------------------- | ------------------------------------------------------------------------------- |
| `TFE_TOKEN`                         | HCP Terraform API token — required for `data.tfe_outputs` cross-workspace reads |
| `TFC_GCP_PROVIDER_AUTH`             | `true` — enables OIDC dynamic credentials                                       |
| `TFC_GCP_WORKLOAD_PROVIDER_NAME`    | Workload identity provider resource name                                        |
| `TFC_GCP_RUN_SERVICE_ACCOUNT_EMAIL` | Service account Terraform runs as                                               |

## Ongoing operations

### Artifact Registry cleanup

Cleanup is managed directly on `google_artifact_registry_repository.docker` and
runs asynchronously after apply. `KEEP` policies take precedence over `DELETE`
policies, so the 20 newest versions of every package remain available for
deployments and rollbacks even when an age-based delete policy also matches.

### Rotating storage credentials

1. Generate a new S3 access key pair in the Supabase dashboard.
2. Add the new values directly in GCP Secret Manager (do **not** use HCP Terraform workspace variables):
   - `armadacms-SUPABASE_STORAGE_ACCESS_KEY_ID`
   - `armadacms-SUPABASE_STORAGE_SECRET_ACCESS_KEY`
3. Verify uploads work, then revoke the old key.

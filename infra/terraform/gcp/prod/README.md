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
- RDS PostgreSQL (AWS) is the database, and S3 (AWS) is the file-storage backend.
- Cloud Run uses direct VPC egress on the `default` VPC with Cloud NAT for a stable outbound IP.
- `invoker_iam_disabled = true` enables public access without an IAM binding.
- Runtime secrets (`DB_PASSWORD`, `jwtsecret_laganda`, `EVENTRO_*`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) come from Secret Manager, while plain env vars (`DB_HOST`, `DB_NAME`, `S3_BUCKET`, `AWS_REGION`) are injected automatically from the `armadacms-aws-prod` workspace outputs.
- The Cloud Run container image is ignored by Terraform after the first deploy so Cloud Build can ship new revisions freely.

## Files

| File                   | Purpose                                                                          |
| ---------------------- | -------------------------------------------------------------------------------- |
| `versions.tf`          | Provider version requirements (google, tfe)                                      |
| `variables.tf`         | Configurable inputs                                                              |
| `locals.tf`            | Derived names and Cloud Run env vars (including cross-workspace values)          |
| `aws_state.tf`         | `data.tfe_outputs.aws_prod` — reads infrastructure values from the AWS workspace |
| `services.tf`          | GCP API enablement                                                               |
| `artifact_registry.tf` | Artifact Registry repository                                                     |
| `iam.tf`               | Runtime service account and Cloud Build permissions                              |
| `secrets.tf`           | Secret Manager secrets                                                           |
| `networking.tf`        | Cloud NAT, Cloud Router, static egress IP, optional VPC connector                |
| `cloud_build.tf`       | GitHub-backed Cloud Build triggers                                               |
| `cloud_run.tf`         | Cloud Run service                                                                |
| `load_balancer.tf`     | External HTTPS load balancer, serverless NEG, proxies, forwarding rules          |
| `outputs.tf`           | Useful outputs including `static_egress_ip` (consumed by the AWS workspace)      |
| `prod.auto.tfvars`     | Committed non-secret production defaults                                         |
| `backend.tf.example`   | HCP Terraform backend template                                                   |

## Workspace dependencies

This root consumes infrastructure outputs from `armadacms-aws-prod` for Cloud Run environment variables and exports `static_egress_ip` for `aws/prod`.

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

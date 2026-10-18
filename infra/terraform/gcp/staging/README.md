# ArmadaCMS Terraform — GCP staging

This Terraform root manages the **Google Cloud staging runtime stack** for `ArmadaCMS`.

## What it manages

- Project APIs
- Secret Manager secrets and IAM bindings
- Cloud Run service
- Cloud Run custom domain mapping (`staging.cms.armada.nu`)
- Optional serverless VPC egress with Cloud NAT and a static outbound IP (supported by the root but currently disabled)
- Environment-specific Workload Identity Federation for GitHub Actions releases
- Cloud Scheduler retry trigger in `europe-west4`; its target remains the photo worker Job in `europe-north1`

Staging shares the production Artifact Registry repository — GitHub Actions publishes
images there and Cloud Run pulls from it. There is no separate Artifact Registry
resource here, and no external HTTPS load balancer (the domain mapping handles
TLS termination instead).

## Architecture notes

- Cloud Run runs the Go API and bundled React-Admin frontend.
- GitHub Actions verifies the staging commit through reusable CI workflows, smoke-tests a tagged revision without traffic, then promotes that revision.
- PostgreSQL is provided by the persistent staging branch of the production Supabase project. Connection details are read from `armadacms-supabase-prod` via `tfe_outputs`.
- File storage uses the production project's shared Supabase Storage bucket via its S3-compatible endpoint.
- `invoker_iam_disabled = true` enables public access without an IAM binding.
- Runtime secrets include `DB_PASSWORD`, `jwtsecret_laganda`, `EVENTRO_*`, `REVALIDATION_SECRET`, and `VERCEL_AUTOMATION_BYPASS_SECRET`. The prefixed Secret Manager IDs `armadacms-staging-SUPABASE_STORAGE_ACCESS_KEY_ID` and `armadacms-staging-SUPABASE_STORAGE_SECRET_ACCESS_KEY` are injected as `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
- Plain env vars include DB settings plus `S3_ENDPOINT`, `S3_PUBLIC_URL`, `S3_BUCKET`, and `S3_REGION`, all read from `armadacms-supabase-prod` unless an explicit DB override is set.
- Cloud Run image, generated revision name and traffic are release-owned and ignored by Terraform. Runtime configuration remains Terraform-owned.
- Cloud Run releases use separate runtime, build and deployment service accounts. Runtime can read only its own secrets; the builder publishes images and caches to Artifact Registry; the deployer reads images, updates its Cloud Run service and photo worker, and acts as the runtime identity through Environment-bound OIDC federation. Pull request container builds run in GitHub Actions without Google credentials or secrets; no PR build identity or trigger is provisioned in GCP.

Recovery snapshots and release outcomes are separate Actions artifacts retained for 90 days. Snapshot publication precedes every Cloud Run mutation. Actions serializes releases per environment; etags and configuration fingerprints protect against external changes.

## Files

| File                  | Purpose                                                                                                  |
| --------------------- | -------------------------------------------------------------------------------------------------------- |
| `versions.tf`         | Provider version requirements (google, tfe)                                                              |
| `variables.tf`        | Configurable inputs                                                                                      |
| `locals.tf`           | Derived names and Cloud Run env vars                                                                     |
| `supabase_state.tf`   | `data.tfe_outputs.supabase_prod` — reads staging branch DB connection values and Supabase Storage config |
| `services.tf`         | GCP API enablement                                                                                       |
| `iam.tf`              | Runtime and deploy service accounts with resource-scoped permissions                                                      |
| `secrets.tf`          | Secret Manager secrets                                                                                   |
| `networking.tf`       | Cloud NAT, Cloud Router, static egress IP, optional VPC connector                                        |
| `actions.tf` | GitHub OIDC federation, environment outputs and Terraform state moves |
| `cloud_run.tf`        | Cloud Run service                                                                                        |
| `domain_mapping.tf`   | Cloud Run custom domain mapping for `staging.cms.armada.nu`                                              |
| `outputs.tf`          | Useful outputs (service account emails, image URI, secret IDs)                                           |
| `staging.auto.tfvars` | Committed non-secret staging defaults                                                                    |
| `backend.tf.example`  | HCP Terraform backend template                                                                           |

## Workspace dependencies

This root reads `staging_db_host`, `staging_db_user`, `staging_db_name`, and all Supabase Storage values from `armadacms-supabase-prod` (staging is a branch of the same Supabase project as production). Grant `armadacms-gcp-staging` remote state read access to `armadacms-supabase-prod` under **Settings → Remote state sharing**.

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

1. Generate a new S3 access key pair in the Supabase dashboard.
2. Add the new values directly in GCP Secret Manager (do **not** use HCP Terraform workspace variables):
   - `armadacms-staging-SUPABASE_STORAGE_ACCESS_KEY_ID`
   - `armadacms-staging-SUPABASE_STORAGE_SECRET_ACCESS_KEY`
3. Deploy a new Cloud Run revision, verify uploads, and then revoke the old key.

### Rotating the JWT secret

Add a new version to `armadacms-staging-jwtsecret_laganda` directly in GCP Secret Manager,
then deploy a new Cloud Run revision. Do not put the value in HCP Terraform variables or
Terraform state. All existing sessions signed with the old value will become invalid once
the new revision serves traffic.

Ensure that `jwtsecret_laganda` in staging Secret Manager is a **different value from
production** so a leaked staging token cannot authenticate against the production API.

### Deploying a new revision

The staging push workflow builds the image in parallel with CI, then persists recovery metadata, verifies the tagged preview and promotes it after successful CI and verification. Terraform does not participate in normal releases.

## Release identity

The `github_environment_variables` output exposes the two non-secret Actions connection values. OIDC trust permits only staging pushes from the ArmadaCMS deployment workflow and the `Staging` GitHub Environment. The deploy account reads the shared Artifact Registry repository and updates this environment's service and worker. Preview uses the tagged `run.app` address; live verification uses `staging.cms.armada.nu`.

The image build uses a separate registry-only service account and branch-bound OIDC provider without a GitHub Environment deployment. The `github_build_repository_variables` output exposes its non-secret repository connection values. CI and the build run concurrently; the Environment-protected release job requires both to succeed. Only the deploy identity can change Cloud Run and impersonate the runtime account. Its Artifact Registry access is read-only.

# ArmadaCMS Terraform — GCP production

This Terraform root manages the **current Google Cloud production runtime stack** for `ArmadaCMS`.

## What it manages

- project APIs required by the stack
- Artifact Registry for container images
- Secret Manager secrets for runtime configuration
- a dedicated Cloud Run runtime service account
- optional Cloud Run service management
- optional serverless VPC egress with Cloud NAT and a stable outbound IP for AWS RDS allow-listing
- optional external HTTPS load balancer resources in front of Cloud Run
- IAM for the existing `cloudbuild.yaml` deployment flow

It intentionally does **not** try to manage GitHub-to-Cloud-Build trigger wiring yet, because the repository connection and trigger shape often differ between projects and are easiest to add once the base infra is under Terraform control.

## What this setup assumes

This Terraform layout is tailored to the current ArmadaCMS deployment model already present in the repo:

- Cloud Run runs the Go API and bundled React-Admin frontend
- Cloud Build updates the Cloud Run service image using `cloudbuild.yaml`
- AWS RDS remains the PostgreSQL database
- AWS S3 remains the file-storage backend
- Secret Manager stores runtime secrets and the GitHub App private key used for Cloud Build deployment tracking
- production Cloud Run currently uses direct VPC egress on the existing `default` VPC/subnet together with Cloud NAT

Runtime secret handling in the current production setup is intentionally mixed:

- `DB_PASSWORD`, `EVENTRO_API`, `EVENTRO_FAIR_ID`, `EVENTRO_ORG`, `jwtsecret_laganda`, `AWS_ACCESS_KEY_ID`, and `AWS_SECRET_ACCESS_KEY` come from Secret Manager
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME`, `S3_BUCKET`, and `AWS_REGION` are plain Cloud Run environment variables
- the current production project already grants the compute default service account project-level `roles/secretmanager.secretAccessor`, so per-secret IAM bindings are disabled by default in this root

A deliberate quirk: Terraform ignores the Cloud Run container image field after the first deploy so that Cloud Build can continue shipping new revisions without Terraform constantly trying to roll the image back.

## Files

- `versions.tf` — Terraform + provider version requirements
- `variables.tf` — configurable inputs
- `services.tf` — API enablement
- `artifact_registry.tf` — Artifact Registry repository
- `iam.tf` — runtime identity and Cloud Build permissions
- `secrets.tf` — Secret Manager secrets and IAM bindings
- `networking.tf` — optional direct-VPC or connector-based Cloud Run egress, Cloud Router, Cloud NAT, and static egress IP
- `load_balancer.tf` — optional external HTTPS load balancer, serverless NEG, proxies, forwarding rules, and certificate wiring
- `cloud_run.tf` — optional Cloud Run service and public invoker binding
- `outputs.tf` — useful outputs after apply
- `terraform.tfvars.example` — example configuration for ArmadaCMS
- `backend.tf.example` — HCP Terraform remote state example

## Recommended rollout order

### 1. Start with bootstrap-only mode

Copy `terraform.tfvars.example` to `terraform.tfvars` and keep:

- `deploy_cloud_run_service = false`

That first apply will set up the GCP plumbing without trying to deploy a Cloud Run service that references secrets which do not have values yet.

### 2. Populate Secret Manager

You have two options:

- **Safer for state:** create secret versions manually in Secret Manager after the bootstrap apply.
- **Faster for setup:** populate `github_app_private_key` and `secret_values` in `terraform.tfvars` and let Terraform create the first versions.

> If you let Terraform manage secret values, those values will be stored in Terraform state. Use an encrypted remote backend and restricted IAM if you choose that route.

### 3. Turn on Cloud Run management

Once the secrets exist, set:

- `deploy_cloud_run_service = true`

and apply again.

### 4. Match the Cloud Run networking mode to production

The current production setup uses:

- `cloud_run_vpc_egress_mode = "DIRECT_VPC"`
- `vpc_network_name = "default"`
- `vpc_subnetwork_name = "default"`
- `manage_vpc_network_resources = false`

That means Terraform references the existing VPC/subnet rather than trying to create a Serverless VPC Access connector.

Use `CONNECTOR` only if you intentionally want to move Cloud Run to a Serverless VPC Access connector later.

### 5. Allow-list the egress IP in AWS RDS

If `enable_vpc_egress = true`, Terraform reserves a static external IP and routes Cloud Run traffic through Cloud NAT. Use the `static_egress_ip` output in the AWS RDS security group allow-list.

### 6. Optionally bring the HTTPS load balancer under Terraform

If production traffic reaches ArmadaCMS through a Google Cloud HTTPS load balancer in front of Cloud Run, you can now model that in `load_balancer.tf`.

Recommended adoption order:

1. Confirm the existing load balancer resource names in the GCP console
2. Update the `lb_*` variables in `terraform.tfvars`
3. Keep `enable_https_load_balancer = false` until naming matches reality
4. Import existing load balancer resources
5. Turn `enable_https_load_balancer = true`
6. Run `terraform plan` and review drift carefully before any apply

## Existing production resources: import before apply

If the project already contains production resources with the same names, import them before the first full apply instead of trying to recreate them.

Common candidates are:

- the Artifact Registry repository
- the Cloud Run service
- the runtime service account if it was created manually and you want Terraform to manage it
- the GitHub App secret if it already exists in Secret Manager
- the HTTPS load balancer resources if they already exist in GCP
- the Cloud Router, Cloud NAT, and static egress IP if they already exist in GCP

Typical import IDs look like this:

- Artifact Registry repository: `projects/<project>/locations/<region>/repositories/<repository>`
- Cloud Run service: `projects/<project>/locations/<region>/services/<service>`
- Service account: `projects/<project>/serviceAccounts/<email>`
- Secret Manager secret: `projects/<project>/secrets/<secret-id>`
- Cloud Router: `projects/<project>/regions/<region>/routers/<name>`
- Cloud NAT: `projects/<project>/regions/<region>/routers/<router>/nats/<name>`
- Regional address: `projects/<project>/regions/<region>/addresses/<name>`
- Serverless NEG: `projects/<project>/regions/<region>/networkEndpointGroups/<name>`
- Global address: `projects/<project>/global/addresses/<name>`
- Backend service: `projects/<project>/global/backendServices/<name>`
- URL map: `projects/<project>/global/urlMaps/<name>`
- Target HTTPS proxy: `projects/<project>/global/targetHttpsProxies/<name>`
- Target HTTP proxy: `projects/<project>/global/targetHttpProxies/<name>`
- Global forwarding rule: `projects/<project>/global/forwardingRules/<name>`
- Managed SSL certificate: `projects/<project>/global/sslCertificates/<name>`

## Remote state with HCP Terraform (recommended)

By default Terraform uses local state, but for shared infrastructure and multi-cloud expansion, this root is documented around **HCP Terraform**.

Why HCP Terraform is a good fit here:

- one remote state model across GCP, AWS, and Vercel
- locking, plan history, and auditability without extra backend plumbing
- workspace-based separation for production and future environments
- easier future expansion if the repo later grows into multiple Terraform roots

If this is a brand-new GCP project, you may still need a one-time manual enablement of the Service Usage API before Terraform can manage the rest of the project APIs.

### Initial setup

1. Create an HCP Terraform organization
2. Run `terraform login` locally, or set `TF_TOKEN_app_terraform_io`
3. Copy `backend.tf.example` to `backend.tf`
4. Replace the placeholder organization and workspace name
5. Run `terraform init`

### Workspace for this root

For this root, use:

- `armadacms-gcp-prod`

### Variables and secrets in HCP Terraform

For HCP Terraform runs, store environment-specific values such as these as workspace variables:

- `project_id`
- `github_app_private_key`
- `secret_values`
- any production overrides for scaling or egress

Mark secrets as sensitive in HCP Terraform so they stay out of plan output.

### Migrating from local state later

If you have already created local state and want to move it into HCP Terraform, copy `backend.tf.example` to `backend.tf`, run `terraform init`, and let Terraform migrate the existing state when prompted.

## Notes for this root

- The defaults mirror the current `cloudbuild.yaml` values: project path shape, region, service name, and GitHub App secret name.
- The example Cloud Run settings now mirror the live production networking shape more closely, including direct VPC egress on the existing `default` network.
- If you do not need a stable egress IP, leave `enable_vpc_egress = false` and keep the setup simpler.
- The HTTPS load balancer resources are disabled by default so you can inspect/import the existing production setup before Terraform starts managing it.

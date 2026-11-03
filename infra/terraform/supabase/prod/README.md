# ArmadaCMS Terraform — Supabase production and staging settings

This root manages the hosted production project, selected platform settings for
production and its existing persistent staging branch, and connection metadata
consumed by the GCP workspaces. The Supabase provider is constrained to the
1.11 patch series and locked to 1.11.0.

## Managed configuration

| File                               | Purpose                                                                                            |
| ---------------------------------- | -------------------------------------------------------------------------------------------------- |
| `project.tf`                       | Existing production project record, protected by `prevent_destroy`                                 |
| `settings.tf`                      | Production and staging API, Auth, Storage, SSL and network settings; production pooler data source |
| `gcp_state.tf`                     | Reads the production Cloud NAT address from `armadacms-gcp-prod` through `tfe_outputs`             |
| `variables.tf`, `prod.auto.tfvars` | Project references, connection metadata and non-secret desired settings                            |
| `outputs.tf`                       | Project and connection metadata shared with GCP                                                    |
| `backend.tf.example`               | Per-developer HCP Terraform backend configuration                                                  |

Only the explicitly configured JSON keys are managed:

- **Data API:** `pg_pgrst_no_exposed_schemas` preserves the hosted no-exposed-schemas
  setting in both environments. The search path is `public,extensions` and the row
  limit is 1000. A search-path entry does not expose that schema through the API.
- **Auth:** new Supabase Auth sign-ups are disabled. ArmadaCMS uses its own Go API
  authentication and direct Postgres connections.
- **Storage:** the existing 5 GiB upload limit, S3 protocol and image transformations
  are retained. Production Storage is shared by the application deployments.
- **SSL:** database SSL enforcement stays enabled in both environments.
- **Network:** production allows only the `/32` for GCP production's
  `static_egress_ip`. A missing or invalid IP fails the resource precondition;
  there is no open-network fallback. Staging retains its existing unrestricted
  IPv4/IPv6 rules because it uses direct IPv6 connectivity without a static NAT IP.

The settings resources configure existing services rather than creating new
projects or branches. The staging branch lifecycle, SQL migrations, bucket
policies, API keys, credentials, Edge Functions, Auth providers and Storage
analytics remain outside this root. No Postgres configuration overrides were
present when inspected. Although provider 1.11.0 declares a `pooler` settings
attribute, its settings resource does not read or update it; pool sizes are not
claimed as managed here.

## Workspace and dependencies

Workspace: `armadacms-supabase-prod`, organization: `THS-Armada`.

- `SUPABASE_ACCESS_TOKEN` authenticates the Supabase provider.
- `organization_id` is the Supabase organization slug.
- `database_password` is sensitive and must match the existing production password;
  the Management API does not return it when reading the project.
- `gcp/prod` exports `static_egress_ip`; this root reads it using `tfe_outputs`.
- `gcp/prod` and `gcp/staging` read this root's existing connection outputs.

These are reads of previously applied workspace outputs, not a dependency between
resources in a single Terraform graph. Keep the connection output names stable.
Do not derive the GCP address from a Supabase output or hardcode a copy of it here.

Runtime secret values remain in GCP Secret Manager. This root does not manage
Supabase secret values or application database schema.

## Development and validation

Copy `backend.tf.example` to the ignored `backend.tf` for authenticated workspace
operations. Use `terraform fmt -check -recursive infra/terraform` and validate this
root locally. `terraform test` in this root uses mocked providers to check that
missing or invalid NAT outputs stop the plan. Remote applies use the reviewed HCP
Terraform workflow.

The configuration was compared with hosted settings using Supabase CLI
`config diff`, `config pull --dry-run`, `branches list`, `network-restrictions get`,
`ssl-enforcement get`, and `postgres-config get`. Those inspection commands do not
change the hosted configuration. The repository's `supabase/config.toml` describes
the local development stack independently of these hosted Terraform settings.

Provider 1.11.0 skips REST health checks only for an empty `db_schema`; the hosted
sentinel is non-empty. It also checks the project and other services before
updating settings. Successful validation or a read-only plan does not establish
that these apply-time health checks will pass. Review any HCP apply failure by
service rather than assuming a timeout increase resolves it.

# ArmadaCMS Terraform — Supabase production and staging settings

This root manages the hosted production project and platform settings for
production and its persistent staging branch. It exports connection metadata
for the GCP workspaces. The Supabase provider is constrained to the
1.11 patch series and locked to 1.11.0.

## Managed configuration

| File                               | Purpose                                                                                            |
| ---------------------------------- | -------------------------------------------------------------------------------------------------- |
| `project.tf`                       | Production project record, protected by `prevent_destroy`                                          |
| `settings.tf`                      | Production and staging API, Auth, Storage, SSL and network settings; production pooler data source |
| `gcp_state.tf`                     | Reads the production Cloud NAT address from `armadacms-gcp-prod` through `tfe_outputs`             |
| `variables.tf`, `prod.auto.tfvars` | Project references, connection metadata and non-secret desired settings                            |
| `outputs.tf`                       | Project and connection metadata shared with GCP                                                    |
| `backend.tf.example`               | Per-developer HCP Terraform backend configuration                                                  |

- **Data API** uses `pg_pgrst_no_exposed_schemas` in both environments, so no application schemas are exposed. The search path is `public,extensions` and the row limit is 1000.
- **Auth** disables Supabase Auth sign-ups. The Go API handles ArmadaCMS authentication.
- **Storage** enables the S3 protocol and image transformations and sets a 5 GiB upload limit. Both application deployments use production Storage.
- **SSL** is required for database connections in both environments.
- **Network** allows only the GCP production Cloud NAT address in production, as a `/32`. A missing or invalid address fails the resource precondition. Staging allows all IPv4 and IPv6 addresses and uses a direct IPv6 connection.

The staging branch lifecycle, database parameter overrides, SQL migrations,
bucket policies, API keys, credentials, Edge Functions, Auth providers, Storage
analytics and pooler sizing are outside this root's scope.

## Workspace and dependencies

Workspace: `armadacms-supabase-prod`, organization: `THS-Armada`.

- `SUPABASE_ACCESS_TOKEN` authenticates the Supabase provider.
- `organization_id` is the Supabase organization slug.
- `database_password` is the sensitive password for the production database.
- `gcp/prod` exports `static_egress_ip`; this root reads it using `tfe_outputs`.
- `gcp/prod` and `gcp/staging` read this root's connection outputs.

These reads use the published workspace outputs. The output names form the
interface between the workspaces.

GCP Secret Manager stores runtime secrets. SQL migrations define the application database schema.

## Development and validation

Copy `backend.tf.example` to the ignored `backend.tf` for authenticated workspace
operations. Use `terraform fmt -check -recursive infra/terraform` and validate this
root locally. `terraform test` in this root uses mocked providers to check that
missing or invalid NAT outputs stop the plan. Remote applies use the reviewed HCP
Terraform workflow.

`supabase/config.toml` configures the local development stack.

Provider 1.11.0 checks project and service health before applying settings.
It skips the REST check only for an empty `db_schema`, so
`pg_pgrst_no_exposed_schemas` still triggers that check.

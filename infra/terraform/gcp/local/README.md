# Local development keys

This root defines a score-1.0 reCAPTCHA Enterprise test key, a separate assessment API key restricted to the reCAPTCHA API, and the local Web Maps Embed browser key. It uses the `THS-Armada/armadacms-gcp-local` workspace with local execution and remote state. It does not manage hosted keys, API enablement, IAM or Cloud Run services. The Maps key remains restricted to localhost origins and the Maps Embed API.

The test key permits changing developer LAN addresses. The local backend still checks token validity, `photo_upload`, score and its configured hostnames. Production and staging retain their existing keys and workload identity.

`scripts/configure-local-recaptcha.ps1` consumes the sensitive `local_recaptcha_config` output and updates only the gitignored local backend and photo-app environment files. The API key never goes to the browser. See [local development](../../../../scripts/README.md#local-development).

## Developer setup

1. Install the Google Cloud CLI and Terraform matching the version configured
   for the existing `THS-Armada/armadacms-gcp-local` workspace.
2. Ask a maintainer for access to that workspace if needed. Use your own HCP
   Terraform and Google accounts.
3. From the ArmadaCMS repository root, authenticate and initialize your checkout:

   ```powershell
   terraform login
   gcloud auth application-default login

   if (-not (Test-Path -LiteralPath 'infra/terraform/gcp/local/backend.tf')) {
       Copy-Item -LiteralPath 'infra/terraform/gcp/local/backend.tf.example' `
           -Destination 'infra/terraform/gcp/local/backend.tf'
   }
   terraform "-chdir=infra/terraform/gcp/local" init
   ```

The backend template already selects `THS-Armada/armadacms-gcp-local`. Verify
those values if you already have a local `backend.tf`, and keep that file
gitignored. Re-run `terraform init` when the backend or providers change.
New developers reuse the existing resources; no workspace creation or apply is
needed for workstation setup.

Terraform uses these Application Default Credentials, which are separate from
`gcloud auth login`. The provider already selects the Armada project for API
quota. Your account needs access to the existing local keys and permission to
use the project's API quota; ask a maintainer if access is missing.

From the ArmadaCMS repository root, run:

```powershell
./scripts/configure-local-recaptcha.ps1
```

This reads the existing workspace output and updates only the ignored backend
and photo-app environment settings. For Web event maps, use the development
`maps_embed_api_key` output to configure `NEXT_PUBLIC_GOOGLE_MAPS_EMBED_API_KEY`
in the sibling `armada.nu/apps/web/.env.local`, preserving its other values.
Do not print key values into logs or commit local environment files. Restart
local applications after changing their environment files.

# Terraform layout

This directory is organized by **provider** and then by **environment/root**.

## Current structure

```text
infra/terraform/
├── README.md
├── gcp/
│   └── prod/      # Current ArmadaCMS runtime stack on Google Cloud
├── aws/           # Reserved for future AWS Terraform roots
└── vercel/        # Reserved for future Vercel Terraform roots
```

## Recommended approach

- Keep **one Terraform root per logical stack**.
- Keep **one HCP Terraform workspace per environment per root**.
- Do not mix unrelated providers in one root just because Terraform technically allows it.

For this repo today:

- `gcp/prod/` is the active Terraform root for the ArmadaCMS runtime stack.
- There is no GCP staging root yet because there is no GCP staging environment.
- AWS and Vercel directories are placeholders for future roots when those parts are ready to be managed separately.

## Suggested workspace naming

Examples:

- `armadacms-gcp-prod`
- `armadacms-aws-prod`
- `armadacms-vercel-prod`

If staging environments are added later, mirror the same pattern:

- `armadacms-gcp-staging`
- `armadacms-aws-staging`
- `armadacms-vercel-staging`

## Active root

See `gcp/prod/README.md` for the current GCP production stack, bootstrap sequence, imports, and HCP Terraform backend setup.

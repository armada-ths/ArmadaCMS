# AWS Terraform roots

This directory is reserved for future AWS-specific Terraform roots.

Recommended pattern when you add AWS infrastructure:

```text
infra/terraform/aws/
└── prod/
```

If you later add a separate staging environment, mirror the same structure:

```text
infra/terraform/aws/
├── prod/
└── staging/
```

Keep AWS state in its own HCP Terraform workspace(s), separate from the GCP runtime root.

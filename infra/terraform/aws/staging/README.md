# ArmadaCMS Terraform — AWS staging

This Terraform root manages the **AWS staging resources** for `ArmadaCMS`: the S3
file-upload bucket together with the IAM user and policy that allow the Cloud Run
service to write to it.

Staging uses a **Supabase project** for PostgreSQL rather than RDS, so there is no
database resource here.

## What it manages

| File     | Resources                                                           |
| -------- | ------------------------------------------------------------------- |
| `s3.tf`  | `aws_s3_bucket.cms_files` — public-read file bucket                 |
|          | `aws_s3_bucket_public_access_block.cms_files`                       |
|          | `aws_s3_bucket_policy.cms_files` — PublicReadGetObject + HTTPS-only |
| `iam.tf` | `aws_iam_policy.s3_uploads` — `s3:PutObject` on the staging bucket  |
|          | `aws_iam_group.s3_uploaders` — `ArmadaCMSStagingUploads`            |
|          | `aws_iam_group_policy_attachment.s3_uploads`                        |
|          | `aws_iam_user.s3_uploader` — `armadacms-staging-s3`                 |
|          | `aws_iam_user_group_membership.s3_uploader`                         |

The IAM access key for `armadacms-staging-s3` is intentionally **not** managed by
Terraform. Create it manually in the AWS console and store the values as GCP Secret
Manager secrets (`armadacms-staging-AWS_ACCESS_KEY_ID` and
`armadacms-staging-AWS_SECRET_ACCESS_KEY`).

## Architecture notes

- This root manages only file-storage infrastructure for the staging Cloud Run service.
- Unlike production, there is no RDS security group or IP-restricted S3 upload policy
  because staging does not use a static Cloud Run NAT IP.
- Bucket name and region are exported as Terraform outputs and consumed by the
  `armadacms-gcp-staging` workspace via `data.tfe_outputs`.

## Files

| File                 | Purpose                                                      |
| -------------------- | ------------------------------------------------------------ |
| `versions.tf`        | Provider version requirements (aws, tfe)                     |
| `variables.tf`       | Configurable inputs                                          |
| `locals.tf`          | Common tags and S3 bucket name                               |
| `s3.tf`              | S3 bucket, public access block, and bucket policy            |
| `iam.tf`             | S3 upload policy, IAM group, user, and group membership      |
| `outputs.tf`         | Bucket name and region consumed by the GCP staging workspace |
| `backend.tf.example` | HCP Terraform backend template                               |

## Workspace dependencies

This root has no upstream workspace dependencies. It exports `s3_bucket_name` and
`s3_bucket_region` for `armadacms-gcp-staging`.

Grant `armadacms-gcp-staging` remote state read access under **Settings → Remote
state sharing** (or "Share with all workspaces").

For the full cross-workspace layout, see [`../../README.md`](../../README.md).

## HCP Terraform workspace setup

Workspace: `armadacms-aws-staging` in the `THS-Armada` organization.

Copy `backend.tf.example` to `backend.tf`, fill in the workspace name, and run
`terraform init`.

**Environment variables** (set in the workspace):

| Variable                | Notes                                                     |
| ----------------------- | --------------------------------------------------------- |
| `TFC_AWS_PROVIDER_AUTH` | `true` — enables OIDC dynamic credentials                 |
| `TFC_AWS_RUN_ROLE_ARN`  | ARN of the `armadacms-hcp-terraform-aws-staging` IAM role |

## Ongoing operations

### Rotating the S3 access key

1. Create a new access key for `armadacms-staging-s3` in the AWS console.
2. Update the corresponding Secret Manager secrets in the GCP staging project
   (`armadacms-staging-AWS_ACCESS_KEY_ID`, `armadacms-staging-AWS_SECRET_ACCESS_KEY`).
3. Verify uploads work, then delete the old key.

No Terraform changes needed.

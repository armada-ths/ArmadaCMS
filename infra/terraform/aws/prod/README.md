# ArmadaCMS Terraform — AWS production

This Terraform root manages the **AWS production resources** for `ArmadaCMS`: the RDS PostgreSQL database and the S3 file-upload bucket, together with the IAM user and policy that allow the Cloud Run service to write to S3.

## What it manages

| File     | Resources                                                                              |
| -------- | -------------------------------------------------------------------------------------- |
| `rds.tf` | `aws_security_group.rds` — PostgreSQL SG (port 5432, ingress from Cloud Run NAT only) |
|          | `aws_db_instance.main` — PostgreSQL 17.5, db.t4g.micro, 20 GB gp2, publicly accessible |
| `s3.tf`  | `aws_s3_bucket.cms_files` — public-read file bucket                                    |
|          | `aws_s3_bucket_public_access_block.cms_files`                                          |
|          | `aws_s3_bucket_policy.cms_files` — PublicReadGetObject + HTTPS-only                    |
| `iam.tf` | `aws_iam_policy.s3_uploads` — `s3:PutObject` restricted to the Cloud Run NAT IP        |
|          | `aws_iam_group.s3_uploaders` — `ArmadaCMSProductionUploads`                            |
|          | `aws_iam_group_policy_attachment.s3_uploads`                                           |
|          | `aws_iam_user.s3_uploader` — `armadacms-prod-s3`                                       |
|          | `aws_iam_user_group_membership.s3_uploader`                                            |

The IAM access key for `armadacms-prod-s3` is intentionally **not** managed by Terraform. Rotate it manually in the AWS console.

## Architecture notes

- This root manages the AWS database and file-storage infrastructure used by the production Cloud Run service.
- It reads `static_egress_ip` from `armadacms-gcp-prod` so the RDS security group and S3 upload policy stay aligned with the current Cloud Run outbound IP automatically.
- It exports database and bucket values consumed by `gcp/prod`.

## Files

| File                 | Purpose                                                                       |
| -------------------- | ----------------------------------------------------------------------------- |
| `versions.tf`        | Provider version requirements (aws, tfe)                                      |
| `variables.tf`       | Configurable inputs                                                           |
| `locals.tf`          | Common tags, bucket/RDS names, NAT CIDR (read from GCP workspace)             |
| `gcp_state.tf`       | `data.tfe_outputs.gcp_prod` — reads `static_egress_ip` from the GCP workspace |
| `rds.tf`             | RDS security group and PostgreSQL instance                                    |
| `s3.tf`              | S3 bucket, public access block, and bucket policy                             |
| `iam.tf`             | S3 upload policy, IAM group, user, and group membership                       |
| `imports.tf`         | Native import blocks for all managed resources                                |
| `outputs.tf`         | Infrastructure values consumed by the GCP workspace                           |
| `prod.auto.tfvars`   | Committed non-secret production defaults                                      |
| `backend.tf.example` | HCP Terraform backend template                                                |

## Workspace dependencies

This root consumes `static_egress_ip` from `armadacms-gcp-prod` and exports the database and bucket outputs consumed by `gcp/prod`.

For the full cross-workspace wiring and remote state sharing setup, see [`../../README.md`](../../README.md).

## HCP Terraform workspace setup

Workspace: `armadacms-aws-prod` in the `THS-Armada` organization.

**Terraform variables** (set in the workspace):

| Variable      | Sensitive | Notes               |
| ------------- | --------- | ------------------- |
| `db_password` | Yes       | RDS master password |

**Environment variables** (set in the workspace):

| Variable                | Notes                                                                           |
| ----------------------- | ------------------------------------------------------------------------------- |
| `TFE_TOKEN`             | HCP Terraform API token — required for `data.tfe_outputs` cross-workspace reads |
| `TFC_AWS_PROVIDER_AUTH` | `true` — enables OIDC dynamic credentials                                       |
| `TFC_AWS_RUN_ROLE_ARN`  | ARN of the `armadacms-hcp-terraform-aws-prod` IAM role                          |

## Ongoing operations

### Rotating the S3 access key

1. Create a new access key for `armadacms-prod-s3` in the AWS console.
2. Update `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in Secret Manager.
3. Verify uploads work, then delete the old key.

No Terraform changes needed.

### Updating the RDS instance class or storage

Edit `rds.tf` and run `terraform apply`. Instance class changes may require a maintenance window restart.

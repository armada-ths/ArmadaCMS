# ArmadaCMS Terraform — AWS production

This Terraform root manages the **AWS production resources** for `ArmadaCMS`: the RDS PostgreSQL database and the S3 file-upload bucket, together with the IAM user and policy that allow the Cloud Run service to write to S3.

All resources already exist in the AWS account (`593054043164`, region `eu-north-1`). The first `terraform apply` will **import** them into Terraform state without making changes.

---

## What it manages

| File     | Resources                                                                              |
| -------- | -------------------------------------------------------------------------------------- |
| `rds.tf` | `aws_security_group.rds` — PostgreSQL SG (port 5432, ingress from Cloud Run NAT only)  |
|          | `aws_db_instance.main` — PostgreSQL 17.5, db.t4g.micro, 20 GB gp2, publicly accessible |
| `s3.tf`  | `aws_s3_bucket.cms_files` — public-read file bucket                                    |
|          | `aws_s3_bucket_public_access_block.cms_files`                                          |
|          | `aws_s3_bucket_policy.cms_files` — PublicReadGetObject + HTTPS-only                    |
| `iam.tf` | `aws_iam_policy.s3_uploads` — s3:PutObject restricted to Cloud Run NAT IP              |
|          | `aws_iam_group.s3_uploaders` — `ArmadaCMSProductionUploads`                            |
|          | `aws_iam_group_policy_attachment.s3_uploads`                                           |
|          | `aws_iam_user.s3_uploader` — `armadacms-prod-s3`                                       |
|          | `aws_iam_group_membership.s3_uploader`                                                 |

The IAM access key for `armadacms-prod-s3` is intentionally **not** managed by Terraform (storing secret key material in state is an unnecessary risk). Rotate it manually in the AWS console.

---

## Prerequisites

- Terraform >= 1.7 installed locally
- AWS credentials with read/write access to RDS, S3, and IAM in the target account
- HCP Terraform workspace `armadacms-aws-prod` created in the `THS-Armada` organization (or remove `backend.tf` for local state)
- If using HCP Terraform: add `db_password` as a **sensitive Terraform variable** in the workspace

---

## First-time setup

### 1. Configure the backend

The default `backend.tf` points to HCP Terraform. If you want to use **local state** for the initial import:

```bash
# Temporarily disable remote backend
Remove-Item backend.tf   # or rename it away
terraform init
```

### 2. Supply the DB password

The `db_password` variable is required but sensitive. Pass it without writing it to disk:

```bash
# PowerShell
$env:TF_VAR_db_password = "your-password"
terraform plan
```

Or add it as a sensitive variable in HCP Terraform.

### 3. Init and import

```bash
terraform init
terraform plan   # shows import actions for all existing resources
terraform apply  # imports everything; no infrastructure is created or destroyed
```

The `imports.tf` file uses Terraform 1.5+ native import blocks, so the plan output will show each resource being imported. After a clean apply all resources will show **"No changes"** on subsequent plans.

### 4. Verify

```bash
terraform output
```

Expected outputs: RDS endpoint, S3 bucket name/ARN, IAM user ARN.

---

## Ongoing operations

### Updating the Cloud Run NAT IP

If the GCP NAT static IP changes, update `cloud_run_nat_ip` in `prod.auto.tfvars` and run `terraform apply`. This will update both the RDS security group ingress rule and the IAM policy IP condition atomically.

### Rotating the S3 access key

1. In the AWS console, create a new access key for `armadacms-prod-s3`.
2. Update `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in Cloud Run (via Secret Manager).
3. Verify uploads work, then delete the old key.

Terraform does not manage access keys, so no `terraform apply` is needed.

### Adjusting RDS storage or instance class

Edit `rds.tf` and run `terraform apply`. RDS instance class changes require a maintenance window or force a brief restart.

---

## Variables

| Variable           | Default                 | Description               |
| ------------------ | ----------------------- | ------------------------- |
| `region`           | `eu-north-1`            | AWS region                |
| `environment`      | `production`            | Environment tag value     |
| `cloud_run_nat_ip` | `34.51.249.94`          | Cloud Run NAT outbound IP |
| `db_password`      | _(required, sensitive)_ | RDS master password       |

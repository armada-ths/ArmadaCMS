output "cloud_build_service_account_email" {
  description = "Cloud Build service account that has deploy permissions."
  value       = local.cloud_build_service_account_email
}

output "runtime_service_account_email" {
  description = "Service account used by the ArmadaCMS staging Cloud Run service."
  value       = var.manage_runtime_service_account ? google_service_account.runtime[0].email : (trimspace(var.cloud_run_service_account_email) != "" ? var.cloud_run_service_account_email : local.default_compute_service_account_email)
}

output "artifact_registry_repository" {
  description = "Shared production Artifact Registry repository path where ArmadaCMS staging images are pushed."
  value       = "${var.prod_artifact_registry_host}/${var.project_id}/${var.prod_artifact_registry_repository_id}"
}

output "default_image_uri" {
  description = "Image URI Terraform expects Cloud Run to use on first deployment."
  value       = local.container_image
}

output "runtime_secret_ids" {
  description = "Secret Manager secret IDs used by ArmadaCMS staging runtime environment variables."
  value       = { for key, secret in google_secret_manager_secret.app : key => secret.secret_id }
}

output "github_app_private_key_secret_id" {
  description = "Secret Manager secret ID used by cloudbuild.yaml for GitHub deployment tracking."
  value       = var.manage_github_app_secret ? google_secret_manager_secret.github_app_private_key[0].secret_id : var.github_app_private_key_secret_id
}

output "cloud_run_service_name" {
  description = "Managed Cloud Run service name, or null when deploy_cloud_run_service is false."
  value       = var.deploy_cloud_run_service ? google_cloud_run_v2_service.armadacms[0].name : null
}

output "cloud_run_service_url" {
  description = "Default Cloud Run service URL, or null when deploy_cloud_run_service is false."
  value       = var.deploy_cloud_run_service ? google_cloud_run_v2_service.armadacms[0].uri : null
}

output "static_egress_ip" {
  description = "Reserved static egress IP when optional staging VPC egress is enabled."
  value       = var.enable_vpc_egress ? google_compute_address.nat[0].address : null
}

output "cloud_run_vpc_network" {
  description = "VPC network used by Cloud Run, or null when VPC egress is disabled."
  value       = var.enable_vpc_egress ? local.selected_vpc_network_name : null
}

output "cloud_run_vpc_subnetwork" {
  description = "VPC subnetwork used by Cloud Run, or null when VPC egress is disabled."
  value       = var.enable_vpc_egress ? local.selected_vpc_subnetwork_name : null
}

output "domain_mapping_resource_records" {
  description = "DNS records that must be added for staging.cms.armada.nu to complete the Cloud Run domain mapping. Add these to your DNS provider, then wait for Google to provision the managed certificate."
  value       = var.enable_domain_mapping && var.deploy_cloud_run_service ? try(google_cloud_run_domain_mapping.staging[0].status[0].resource_records, []) : []
}

output "recaptcha_site_key_id" {
  description = "reCAPTCHA Enterprise site key ID for the armada.nu website, or null when disabled."
  value       = var.enable_recaptcha ? reverse(split("/", google_recaptcha_enterprise_key.website[0].name))[0] : null
}

output "recaptcha_assessment_api_key" {
  description = "GCP API key for server-side reCAPTCHA Enterprise assessment calls (set as RECAPTCHA_SECRET_KEY in Vercel), or null when disabled."
  value       = var.enable_recaptcha ? google_apikeys_key.recaptcha_assessment[0].key_string : null
  sensitive   = true
}

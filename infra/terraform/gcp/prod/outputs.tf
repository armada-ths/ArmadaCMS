output "cloud_build_service_account_email" {
  description = "Cloud Build service account that has deploy permissions."
  value       = local.cloud_build_service_account_email
}

output "runtime_service_account_email" {
  description = "Service account used by the ArmadaCMS Cloud Run service."
  value       = var.manage_runtime_service_account ? google_service_account.runtime[0].email : (trimspace(var.cloud_run_service_account_email) != "" ? var.cloud_run_service_account_email : local.default_compute_service_account_email)
}

output "artifact_registry_repository" {
  description = "Artifact Registry repository path used for ArmadaCMS images."
  value       = "${local.artifact_registry_host}/${var.project_id}/${google_artifact_registry_repository.docker.repository_id}"
}

output "default_image_uri" {
  description = "Image URI Terraform expects Cloud Run to use on first deployment."
  value       = local.container_image
}

output "runtime_secret_ids" {
  description = "Secret Manager secret IDs used by ArmadaCMS runtime environment variables."
  value       = { for key, secret in google_secret_manager_secret.app : key => secret.secret_id }
}

output "github_app_private_key_secret_id" {
  description = "Secret Manager secret ID used by cloudbuild.yaml for GitHub deployment tracking."
  value       = google_secret_manager_secret.github_app_private_key.secret_id
}

output "cloud_run_service_name" {
  description = "Managed Cloud Run service name, or null when deploy_cloud_run_service is false."
  value       = var.deploy_cloud_run_service ? google_cloud_run_v2_service.armadacms[0].name : null
}

output "cloud_run_service_url" {
  description = "Managed Cloud Run service URL, or null when deploy_cloud_run_service is false."
  value       = var.deploy_cloud_run_service ? google_cloud_run_v2_service.armadacms[0].uri : null
}

output "vpc_connector_id" {
  description = "Serverless VPC Access connector ID when enable_vpc_egress is true."
  value       = var.enable_vpc_egress && var.cloud_run_vpc_egress_mode == "CONNECTOR" ? google_vpc_access_connector.serverless[0].id : null
}

output "static_egress_ip" {
  description = "Reserved static egress IP to allow-list on AWS RDS when enable_vpc_egress is true."
  value       = var.enable_vpc_egress ? google_compute_address.nat[0].address : null
}

output "cloud_run_vpc_network" {
  description = "VPC network used by Cloud Run for direct or connector-based egress, or null when disabled."
  value       = var.enable_vpc_egress ? local.selected_vpc_network_name : null
}

output "cloud_run_vpc_subnetwork" {
  description = "VPC subnetwork used by Cloud Run for direct egress and Cloud NAT, or null when disabled."
  value       = var.enable_vpc_egress ? local.selected_vpc_subnetwork_name : null
}

output "load_balancer_ip_address" {
  description = "Global IP address of the HTTPS load balancer when lb_reserve_global_ip is enabled."
  value       = var.enable_https_load_balancer && var.lb_reserve_global_ip ? google_compute_global_address.cloud_run_lb[0].address : null
}

output "load_balancer_backend_service" {
  description = "Backend service name for the external HTTPS load balancer, or null when disabled."
  value       = var.enable_https_load_balancer ? google_compute_backend_service.cloud_run_lb[0].name : null
}

output "load_balancer_serverless_neg" {
  description = "Serverless NEG name for the Cloud Run-backed load balancer, or null when disabled."
  value       = var.enable_https_load_balancer ? google_compute_region_network_endpoint_group.cloud_run_lb_neg[0].name : null
}

output "recaptcha_site_key_id" {
  description = "reCAPTCHA Enterprise site key ID for the armada.nu website, or null when disabled."
  value       = var.enable_recaptcha ? reverse(split("/", google_recaptcha_enterprise_key.website[0].name))[0] : null
}

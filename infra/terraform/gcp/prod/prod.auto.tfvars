project_id = "just-sunrise-491718-m9"
region     = "europe-north2"

storage_provider = "supabase"
# Storage cutover: uncomment the block below and change storage_provider to "supabase"
# after all S3 files have been copied and the URL rewrite SQL has been run.
# Then add SUPABASE_STORAGE_ACCESS_KEY_ID and SUPABASE_STORAGE_SECRET_ACCESS_KEY
# directly in GCP Secret Manager (NOT via HCP Terraform workspace variables).
# supabase_url                 = "https://rsdjnixgxqauonaofrwr.supabase.co"
# supabase_storage_s3_endpoint = "https://rsdjnixgxqauonaofrwr.storage.supabase.co/storage/v1/s3"
# supabase_storage_bucket      = "armadacms-files"
# supabase_storage_region      = "eu-north-1"

deploy_cloud_run_service = true
deletion_protection      = true
cloud_run_ingress        = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

manage_runtime_service_account    = false
cloud_run_service_account_email   = "475154911163-compute@developer.gserviceaccount.com"
cloud_build_service_account_email = "475154911163-compute@developer.gserviceaccount.com"
manage_cloud_build_triggers       = true

cloud_run_cpu    = "1000m"
cloud_run_memory = "1Gi"
min_instances    = 0
max_instances    = 2

enable_vpc_egress   = true
vpc_network_name    = "default"
vpc_subnetwork_name = "default"

enable_https_load_balancer     = true
lb_managed_certificate_domains = ["cms.armada.nu"]

enable_recaptcha          = true
recaptcha_allowed_domains = ["armada.nu"]

revalidation_url = "https://armada.nu/api/revalidate"

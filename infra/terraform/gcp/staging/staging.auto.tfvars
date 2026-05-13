project_id   = "just-sunrise-491718-m9"
region       = "europe-north1"
environment  = "staging"
service_name = "armadacms-staging"
name_prefix  = "armadacms-staging"

storage_provider             = "supabase"
supabase_url                 = "https://rsdjnixgxqauonaofrwr.supabase.co"
supabase_storage_s3_endpoint = "https://rsdjnixgxqauonaofrwr.storage.supabase.co/storage/v1/s3"
supabase_storage_bucket      = "armadacms-files"
supabase_storage_region      = "eu-north-1"

deploy_cloud_run_service = true
deletion_protection      = false

cloud_run_ingress = "INGRESS_TRAFFIC_ALL"

manage_runtime_service_account    = false
cloud_run_service_account_email   = "475154911163-compute@developer.gserviceaccount.com"
cloud_build_service_account_email = "475154911163-compute@developer.gserviceaccount.com"
manage_cloud_build_triggers       = true

cloud_run_cpu    = "1000m"
cloud_run_memory = "512Mi"
min_instances    = 0
max_instances    = 1

enable_vpc_egress = false

enable_domain_mapping   = true
domain_mapping_hostname = "staging.cms.armada.nu"

db_host = "db.dqeikqjiztvmifmnbzbf.supabase.co"
db_user = "postgres"
db_name = "postgres"

revalidation_url = "https://staging.armada.nu/api/revalidate"

# Initial admin user seeded on first startup when the DB is empty.
# Set secret_values["INITIAL_ADMIN_PASSWORD"] in HCP Terraform to activate.
# Once the account is created the env vars have no further effect.
initial_admin_username = "admin"

# The github-app-private-key secret already exists from the production
# workspace — set manage_github_app_secret = false to reuse it.
manage_github_app_secret = false

enable_https_load_balancer = false

enable_recaptcha          = true
recaptcha_allowed_domains = ["staging.armada.nu"]

project_id = "just-sunrise-491718-m9"
region     = "europe-north2"

deploy_cloud_run_service = true
deletion_protection      = true
cloud_run_ingress        = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

manage_runtime_service_account     = true
manage_cloud_build_service_account = true
manage_secret_accessor_bindings    = true
manage_cloud_build_triggers        = true

cloud_run_cpu    = "1000m"
cloud_run_memory = "1Gi"
min_instances    = 0
max_instances    = 2

enable_vpc_egress            = true
manage_vpc_network_resources = true
vpc_network_name             = "armadacms-serverless"
vpc_subnetwork_name          = "armadacms-serverless-europe-north2"

enable_https_load_balancer     = true
lb_managed_certificate_domains = ["cms.armada.nu"]

enable_recaptcha          = true
recaptcha_allowed_domains = ["armada.nu"]

revalidation_url = "https://armada.nu/api/revalidate"

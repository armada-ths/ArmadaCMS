data "google_compute_network" "existing" {
  count = var.enable_vpc_egress && !var.manage_vpc_network_resources ? 1 : 0

  project = var.project_id
  name    = var.vpc_network_name
}

resource "google_compute_network" "serverless" {
  count = var.enable_vpc_egress && var.manage_vpc_network_resources ? 1 : 0

  project                 = var.project_id
  name                    = var.vpc_network_name
  auto_create_subnetworks = false
  routing_mode            = "REGIONAL"

  depends_on = [google_project_service.enabled]
}

data "google_compute_subnetwork" "existing" {
  count = var.enable_vpc_egress && !var.manage_vpc_network_resources ? 1 : 0

  project = var.project_id
  region  = var.region
  name    = var.vpc_subnetwork_name
}

resource "google_compute_subnetwork" "serverless" {
  count = var.enable_vpc_egress && var.manage_vpc_network_resources ? 1 : 0

  project                  = var.project_id
  region                   = var.region
  name                     = var.vpc_subnetwork_name
  ip_cidr_range            = var.vpc_subnet_cidr
  network                  = google_compute_network.serverless[0].id
  private_ip_google_access = true
}

locals {
  selected_vpc_network_name    = var.enable_vpc_egress ? (var.manage_vpc_network_resources ? google_compute_network.serverless[0].name : data.google_compute_network.existing[0].name) : null
  selected_vpc_network_id      = var.enable_vpc_egress ? (var.manage_vpc_network_resources ? google_compute_network.serverless[0].id : data.google_compute_network.existing[0].id) : null
  selected_vpc_subnetwork_name = var.enable_vpc_egress ? (var.manage_vpc_network_resources ? google_compute_subnetwork.serverless[0].name : data.google_compute_subnetwork.existing[0].name) : null
  selected_vpc_subnetwork_id   = var.enable_vpc_egress ? (var.manage_vpc_network_resources ? google_compute_subnetwork.serverless[0].id : data.google_compute_subnetwork.existing[0].id) : null
}

resource "google_compute_address" "nat" {
  count = var.enable_vpc_egress ? 1 : 0

  project = var.project_id
  region  = var.region
  name    = var.nat_ip_name
}

resource "google_compute_router" "nat" {
  count = var.enable_vpc_egress ? 1 : 0

  project = var.project_id
  region  = var.region
  name    = var.nat_router_name
  network = local.selected_vpc_network_id
}

resource "google_compute_router_nat" "serverless" {
  count = var.enable_vpc_egress ? 1 : 0

  project                            = var.project_id
  region                             = var.region
  name                               = var.nat_name
  router                             = google_compute_router.nat[0].name
  nat_ip_allocate_option             = "MANUAL_ONLY"
  nat_ips                            = [google_compute_address.nat[0].self_link]
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"

  subnetwork {
    name                    = local.selected_vpc_subnetwork_id
    source_ip_ranges_to_nat = ["PRIMARY_IP_RANGE"]
  }

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

resource "google_vpc_access_connector" "serverless" {
  count = var.enable_vpc_egress && var.cloud_run_vpc_egress_mode == "CONNECTOR" ? 1 : 0

  project       = var.project_id
  region        = var.region
  name          = var.vpc_connector_name
  network       = local.selected_vpc_network_name
  ip_cidr_range = var.vpc_connector_cidr

  depends_on = [google_compute_subnetwork.serverless]
}

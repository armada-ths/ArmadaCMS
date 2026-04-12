# Cloud Run custom domain mapping for staging.cms.armada.nu.
#
# This uses the Cloud Run domain mapping API (available in europe-north1) to
# serve the staging service under a custom domain without a load balancer.
#
# After applying, run `terraform output domain_mapping_resource_records` to see
# the DNS records that must be added to staging.cms.armada.nu. The records are
# typically a CNAME pointing to ghs.googlehosted.com (or A/AAAA for root
# domains). DNS verification and certificate provisioning can take several
# minutes.
#
# PREREQUISITE: The domain staging.cms.armada.nu must be verified in Search
# Console or via the Google Domains verification flow for the GCP project.

import {
  id = "locations/europe-north1/namespaces/just-sunrise-491718-m9/domainmappings/staging.cms.armada.nu"
  to = google_cloud_run_domain_mapping.staging[0]
}

resource "google_cloud_run_domain_mapping" "staging" {
  count = var.enable_domain_mapping && var.deploy_cloud_run_service ? 1 : 0

  project  = var.project_id
  location = var.region
  name     = var.domain_mapping_hostname

  metadata {
    # Use the project number (not the project ID) to match what the Cloud Run
    # API stores internally. Using the project ID here causes a namespace
    # mismatch on import which triggers a force-replace.
    namespace = data.google_project.current.number
  }

  spec {
    route_name = google_cloud_run_v2_service.armadacms[0].name
  }

  lifecycle {
    ignore_changes = [metadata]
  }

  depends_on = [
    google_cloud_run_v2_service.armadacms,
    google_project_service.enabled,
  ]
}

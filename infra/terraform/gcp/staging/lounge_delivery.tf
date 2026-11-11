resource "google_cloud_scheduler_job" "lounge_delivery" {
  count            = var.deploy_cloud_run_service && var.enable_lounge_orders ? 1 : 0
  project          = var.project_id
  region           = var.scheduler_region
  name             = "${var.service_name}-lounge-delivery"
  description      = "Dispatch queued exhibitor lounge Slack notifications"
  schedule         = "* * * * *"
  time_zone        = "Europe/Stockholm"
  attempt_deadline = "60s"

  http_target {
    http_method = "POST"
    uri         = local.lounge_dispatch_url
    oidc_token {
      service_account_email = local.photo_worker_identity
      audience              = local.lounge_dispatch_url
    }
  }

  depends_on = [google_project_service.enabled]
}

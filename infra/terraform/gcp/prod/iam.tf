resource "google_service_account" "runtime" {
  count = var.manage_runtime_service_account ? 1 : 0

  project      = var.project_id
  account_id   = local.runtime_service_account_id
  display_name = "ArmadaCMS Cloud Run runtime"
  description  = "Runtime identity for the ArmadaCMS Cloud Run service."
}

resource "google_project_iam_member" "cloud_build_roles" {
  for_each = toset([
    "roles/artifactregistry.writer",
    "roles/run.admin",
  ])

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${local.cloud_build_service_account_email}"

  depends_on = [google_project_service.enabled]
}

resource "google_service_account_iam_member" "cloud_build_act_as_runtime" {
  count = var.manage_runtime_service_account ? 1 : 0

  service_account_id = google_service_account.runtime[0].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${local.cloud_build_service_account_email}"
}

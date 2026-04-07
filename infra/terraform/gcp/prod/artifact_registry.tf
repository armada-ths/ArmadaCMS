resource "google_artifact_registry_repository" "docker" {
  project       = var.project_id
  location      = var.region
  repository_id = var.artifact_registry_repository_id
  description   = "Cloud Run Source Deployments"
  format        = "DOCKER"
  labels        = local.labels

  depends_on = [google_project_service.enabled]
}

resource "google_artifact_registry_repository" "docker" {
  project       = var.project_id
  location      = var.region
  repository_id = var.artifact_registry_repository_id
  description   = "Docker images for ArmadaCMS Cloud Run deployments"
  format        = "DOCKER"
  labels        = local.labels

  depends_on = [google_project_service.enabled]
}

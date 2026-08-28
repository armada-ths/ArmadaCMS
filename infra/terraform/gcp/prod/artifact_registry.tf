resource "google_artifact_registry_repository" "docker" {
  project       = var.project_id
  location      = var.region
  repository_id = var.artifact_registry_repository_id
  description   = "Cloud Run Source Deployments"
  format        = "DOCKER"
  labels        = local.labels

  # KEEP policies take precedence over DELETE policies.

  cleanup_policy_dry_run = false

  cleanup_policies {
    id     = "delete-untagged-after-14d"
    action = "DELETE"

    condition {
      tag_state  = "UNTAGGED"
      older_than = "1209600s"
    }
  }

  cleanup_policies {
    id     = "delete-pr-after-30d"
    action = "DELETE"

    condition {
      tag_state    = "TAGGED"
      tag_prefixes = ["pr-"]
      older_than   = "2592000s"
    }
  }

  cleanup_policies {
    id     = "delete-any-after-180d"
    action = "DELETE"

    condition {
      tag_state  = "ANY"
      older_than = "15552000s"
    }
  }

  cleanup_policies {
    id     = "keep-20-most-recent"
    action = "KEEP"

    most_recent_versions {
      keep_count = 20
    }
  }

  depends_on = [google_project_service.enabled]
}

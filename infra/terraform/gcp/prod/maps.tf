resource "google_apikeys_key" "maps_embed_production" {
  project      = var.project_id
  name         = "armadanu-maps-embed-production"
  display_name = "armada.nu Maps Embed (production)"

  restrictions {
    api_targets {
      service = "maps-embed-backend.googleapis.com"
    }
    browser_key_restrictions {
      allowed_referrers = var.maps_embed_allowed_referrers
    }
  }

  lifecycle {
    prevent_destroy = true
  }

  depends_on = [
    google_project_service.enabled["apikeys.googleapis.com"],
    google_project_service.enabled["maps-embed-backend.googleapis.com"],
  ]
}

variable "maps_embed_allowed_referrers" {
  description = "Origins allowed to use this environment's Maps Embed browser key. Add specific preview domains rather than all vercel.app sites."
  type        = list(string)
  default     = ["https://armada.nu/*", "https://www.armada.nu/*"]

  validation {
    condition     = length(var.maps_embed_allowed_referrers) > 0 && alltrue([for referrer in var.maps_embed_allowed_referrers : can(regex("^https?://[^/]+/\\*$", referrer)) && !strcontains(referrer, "*.vercel.app")])
    error_message = "Provide HTTP(S) origin referrers ending in /*; do not allow all vercel.app tenants."
  }
}

output "maps_embed_api_key" {
  description = "This environment's browser-restricted key for Web's NEXT_PUBLIC_GOOGLE_MAPS_EMBED_API_KEY. Never log or commit the value."
  value       = google_apikeys_key.maps_embed_production.key_string
  sensitive   = true
}

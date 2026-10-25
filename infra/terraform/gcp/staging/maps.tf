resource "google_apikeys_key" "maps_embed_staging" {
  project      = var.project_id
  name         = "armadanu-maps-embed-preview"
  display_name = "armada.nu Maps Embed (preview)"

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
}

variable "maps_embed_allowed_referrers" {
  description = "Origins allowed to use this environment's Maps Embed browser key. Add specific preview domains rather than all vercel.app sites."
  type        = list(string)
  default     = ["https://staging.armada.nu/*"]

  validation {
    condition     = length(var.maps_embed_allowed_referrers) > 0 && alltrue([for referrer in var.maps_embed_allowed_referrers : can(regex("^https?://[^/]+/\\*$", referrer)) && !strcontains(referrer, "*.vercel.app")])
    error_message = "Provide HTTP(S) origin referrers ending in /*; do not allow all vercel.app tenants."
  }
}

output "maps_embed_api_key" {
  description = "This environment's browser-restricted key for Web's NEXT_PUBLIC_GOOGLE_MAPS_EMBED_API_KEY. Never log or commit the value."
  value       = google_apikeys_key.maps_embed_staging.key_string
  sensitive   = true
}

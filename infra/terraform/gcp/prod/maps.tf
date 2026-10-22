# Browser-visible keys: separate environments and restrict both API and referrer.
resource "google_apikeys_key" "maps_embed" {
  for_each = var.maps_embed_allowed_referrers

  project      = var.project_id
  name         = "armadanu-maps-embed-${each.key}"
  display_name = "armada.nu Maps Embed (${each.key})"

  restrictions {
    api_targets {
      service = "maps-embed-backend.googleapis.com"
    }
    browser_key_restrictions {
      allowed_referrers = each.value
    }
  }

  depends_on = [
    google_project_service.enabled["apikeys.googleapis.com"],
    google_project_service.enabled["maps-embed-backend.googleapis.com"],
  ]
}

output "maps_embed_api_keys" {
  description = "Browser-restricted Maps Embed keys for the Web Vercel environments. Transfer to NEXT_PUBLIC_GOOGLE_MAPS_EMBED_API_KEY without logging values."
  value       = { for environment, key in google_apikeys_key.maps_embed : environment => key.key_string }
  sensitive   = true
}

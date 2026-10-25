# Retired multi-environment resource address. Existing keys are retained.
removed {
  from = google_apikeys_key.maps_embed
  lifecycle {
    destroy = false
  }
}

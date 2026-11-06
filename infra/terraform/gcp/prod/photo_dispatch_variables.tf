variable "photo_dispatch_url" {
  type        = string
  default     = "https://cms.armada.nu/api/v1/internal/photos/dispatch"
  description = "Public CMS photo dispatcher URL, also used as the exact OIDC audience."
  validation {
    condition     = can(regex("^https://[^/?#]+/api/v1/internal/photos/dispatch$", var.photo_dispatch_url))
    error_message = "Use the HTTPS CMS photo dispatch endpoint without query parameters."
  }
}

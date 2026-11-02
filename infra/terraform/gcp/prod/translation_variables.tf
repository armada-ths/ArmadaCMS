
variable "enable_translation_worker" {
  type        = bool
  default     = false
  description = "Provision the optional AI translation worker and recovery schedule."
}
variable "translation_ai_enabled" {
  type        = bool
  default     = false
  description = "Enable on-demand translation after the schema and worker are ready."
}
variable "translation_model" {
  type    = string
  default = "gemini-3.5-flash-lite"
}
variable "translation_region" {
  type    = string
  default = "eu"
}
variable "translation_daily_token_budget" {
  type    = number
  default = 1000000
  validation {
    condition     = var.translation_daily_token_budget > 0 && floor(var.translation_daily_token_budget) == var.translation_daily_token_budget
    error_message = "The token budget must be a positive integer."
  }
}
check "translation_worker_before_activation" {
  assert {
    condition     = !var.translation_ai_enabled || var.enable_translation_worker
    error_message = "Provision the translation worker before enabling generation."
  }
}

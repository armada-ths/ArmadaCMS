variable "region" {
  description = "AWS region for all resources."
  type        = string
  default     = "eu-north-1"
}

variable "environment" {
  description = "Environment label applied to resource tags."
  type        = string
  default     = "staging"
}



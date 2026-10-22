terraform {
  required_version = ">= 1.7.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.5"
    }
  }
}

variable "project_id" {
  description = "Existing Armada Google Cloud project; APIs are managed by the hosted roots."
  type        = string
  default     = "just-sunrise-491718-m9"
}

provider "google" {
  project = var.project_id
}

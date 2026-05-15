terraform {
  required_version = ">= 1.7.0"

  required_providers {
    supabase = {
      source  = "supabase/supabase"
      version = "~> 1.9"
    }
    tfe = {
      source  = "hashicorp/tfe"
      version = "~> 0.61"
    }
  }
}

provider "supabase" {}

terraform {
  required_version = ">= 1.7.0"

  required_providers {
    tfe = {
      source  = "hashicorp/tfe"
      version = "~> 0.61"
    }

    supabase = {
      source  = "supabase/supabase"
      version = "~> 1.11.0"
    }
  }
}

provider "supabase" {}

terraform {
  required_version = ">= 1.7.0"

  required_providers {
    supabase = {
      source  = "supabase/supabase"
      version = "~> 1.0"
    }
  }
}

provider "supabase" {}

mock_provider "supabase" {}
mock_provider "tfe" {}

variables {
  organization_id   = "test-organization"
  database_password = "unused-by-mocked-provider"
}

run "valid_nat_output" {
  command = plan

  override_data {
    target = data.tfe_outputs.gcp_prod
    values = {
      values = { static_egress_ip = "203.0.113.10" }
    }
  }
}

run "missing_nat_output_fails_closed" {
  command = plan

  override_data {
    target = data.tfe_outputs.gcp_prod
    values = {
      values = {}
    }
  }

  expect_failures = [supabase_settings.production]
}

run "invalid_nat_output_fails_closed" {
  command = plan

  override_data {
    target = data.tfe_outputs.gcp_prod
    values = {
      values = { static_egress_ip = "not-an-ip" }
    }
  }

  expect_failures = [supabase_settings.production]
}

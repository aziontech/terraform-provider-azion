data "azion_application_rules_engine" "example" {
  application_id = 1234567890
  results = [{
    phase = "request"
  }]
}
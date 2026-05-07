data "azion_application_rule_engine" "example" {
  application_id = 1234567890
  results = {
    phase = "request"
    id    = 123456
  }
}
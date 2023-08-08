data "azion_edge_application_rule_engine" "example" {
  edge_application_id = 1234567890
  results = {
    phase = "request"
    id = 123456
  }
}
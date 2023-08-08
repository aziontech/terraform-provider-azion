data "azion_edge_application_rules_engine" "example" {
  edge_application_id = 1234567890
  results =  [{
    phase = "request"
  }]
}
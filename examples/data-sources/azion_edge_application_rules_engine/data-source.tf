data "azion_edge_application_rules_engine" "example" {
  edge_application_id = <edge_application_id>
  results =  [{
    phase = <request> or <response>
  }]
}
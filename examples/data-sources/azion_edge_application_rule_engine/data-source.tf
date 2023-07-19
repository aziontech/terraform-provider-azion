data "azion_edge_application_rule_engine" "example" {
  edge_application_id = <edge_application_id>
  results = {
    phase = <request> or <response>
    id = <rule ID>
  }
}
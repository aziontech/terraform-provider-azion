resource "azion_edge_application_rules_engine" "example" {
  edge_application_id = <edge_application_id>
  results = {
    name = "Terraform Example"
    phase = "request"
    description = "My rule engine"
    behaviors = [
      {
        name = "deliver"
        target = ""
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable= "$${uri}"
            operator= "is_equal"
            conditional= "if"
            input_value= "/"
          }
        ]
      }
    ]
  }
}
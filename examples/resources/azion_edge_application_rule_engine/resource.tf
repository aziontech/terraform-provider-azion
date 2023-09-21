resource "azion_edge_application_rule_engine" "example" {
  edge_application_id = 1234567890
  results = {
    name        = "Terraform Example"
    phase       = "request"
    description = "My rule engine"
    behaviors = [
      {
        name = "deliver"
        "target_object" : {
          "target" = ""
          "regex" : "2379",
          "captured_array" : "Terraform",
          "subject" : "$${device_group}"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${uri}"
            operator    = "is_equal"
            conditional = "if"
            input_value = "/"
          }
        ]
      }
    ]
  }
}
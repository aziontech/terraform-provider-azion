resource "azion_edge_application_rules_engine" "example" {
  edge_application_id = <edge_application_id>
  results = {
    name = "Terraform Example"
    phase = <request> or <response> or <default>
    behaviors: [
      {
        "name": "set_origin",
        "target": "null"
      }
    ]
    criteria: [
      {
        entries : [
          {
            "variable" : "$${uri}",
            "operator" : "is_equal",
            "conditional" : "if",
            "input_value" : "/page"
          }
        ],
      },
      {
        entries : [
          {
            "variable": "$${uri}",
            "operator": "is_equal",
            "conditional": "if",
            "input_value": "/"
          }
        ],
      }
    ]
  }
}
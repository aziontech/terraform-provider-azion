# Example: ordering rule engine rules for an application
# First, create the parent application
resource "azion_application_main_setting" "example" {
  application = {
    name   = "My Application"
    active = true
  }
}

# Create two request-phase rules
resource "azion_application_rule_engine" "first" {
  application_id = azion_application_main_setting.example.application.application_id
  results = {
    name  = "First rule"
    phase = "request"
    behaviors = [
      {
        behavior = {
          type = "deliver"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            criterion = {
              variable    = "$${uri}"
              operator    = "starts_with"
              conditional = "if"
              argument    = "/a/"
            }
          }
        ]
      }
    ]
  }
}

resource "azion_application_rule_engine" "second" {
  application_id = azion_application_main_setting.example.application.application_id
  results = {
    name  = "Second rule"
    phase = "request"
    behaviors = [
      {
        behavior = {
          type = "deliver"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            criterion = {
              variable    = "$${uri}"
              operator    = "starts_with"
              conditional = "if"
              argument    = "/b/"
            }
          }
        ]
      }
    ]
  }
}

# Reorder them: "second" is evaluated before "first" in the request phase.
resource "azion_application_rule_engine_order" "request_order" {
  application_id = azion_application_main_setting.example.application.application_id
  phase          = "request"
  order = [
    azion_application_rule_engine.second.results.id,
    azion_application_rule_engine.first.results.id,
  ]
}

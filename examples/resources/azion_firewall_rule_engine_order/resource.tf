# Example: ordering rule engine rules for a firewall
# First, create the parent firewall
resource "azion_firewall_main_setting" "example" {
  data = {
    name   = "My Firewall"
    active = true
  }
}

# Create two rules
resource "azion_firewall_rule_engine" "first" {
  firewall_id = azion_firewall_main_setting.example.data.id
  results = {
    name = "Block /admin"
    behaviors = [
      {
        behavior = {
          type = "drop"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            criterion = {
              variable    = "$${request_uri}"
              operator    = "matches"
              conditional = "if"
              argument    = "/admin.*"
            }
          }
        ]
      }
    ]
  }
}

resource "azion_firewall_rule_engine" "second" {
  firewall_id = azion_firewall_main_setting.example.data.id
  results = {
    name = "Block /internal"
    behaviors = [
      {
        behavior = {
          type = "drop"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            criterion = {
              variable    = "$${request_uri}"
              operator    = "matches"
              conditional = "if"
              argument    = "/internal.*"
            }
          }
        ]
      }
    ]
  }
}

# Reorder them: "second" is evaluated before "first".
resource "azion_firewall_rule_engine_order" "example" {
  firewall_id = azion_firewall_main_setting.example.data.id
  order = [
    azion_firewall_rule_engine.second.results.id,
    azion_firewall_rule_engine.first.results.id,
  ]
}

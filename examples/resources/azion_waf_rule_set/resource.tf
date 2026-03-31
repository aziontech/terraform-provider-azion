# Create a WAF rule set with basic settings
resource "azion_waf_rule_set" "example" {
  waf_id = 12345
  result = {
    name   = "My WAF Exception"
    active = true

    conditions = [
      {
        match          = "does_not_match"
        condition_type = "generic"
      }
    ]
  }
}

# Create a WAF rule set with more detailed configuration
resource "azion_waf_rule_set" "detailed_example" {
  waf_id = 12345
  result = {
    name     = "Detailed WAF Exception"
    path     = "/api/*"
    operator = "regex"
    active   = true

    conditions = [
      {
        match          = "matches"
        name           = "User-Agent"
        value          = "bot"
        condition_type = "specific_on_name"
      },
      {
        match          = "does_not_match"
        condition_type = "generic"
      }
    ]
  }
}

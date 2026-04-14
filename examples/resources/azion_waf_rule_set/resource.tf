# Create a WAF rule set with a generic condition
# Generic conditions only use the 'match' field
resource "azion_waf_rule_set" "example" {
  waf_id = 12345
  result = {
    name     = "My WAF Exception"
    path     = "/api/*"
    active   = true
    operator = "regex"
    rule_id  = 0

    conditions = [
      {
        match          = "any_url"
        condition_type = "generic"
      }
    ]
  }
}

# Create a WAF rule set with specific condition on name
# Specific on name conditions require the 'name' field
resource "azion_waf_rule_set" "header_example" {
  waf_id = 12345
  result = {
    name     = "Header Exception"
    active   = true
    rule_id  = 0

    conditions = [
      {
        match          = "specific_http_header_name"
        name           = "X-Custom-Header"
        condition_type = "specific_on_name"
      }
    ]
  }
}

# Create a WAF rule set with specific condition on value
# Specific on value conditions require the 'value' field
resource "azion_waf_rule_set" "value_example" {
  waf_id = 12345
  result = {
    name     = "Query String Exception"
    active   = true
    rule_id  = 0

    conditions = [
      {
        match          = "specific_query_string_value"
        value          = "trusted_value"
        condition_type = "specific_on_value"
      }
    ]
  }
}

# Create a WAF rule set with multiple conditions
resource "azion_waf_rule_set" "multi_condition_example" {
  waf_id = 12345
  result = {
    name     = "Multi Condition Exception"
    path     = "/api/v1/*"
    operator = "regex"
    active   = true
    rule_id  = 0

    conditions = [
      {
        match          = "any_url"
        condition_type = "generic"
      },
      {
        match          = "specific_http_header_name"
        name           = "Authorization"
        condition_type = "specific_on_name"
      }
    ]
  }
}

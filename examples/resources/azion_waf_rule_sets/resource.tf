# First, create the parent WAF
resource "azion_waf" "example" {
  result {
    name   = "My WAF"
    active = true
  }
}

# Then declare every exception of that WAF, one rule_set block each. Exceptions
# not listed here are deleted on apply, including ones created in Azion Console.
# Declaring no rule_set block at all deletes every exception of the WAF.
resource "azion_waf_rule_sets" "example" {
  waf_id = azion_waf.example.id

  rule_set {
    name     = "Allow health check"
    rule_id  = 0
    path     = "/health"
    operator = "contains"

    conditions {
      condition {
        match          = "any_url"
        condition_type = "generic"
      }
    }
  }

  rule_set {
    name    = "Allow custom header"
    rule_id = 0

    conditions {
      condition {
        match          = "specific_http_header_name"
        name           = "X-Custom-Header"
        condition_type = "specific_on_name"
      }
    }
  }

  rule_set {
    name    = "Allow trusted query string"
    rule_id = 0
    active  = false

    conditions {
      condition {
        match          = "specific_query_string_value"
        value          = "trusted_value"
        condition_type = "specific_on_value"
      }
    }
  }
}

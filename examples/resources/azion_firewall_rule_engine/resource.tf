# Example: Basic firewall rule with drop behavior
resource "azion_firewall_rule_engine" "block_admin" {
  firewall_id = 1234567890
  results = {
    name        = "Block Admin Path"
    description = "Block access to admin paths"
    active      = true
    behaviors = [
      {
        type = "drop"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${request_uri}"
            operator    = "matches"
            conditional = "if"
            argument    = "/admin.*"
          }
        ]
      }
    ]
  }
}

# Example: Firewall rule with run_function behavior
resource "azion_firewall_rule_engine" "run_function_example" {
  firewall_id = 1234567890
  results = {
    name        = "Execute Function on API"
    description = "Run edge function for API requests"
    active      = true
    behaviors = [
      {
        type = "run_function"
        attributes = {
          value = 4305 # Function instance ID
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${request_uri}"
            operator    = "starts_with"
            conditional = "if"
            argument    = "/api/"
          }
        ]
      }
    ]
  }
}

# Example: Firewall rule with set_custom_response behavior
resource "azion_firewall_rule_engine" "custom_response_example" {
  firewall_id = 1234567890
  results = {
    name        = "Maintenance Page"
    description = "Return custom maintenance page"
    active      = true
    behaviors = [
      {
        type = "set_custom_response"
        attributes = {
          status_code  = 503
          content_type = "text/html"
          content_body = "<html><body><h1>Under Maintenance</h1><p>Please try again later.</p></body></html>"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${host}"
            operator    = "is_equal"
            conditional = "if"
            argument    = "maintenance.example.com"
          }
        ]
      }
    ]
  }
}

# Example: Firewall rule with set_waf behavior
resource "azion_firewall_rule_engine" "waf_example" {
  firewall_id = 1234567890
  results = {
    name        = "Enable WAF Protection"
    description = "Apply WAF rules"
    active      = true
    behaviors = [
      {
        type = "set_waf"
        attributes = {
          waf_id = 98765
          mode   = "blocking"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${host}"
            operator    = "is_equal"
            conditional = "if"
            argument    = "api.example.com"
          }
        ]
      }
    ]
  }
}

# Example: Firewall rule with set_rate_limit behavior
resource "azion_firewall_rule_engine" "rate_limit_example" {
  firewall_id = 1234567890
  results = {
    name        = "Rate Limit API"
    description = "Limit API request rate per client"
    active      = true
    behaviors = [
      {
        type = "set_rate_limit"
        attributes = {
          type               = "second"
          limit_by           = "client_ip"
          average_rate_limit = 100
          maximum_burst_size = 200
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${request_uri}"
            operator    = "starts_with"
            conditional = "if"
            argument    = "/api/"
          }
        ]
      }
    ]
  }
}

# Example: Complex rule with multiple criteria
resource "azion_firewall_rule_engine" "complex_example" {
  firewall_id = 1234567890
  results = {
    name        = "Complex Security Rule"
    description = "Multi-condition security rule"
    active      = true
    behaviors = [
      {
        type = "drop"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${request_uri}"
            operator    = "matches"
            conditional = "if"
            argument    = "/admin.*"
          }
        ]
      },
      {
        entries = [
          {
            variable    = "$${network}"
            operator    = "is_not_in_list"
            conditional = "and"
            argument    = "12345" # Allowed network list ID
          }
        ]
      }
    ]
  }
}

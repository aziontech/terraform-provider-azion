# Example: Complete setup with parent application
# First, create the parent application
resource "azion_application_main_setting" "example" {
  application = {
    name   = "My Application"
    active = true
  }
}

# Then create the rule engine for that application
resource "azion_application_rule_engine" "example" {
  application_id = azion_application_main_setting.example.application.application_id
  results = {
    name        = "Terraform Example"
    phase       = "request"
    description = "My rule engine"
    behaviors = [
      {
        type = "deliver"
      },
      {
        type = "bypass_cache"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${uri}"
            operator    = "is_equal"
            conditional = "if"
            argument    = "/"
          }
        ]
      }
    ]
  }
}

# Example 1: Request phase rule with no-args behavior
resource "azion_application_rule_engine" "example_simple" {
  application_id = 1234567890
  results = {
    name        = "Terraform Example"
    phase       = "request"
    description = "My rule engine"
    behaviors = [
      {
        type = "deliver"
      },
      {
        type = "bypass_cache"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${uri}"
            operator    = "is_equal"
            conditional = "if"
            argument    = "/"
          }
        ]
      }
    ]
  }
}

# Example 2: Request phase rule with behavior that has arguments
resource "azion_application_rule_engine" "example_with_args" {
  application_id = 1234567890
  results = {
    name        = "Add Header Example"
    phase       = "request"
    active      = true
    description = "Rule with behavior that has arguments"

    behaviors = [
      {
        type = "add_request_header"
        attributes = {
          value = "X-Custom-Header: MyValue"
        }
      }
    ]

    criteria = [
      {
        entries = [
          {
            variable    = "$${uri}"
            operator    = "starts_with"
            conditional = "if"
            argument    = "/api/"
          }
        ]
      }
    ]
  }
}

# Example 3: Request phase rule with capture_match_groups behavior
resource "azion_application_rule_engine" "example_capture" {
  application_id = 1234567890
  results = {
    name        = "Capture Match Groups Example"
    phase       = "request"
    description = "Rule with capture_match_groups behavior"

    behaviors = [
      {
        type = "capture_match_groups"
        capture_attributes = {
          subject        = "$${uri}"
          regex          = "/api/([a-z]+)"
          captured_array = "api_paths"
        }
      }
    ]

    criteria = [
      {
        entries = [
          {
            variable    = "$${uri}"
            operator    = "matches"
            conditional = "if"
            argument    = "^/api/"
          }
        ]
      }
    ]
  }
}

# Example 4: Response phase rule
resource "azion_application_rule_engine" "example_response" {
  application_id = 1234567890
  results = {
    name        = "Response Phase Example"
    phase       = "response"
    description = "Response phase rule"

    behaviors = [
      {
        type = "add_response_header"
        attributes = {
          value = "X-Response-Processed: true"
        }
      }
    ]

    criteria = [
      {
        entries = [
          {
            variable    = "$${status}"
            operator    = "is_equal"
            conditional = "if"
            argument    = "200"
          }
        ]
      }
    ]
  }
}

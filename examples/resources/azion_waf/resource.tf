# Create a WAF with basic settings
resource "azion_waf" "example" {
  result = {
    name   = "My WAF"
    active = true
  }
}

# Create a WAF with full engine settings
resource "azion_waf" "full_example" {
  result = {
    name   = "My Full WAF"
    active = true

    engine_settings = {
      engine_version = "2021-Q3"
      type           = "score"

      attributes = {
        rulesets = [1, 2, 3]

        thresholds = [
          {
            threat      = "sql_injection"
            sensitivity = "high"
          },
          {
            threat      = "cross_site_scripting"
            sensitivity = "highest"
          },
          {
            threat      = "directory_traversal"
            sensitivity = "medium"
          }
        ]
      }
    }
  }
}

# Output the WAF ID
output "waf_id" {
  value = azion_waf.example.result.id
}

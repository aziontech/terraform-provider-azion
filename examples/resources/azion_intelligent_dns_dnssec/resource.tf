# Example: Complete setup with parent DNS zone
# First, create the parent DNS zone
resource "azion_intelligent_dns_zone" "example" {
  zone = {
    name    = "example.com"
    active  = true
    domain  = "example.com"
  }
}

# Then configure DNSSEC for that zone
resource "azion_intelligent_dns_dnssec" "example_with_parent" {
  zone_id = azion_intelligent_dns_zone.example.id
  dnssec = {
    is_enabled = true
  }
}

# Example: Using hardcoded zone ID
resource "azion_intelligent_dns_dnssec" "examples" {
  zone_id = "12345"
  dnssec = {
    is_enabled = true
  }
}

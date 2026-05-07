# Example: Complete setup with parent DNS zone
# First, create the parent DNS zone
resource "azion_intelligent_dns_zone" "example" {
  zone = {
    name    = "example.com"
    active  = true
    domain  = "example.com"
  }
}

# Then create the DNS record for that zone
resource "azion_intelligent_dns_record" "example_with_parent" {
  zone_id = azion_intelligent_dns_zone.example.id
  record = {
    type = "A"
    name = "site"
    rdata = [
      "8.8.8.8"
    ]
    policy      = "weighted"
    weight      = 50
    description = "This is a description"
    ttl         = 20
  }
}

# Example: Using hardcoded zone ID
resource "azion_intelligent_dns_record" "example" {
  zone_id = "12345"
  record = {
    type = "A"
    name = "site"
    rdata = [
      "8.8.8.8"
    ]
    policy      = "weighted"
    weight      = 50
    description = "This is a description"
    ttl         = 20
  }
}

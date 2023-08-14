resource "azion_intelligent_dns_dnssec" "examples" {
  zone_id = "12345"
  dns_sec = {
    is_enabled = true
  }
}

resource "azion_intelligent_dns_dnssec" "examples" {
  zone_id = "12345"
  dnssec = {
    is_enabled = true
  }
}

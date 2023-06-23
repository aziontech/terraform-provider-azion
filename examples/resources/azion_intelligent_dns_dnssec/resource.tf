resource "azion_intelligent_dns_dnssec" "examples" {
  zone_id = "<zone_id>"
  dns_sec = {
      is_enabled = true
    }
}

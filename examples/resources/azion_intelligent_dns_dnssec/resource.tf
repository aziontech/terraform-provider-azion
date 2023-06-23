resource "azion_intellignet_dns_dnssec" "examples" {
  zone_id = "<zone_id>"
  dns_sec = {
      is_enabled = true
    }
}

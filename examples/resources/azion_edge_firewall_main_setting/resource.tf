resource "azion_edge_firewall_main_setting" "example" {
  results = {
    name                       = "New EdgeFirewall in terraform"
    is_active                  = true
    edge_functions_enabled     = true
    network_protection_enabled = true
    waf_enabled                = true
    domains                    = []
  }
}
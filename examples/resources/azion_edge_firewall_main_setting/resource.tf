resource "azion_edge_firewall_main_setting" "example" {
  data = {
    name   = "New EdgeFirewall in terraform"
    active = true
    debug  = false
    
    modules = {
      functions = {
        enabled = true
      }
      
      network_protection = {
        enabled = true
      }
      
      waf = {
        enabled = true
      }
    }
  }
}
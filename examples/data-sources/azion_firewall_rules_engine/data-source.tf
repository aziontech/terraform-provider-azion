data "azion_firewall_rules_engine" "example" {
  firewall_id = 12345
  page        = 1
  page_size   = 10
}

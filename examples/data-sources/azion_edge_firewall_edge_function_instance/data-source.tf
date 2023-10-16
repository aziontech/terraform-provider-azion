data "azion_edge_firewall_edge_function_instance" "example" {
  edge_firewall_id = 1234567890
  results = {
    edge_function_instance_id = 123456
  }
}
resource "azion_edge_firewall_edge_functions_instance" "example" {
  edge_firewall_id = 12464
  results = {
    name = "Terraform Test"
    "edge_function_id" : 9359
    "args" : jsonencode(
    { a = "b" })
  }
}

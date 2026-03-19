resource "azion_firewall_functions_instance" "example" {
  firewall_id = 12464
  data = {
    name     = "Terraform Test"
    function = 9359
    args = jsonencode({
      a = "b"
    })
  }
}

resource "azion_application_function_instance" "example" {
  application_id = 1234567890
  data = {
    name        = "Terraform Example"
    function_id = 12345
    active      = true
    args = jsonencode({
      key     = "Value"
      Example = "example"
    })
  }
}
resource "azion_edge_function" "example" {
  edge_function = {
    name           = "Function Terraform Example"
    code           = file("${path.module}/example.txt")
    language       = "javascript"
    initiator_type = "edge_application"
    json_args      = jsonencode(
      {"key" = "Value",
       "key" = "example"
      })
    active         = true/false
  }
}
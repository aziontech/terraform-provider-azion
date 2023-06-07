resource "local_file" "content_file" {
  filename = "${path.module}/example.txt"
  content  = file("${path.module}/example.txt")
}

resource "azion_edge_function" "example" {
  edge_function = {
    name           = "Function Terraform Example"
    code =         local_file.content_file.content or file("${path.module}/example.txt")
    language       = "javascript"
    initiator_type = "edge_application"
    json_args      = jsonencode(
      {"key" = "Value",
       "key" = "example"
      })
    active         = true/false
  }
}
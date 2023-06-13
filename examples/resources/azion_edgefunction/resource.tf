resource "local_file" "content_file" {
  filename = "${path.module}/example.txt"
  content  = file("${path.module}/example.txt")
}

resource "azion_edge_function" "example" {
  edge_function = {
    name           = "Function Terraform Test 5 API"
    code =         local_file.content_file.content
    language       = "javascript"
    initiator_type = "edge_application"
    json_args      = jsonencode(
      {"key": "Value",
        "chave":"4",
        "oi" = "false",
        "teste" = false
      })
    active         = true
  }
}


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
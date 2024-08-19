resource "local_file" "content_file" {
  filename = "${path.module}/example.txt"
  content  = file("${path.module}/example.txt")
}

resource "azion_edge_function" "example1" {
  edge_function = {
    name           = "Function Terraform Example"
    code           = trimspace(local_file.content_file.content)
    language       = "javascript"
    initiator_type = "edge_application"
    json_args      = jsonencode({ "key" = "Value" })
    active         = true
  }
}


resource "azion_edge_function" "example2" {
  edge_function = {
    name           = "Function Terraform Example"
    code           = trimspace(file("${path.module}/example.txt"))
    language       = "javascript"
    initiator_type = "edge_application"
    json_args      = jsonencode({ "key" = "Value" })
    active         = true
  }
}


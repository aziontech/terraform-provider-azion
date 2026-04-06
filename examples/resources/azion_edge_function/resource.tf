resource "local_file" "content_file" {
  filename = "${path.module}/example.txt"
  content  = file("${path.module}/example.txt")
}

resource "azion_edge_function" "example1" {
  edge_function = {
    name                 = "Function Terraform Example"
    code                 = trimspace(local_file.content_file.content)
    active               = true
    default_args         = jsonencode({ "key" = "Value" })
    execution_environment = "default"
    runtime              = "nodejs20.x"
  }
}


resource "azion_edge_function" "example2" {
  edge_function = {
    name                 = "Function Terraform Example"
    code                 = trimspace(file("${path.module}/example.txt"))
    active               = true
    default_args         = jsonencode({ "key" = "Value" })
    execution_environment = "default"
    runtime              = "nodejs20.x"
  }
}

# Example with inline code
resource "azion_edge_function" "example" {
  edge_function = {
    name                  = "Function Terraform Example"
    code                  = "console.log('Hello World');"
    active                = true
    default_args          = jsonencode({ "key" = "Value" })
    execution_environment = "default"
    runtime               = "nodejs20.x"
  }
}

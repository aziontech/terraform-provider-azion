# Example with inline code
resource "azion_function" "example" {
  function = {
    name                  = "Function Terraform Example"
    code                  = "console.log('Hello World');"
    active                = true
    default_args          = jsonencode({ "key" = "Value" })
    execution_environment = "application"
    runtime               = "azion_js"
  }
}
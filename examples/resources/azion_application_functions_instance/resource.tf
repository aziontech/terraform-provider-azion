# Example: Complete setup with parent application
# First, create the parent application with functions module enabled
resource "azion_application_main_setting" "example" {
  application = {
    name   = "My Application"
    active = true
    modules = {
      functions = {
        enabled = true
      }
    }
  }
}

# Then create the function instance for that application
resource "azion_application_function_instance" "example" {
  application_id = azion_application_main_setting.example.application.application_id
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

# Example: Using hardcoded application ID
resource "azion_application_function_instance" "example_simple" {
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
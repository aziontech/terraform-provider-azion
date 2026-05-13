# Example: Complete setup with parent application
# First, create the parent application
resource "azion_application_main_setting" "example" {
  application = {
    name   = "My Application"
    active = true
  }
}

# Then create the device group for that application
resource "azion_application_device_group" "example_with_parent" {
  application_id = azion_application_main_setting.example.application.application_id
  device_group = {
    name       = "mobiledevices"
    user_agent = ".*(Mobile|Android|iPhone).*"
  }
}

# Example: Using hardcoded application ID
resource "azion_application_device_group" "example" {
  application_id = 12345
  device_group = {
    name       = "mobiledevices"
    user_agent = ".*(Mobile|Android|iPhone).*"
  }
}

resource "azion_application_device_group" "example" {
  application_id = 12345
  device_group = {
    name       = "mobiledevices"
    user_agent = ".*(Mobile|Android|iPhone).*"
  }
}

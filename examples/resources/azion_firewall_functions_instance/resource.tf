# Example: Complete setup with parent firewall
# First, create the parent firewall with functions module enabled
resource "azion_firewall_main_setting" "example" {
  data = {
    name   = "My Firewall"
    active = true
    modules = {
      functions = {
        enabled = true
      }
    }
  }
}

# Then create the function instance for that firewall
resource "azion_firewall_functions_instance" "example" {
  firewall_id = azion_firewall_main_setting.example.data.id
  data = {
    name     = "Terraform Test"
    function = 9359
    args = jsonencode({
      a = "b"
    })
  }
}

# Example: Using hardcoded firewall ID
resource "azion_firewall_functions_instance" "example_simple" {
  firewall_id = 12464
  data = {
    name     = "Terraform Test"
    function = 9359
    args = jsonencode({
      a = "b"
    })
  }
}

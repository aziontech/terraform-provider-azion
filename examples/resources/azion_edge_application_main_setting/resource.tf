resource "azion_application_main_setting" "example" {
  application = {
    name   = "Terraform Examples"
    active = true
    debug  = false
    modules = {
      edge_cache = {
        enabled = true
      }
      functions = {
        enabled = false
      }
      application_accelerator = {
        enabled = false
      }
      image_processor = {
        enabled = false
      }
    }
  }
}

resource "azion_application_cache_setting" "example" {
  application_id = azion_application_main_setting.example.application.application_id
  cache_setting = {
    name = "Terraform Main Settings Example"
    browser_cache = {
      behavior = "override"
      max_age  = 20
    }
    modules = {
      cache = {
        behavior = "override"
        max_age  = 60
        stale_cache = {
          enabled = true
        }
      }
    }
  }
  depends_on = [
    azion_application_main_setting.example
  ]
}

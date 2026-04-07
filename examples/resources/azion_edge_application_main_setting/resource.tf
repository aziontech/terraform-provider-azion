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

resource "azion_edge_application_origin" "example" {
  edge_application_id = azion_application_main_setting.example.application.application_id
  origin = {
    name        = "Terraform Main Settings Example"
    origin_type = "single_origin"
    addresses = [
      {
        address = "httpExample.org"
      }
    ]
    origin_protocol_policy = "https"
    host_header            = "$${host}"
  }
  depends_on = [
    azion_application_main_setting.example
  ]
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
    azion_application_main_setting.example,
    azion_edge_application_origin.example
  ]
}

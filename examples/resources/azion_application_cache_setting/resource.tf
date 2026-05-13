# Example: Complete setup with parent application
# First, create the parent application with edge cache enabled
resource "azion_application_main_setting" "example" {
  application = {
    name   = "My Application"
    active = true
    modules = {
      edge_cache = {
        enabled = true
      }
    }
  }
}

# Then create the cache setting for that application
resource "azion_application_cache_setting" "example_with_parent" {
  application_id = azion_application_main_setting.example.application.application_id
  cache_setting = {
    name = "Terraform Cache Setting Example"
    browser_cache = {
      behavior = "override"
      max_age  = 3600
    }
    modules = {
      cache = {
        behavior = "override"
        max_age  = 13660
        stale_cache = {
          enabled = true
        }
        tiered_cache = {
          topology = "nearest-region"
          enabled  = true
        }
      }
      application_accelerator = {
        cache_vary_by_querystring = {
          behavior = "ignore"
        }
        cache_vary_by_cookies = {
          behavior = "ignore"
        }
        cache_vary_by_devices = {
          behavior = "ignore"
        }
      }
    }
  }
}

# Example: Using hardcoded application ID
resource "azion_application_cache_setting" "example" {
  application_id = 1234567890
  cache_setting = {
    name = "Terraform Cache Setting Example"
    browser_cache = {
      behavior = "override"
      max_age  = 3600
    }
    modules = {
      cache = {
        behavior = "override"
        max_age  = 13660
        stale_cache = {
          enabled = true
        }
        tiered_cache = {
          topology = "nearest-region"
          enabled  = true
        }
      }
      application_accelerator = {
        cache_vary_by_querystring = {
          behavior = "ignore"
        }
        cache_vary_by_cookies = {
          behavior = "ignore"
        }
        cache_vary_by_devices = {
          behavior = "ignore"
        }
      }
    }
  }
}
---
subcategory: ""
description: |-
  Provides an Application Cache Setting resource using the V4 API.
---

# azion_application_cache_setting (Resource)

Creates and manages an Application Cache Setting using the Azion V4 API.

## Configuration is the desired state

This resource asserts the **complete** cache setting on every apply, using a full
replacement (`PUT`) rather than a partial update. Two things follow from that, and
both are intentional:

- A change made outside Terraform — in Azion Console, for example — is **reverted**
  on the next `terraform apply`. Inside a block you declare, it also appears in
  `terraform plan` as a diff, including for arguments you leave out of that block.
  A block you leave out (for example `browser_cache` or `modules { cache { tiered_cache { ... } } }`)
  is still reset to its defaults on every apply, but a console change to it does
  not show up in `terraform plan`; declare the block, even empty, to have its drift
  detected.
- Leaving a field or block out of your configuration is not "don't care". It means
  "use the Azion default", and the field is reset to that default. To keep a
  non-default value, declare it explicitly.

The defaults enforced for omitted fields are:

| Field | Enforced default |
|---|---|
| `browser_cache.behavior` | `honor` |
| `browser_cache.max_age` | `0` |
| `modules.cache.behavior` | `override` |
| `modules.cache.max_age` | `60` |
| `modules.cache.stale_cache.enabled` | `true` |
| `modules.cache.large_file_cache.enabled` | `false` |
| `modules.cache.large_file_cache.offset` | `1024` |
| `modules.cache.tiered_cache.enabled` | `false` |
| `modules.cache.tiered_cache.topology` | null (see below) |
| `modules.application_accelerator.cache_vary_by_method` | `[]` |
| `modules.application_accelerator.cache_vary_by_querystring.behavior` | `ignore` |
| `modules.application_accelerator.cache_vary_by_querystring.fields` | `[]` |
| `modules.application_accelerator.cache_vary_by_querystring.sort_enabled` | `false` |
| `modules.application_accelerator.cache_vary_by_cookies.behavior` | `ignore` |
| `modules.application_accelerator.cache_vary_by_cookies.cookie_names` | `[]` |
| `modules.application_accelerator.cache_vary_by_devices.behavior` | `ignore` |
| `modules.application_accelerator.cache_vary_by_devices.device_group` | `[]` |

`modules.cache.tiered_cache.topology` is the one exception to the table above. It
is enforced as null only while tiered cache is disabled, which matches what the
API reports for a tiered cache that was never configured. If you enable tiered
cache without naming a topology, the API assigns one and Terraform keeps it.

Note that drift is only reverted when an apply runs, and only when state is
refreshed — `terraform plan -refresh=false` will not detect it. Terraform also
cannot remove cache settings that were created outside Terraform, since those are
not in state.

Because every apply sends the complete object, any API field this provider does
not model is also reset to its API default.

## Example Usage

### With Parent Application

```terraform
# First, create the parent application with edge cache enabled
resource "azion_application_main_setting" "example" {
  application {
    name   = "My Application"
    active = true
    modules {
      cache {
        enabled = true
      }
    }
  }
}

# Then create the cache setting for that application
resource "azion_application_cache_setting" "example" {
  application_id = azion_application_main_setting.example.application.application_id
  cache_setting {
    name = "Terraform Cache Setting Example"
    browser_cache {
      behavior = "override"
      max_age  = 3600
    }
    modules {
      cache {
        behavior = "override"
        max_age  = 13660
        stale_cache {
          enabled = true
        }
        tiered_cache {
          topology = "nearest-region"
          enabled  = true
        }
      }
      application_accelerator {
        cache_vary_by_querystring {
          behavior = "ignore"
        }
        cache_vary_by_cookies {
          behavior = "ignore"
        }
        cache_vary_by_devices {
          behavior = "ignore"
        }
      }
    }
  }
}
```

### Using Hardcoded Application ID

```terraform
resource "azion_application_cache_setting" "example" {
  application_id = 1234567890
  cache_setting {
    name = "Terraform Cache Setting Example"
    browser_cache {
      behavior = "override"
      max_age  = 3600
    }
    modules {
      cache {
        behavior = "override"
        max_age  = 13660
        stale_cache {
          enabled = true
        }
        tiered_cache {
          topology = "nearest-region"
          enabled  = true
        }
      }
      application_accelerator {
        cache_vary_by_querystring {
          behavior = "ignore"
        }
        cache_vary_by_cookies {
          behavior = "ignore"
        }
        cache_vary_by_devices {
          behavior = "ignore"
        }
      }
    }
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `application_id` (Number) Numeric identifier of the Application.

### Optional

- `cache_setting` (Block, Optional) Required. Cache setting configuration. (see [below for nested schema](#nestedblock--cache_setting))

### Read-Only

- `id` (Number) Resource identifier.
- `last_updated` (String) Timestamp of the last Terraform update of the resource.

<a id="nestedblock--cache_setting"></a>
### Nested Schema for `cache_setting`

Required:

- `name` (String) Name of the cache setting.

Optional:

- `browser_cache` (Block, Optional) Browser cache settings. Omitting this block resets it to the Azion defaults on every apply, but its drift is only detected while the block is declared. (see [below for nested schema](#nestedblock--cache_setting--browser_cache))
- `modules` (Block, Optional) Cache settings modules. A module left out is reset to the Azion defaults on every apply, but its drift is only detected while its block is declared. (see [below for nested schema](#nestedblock--cache_setting--modules))

Read-Only:

- `created_at` (String) The creation timestamp.
- `id` (Number) Cache setting identifier.

<a id="nestedblock--cache_setting--browser_cache"></a>
### Nested Schema for `cache_setting.browser_cache`

Optional:

- `behavior` (String) Browser cache behavior: `override`, `honor`, `no-cache`.
- `max_age` (Number) Maximum TTL for browser cache in seconds.


<a id="nestedblock--cache_setting--modules"></a>
### Nested Schema for `cache_setting.modules`

Optional:

- `application_accelerator` (Block, Optional) Application accelerator module settings. Requires the application accelerator module on the application. (see [below for nested schema](#nestedblock--cache_setting--modules--application_accelerator))
- `cache` (Block, Optional) Edge cache module settings. (see [below for nested schema](#nestedblock--cache_setting--modules--cache))

<a id="nestedblock--cache_setting--modules--application_accelerator"></a>
### Nested Schema for `cache_setting.modules.application_accelerator`

Optional:

- `cache_vary_by_cookies` (Block, Optional) Cookies cache variation settings. (see [below for nested schema](#nestedblock--cache_setting--modules--application_accelerator--cache_vary_by_cookies))
- `cache_vary_by_devices` (Block, Optional) Devices cache variation settings. (see [below for nested schema](#nestedblock--cache_setting--modules--application_accelerator--cache_vary_by_devices))
- `cache_vary_by_method` (List of String) HTTP methods to vary cache by.
- `cache_vary_by_querystring` (Block, Optional) Query string cache variation settings. (see [below for nested schema](#nestedblock--cache_setting--modules--application_accelerator--cache_vary_by_querystring))

<a id="nestedblock--cache_setting--modules--application_accelerator--cache_vary_by_cookies"></a>
### Nested Schema for `cache_setting.modules.application_accelerator.cache_vary_by_cookies`

Optional:

- `behavior` (String) Cookies behavior: `ignore`, `all`, `allowlist`, `denylist`.
- `cookie_names` (List of String) Cookie names for allowlist/denylist.


<a id="nestedblock--cache_setting--modules--application_accelerator--cache_vary_by_devices"></a>
### Nested Schema for `cache_setting.modules.application_accelerator.cache_vary_by_devices`

Optional:

- `behavior` (String) Devices behavior: `ignore`, `allowlist`.
- `device_group` (List of Number) Device group IDs for allowlist.


<a id="nestedblock--cache_setting--modules--application_accelerator--cache_vary_by_querystring"></a>
### Nested Schema for `cache_setting.modules.application_accelerator.cache_vary_by_querystring`

Optional:

- `behavior` (String) Query string behavior: `ignore`, `all`, `allowlist`, `denylist`.
- `fields` (List of String) Query string fields for allowlist/denylist.
- `sort_enabled` (Boolean) Enable query string sorting.



<a id="nestedblock--cache_setting--modules--cache"></a>
### Nested Schema for `cache_setting.modules.cache`

Optional:

- `behavior` (String) Cache behavior: `honor`, `override`.
- `large_file_cache` (Block, Optional) Large file cache settings. (see [below for nested schema](#nestedblock--cache_setting--modules--cache--large_file_cache))
- `max_age` (Number) Maximum TTL for edge cache in seconds.
- `stale_cache` (Block, Optional) Stale cache settings. (see [below for nested schema](#nestedblock--cache_setting--modules--cache--stale_cache))
- `tiered_cache` (Block, Optional) Tiered cache settings. Requires the tiered cache module on the application. (see [below for nested schema](#nestedblock--cache_setting--modules--cache--tiered_cache))

<a id="nestedblock--cache_setting--modules--cache--large_file_cache"></a>
### Nested Schema for `cache_setting.modules.cache.large_file_cache`

Optional:

- `enabled` (Boolean) Enable large file cache.
- `offset` (Number) Offset for large file cache slicing.


<a id="nestedblock--cache_setting--modules--cache--stale_cache"></a>
### Nested Schema for `cache_setting.modules.cache.stale_cache`

Optional:

- `enabled` (Boolean) Enable stale cache.


<a id="nestedblock--cache_setting--modules--cache--tiered_cache"></a>
### Nested Schema for `cache_setting.modules.cache.tiered_cache`

Optional:

- `enabled` (Boolean) Enable tiered cache.
- `topology` (String) Tiered cache topology: `nearest-region`, `br-east-1`, `us-east-1`.

## Import

Import is supported using the following syntax:

```shell
terraform import azion_application_cache_setting.example <application_id>/<cache_setting_id>

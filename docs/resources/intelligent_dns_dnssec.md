---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azion_intelligent_dns_dnssec Resource - terraform-provider-azion"
subcategory: ""
description: |-
  
---

# azion_intelligent_dns_dnssec (Resource)



## Example Usage

```terraform
resource "azion_intelligent_dns_dnssec" "examples" {
  zone_id = "12345"
  dns_sec = {
    is_enabled = true
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `dns_sec` (Attributes) (see [below for nested schema](#nestedatt--dns_sec))
- `zone_id` (String) The zone identifier to target for the resource.

### Read-Only

- `last_updated` (String) Timestamp of the last Terraform update of the order.
- `schema_version` (Number) Schema Version.

<a id="nestedatt--dns_sec"></a>
### Nested Schema for `dns_sec`

Required:

- `is_enabled` (Boolean) Zone DNSSEC flags for enabled.

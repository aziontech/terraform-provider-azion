---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azion_zone Resource - azion"
subcategory: ""
description: |-
  
---

# azion_zone (Resource)



## Example Usage

```terraform
resource "azion_zone" "example" {
  zone = {
      domain: "example.com",
      is_active: true,
      name: "example"
    }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `zone` (Attributes) (see [below for nested schema](#nestedatt--zone))

### Read-Only

- `id` (String) Numeric identifier of the order.
- `last_updated` (String) Timestamp of the last Terraform update of the order.
- `schema_version` (Number) Schema Version.

<a id="nestedatt--zone"></a>
### Nested Schema for `zone`

Required:

- `domain` (String) Domain description of the DNS.
- `is_active` (Boolean) Enable description of the DNS.
- `name` (String) Name description of the DNS.

Read-Only:

- `expiry` (Number)
- `id` (Number)
- `nameservers` (List of String)
- `nxttl` (Number)
- `refresh` (Number)
- `retry` (Number)
- `soattl` (Number)

## Import

Import is supported using the following syntax:

```shell
terraform import azion_zone.example <zone_id>
```
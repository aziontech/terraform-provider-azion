---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azion_edge_application_origin Data Source - terraform-provider-azion"
subcategory: ""
description: |-
  
---

# azion_edge_application_origin (Data Source)



## Example Usage

```terraform
data "azion_edge_application_origin" "example" {
  edge_application_id = "1234567890"
  origin = {
    origin_key = "123456"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `edge_application_id` (Number) The edge application identifier.
- `origin` (Attributes) (see [below for nested schema](#nestedatt--origin))

### Read-Only

- `id` (String) Identifier of the data source.
- `schema_version` (Number) Schema Version.

<a id="nestedatt--origin"></a>
### Nested Schema for `origin`

Required:

- `origin_key` (String) Origin key.

Read-Only:

- `addresses` (Attributes List) (see [below for nested schema](#nestedatt--origin--addresses))
- `connection_timeout` (Number) Connection timeout in seconds.
- `hmac_access_key` (String) HMAC access key.
- `hmac_authentication` (Boolean) Whether HMAC authentication is enabled.
- `hmac_region_name` (String) HMAC region name.
- `hmac_secret_key` (String) HMAC secret key.
- `host_header` (String) Host header value.
- `is_origin_redirection_enabled` (Boolean) Whether origin redirection is enabled.
- `method` (String) HTTP method used by the origin.
- `name` (String) Name of the origin.
- `origin_id` (Number) The origin identifier to target for the resource.
- `origin_path` (String) Path of the origin.
- `origin_protocol_policy` (String) Origin protocol policy.
- `origin_type` (String) Type of the origin.
- `timeout_between_bytes` (Number) Timeout between bytes in seconds.

<a id="nestedatt--origin--addresses"></a>
### Nested Schema for `origin.addresses`

Read-Only:

- `address` (String) Address of the origin.
- `is_active` (Boolean) Status of the origin.
- `server_role` (String) Server role of the origin.
- `weight` (Number) Weight of the origin.

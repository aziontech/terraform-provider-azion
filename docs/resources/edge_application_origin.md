---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azion_edge_application_origin Resource - terraform-provider-azion"
subcategory: ""
description: |-
  
---

# azion_edge_application_origin (Resource)



## Example Usage

```terraform
resource "azion_edge_application_origin" "example" {
  edge_application_id = 1234567890
  origin = {
    name        = "Terraform Example"
    origin_type = "single_origin"
    addresses : [
      {
        "address" : "terraform.org"
      }
    ],
    origin_protocol_policy : "http",
    host_header : "$${host}",
    origin_path : "/requests",
    hmac_authentication : false,
    hmac_region_name : "",
    hmac_access_key : "",
    hmac_secret_key : ""
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `edge_application_id` (Number) The edge application identifier.
- `origin` (Attributes) Origin configuration. (see [below for nested schema](#nestedatt--origin))

### Read-Only

- `id` (String) The ID of this resource.
- `last_updated` (String) Timestamp of the last Terraform update of the resource.
- `schema_version` (Number)

<a id="nestedatt--origin"></a>
### Nested Schema for `origin`

Required:

- `addresses` (Attributes List) (see [below for nested schema](#nestedatt--origin--addresses))
- `host_header` (String) Host header value that will be delivered to the origin.
~> **Note about Host Header**
Accepted values: `${host}`(default) and must be specified with `$${host}`
- `name` (String) Origin name.

Optional:

- `connection_timeout` (Number) Connection timeout in seconds.
- `hmac_access_key` (String) HMAC access key.
- `hmac_authentication` (Boolean) Whether HMAC authentication is enabled.
- `hmac_region_name` (String) HMAC region name.
- `hmac_secret_key` (String) HMAC secret key.
- `origin_path` (String) Path of the origin.
- `origin_protocol_policy` (String) Protocols for connection to the origin.
~> **Note about Origin Protocol Policy**
Accepted values: `preserve`(default), `http` and `https`
- `origin_type` (String) Identifies the source of a record.
~> **Note about Origin Type**
Accepted values: `single_origin`(default), `load_balancer` and `live_ingest`
- `timeout_between_bytes` (Number) Timeout between bytes in seconds.

Read-Only:

- `is_origin_redirection_enabled` (Boolean) Whether origin redirection is enabled.
- `method` (String) HTTP method used by the origin.
- `origin_id` (Number) Origin identifier.
- `origin_key` (String) Origin key.

<a id="nestedatt--origin--addresses"></a>
### Nested Schema for `origin.addresses`

Required:

- `address` (String) Address of the origin.

Optional:

- `is_active` (Boolean) Status of the origin.
- `server_role` (String) Server role of the origin.
- `weight` (Number) Weight of the origin.

## Import

Import is supported using the following syntax:

```shell
terraform import azion_edge_application_origin.example <edge_application_id>/<origin_key>
```

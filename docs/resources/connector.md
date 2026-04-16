---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_connector"
description: |-
  Provides a connector resource for connecting to different origin types.
---

# azion_connector

Creates a connector resource. Connectors are polymorphic and support different types (http, storage).

## Example Usage

### Storage Connector

```hcl
resource "azion_connector" "storage_connector" {
  connector = {
    name   = "My Storage Connector"
    type   = "storage"
    active = true
    storage_attributes = {
      bucket = "my-bucket"
      prefix = "path/to/files/"
    }
  }
}
```

### HTTP Connector

```hcl
resource "azion_connector" "http_connector" {
  connector = {
    name   = "My HTTP Connector"
    type   = "http"
    active = true
    http_attributes = {
      addresses = [
        {
          address   = "192.168.1.100"
          http_port = 80
          active    = true
        }
      ]
    }
  }
}
```

### HTTP Connector with All Options

```hcl
resource "azion_connector" "http_connector_full" {
  connector = {
    name   = "My HTTP Connector Full"
    type   = "http"
    active = true
    http_attributes = {
      addresses = [
        {
          address    = "192.168.1.100"
          http_port  = 80
          https_port = 443
          active     = true
          modules = {
            load_balancer = {
              server_role = "primary"
              weight      = 1
            }
          }
        }
      ]
      connection_options = {
        dns_resolution      = "both"
        following_redirect  = false
        host                = "$${host}"
        http_version_policy = "http1_1"
        path_prefix         = ""
        real_ip_header      = "X-Real-IP"
        real_port_header    = "X-Real-PORT"
        transport_policy    = "preserve"
      }
      modules = {
        load_balancer = {
          enabled = false
        }
        origin_shield = {
          enabled = false
        }
      }
    }
  }
}
```

### Using Data Sources

```hcl
# Read a single connector by ID
data "azion_connector" "by_id" {
  id = azion_connector.http_connector.connector.id
}

# List all connectors in the account
data "azion_connectors" "all" {}
```

## Import

```sh
terraform import azion_connector.example 12345
```

## Argument Reference

### Common Arguments

* `connector` - (Required) The connector configuration block. Contains:
  * `name` - (Required) Name of the connector.
  * `type` - (Required) Type of the connector. Must be one of: `http` or `storage`.
  * `active` - (Optional) Status of the connector. Default is `true`.
  * `id` - (Computed) The connector identifier.
  * `created_at` - (Computed) The creation timestamp of the connector.
  * `last_editor` - (Computed) The last editor of the connector.
  * `last_modified` - (Computed) Last modified timestamp of the connector.
  * `product_version` - (Computed) Product version of the connector.

### Storage Connector Arguments

When `type = "storage"`, include `storage_attributes` inside the `connector` block:

* `storage_attributes` - (Required) Attributes for storage type connectors:
  * `bucket` - (Required) The name of the bucket.
  * `prefix` - (Optional) The prefix path within the bucket.

### HTTP Connector Arguments

When `type = "http"`, include `http_attributes` inside the `connector` block:

* `http_attributes` - (Required) Attributes for HTTP type connectors:
  * `addresses` - (Required) List of origin addresses:
    * `address` - (Required) The origin address (IP or hostname).
    * `active` - (Optional) Whether the address is active.
    * `http_port` - (Optional) HTTP port number.
    * `https_port` - (Optional) HTTPS port number.
    * `modules` - (Optional) Address-level modules:
      * `load_balancer` - (Optional) Load balancer module:
        * `server_role` - (Optional) Role in load balancing (`primary` or `backup`).
        * `weight` - (Optional) Weight for load balancing strategy.
  * `connection_options` - (Optional) HTTP connection options (Computed with API defaults):
    * `dns_resolution` - (Optional) DNS resolution strategy (`both` or `force_ipv4`).
    * `following_redirect` - (Optional) Whether to follow redirects.
    * `host` - (Optional) Host header value. Use `$${host}` for original host (escaped Terraform interpolation).
    * `http_version_policy` - (Optional) HTTP version policy.
    * `path_prefix` - (Optional) Path prefix for requests.
    * `real_ip_header` - (Optional) Header for real IP.
    * `real_port_header` - (Optional) Header for real port.
    * `transport_policy` - (Optional) Transport policy.
  * `modules` - (Optional) HTTP modules configuration (Computed with API defaults):
    * `load_balancer` - (Optional) Load balancer module:
      * `enabled` - (Optional) Whether load balancer is enabled.
    * `origin_shield` - (Optional) Origin shield module:
      * `enabled` - (Optional) Whether origin shield is enabled.

## Attribute Reference

The following attributes are exported:

* `id` - The ID of the connector (as a string for Terraform state management).
* `last_updated` - Timestamp of the last Terraform update.
* `schema_version` - Schema version.

## Timeouts

This resource does not have configurable timeouts.

---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_workload"
description: |-
  Provides a resource for managing Azion Workloads.
---

# azion_workload

Creates and manages an Azion Workload resource.

## Example Usage

```hcl
resource "azion_workload" "example" {
  workload = {
    name           = "My Workload"
    active         = true
    infrastructure = 1

    tls = {
      certificate     = 1234
      ciphers         = 3
      minimum_version = "tls_1_2"
    }

    protocols = {
      http = {
        versions    = ["http1", "http2", "http3"]
        http_ports  = [80]
        https_ports = [443]
        quic_ports  = [443]
      }
    }

    mtls = {
      enabled = true
      config = {
        certificate  = 5678
        crl          = [9012]
        verification = "enforce"
      }
    }

    domains                      = ["example.com", "www.example.com"]
    workload_domain_allow_access = true
  }
}
```

## Import

```sh
terraform import azion_workload.example 123456
```

## Argument Reference

The following arguments are supported:

* `workload` - (Required) The workload configuration. See [Workload Configuration](#workload-configuration) below.

### Workload Configuration

The `workload` block supports:

* `name` - (Required) Name of the workload.
* `active` - (Optional) Status of the workload. Default is `false`.
* `infrastructure` - (Optional) Infrastructure type: `1` for Production (All Locations), `2` for Staging.
* `tls` - (Optional) TLS configuration for the workload. See [TLS Configuration](#tls-configuration) below.
* `protocols` - (Optional) Protocol configurations for the workload. See [Protocols Configuration](#protocols-configuration) below.
* `mtls` - (Optional) Mutual TLS configuration for the workload. See [MTLS Configuration](#mtls-configuration) below.
* `domains` - (Optional) Set of domains associated with the workload. Order is not significant and duplicates are not allowed.
* `workload_domain_allow_access` - (Optional) Whether domain access is allowed.

### TLS Configuration

The `tls` block supports:

* `certificate` - (Optional) Certificate ID for TLS.
* `ciphers` - (Optional) Cipher suite configuration. Valid values are:
  * `1` - TLSv1.2_2018
  * `2` - TLSv1.2_2019
  * `3` - TLSv1.3_2022
  * `4` - TLSv1.2_2021
  * `5` - Legacy_v2025Q1
  * `6` - Compatible_v2025Q1
  * `7` - Modern_v2025Q1
  * `8` - Legacy_v2017Q1
* `minimum_version` - (Optional) Minimum TLS version. Valid values: `tls_1_0`, `tls_1_1`, `tls_1_2`, `tls_1_3`.

### Protocols Configuration

The `protocols` block supports:

* `http` - (Optional) HTTP protocol configuration. See [HTTP Protocol Configuration](#http-protocol-configuration) below.

#### HTTP Protocol Configuration

The `http` block supports:

* `versions` - (Optional) List of HTTP versions supported. Valid values: `http1`, `http2`, `http3`.
* `http_ports` - (Optional) List of HTTP ports.
* `https_ports` - (Optional) List of HTTPS ports.
* `quic_ports` - (Optional) List of QUIC ports. When the `http` block is set but `quic_ports` is omitted, the provider defaults this attribute to `[443]` to match the API default and avoid drift between plan and state. Omitting the entire `http` block does not trigger the default.

### MTLS Configuration

The `mtls` block supports:

* `enabled` - (Optional) Whether MTLS is enabled.
* `config` - (Optional) MTLS configuration. See [MTLS Config](#mtls-config) below.

#### MTLS Config

The `config` block supports:

* `certificate` - (Optional) MTLS certificate ID.
* `crl` - (Optional) List of Certificate Revocation List IDs.
* `verification` - (Optional) MTLS verification type. Valid values: `enforce`, `permissive`.

## Attribute Reference

The following attributes are exported:

* `id` - The ID of the workload.
* `last_updated` - Timestamp of the last Terraform update of the resource.

### Workload Attributes

The following attributes are exported in the `workload` block:

* `id` - The workload identifier.
* `last_editor` - The last editor of the workload.
* `last_modified` - Last modified timestamp of the workload.
* `created_at` - Creation timestamp of the workload.
* `workload_domain` - The workload domain.
* `product_version` - Product version of the workload.

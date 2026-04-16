---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_workload"
description: |-
  Provides a data source to read a specific workload.
---

# azion_workload (Data Source)

Use this data source to read a specific workload by its ID.

## Example Usage

```terraform
data "azion_workload" "example" {
  id = "12345"
}
```

## Argument Reference

* `id` - (Required) The ID of the workload to retrieve.

## Attribute Reference

* `data` - The workload data.
  * `id` - The workload identifier.
  * `name` - Name of the workload.
  * `active` - Status of the workload.
  * `last_editor` - The last editor of the workload.
  * `last_modified` - Last modified timestamp of the workload.
  * `created_at` - Creation timestamp of the workload.
  * `infrastructure` - Infrastructure type: 1 for Production (All Locations), 2 for Staging.
  * `tls` - TLS configuration for the workload.
    * `certificate` - Certificate ID for TLS.
    * `ciphers` - Cipher suite configuration.
    * `minimum_version` - Minimum TLS version.
  * `protocols` - Protocol configurations for the workload.
    * `http` - HTTP protocol configuration.
      * `versions` - HTTP versions supported.
      * `http_ports` - HTTP ports.
      * `https_ports` - HTTPS ports.
      * `quic_ports` - QUIC ports.
  * `mtls` - Mutual TLS configuration for the workload.
    * `enabled` - Whether MTLS is enabled.
    * `config` - MTLS configuration.
      * `certificate` - MTLS certificate ID.
      * `crl` - Certificate Revocation List.
      * `verification` - MTLS verification type: enforce or permissive.
  * `domains` - List of domains associated with the workload.
  * `workload_domain_allow_access` - Whether domain access is allowed.
  * `workload_domain` - The workload domain.
  * `product_version` - Product version of the workload.

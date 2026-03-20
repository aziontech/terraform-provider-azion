---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_workloads"
description: |-
  Provides a data source to list all workloads.
---

# azion_workloads (Data Source)

Use this data source to list all workloads in your account.

## Example Usage

```terraform
data "azion_workloads" "example" {
}
```

## Argument Reference

This data source has no arguments.

## Attribute Reference

* `id` - Identifier of the data source.
* `counter` - The total count of workloads.
* `results` - List of workloads.
  * `id` - The workload identifier.
  * `name` - Name of the workload.
  * `active` - Status of the workload.
  * `last_editor` - The last editor of the workload.
  * `last_modified` - Last modified timestamp of the workload.
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

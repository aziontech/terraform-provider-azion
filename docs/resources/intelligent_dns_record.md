---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_intelligent_dns_record Resource"
description: |-
  Provides a resource for managing DNS records in an Intelligent DNS zone.
---

# azion_intelligent_dns_record (Resource)

Creates and manages DNS records in an Intelligent DNS zone.

## Example Usage

```terraform
resource "azion_intelligent_dns_record" "example" {
  zone_id = "12345"
  record = {
    type = "A"
    name = "site"
    rdata = [
      "8.8.8.8"
    ]
    policy      = "weighted"
    weight      = 50
    description = "This is a description"
    ttl         = 20
  }
}
```

## Import

```sh
terraform import azion_intelligent_dns_record.example "zone_id/record_id"
```

## Schema

### Required

* `zone_id` (String) The zone identifier to target for the resource.
* `record` (Attributes) The record configuration. (see below for nested schema)

### Read-Only

* `last_updated` (String) Timestamp of the last Terraform update.

<a id="nestedatt--record"></a>
### Nested Schema for `record`

Required:
* `name` (String) The name of the DNS record.
* `type` (String) DNS record type (A, AAAA, ANAME, CNAME, MX, NS, PTR, SRV, TXT, CAA, DS).
* `rdata` (List of String) List of answers replied by DNS Authoritative to that Record.
* `policy` (String) Must be 'simple' or 'weighted'.
* `ttl` (Number) Time-to-live defines max-time for packets life in seconds.

Optional:
* `weight` (Number) You can only use this field when policy is 'weighted'.
* `description` (String) Description of the record.

Read-Only:
* `id` (Number) The record identifier.

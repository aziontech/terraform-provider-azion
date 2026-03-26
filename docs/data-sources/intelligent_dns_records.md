---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_intelligent_dns_records Data Source"
description: |-
  Provides a data source for listing DNS records in an Intelligent DNS zone.
---

# azion_intelligent_dns_records (Data Source)

Use this data source to list DNS records from a specific Intelligent DNS zone.

## Example Usage

```terraform
data "azion_intelligent_dns_records" "example" {
  zone_id = 1234
}
```

## Argument Reference

* `zone_id` - (Required) The zone identifier to target for the resource.
* `page` - (Optional) The page number of Records. Defaults to 1.
* `page_size` - (Optional) The page size number of Records. Defaults to 10.

## Attribute Reference

* `id` - The ID of this resource.
* `counter` - The total number of records.
* `total_pages` - The total number of pages.
* `links` - Pagination links.
  * `previous` - URL to the previous page.
  * `next` - URL to the next page.
* `results` - List of records.
  * `record_id` - The record identifier.
  * `name` - The name of the DNS record.
  * `type` - DNS record type (A, AAAA, ANAME, CNAME, MX, NS, PTR, SRV, TXT, CAA, DS).
  * `rdata` - List of answers replied by DNS Authoritative to that Record.
  * `policy` - Must be 'simple' or 'weighted'.
  * `ttl` - Time-to-live defines max-time for packets life in seconds.
  * `weight` - Weight for weighted policy records.
  * `description` - Description of the record.

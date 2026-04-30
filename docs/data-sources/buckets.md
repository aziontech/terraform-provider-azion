---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_buckets"
description: |-
  Provides a data source to list all storage buckets.
---

# azion_buckets (Data Source)

Use this data source to list all storage buckets in your account.

## Example Usage

```terraform
data "azion_buckets" "example" {
}
```

## Argument Reference

This data source has no required arguments.

## Attribute Reference

* `id` - The identifier of the data source.
* `counter` - The total count of buckets.
* `total_pages` - The total number of pages.
* `page` - The current page number.
* `page_size` - The number of items per page.
* `results` - List of buckets.
  * `name` - Name of the bucket.
  * `workloads_access` - Access type for workloads: `read_only`, `read_write`, or `restricted`.
  * `last_editor` - The last editor of the bucket.
  * `last_modified` - Last modified timestamp of the bucket.
  * `product_version` - Product version of the bucket.

---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_bucket"
description: |-
  Provides a data source to read a specific storage bucket.
---

# azion_bucket (Data Source)

Use this data source to read a specific storage bucket by its name.

## Example Usage

```terraform
data "azion_bucket" "example" {
  name = "my-bucket-name"
}
```

## Argument Reference

* `name` - (Required) The name of the bucket to retrieve.

## Attribute Reference

* `id` - The identifier of the data source (same as bucket name).
* `data` - The bucket data.
  * `name` - Name of the bucket.
  * `workloads_access` - Access type for workloads: `read_only`, `read_write`, or `restricted`.
  * `last_editor` - The last editor of the bucket.
  * `last_modified` - Last modified timestamp of the bucket.
  * `product_version` - Product version of the bucket.

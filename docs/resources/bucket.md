---
subcategory: "Storage"
layout: "azion"
page_title: "Azion: azion_bucket"
description: |-
  Provides an Azion Storage Bucket resource.
---

# azion_bucket

Provides an Azion Storage Bucket resource. This allows you to create, update, and delete storage buckets.

## Example Usage

```hcl
resource "azion_bucket" "example" {
  bucket = {
    name              = "my-bucket-name"
    workloads_access  = "read_write"
  }
}
```

## Argument Reference

The `bucket` block contains:

* `name` - (Required) The name of the bucket. Must be unique. Changing this will recreate the bucket.
* `workloads_access` - (Required) Access type for workloads. Valid values are:
  - `read_only` - Workloads can only read from the bucket
  - `read_write` - Workloads can read and write to the bucket
  - `restricted` - Access is restricted

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the bucket (same as the bucket name).
* `last_updated` - Timestamp of the last Terraform update of the resource.

The `bucket` block also exports:

* `last_editor` - The last editor of the bucket.
* `last_modified` - The last modified timestamp of the bucket.
* `product_version` - The product version of the bucket.

## Import

Buckets can be imported using the `name` attribute:

```sh
terraform import azion_bucket.example my-bucket-name
```

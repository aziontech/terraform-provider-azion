---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_data_streams"
description: |-
  Provides a Data Streams data source for listing every stream in the account.
---

# azion_data_streams (Data Source)

Use this data source to list every Data Stream in the account.

Each result carries the same fields as [`azion_data_stream`](data_stream.md), with the polymorphic `inputs`, `transform` and `outputs` lists exposed as JSON strings.

## Example Usage

### List all data streams

```terraform
data "azion_data_streams" "example" {
}

output "azion_data_streams" {
  value = data.azion_data_streams.example
}
```

### Extract the names of every stream

```terraform
output "azion_data_streams_names" {
  value = [for stream in data.azion_data_streams.example.results : stream.name]
}
```

## Argument Reference

This data source takes no arguments.

## Attribute Reference

* `id` - Identifier of the data source.
* `counter` - The total count of data streams.
* `results` - List of data streams.
  * `id` - The data stream identifier.
  * `name` - Name of the data stream.
  * `active` - Status of the data stream.
  * `last_editor` - The last editor of the data stream.
  * `created_at` - The creation timestamp of the data stream.
  * `last_modified` - Last modified timestamp of the data stream.
  * `product_version` - Product version of the data stream.
  * `inputs` - Inputs of the stream as a JSON string.
  * `transform` - Transforms of the stream as a JSON string.
  * `outputs` - Outputs of the stream as a JSON string.

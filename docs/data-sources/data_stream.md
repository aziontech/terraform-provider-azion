---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_data_stream"
description: |-
  Provides a Data Stream data source for reading a single stream by ID.
---

# azion_data_stream (Data Source)

Use this data source to read a specific Data Stream by its identifier.

The polymorphic `inputs`, `transform` and `outputs` lists are exposed as JSON strings so every endpoint and transform type is readable without a per-type schema. Decode them with `jsondecode` when a specific field is needed.

## Example Usage

### Read a data stream by ID

```terraform
data "azion_data_stream" "example" {
  id = "1234"
}

output "azion_data_stream" {
  value = data.azion_data_stream.example
}
```

### Read a nested field out of the JSON attributes

```terraform
output "azion_data_stream_first_output_type" {
  value = jsondecode(data.azion_data_stream.example.data.outputs)[0].type
}
```

### Read a data stream created by a resource

```terraform
resource "azion_data_stream" "example" {
  data_stream = {
    name   = "Applications to HTTP endpoint"
    active = true
    inputs = [
      {
        type = "raw_logs"
        attributes = {
          data_source = "http"
        }
      }
    ]
    outputs = [
      {
        type = "standard"
        standard_attributes = {
          url     = "https://logs.example.com/ingest"
          headers = {}
        }
      }
    ]
  }
}

data "azion_data_stream" "from_resource" {
  id = azion_data_stream.example.id
}
```

## Argument Reference

* `id` - (Required) The identifier of the data stream, as a string.

## Attribute Reference

* `data` - The data stream.
  * `id` - The data stream identifier.
  * `name` - Name of the data stream.
  * `active` - Status of the data stream.
  * `last_editor` - The last editor of the data stream.
  * `created_at` - The creation timestamp of the data stream.
  * `last_modified` - Last modified timestamp of the data stream.
  * `product_version` - Product version of the data stream.
  * `inputs` - Inputs of the stream as a JSON string. Each entry has a `type` and `attributes.data_source`.
  * `transform` - Transforms of the stream as a JSON string. Structure varies by `type` (`sampling`, `filter_workloads`, `render_template`).
  * `outputs` - Outputs of the stream as a JSON string. Structure varies by `type` (`standard`, `kafka`, `s3`, `big_query`, `elasticsearch`, `splunk`, `aws_kinesis_firehose`, `datadog`, `qradar`, `azure_monitor`, `azure_blob_storage`).

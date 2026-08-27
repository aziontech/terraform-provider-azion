---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_data_stream_template"
description: |-
  Provides a Data Stream template resource for defining the payload shape of a stream.
---

# azion_data_stream_template

Creates a Data Stream template. A template defines the payload shape (`data_set`) that the `render_template` transform of an [`azion_data_stream`](data_stream.md) applies to each record.

Azion ships built-in templates that every account can reference. Those are read-only — use the [`azion_data_stream_templates`](../data-sources/data_stream_templates.md) data source to discover their IDs. This resource manages **custom** templates, so `custom` is always `true` for anything Terraform creates here.

## Example Usage

### A JSON payload template

```terraform
resource "azion_data_stream_template" "example" {
  template = {
    name   = "My HTTP Events Template"
    active = true
    data_set = jsonencode({
      time        = "$time_iso8601"
      host        = "$host"
      remote_addr = "$remote_addr"
      request_uri = "$request_uri"
      status      = "$status"
      bytes_sent  = "$bytes_sent"
    })
  }
}
```

### Wiring a template into a stream

The `render_template` transform takes the template's numeric ID, so it can be referenced directly instead of hardcoded.

```terraform
resource "azion_data_stream_template" "example" {
  template = {
    name     = "My HTTP Events Template"
    active   = true
    data_set = jsonencode({ time = "$time_iso8601", status = "$status" })
  }
}

resource "azion_data_stream" "with_template" {
  data_stream = {
    name   = "Rendered logs to HTTP endpoint"
    active = true

    inputs = [
      {
        type = "raw_logs"
        attributes = {
          data_source = "http"
        }
      }
    ]

    transform = [
      # A no-op sampling entry: `render_template` cannot be the only transform,
      # the API requires either sampling or filter_workloads alongside it.
      {
        type = "sampling"
        sampling_attributes = {
          rate = 100
        }
      },
      {
        type = "render_template"
        render_template_attributes = {
          template = azion_data_stream_template.example.template.id
        }
      }
    ]

    outputs = [
      {
        type = "standard"
        standard_attributes = {
          url = "https://logs.example.com/ingest"
          headers = {
            "Authorization" = "Bearer my-token"
          }
        }
      }
    ]
  }
}
```

### A plain-text payload template

`data_set` is an opaque string, so it does not have to hold JSON.

```terraform
resource "azion_data_stream_template" "plain_text" {
  template = {
    name     = "My Plain Text Template"
    active   = true
    data_set = "$time_iso8601 $remote_addr $request_method $request_uri $status"
  }
}
```

## Import

```sh
terraform import azion_data_stream_template.example 1234
```

## Argument Reference

* `template` - (Required) The template configuration.
  * `name` - (Required) Name of the template.
  * `active` - (Optional) Status of the template. Computed when omitted.
  * `data_set` - (Required) The payload template. A string holding the record layout with `$variable` placeholders. The API stores the content verbatim but strips leading and trailing whitespace — see the notes below.

## Attribute Reference

* `id` - The template identifier, as a string.
* `last_updated` - Timestamp of the last Terraform update of the resource.
* `schema_version` - Schema version of the resource.
* `template`
  * `id` - The template identifier.
  * `custom` - Whether the template is user-defined. Always `true` for templates managed by Terraform; Azion's built-in templates are `false`.
  * `last_editor` - The last editor of the template.
  * `created_at` - The creation timestamp of the template.
  * `last_modified` - Last modified timestamp of the template.

## Notes

* Updates use `PATCH` (`PartialUpdateTemplate`) with all three mutable fields sent on every apply. This differs from [`azion_data_stream`](data_stream.md), which must use `PUT` because its partial-update payload cannot carry `outputs`.
* **The API strips leading and trailing whitespace from `data_set`.** A heredoc (`<<-EOT`) always ends in a newline, so the stored value comes back one byte shorter than it was sent. The provider keeps the configured value in state whenever the API's echo differs only by surrounding whitespace; without that, every heredoc `data_set` would fail apply with `Provider produced inconsistent result after apply`. A genuine server-side rewrite still differs after trimming and surfaces as real drift. Use `jsonencode` to keep the local value canonical.
* Deleting a template that a stream's `render_template` transform still references is rejected by the API with `400` code `32005` (`Cannot delete a template with associated data streams`). Terraform orders this correctly on its own — a stream that references `azion_data_stream_template.x.template.id` is destroyed before the template — so no `depends_on` is needed. The provider also retries this specific error a few times to absorb any lag in the API releasing the association.

  If the error persists, the cause is usually a stream that exists in the API but not in Terraform state, which nothing in the configuration will clean up. List the streams and check the `template` values under `transform` to find it:

  ```sh
  curl -s -H "Authorization: token $AZION_TOKEN" \
    "https://api.azion.com/v4/workspace/stream/streams?page_size=100"
  ```

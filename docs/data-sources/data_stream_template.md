---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_data_stream_template"
description: |-
  Provides a Data Stream template data source for reading a single template by ID.
---

# azion_data_stream_template (Data Source)

Use this data source to read a specific Data Stream template by its identifier. It reads both Azion's built-in templates and the custom templates in the account.

## Example Usage

### Read a template by ID

```terraform
data "azion_data_stream_template" "example" {
  id = "1234"
}

output "azion_data_stream_template" {
  value = data.azion_data_stream_template.example
}
```

### Read the payload template only

```terraform
output "azion_data_stream_template_data_set" {
  value = data.azion_data_stream_template.example.data.data_set
}
```

### Read a template created by a resource

```terraform
resource "azion_data_stream_template" "example" {
  template = {
    name     = "My HTTP Events Template"
    active   = true
    data_set = jsonencode({ time = "$time_iso8601", status = "$status" })
  }
}

data "azion_data_stream_template" "from_resource" {
  id = azion_data_stream_template.example.id
}
```

## Argument Reference

* `id` - (Required) The identifier of the template, as a string.

## Attribute Reference

* `data` - The template.
  * `id` - The template identifier.
  * `name` - Name of the template.
  * `active` - Status of the template.
  * `data_set` - The payload template, holding the record layout with `$variable` placeholders.
  * `custom` - Whether the template is user-defined. Azion's built-in templates are `false`.
  * `last_editor` - The last editor of the template.
  * `created_at` - The creation timestamp of the template.
  * `last_modified` - Last modified timestamp of the template.

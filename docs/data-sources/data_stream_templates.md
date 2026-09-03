---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_data_stream_templates"
description: |-
  Provides a Data Stream templates data source for listing every template available to the account.
---

# azion_data_stream_templates (Data Source)

Use this data source to list every Data Stream template available to the account — both Azion's built-in templates and the custom ones created in the account.

This is the practical way to discover the numeric template ID required by a stream's `render_template` transform, since built-in templates have no corresponding resource.

Each result carries the same fields as [`azion_data_stream_template`](data_stream_template.md).

## Example Usage

### List all templates

```terraform
data "azion_data_stream_templates" "example" {
}

output "azion_data_stream_templates" {
  value = data.azion_data_stream_templates.example
}
```

### Look up built-in templates by name

`custom = false` marks Azion's built-in templates.

```terraform
output "azion_data_stream_builtin_templates" {
  value = {
    for template in data.azion_data_stream_templates.example.results :
    template.name => template.id
    if template.custom == false
  }
}
```

## Argument Reference

This data source takes no arguments.

## Attribute Reference

* `id` - Identifier of the data source.
* `counter` - The total count of templates.
* `results` - List of templates.
  * `id` - The template identifier.
  * `name` - Name of the template.
  * `active` - Status of the template.
  * `data_set` - The payload template.
  * `custom` - Whether the template is user-defined.
  * `last_editor` - The last editor of the template.
  * `created_at` - The creation timestamp of the template.
  * `last_modified` - Last modified timestamp of the template.

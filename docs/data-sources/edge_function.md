---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azion_edge_function Data Source - terraform-provider-azion"
subcategory: ""
description: |-
  
---

# azion_edge_function (Data Source)



## Example Usage

```terraform
data "azion_edge_function" "example" {
  id = "1234567890"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) Numeric identifier of the data source.

### Read-Only

- `results` (Attributes) (see [below for nested schema](#nestedatt--results))
- `schema_version` (Number) Schema Version.

<a id="nestedatt--results"></a>
### Nested Schema for `results`

Read-Only:

- `active` (Boolean) Status of the function.
- `code` (String) Code of the function.
- `function_id` (Number) The function identifier.
- `function_to_run` (String) The function to run.
- `initiator_type` (String) Initiator type of the function.
- `json_args` (String) JSON arguments of the function.
- `language` (String) Language of the function.
- `last_editor` (String) The last editor of the function.
- `modified` (String) Last modified timestamp of the function.
- `name` (String) Name of the function.
- `reference_count` (Number) The reference count of the function.
- `version` (String) Version of the function.

---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azion_edge_application_edge_functions_instance Resource - terraform-provider-azion"
subcategory: ""
description: |-
  
---

# azion_edge_application_edge_functions_instance (Resource)



## Example Usage

```terraform
resource "azion_edge_application_edge_functions_instance" "example" {
    edge_application_id = <edge_application_id>
    results = {
    name = "Terraform Example"
    "edge_function_id": <edge_function_id>,
    "args": jsonencode(
            { "key" = "Value",
              "Example" = "example"
            })
    }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `edge_application_id` (Number) The edge application identifier.
- `results` (Attributes) (see [below for nested schema](#nestedatt--results))

### Read-Only

- `id` (String) The ID of this resource.
- `last_updated` (String) Timestamp of the last Terraform update of the resource.
- `schema_version` (Number)

<a id="nestedatt--results"></a>
### Nested Schema for `results`

Required:

- `args` (String) JSON arguments of the function.
- `edge_function_id` (Number) The edge function identifier.
- `name` (String) Name of the function.

Read-Only:

- `id` (Number) The edge function instance identifier.

## Import

Import is supported using the following syntax:

```shell
terraform import azion_edge_application_edge_functions_instance.example <edge_application_id>/<edge_function_instance_id>
```
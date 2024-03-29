---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azion_edge_application_rules_engine Data Source - terraform-provider-azion"
subcategory: ""
description: |-
  
---

# azion_edge_application_rules_engine (Data Source)



## Example Usage

```terraform
data "azion_edge_application_rules_engine" "example" {
  edge_application_id = 1234567890
  results = [{
    phase = "request"
  }]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `edge_application_id` (Number) The edge application identifier.
- `results` (Attributes List) (see [below for nested schema](#nestedatt--results))

### Optional

- `page` (Number) The page number of edge applications.
- `page_size` (Number) The Page Size number of edge applications.

### Read-Only

- `counter` (Number) The total number of edge applications.
- `id` (String) Identifier of the data source.
- `links` (Attributes) (see [below for nested schema](#nestedatt--links))
- `schema_version` (Number) Schema Version.
- `total_pages` (Number) The total number of pages.

<a id="nestedatt--results"></a>
### Nested Schema for `results`

Required:

- `phase` (String) The phase in which the rule is executed (e.g., default, request, response).

Read-Only:

- `behaviors` (Attributes List) (see [below for nested schema](#nestedatt--results--behaviors))
- `criteria` (Attributes List) (see [below for nested schema](#nestedatt--results--criteria))
- `description` (String) The description of the rules engine rule.
- `id` (Number) The ID of the rules engine rule.
- `is_active` (Boolean) The status of the rules engine rule.
- `name` (String) The name of the rules engine rule.
- `order` (Number) The order of the rule in the rules engine.

<a id="nestedatt--results--behaviors"></a>
### Nested Schema for `results.behaviors`

Required:

- `target_object` (Attributes) (see [below for nested schema](#nestedatt--results--behaviors--target_object))

Read-Only:

- `name` (String) The name of the behavior.

<a id="nestedatt--results--behaviors--target_object"></a>
### Nested Schema for `results.behaviors.target_object`

Read-Only:

- `captured_array` (String) The name of the behavior.
- `regex` (String) The target of the behavior.
- `subject` (String) The target of the behavior.
- `target` (String) The target of the behavior.



<a id="nestedatt--results--criteria"></a>
### Nested Schema for `results.criteria`

Read-Only:

- `entries` (Attributes List) (see [below for nested schema](#nestedatt--results--criteria--entries))

<a id="nestedatt--results--criteria--entries"></a>
### Nested Schema for `results.criteria.entries`

Read-Only:

- `conditional` (String) The conditional operator used in the rule's criteria (e.g., if, and, or).
- `input_value` (String) The input value used in the rule's criteria.
- `operator` (String) The operator used in the rule's criteria.
- `variable` (String) The variable used in the rule's criteria.




<a id="nestedatt--links"></a>
### Nested Schema for `links`

Read-Only:

- `next` (String)
- `previous` (String)

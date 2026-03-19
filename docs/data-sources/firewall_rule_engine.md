---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_firewall_rule_engine"
description: |-
  Provides a data source for a specific firewall rules engine rule.
---

# azion_firewall_rule_engine

Use this data source to read a specific rules engine rule from a firewall.

## Example Usage

```hcl
data "azion_firewall_rule_engine" "example" {
  firewall_id = 12345
  results = {
    id = 67890
  }
}
```

## Argument Reference

* `firewall_id` - (Required) The firewall identifier.
* `results` - (Required) The results block containing the rule details.
  * `id` - (Required) The ID of the rules engine rule to retrieve.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Identifier of the data source.
* `results` - The results block containing the rule details.
  * `name` - Name of the rules engine rule.
  * `active` - Whether the rule is active.
  * `criteria` - List of criteria groups for the rule.
    * `entries` - List of criteria entries within the group.
      * `conditional` - Conditional operator (if, and, or).
      * `variable` - Variable to evaluate.
      * `operator` - Comparison operator.
      * `argument` - Argument for comparison.
  * `behaviors` - List of behaviors for the rule.
    * `type` - Type of behavior.
    * `attributes` - Behavior attributes.
      * `value` - Value for run_function behavior (function instance ID).
      * `status_code` - Status code for set_custom_response behavior.
      * `content_type` - Content type for set_custom_response behavior.
      * `content_body` - Content body for set_custom_response behavior.
      * `waf_id` - WAF ID for set_waf behavior.
      * `mode` - Mode for set_waf or set_rate_limit behavior.
      * `type` - Type for set_rate_limit behavior.
      * `limit_by` - Limit by for set_rate_limit behavior.
      * `average_rate_limit` - Average rate limit for set_rate_limit behavior.
      * `maximum_burst_size` - Maximum burst size for set_rate_limit behavior.
  * `description` - Description of the rule.
  * `order` - Order of the rule.
  * `last_editor` - Last editor of the rule.
  * `last_modified` - Last modified timestamp.

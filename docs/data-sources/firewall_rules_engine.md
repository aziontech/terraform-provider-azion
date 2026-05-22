---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_firewall_rules_engine"
description: |-
  Provides a data source to list all firewall rules engine rules.
---

# azion_firewall_rules_engine

Use this data source to list all rules engine rules from a specific firewall.

## Example Usage

```hcl
data "azion_firewall_rules_engine" "example" {
  firewall_id = 12345
}
```

## Argument Reference

* `firewall_id` - (Required) The firewall identifier.
* `page` - (Optional) The page number for pagination. Defaults to 1.
* `page_size` - (Optional) The number of items per page. Defaults to 10.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Identifier of the data source.
* `counter` - The total number of rules.
* `total_pages` - The total number of pages.
* `links` - Pagination links.
  * `previous` - Link to the previous page.
  * `next` - Link to the next page.
* `results` - List of rules engine rules.
  * `id` - The ID of the rules engine rule.
  * `name` - Name of the rules engine rule.
  * `active` - Whether the rule is active.
  * `criteria` - List of criteria groups for the rule.
    * `entries` - List of criteria entries within the group. Each item contains a single `criterion` object.
      * `criterion` - A single criterion entry.
        * `conditional` - Conditional operator (if, and, or).
        * `variable` - Variable to evaluate.
        * `operator` - Comparison operator.
        * `argument` - Argument for comparison.
  * `behaviors` - List of behaviors for the rule. Each item contains a single `behavior` object.
    * `behavior` - A single behavior for the rule.
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
  * `created_at` - Creation timestamp.

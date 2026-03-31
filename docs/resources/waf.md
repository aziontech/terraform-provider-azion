---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_waf"
description: |-
  Provides a WAF (Web Application Firewall) resource.
---

# azion_waf

Creates a WAF (Web Application Firewall) resource. This resource represents the main WAF configuration that can have associated WAF Rule Sets (Exceptions).

## Example Usage

```hcl
resource "azion_waf" "example" {
  result = {
    name   = "My WAF"
    active = true
    
    engine_settings = {
      engine_version = "2021-Q3"
      type           = "score"
      
      attributes = {
        rulesets = [1, 2, 3]
        
        thresholds = [
          {
            threat      = "sql_injection"
            sensitivity = "high"
          },
          {
            threat      = "cross_site_scripting"
            sensitivity = "highest"
          }
        ]
      }
    }
  }
}
```

## Import

```sh
terraform import azion_waf.example 12345
```

## Argument Reference

* `result` - (Required) The WAF configuration.
  * `name` - (Required) Name of the WAF.
  * `active` - (Optional) Whether the WAF is active.
  * `product_version` - (Optional) Product version of the WAF.
  * `engine_settings` - (Optional) Engine settings for the WAF.
    * `engine_version` - (Optional) Engine version for the WAF (e.g., `2021-Q3`).
    * `type` - (Optional) Type of the WAF engine (e.g., `score`).
    * `attributes` - (Optional) Attributes for the WAF engine settings.
      * `rulesets` - (Optional) List of ruleset IDs.
      * `thresholds` - (Optional) Threshold configurations for the WAF.
        * `threat` - (Required) The threat type for the threshold. Valid values: `cross_site_scripting`, `directory_traversal`, `evading_tricks`, `file_upload`, `identified_attack`, `remote_file_inclusion`, `sql_injection`, `unwanted_access`.
        * `sensitivity` - (Optional) The sensitivity level for the threshold. Valid values: `highest`, `high`, `medium`, `low`, `lowest`.

## Attribute Reference

* `id` - The ID of the WAF.
* `last_editor` - Last editor of the WAF.
* `last_modified` - Last modified timestamp.
* `last_updated` - Timestamp of the last Terraform update of the resource.

## Related Resources

* [azion_waf_rule_set](./waf_rule_set.md) - Manage WAF Rule Sets (Exceptions) that belong to this WAF.

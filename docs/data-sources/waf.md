---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_waf"
description: |-
  Provides a data source to read a specific WAF.
---

# azion_waf

Use this data source to read a specific WAF (Web Application Firewall).

## Example Usage

```hcl
data "azion_waf" "example" {
  waf_id = 12345
}
```

## Argument Reference

* `waf_id` - (Required) The ID of the WAF.

## Attribute Reference

* `id` - Identifier of the data source.
* `results` - The WAF data.
  * `id` - The ID of the WAF.
  * `name` - Name of the WAF.
  * `active` - Whether the WAF is active.
  * `last_editor` - Last editor of the WAF.
  * `last_modified` - Last modified timestamp.
  * `product_version` - Product version of the WAF.
  * `engine_settings` - Engine settings for the WAF.
    * `engine_version` - Engine version for the WAF.
    * `type` - Type of the WAF engine.
    * `attributes` - Attributes for the WAF engine settings.
      * `rulesets` - List of ruleset IDs.
      * `thresholds` - Threshold configurations for the WAF. Each item contains a single `threshold` object.
        * `threshold` - A single threshold configuration.
          * `threat` - The threat type for the threshold.
          * `sensitivity` - The sensitivity level for the threshold.

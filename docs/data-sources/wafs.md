---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_wafs"
description: |-
  Provides a data source to list WAFs.
---

# azion_wafs

Use this data source to list all WAFs (Web Application Firewalls).

## Example Usage

```hcl
data "azion_wafs" "example" {
  page = 1
  page_size = 10
}
```

## Argument Reference

* `page` - (Optional) The page number.
* `page_size` - (Optional) The page size number.

## Attribute Reference

* `id` - Identifier of the data source.
* `counter` - The total number of WAFs.
* `total_pages` - The total number of pages.
* `links` - Pagination links.
  * `previous` - URL to the previous page.
  * `next` - URL to the next page.
* `results` - List of WAFs.
  * `id` - The ID of the WAF.
  * `name` - Name of the WAF.
  * `active` - Whether the WAF is active.
  * `last_editor` - Last editor of the WAF.
  * `last_modified` - Last modified timestamp.
  * `product_version` - Product version of the WAF.
  * `is_versioned` - Whether the WAF is versioned.
  * `version` - The current version of the WAF.
  * `version_state` - The state of the current WAF version.
  * `version_id` - The identifier of the current WAF version.
  * `engine_settings` - Engine settings for the WAF.
    * `engine_version` - Engine version for the WAF.
    * `type` - Type of the WAF engine.
    * `attributes` - Attributes for the WAF engine settings.
      * `rulesets` - List of ruleset IDs.
      * `thresholds` - Threshold configurations for the WAF. Each item contains a single `threshold` object.
        * `threshold` - A single threshold configuration.
          * `threat` - The threat type for the threshold.
          * `sensitivity` - The sensitivity level for the threshold.

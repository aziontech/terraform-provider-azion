---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_waf"
description: |-
  Provides a WAF (Web Application Firewall) resource.
---

# azion_waf

## Drift and enforcement

Fields this resource enforces are reset to their default on every apply, so a
change made outside Terraform — in Azion Console, for example — is reverted and
shows up in `terraform plan` as a diff.

| Field | Enforced default |
|---|---|
| `active` | `true` |
| `engine_settings.engine_version` | `2021-Q3` (the only value the API accepts) |
| `engine_settings.type` | `score` (the only value the API accepts) |

`engine_settings.thresholds[].threshold.sensitivity` is refreshed from the API but
**not** defaulted: the API documents five levels (`highest`, `high`, `medium`,
`low`, `lowest`) with no stated default. A change to a declared threshold's
sensitivity still produces a diff, because state reflects what the API reports.

Four fields stay **unmanaged unless you declare them** — `engine_settings`,
`engine_settings.attributes`, `attributes.rulesets` and `attributes.thresholds`:

- Ruleset IDs are account-specific and there is no default that could be applied
  without wiping a practitioner's tuning.
- If you omit one of these, whatever the API holds is left alone rather than being
  reset, and it is kept out of state so the resource does not diff on every plan.

To bring them under Terraform's control, declare them. Once declared, they are
refreshed from the API on every read, so an out-of-band change to a declared
ruleset list or threshold set does appear as a diff.

Note that drift is only reverted when an apply runs, and only when state is
refreshed — `terraform plan -refresh=false` will not detect it. Terraform also
cannot remove WAFs created outside Terraform, since those are not in state.

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
            threshold = {
              threat      = "sql_injection"
              sensitivity = "high"
            }
          },
          {
            threshold = {
              threat      = "cross_site_scripting"
              sensitivity = "highest"
            }
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
  * `product_version` - (Computed) Product version of the WAF.
  * `engine_settings` - (Optional) Engine settings for the WAF.
    * `engine_version` - (Optional) Engine version for the WAF (e.g., `2021-Q3`).
    * `type` - (Optional) Type of the WAF engine (e.g., `score`).
    * `attributes` - (Optional) Attributes for the WAF engine settings.
      * `rulesets` - (Optional) List of ruleset IDs.
      * `thresholds` - (Optional) Threshold configurations for the WAF. Each item must contain a single `threshold` object.
        * `threshold` - (Required) A single threshold configuration.
          * `threat` - (Required) The threat type for the threshold. Valid values: `cross_site_scripting`, `directory_traversal`, `evading_tricks`, `file_upload`, `identified_attack`, `remote_file_inclusion`, `sql_injection`, `unwanted_access`.
          * `sensitivity` - (Optional) The sensitivity level for the threshold. Valid values: `highest`, `high`, `medium`, `low`, `lowest`.

## Attribute Reference

* `id` - The ID of the WAF.
* `last_editor` - Last editor of the WAF.
* `last_modified` - Last modified timestamp.
* `last_updated` - Timestamp of the last Terraform update of the resource.
* `result`
  * `state` - The state of the current WAF version.
  * `version_id` - The identifier of the current WAF version.

## Related Resources

* [azion_waf_rule_set](./waf_rule_set.md) - Manage WAF Rule Sets (Exceptions) that belong to this WAF.

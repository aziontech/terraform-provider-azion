---
page_title: "azion_firewall_rule_engine_order Resource - terraform-provider-azion"
subcategory: ""
description: |-
  Manages the evaluation order of a firewall's rule engine rules.
---

# azion_firewall_rule_engine_order (Resource)

Manages the evaluation order of the rule engine rules of an Azion Firewall.

The Azion API exposes this as a dedicated PUT endpoint (`/edge_firewall/firewalls/{firewall_id}/rules/order`) whose body is the full list of rule IDs in the desired order. Because ordering is a **collective** operation (one PUT replaces the whole order), it is modeled as its own resource rather than as an attribute on each rule.

Unlike the application rules engine, firewall rules have a single phase, so no `phase` attribute is needed.

This resource requires the parent rules to already exist; reference their IDs through Terraform interpolation and the implicit dependency graph will ensure the order resource runs after the rules.

> **Note on destroy:** the API has no inverse operation for ordering. When this resource is destroyed, it is removed from Terraform state but the rules remain in their last applied order on the Azion side. A warning is emitted to make this explicit.

## Example Usage

### Full setup with parent firewall and rules

```terraform
resource "azion_firewall_main_setting" "example" {
  data = {
    name   = "My Firewall"
    active = true
  }
}

resource "azion_firewall_rule_engine" "first" {
  firewall_id = azion_firewall_main_setting.example.data.id
  results = {
    name = "Block /admin"
    behaviors = [
      {
        behavior = {
          type = "drop"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            criterion = {
              variable    = "$${request_uri}"
              operator    = "matches"
              conditional = "if"
              argument    = "/admin.*"
            }
          }
        ]
      }
    ]
  }
}

resource "azion_firewall_rule_engine" "second" {
  firewall_id = azion_firewall_main_setting.example.data.id
  results = {
    name = "Block /internal"
    behaviors = [
      {
        behavior = {
          type = "drop"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            criterion = {
              variable    = "$${request_uri}"
              operator    = "matches"
              conditional = "if"
              argument    = "/internal.*"
            }
          }
        ]
      }
    ]
  }
}

# Evaluate "second" before "first".
resource "azion_firewall_rule_engine_order" "example" {
  firewall_id = azion_firewall_main_setting.example.data.id
  order = [
    azion_firewall_rule_engine.second.results.id,
    azion_firewall_rule_engine.first.results.id,
  ]
}
```

## Import

Existing firewall rule ordering can be imported using the firewall ID:

```sh
terraform import azion_firewall_rule_engine_order.example 1234567890
```

## Argument Reference

* `firewall_id` - (Required) The firewall identifier whose rule order is being managed.
* `order` - (Required) Ordered list of rule IDs. Every firewall rule that you want to control must appear in this list; the first ID is evaluated first.

## Attribute Reference

* `id` - The resource identifier (same as `firewall_id`).
* `last_updated` - Timestamp of the last Terraform update of the resource.

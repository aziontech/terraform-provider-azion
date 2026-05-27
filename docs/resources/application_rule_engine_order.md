---
page_title: "azion_application_rule_engine_order Resource - terraform-provider-azion"
subcategory: ""
description: |-
  Manages the evaluation order of an application's rule engine rules for a given phase.
---

# azion_application_rule_engine_order (Resource)

Manages the evaluation order of the rule engine rules of an Azion application for a single phase (`request` or `response`).

The Azion API exposes this as a dedicated PUT endpoint (`/workspace/applications/{application_id}/request_rules/order` and the equivalent `response_rules/order`) whose body is the full list of rule IDs in the desired order. Because ordering is a **collective** operation (one PUT replaces the whole order), it is modeled as its own resource rather than as an attribute on each rule.

This resource requires the parent rules to already exist; reference their IDs through Terraform interpolation and `depends_on` is taken care of for you implicitly.

> **Note on destroy:** the API has no inverse operation for ordering. When this resource is destroyed, it is removed from Terraform state but the rules remain in their last applied order on the Azion side. A warning is emitted to make this explicit.

## Example Usage

### Full setup with parent application and rules

```terraform
resource "azion_application_main_setting" "example" {
  application = {
    name   = "My Application"
    active = true
  }
}

resource "azion_application_rule_engine" "first" {
  application_id = azion_application_main_setting.example.application.application_id
  results = {
    name  = "First rule"
    phase = "request"
    behaviors = [
      {
        behavior = {
          type = "deliver"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            criterion = {
              variable    = "$${uri}"
              operator    = "starts_with"
              conditional = "if"
              argument    = "/a/"
            }
          }
        ]
      }
    ]
  }
}

resource "azion_application_rule_engine" "second" {
  application_id = azion_application_main_setting.example.application.application_id
  results = {
    name  = "Second rule"
    phase = "request"
    behaviors = [
      {
        behavior = {
          type = "deliver"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            criterion = {
              variable    = "$${uri}"
              operator    = "starts_with"
              conditional = "if"
              argument    = "/b/"
            }
          }
        ]
      }
    ]
  }
}

# Reorder the request phase rules: evaluate "second" before "first".
resource "azion_application_rule_engine_order" "request_order" {
  application_id = azion_application_main_setting.example.application.application_id
  phase          = "request"
  order = [
    azion_application_rule_engine.second.results.id,
    azion_application_rule_engine.first.results.id,
  ]
}
```

## Import

Existing rule ordering can be imported using the form `{application_id}/{phase}`:

```sh
terraform import azion_application_rule_engine_order.request_order 1234567890/request
```

## Argument Reference

* `application_id` - (Required) The application identifier whose rule order is being managed.
* `phase` - (Required) The phase of the rules to order. Must be `request` or `response`.
* `order` - (Required) Ordered list of rule IDs. Every rule of the chosen phase that you want to control must appear in this list; the first ID is evaluated first.

## Attribute Reference

* `id` - The resource identifier in the form `{application_id}/{phase}`.
* `last_updated` - Timestamp of the last Terraform update of the resource.

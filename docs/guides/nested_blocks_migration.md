---
subcategory: ""
layout: "azion"
page_title: "Guide: Migrating to Block Syntax (v2.12.0)"
description: |-
  Starting with v2.12.0, nested objects in Azion resources are written as blocks instead of attributes, so Terraform rejects unknown arguments.
---

# Migrating to Block Syntax (v2.12.0)

Starting with **v2.12.0**, every nested object you write in an Azion resource is a
**block** instead of an **attribute**. Configurations written for earlier versions
must be updated before running `terraform plan`.

## Why this changed

Earlier versions declared nested objects as attributes (`application = { ... }`).
Terraform converts the value of such an attribute to the expected object type, and
that conversion **silently drops any argument the provider does not define**. A
misspelled argument, or an argument written one level too deep or too shallow, was
ignored without warning, and the resource was created with a configuration other
than the one you wrote.

Blocks are validated by Terraform itself. The same mistake now fails at
`terraform validate`:

```
Error: Unsupported argument

  on main.tf line 16, in resource "azion_application_main_setting" "example":
  16:       enabled = true

An argument named "enabled" is not expected here.
```

## What to change

Argument names and nesting are unchanged. Only the syntax changes.

### 1. Single objects: remove the `=`

```terraform
# Before
resource "azion_application_main_setting" "example" {
  application = {
    name = "My Application"
    modules = {
      cache = {
        enabled = true
      }
    }
  }
}

# After
resource "azion_application_main_setting" "example" {
  application {
    name = "My Application"
    modules {
      cache {
        enabled = true
      }
    }
  }
}
```

### 2. Lists of objects: one block per element

```terraform
# Before
behaviors = [
  {
    behavior = {
      type = "deliver"
    }
  },
  {
    behavior = {
      type = "set_cache_policy"
      attributes = {
        value = 123
      }
    }
  }
]

# After
behaviors {
  behavior {
    type = "deliver"
  }
}
behaviors {
  behavior {
    type = "set_cache_policy"
    attributes {
      value = 123
    }
  }
}
```

Block order is preserved for lists, so rules, behaviors and criteria keep the
order you write them in.

### 3. Objects built from expressions: use `dynamic` blocks

A block cannot be assigned from a variable, a `for` expression or a `merge(...)`.
Use a `dynamic` block instead:

```terraform
# Before
behaviors = [for b in var.behaviors : { behavior = b }]

# After
dynamic "behaviors" {
  for_each = var.behaviors
  content {
    behavior {
      type = behaviors.value.type
    }
  }
}
```

### 4. `azion_waf_rule_sets`: map entries become `rule_set` blocks

The map of exceptions keyed by name is now a repeated `rule_set` block with an
explicit `name`. Existing state is upgraded automatically.

```terraform
# Before
rule_sets = {
  "Allow health check" = {
    rule_id = 0
    path    = "/health"
    conditions = [{ condition = { match = "any_url" } }]
  }
}

# After
rule_set {
  name    = "Allow health check"
  rule_id = 0
  path    = "/health"
  conditions {
    condition {
      match = "any_url"
    }
  }
}
```

Declaring no `rule_set` block deletes every exception of the WAF, as an empty map
did before.

## Reading values

References to nested values do not change. For single blocks, keep using the same
path, for example
`azion_application_main_setting.example.application.application_id`.
For list blocks, keep using an index, for example
`azion_application_rule_engine.example.results.behaviors[0].behavior.type`.

The one exception is bracket syntax on a single object. Terraform accepts
`results["status"]` on an attribute but not on a block, so write
`results.status` instead. The error is `Invalid index operation: Only attribute
access is allowed here`.

In `azion_waf_rule_sets`, `rule_set` is a set, so it has no key or index. Look an
exception up by its name:

```terraform
output "health_check_id" {
  value = one([for e in azion_waf_rule_sets.this.rule_set : e.exception_id if e.name == "Allow health check"])
}
```

## Your existing state

You do not need to re-import or recreate anything. State written by earlier
versions is read by v2.12.0 as is.

After you convert the configuration, the first `terraform plan` shows **No
changes** for every object you declare. One case can show an in-place update once:

- **Optional objects that you never declared** but that the provider used to store
  in state with their defaults. The plan removes them from state. The apply sends
  the same values the provider sent before (the defaults), so nothing changes in
  Azion. This applies to:
  - `modules` in `azion_application_main_setting` and `azion_firewall_main_setting`
  - `browser_cache` and `modules` (and their nested blocks) in
    `azion_application_cache_setting`
  - `tls`, `protocols` and `mtls` in `azion_workload`

`azion_waf_rule_sets` state is converted from the old map to `rule_set` blocks
automatically on the first plan.

## Validation that is now stricter

- Lists that were required but accepted an empty value (`= []`) now need at least
  one block where the API needs one: `behaviors` and `entries` in
  `azion_application_rule_engine`, `criteria`, `entries` and `behaviors` in
  `azion_firewall_rule_engine`, `conditions` in `azion_waf_rule_set`, `pages` in
  `azion_custom_page`. `criteria` in `azion_application_rule_engine` stays
  optional.
- A required argument missing inside an optional block (for example `rate` in a
  `sampling_attributes` block of `azion_data_stream`) is reported as
  `Invalid Attribute Combination: Attribute "...rate" must be specified when
  "...sampling_attributes" is specified`.

## Behavior change for omitted optional blocks

Some optional objects are reset to their defaults on every apply when you leave
them out, such as `modules` in `azion_application_main_setting`. This still
happens. But a change made to them outside Terraform (in Azion Console, for
example) now shows up in `terraform plan` **only while the block is declared** in
your configuration. To have Terraform detect and revert such changes, declare the
block, even with only its defaults:

```terraform
application {
  name = "My Application"
  modules {
    cache {}
    functions {}
    application_accelerator {}
    image_processor {}
  }
}
```

Each resource page lists the defaults that apply.

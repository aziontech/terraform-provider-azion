# Firewall Rules Engine Order Resource - Agent Documentation

This document provides detailed information about the `azion_firewall_rule_engine_order` resource implementation for AI agents working on this Terraform provider. It is the firewall counterpart of [RULES_ENGINE_ORDER.md](RULES_ENGINE_ORDER.md) and follows the same architectural pattern.

## Overview

The `azion_firewall_rule_engine_order` resource manages the evaluation order of the rule-engine rules of an Azion Firewall.

The Azion API exposes ordering through a dedicated PUT-only endpoint whose body is the **complete ordered list** of rule IDs:

| Resource | Endpoint |
|----------|----------|
| Firewall rules | `PUT /edge_firewall/firewalls/{firewall_id}/rules/order` |

Unlike the application rule engine, firewall rules have a **single phase** — there is no `request`/`response` split — so this resource has no `phase` attribute. Otherwise the architecture is identical to `azion_application_rule_engine_order`.

Because the operation is collective (one PUT replaces the whole order), this is modeled as its own resource — NOT as an attribute on `azion_firewall_rule_engine`.

## SDK Information

### API Endpoints

| Operation | Endpoint | API Service |
|-----------|----------|-------------|
| Create / Update | `PUT /edge_firewall/firewalls/{firewall_id}/rules/order` | `FirewallsRulesEngineAPIService.OrderFirewallRules` |
| Read | `GET /edge_firewall/firewalls/{firewall_id}/rules?ordering=order` | `FirewallsRulesEngineAPIService.ListFirewallRules` |
| Delete | No API call | No-op with diagnostic warning |

### SDK Types

```go
// Request body for firewall ordering.
azionapi.FirewallRuleEngineOrderRequest

// Response type — the PUT endpoint returns the paginated rule list
// reflecting the new order.
azionapi.PaginatedFirewallRuleList
```

The JSON body has the same shape as the application ordering endpoints:

```json
{ "order": [12345, 67890, 54321] }
```

## Implementation Details

### File Location

`internal/resource_firewall_rule_engine_order.go`

### Schema

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| `id` | string | — | yes | Same value as `firewall_id` (string-encoded) |
| `firewall_id` | int64 | yes | — | The firewall whose order is being managed |
| `order` | list of int64 | yes | — | Ordered rule IDs; first ID is evaluated first |
| `last_updated` | string | — | yes | Timestamp of the last Terraform update |

Note the absence of a `phase` attribute — firewall rules don't have phases.

### CRUD Lifecycle

#### Create / Update

Both call the same private helper `applyOrder`:

1. Translate `[]types.Int64` → `[]int64`, rejecting null/unknown elements and empty lists.
2. `NewFirewallRuleEngineOrderRequest(orderIDs)` → `OrderFirewallRules(ctx, firewallID).FirewallRuleEngineOrderRequest(*body).Execute()`.
3. 429 handled via `utils.RetryOn429` (5 retries).
4. Other errors surface the response body via `appendBodyError` (shared with the application order resource).

#### Read (drift detection)

`listOrderedRuleIDs` paginates through `ListFirewallRules` with `Ordering("order")` and `PageSize(100)`, sorts by the rule's `order` field, and returns the resulting `[]int64`. Manual reorderings outside Terraform show up as drift on the next plan.

A 404 from the list endpoint flags the resource for removal from state (`resp.State.RemoveResource`).

> **Pagination note**: page size is hardcoded at 100. If a firewall has more than 100 rules, the loop continues until `page >= TotalPages`. The per-page response body is closed inside `fetchRulePage`, NOT deferred inside the loop.

#### Delete

No-op. The Azion API has no inverse operation for ordering, so destroying the resource simply removes it from Terraform state and emits a warning diagnostic. Rules retain their last-applied order on the server.

#### Import

ID format: `{firewall_id}`. Example:

```bash
terraform import azion_firewall_rule_engine_order.example 1234567890
```

ImportState uses `resp.State.SetAttribute` directly so the subsequent Read can fully populate `order`.

### Shared Helpers

The firewall order resource reuses two helpers defined in `resource_application_rule_engine_order.go`:

- `diagAccumulator` — the small interface accepted by `applyOrder` so Create and Update can share the write path.
- `appendBodyError(diags, response, err)` — common error-detail extractor.
- `intSliceToInt64TypeSlice([]int64) []types.Int64` — list conversion.
- `ruleIDOrder` struct — `{id, order int64}` used by `fetchRulePage`.

Keep these helpers in the application order file (they appeared there first). If you add a third order resource, leave the shared helpers where they are; do not move them to `utils/` unless a clear common location emerges.

## Common Pitfalls

1. **Different API service name.** Use `r.client.api.FirewallsRulesEngineAPI` (note: plural "Firewalls", singular "RulesEngine"). The method is `OrderFirewallRules` (NOT `UpdateFirewallRulesOrder` — that's the application naming convention).
2. **No phase attribute.** Firewall rules are single-phase. Resist the urge to mirror the application order schema 1:1.
3. **Don't add an `order` attribute to `azion_firewall_rule_engine`.** Same reasoning as on the application side: per-rule ordering doesn't match the API shape and creates write amplification.
4. **Don't `defer response.Body.Close()` inside a pagination loop.** Defers accumulate until function return. Close per-iteration in the `fetchRulePage` helper.
5. **Required `order` works with "known after apply" values.** Users will normally pass `azion_firewall_rule_engine.foo.results.id` references, which are unknown at plan time. The framework handles this correctly.
6. **No validator import needed.** Without a `phase` attribute, there's no `stringvalidator.OneOf` to import; the file only needs `terraform-plugin-framework` packages plus `azionapi` and `utils`.

## Provider Registration

Registered in `internal/provider.go` under `Resources()` adjacent to `NewFirewallRuleEngineResource`:

```go
NewFirewallRuleEngineResource,
NewFirewallRuleEngineOrderResource,
```

## Documentation & Examples

| File | Purpose |
|------|---------|
| `docs/resources/firewall_rule_engine_order.md` | User-facing docs (terraform-plugin-docs format) |
| `examples/resources/azion_firewall_rule_engine_order/resource.tf` | Runnable example with parent firewall + two rules + reordering |

Per the project-wide child-resource doc convention (see `AGENTS.md` → "Child Resource Documentation"), the docs and example MUST show the parent `azion_firewall_main_setting` and at least two `azion_firewall_rule_engine` resources whose IDs are referenced by the order resource.

## Related Resources

- [`azion_firewall_main_setting`](../docs/resources/firewall_main_setting.md) — parent firewall
- [`azion_firewall_rule_engine`](../docs/resources/firewall_rule_engine.md) — the individual rules being ordered
- See [FIREWALL_RULES_ENGINE.md](FIREWALL_RULES_ENGINE.md) for the rule resource itself
- See [RULES_ENGINE_ORDER.md](RULES_ENGINE_ORDER.md) for the analogous application order resource

## Future Considerations

- **Drift in the rule resource.** `azion_firewall_rule_engine` still surfaces a computed `order` attribute. When the order resource changes ordering, the rule resources will show that attribute drifting on the next plan. A follow-up could mark `order` on `azion_firewall_rule_engine` as informational-only or remove it.

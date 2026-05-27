# Application Rules Engine Order Resource - Agent Documentation

This document provides detailed information about the `azion_application_rule_engine_order` resource implementation for AI agents working on this Terraform provider. It is intentionally separate from `RULES_ENGINE.md`: the two resources share a problem domain but have completely different API shapes and lifecycle semantics.

## Overview

The `azion_application_rule_engine_order` resource manages the evaluation order of an Azion Application's rule-engine rules for a single phase (`request` or `response`).

The Azion API exposes ordering through a dedicated PUT-only endpoint whose body is the **complete ordered list** of rule IDs. The "old" per-rule `order` field on the create/update endpoints is no longer the source of truth for ordering — it MUST go through these endpoints:

| Phase | Endpoint |
|-------|----------|
| Request | `PUT /workspace/applications/{application_id}/request_rules/order` |
| Response | `PUT /workspace/applications/{application_id}/response_rules/order` |

Because the operation is collective (one PUT replaces the whole order), this is modeled as its own resource — NOT as an attribute on `azion_application_rule_engine`. The two resources are independent: `azion_application_rule_engine` manages a rule's content; `azion_application_rule_engine_order` manages the sequence in which rules are evaluated.

## SDK Information

### API Endpoints

| Operation | Endpoint | API Service |
|-----------|----------|-------------|
| Create / Update (request phase) | `PUT /workspace/applications/{application_id}/request_rules/order` | `ApplicationsRequestRulesAPIService.UpdateApplicationRequestRulesOrder` |
| Create / Update (response phase) | `PUT /workspace/applications/{application_id}/response_rules/order` | `ApplicationsResponseRulesAPIService.UpdateApplicationResponseRulesOrder` |
| Read (request phase) | `GET /workspace/applications/{application_id}/request_rules?ordering=order` | `ApplicationsRequestRulesAPIService.ListApplicationRequestRules` |
| Read (response phase) | `GET /workspace/applications/{application_id}/response_rules?ordering=order` | `ApplicationsResponseRulesAPIService.ListApplicationResponseRules` |
| Delete | No API call | No-op with diagnostic warning |

### SDK Types

```go
// Request body for request-phase ordering.
azionapi.ApplicationRequestPhaseRuleEngineOrder

// Request body for response-phase ordering.
// Note the asymmetric naming: request phase has no "Request" suffix,
// response phase has "Request" suffix.
azionapi.ApplicationResponsePhaseRuleEngineOrderRequest

// Response types — both endpoints return the paginated rule list
// reflecting the new order.
azionapi.PaginatedRequestPhaseRuleList
azionapi.PaginatedResponsePhaseRuleList
```

The JSON body for both endpoints has identical shape:

```json
{ "order": [12345, 67890, 54321] }
```

## Implementation Details

### File Location

`internal/resource_application_rule_engine_order.go`

### Schema

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| `id` | string | — | yes | `{application_id}/{phase}` |
| `application_id` | int64 | yes | — | The application whose order is being managed |
| `phase` | string | yes | — | `"request"` or `"response"` (validated with `stringvalidator.OneOf`) |
| `order` | list of int64 | yes | — | Ordered rule IDs; first ID is evaluated first |
| `last_updated` | string | — | yes | Timestamp of the last Terraform update |

### CRUD Lifecycle

#### Create / Update

Both call the same private helper `applyOrder`:

1. Translate `[]types.Int64` → `[]int64`, rejecting null/unknown elements and empty lists.
2. Branch on phase:
   - **request**: `NewApplicationRequestPhaseRuleEngineOrder(orderIDs)` → `UpdateApplicationRequestRulesOrder(ctx, applicationID).ApplicationRequestPhaseRuleEngineOrder(*body).Execute()`
   - **response**: `NewApplicationResponsePhaseRuleEngineOrderRequest(orderIDs)` → `UpdateApplicationResponseRulesOrder(ctx, applicationID).ApplicationResponsePhaseRuleEngineOrderRequest(*body).Execute()`
3. 429 handled via `utils.RetryOn429` (5 retries).
4. Other errors surface the response body via `appendBodyError`.

#### Read (drift detection)

`listOrderedRuleIDs` paginates through `ListApplicationRequestRules` / `ListApplicationResponseRules` with `Ordering("order")` and `PageSize(100)`, sorts by the rule's `order` field, and returns the resulting `[]int64`. This means manual reorderings made outside Terraform show up as drift on the next plan.

A 404 from the list endpoint flags the resource for removal from state (`resp.State.RemoveResource`).

> **Pagination note**: page size is hardcoded at 100. If an application has more than 100 rules in a single phase, the loop continues until `page >= TotalPages`. The per-page response body is closed inside `fetchRulePage`, NOT deferred inside the loop (deferred close in a loop is a `bodyclose` lint anti-pattern).

#### Delete

No-op. The Azion API has no inverse operation for ordering, so destroying the resource simply removes it from Terraform state and emits a warning diagnostic. Rules retain their last-applied order on the server.

#### Import

ID format: `{application_id}/{phase}`. Example:

```bash
terraform import azion_application_rule_engine_order.request_order 1234567890/request
```

ImportState uses `resp.State.SetAttribute` directly (rather than constructing a model) so the subsequent Read can fully populate `order`.

### Helper: `diagAccumulator`

Because Create and Update share the same write path, `applyOrder` accepts a small `diagAccumulator` interface with `AddError` and `AddWarning`. `*diag.Diagnostics` (the pointer type of `resp.Diagnostics`) satisfies it, so callers pass `&resp.Diagnostics` directly. Pointer-to-interface is NOT used — pass the plain interface.

## Common Pitfalls

1. **Asymmetric SDK type names.** The request-phase body type is `ApplicationRequestPhaseRuleEngineOrder` (no `Request` suffix); the response-phase body type is `ApplicationResponsePhaseRuleEngineOrderRequest` (with `Request` suffix). This is an artifact of the OpenAPI generator and easy to mistype.
2. **Don't add an `order` attribute to `azion_application_rule_engine`.** Per-rule ordering does not match the API shape and creates write amplification + race conditions when Terraform applies rules in parallel.
3. **Don't `defer response.Body.Close()` inside a pagination loop.** Defers accumulate until function return. Close per-iteration in a helper.
4. **Required `order` works with "known after apply" values.** Users will normally pass `azion_application_rule_engine.foo.results.id` references, which are unknown at plan time. The framework handles this correctly; no special schema modifier is required.
5. **Rules not in the `order` list.** API behavior for rules omitted from the PUT body is not formally documented. Empirically test before assuming. Document for users that the resource only manages rules listed in `order`.
6. **The Default Rule.** Applications automatically get a Default Rule in the request phase. Including or excluding its ID in `order` has not yet been verified end-to-end — flag any unexpected behavior to users.
7. **Validator import.** Use `github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator` for `stringvalidator.OneOf`, NOT a `schema/validator/stringvalidator` path under the core framework.

## Provider Registration

Registered in `internal/provider.go` under `Resources()` adjacent to `NewApplicationRulesEngineResource`:

```go
NewApplicationRulesEngineResource,
NewApplicationRuleEngineOrderResource,
```

## Documentation & Examples

| File | Purpose |
|------|---------|
| `docs/resources/application_rule_engine_order.md` | User-facing docs (terraform-plugin-docs format) |
| `examples/resources/azion_application_rule_engine_order/resource.tf` | Runnable example with parent app + two rules + reordering |

Per the project-wide child-resource doc convention (see `AGENTS.md` → "Child Resource Documentation"), the docs and example MUST show the parent `azion_application_main_setting` and at least two `azion_application_rule_engine` resources whose IDs are referenced by the order resource.

## Related Resources

- [`azion_application_main_setting`](../docs/resources/application_main_setting.md) — parent application
- [`azion_application_rule_engine`](../docs/resources/application_rule_engine.md) — the individual rules being ordered
- See [RULES_ENGINE.md](RULES_ENGINE.md) for the rule resource itself

## Future Considerations

- **Firewall ordering.** The SDK also exposes `FirewallRuleEngineOrderRequest` for `azion_firewall_rule_engine`. A parallel `azion_firewall_rule_engine_order` resource would follow the same pattern; only the API service names change.
- **Drift in the rule resource.** `azion_application_rule_engine` still surfaces a computed `order` attribute. When the order resource changes ordering, the rule resources will show that attribute drifting on the next plan. A follow-up could mark `order` on `azion_application_rule_engine` as informational-only or remove it.

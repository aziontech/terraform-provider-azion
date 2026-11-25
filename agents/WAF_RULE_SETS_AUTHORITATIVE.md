# WAF Rule Sets (Authoritative) - Agent Documentation

This document covers `azion_waf_rule_sets`, the authoritative resource that owns **every** exception of one WAF. It is separate from [WAF_RULE_SETS.md](WAF_RULE_SETS.md), which covers `azion_waf_rule_set` (one exception per resource). Both resources use the same API, but their lifecycles are different.

Origin: FR-993 / ENG-49914. A customer created exceptions in Azion Console, and `terraform apply` neither showed a diff for them nor removed them. A per-item resource cannot see items it does not hold in state, so this resource was added.

## The two resources

| | `azion_waf_rule_set` | `azion_waf_rule_sets` |
|---|---|---|
| Scope | One exception | All exceptions of a WAF |
| Sees exceptions created outside Terraform | No | Yes, planned for deletion |
| Import ID | `{waf_id}/{exception_id}` | `{waf_id}` |
| Destroy | Deletes one exception | Deletes every exception of the WAF |

This is the same split as `google_project_iam_member` vs `google_project_iam_policy`. **The two must never manage the same WAF**, because each would undo the other's work on every apply. Keep this warning in both docs.

There is also a plural **data source** named `azion_waf_rule_sets` (`data_source_waf_rule_sets.go`). Resource and data source names live in separate namespaces, so the shared name is fine. Do not confuse the two files.

## API

There is no batch endpoint. Everything is built from the per-exception endpoints of `WAFsExceptionsAPI`:

| Use | Call |
|---|---|
| Read (all pages, `page_size=100`) | `ListWafExceptions(ctx, wafID).Page(p).PageSize(100)` |
| Create | `CreateWafException(ctx, wafID)` |
| Update (**PUT**) | `UpdateWafException(ctx, exceptionID, wafID)` |
| Delete | `DeleteWafException(ctx, exceptionID, wafID)` |

## Schema

```
Version 1
id            string  computed (= waf_id), UseStateForUnknown
waf_id        int64   required, RequiresReplace
last_updated  string  computed
rule_set      SetNestedBlock, optional; zero blocks = delete every exception
  Validators: wafRuleSetsNamesValidator (unique names, no "<name>#<digits>")
  name          string required (identifies the exception)
  exception_id  int64  computed (NO UseStateForUnknown, see below)
  rule_id / path / operator      Optional+Computed, no default
  active                         Optional+Computed, default true
  conditions {condition {match, condition_type, name, value}}  ListNestedBlock (wafExceptionConditionsBlock):
                                 listvalidator.IsRequired + SizeAtLeast(1); condition is objectvalidator.IsRequired
  last_editor / last_modified    computed
```

```hcl
rule_set {
  name = "Allow health check"
  path = "/health"
  conditions {
    condition {
      match          = "any_url"
      condition_type = "generic"
    }
  }
}
```

The Go model holds `RuleSets []WafRuleSetsItemModel` (`tfsdk:"rule_set"`, each with `Name`). All internal logic (`reconcile`, `planWafRuleSetsOps`, `buildWafRuleSetsState`, ...) still works on a `map[string]WafRuleSetsItemModel` keyed by name: convert with `wafRuleSetsByName` (slice → map) and `wafRuleSetsList` (map → slice sorted by name; sets `Name` from the key). Zero `rule_set` blocks arrive as a **null** set → nil slice → empty map → every exception deleted. Do **not** add `setvalidator.IsRequired()`: "no blocks" is the documented way to remove all exceptions.

**Why a set, not a list.** The exceptions have no order (the API lists by ID). A list compares positionally, so it would diff whenever configuration order differs from the order state was written in, and reordering blocks would plan changes that apply nothing. A set has no order. The cost: Terraform shows a changed exception as one element removed and one added, and the framework cannot correlate set elements with the configuration (see ModifyPlan below).

### Schema version 0 → 1 (`UpgradeState`)

v2.11.0 shipped `rule_sets = { "<name>" = {...} }`, a `MapNestedAttribute`. The framework has no map block, so it became the `rule_set` set block and the state type changed (map → set). `UpgradeState` declares the v0 schema (`wafRuleSetsSchemaV0`, types only), reads it into `wafRuleSetsResourceModelV0` and writes each map key as `name` (`upgradeWafRuleSetsStateV0`); everything else is copied. An empty map becomes no blocks. Verified: hand-written v0 state + the configuration rewritten as blocks → `terraform plan -refresh=false` shows "No changes". Tests: `TestWAFRuleSetsUpgradeStateV0` (through the framework's real `UpgradeResourceState`) and `TestWAFRuleSetsUpgradeStateV0EmptyMap`. Keep the v0 schema and upgrader as long as v2.11.0 state may exist.

**Never use `UseStateForUnknown` on any attribute inside `rule_set`.** It only skips when the *whole resource* has no state. For an exception added to an existing resource (a new exception, one deleted in Console and being recreated, a rename, a rollback), it copies that element's **null** prior value into the plan, and apply fails with `Provider produced inconsistent result after apply: ... exception_id: was null, but now ...`. This shipped once and was caught in manual testing. `ModifyPlan` carries `exception_id` over for names that state holds, and explicitly marks the computed fields of new exceptions as unknown.

The Optional+Computed and "only `active` has a default" rules are the same as in `azion_waf_rule_set`, for the same reasons. See WAF_RULE_SETS.md. Do not add `UseStateForUnknown` to `rule_id`, `path`, `operator` or the condition `name`/`value`. When the plan carries a stale value and the API returns something else, apply fails with "Provider produced inconsistent result". `ModifyPlan` (below) already removes the noise for unchanged exceptions.

## Key design points

### Duplicate names (`buildWafRuleSetsState`)

The API does not document names as unique per WAF. When several exceptions share a name, the one whose ID prior state already holds under that name keeps it. Without prior state, the lowest ID keeps it. Every other duplicate is keyed, and written to state with the name, `<name>#<exception_id>`, so the plan shows it as a deletion, and Read emits a warning. `wafRuleSetsNamesValidator` rejects declared names of that form (`isWafRuleSetsDuplicateKey`), so they can never collide, and rejects two `rule_set` blocks with the same name. (Two byte-identical blocks collapse into one set element before validation; that is harmless.)

### Reconcile order (`reconcile`, `planWafRuleSetsOps`)

The order is **delete → update → create**, with keys sorted within each step. Deleting first frees names for creations. On the first error, reconcile stops and returns what was applied. Create, Update and Delete all save that partial result to state.

- An update that gets a 404 (deleted since the last refresh) falls back to a create.
- A delete that gets a 404 counts as success.

### Updates are PUT: send undeclared fields back (`buildWafRuleSetsRequest`)

A field the configuration leaves to the API is unknown in the plan. On update, the prior (refreshed) value is sent instead of omitting the field, which could reset it. On create there is no prior value, so the field is omitted.

### What counts as a change (`wafRuleSetsItemNeedsUpdate`)

The plan and state differ on a field only when the planned value is **known** and not equal. An unknown planned value means "not declared", so it never forces an update.

### Plan noise and unknowns (`ModifyPlan`, `wafRuleSetsItemPlannedForApply`)

`ModifyPlan` matches planned exceptions to state **by name** and replaces each one that does not need an update with its state value; when nothing changes at all it also restores `last_updated`. The result: the plan lists only the exceptions that actually change, and a converged WAF plans empty.

For exceptions that apply creates or updates, `ModifyPlan` itself marks unknown everything the API supplies: `last_editor`, `last_modified`, `exception_id` (new only), and any null `rule_id`/`path`/`operator`/condition `name`/`value` (null in a block = not declared). The framework's `MarkComputedNilsAsUnknown` cannot be relied on inside a set block: it locates each element in the configuration by value, and an element whose value differs (a default applied, prior values filled in by core) is not found and keeps its nulls, which apply would then contradict. `ModifyPlan` therefore also runs on create (null prior state). If any planned `name` is unknown it does nothing.

### Create does not delete what the plan did not show

On Create, the provider lists the WAF and adopts any exception that has a declared name: it is updated, not duplicated. Undeclared exceptions are **left in place** with a warning. Deleting them would remove something that never appeared in the plan. The next refresh reads them, and the next plan removes them. Import is the recommended adoption path.

### Delete

Delete removes every exception in state, which is every exception of the WAF as of the last refresh.

## Tests

`internal/resource_waf_rule_sets_test.go`:

- Pure functions: `planWafRuleSetsOps`, `wafRuleSetsItemNeedsUpdate`, `buildWafRuleSetsState` (duplicates), `isWafRuleSetsDuplicateKey`, `buildWafRuleSetsRequest`.
- `wafRuleSetsNamesValidator`, `UpgradeState` (v0 map → v1 blocks), zero blocks deleting everything, and create planning API values unknown (`resource_waf_rule_sets_plan_test.go`).
- `ModifyPlan` driven with real `tfsdk.Plan`/`tfsdk.State`.
- `resource_waf_rule_sets_plan_test.go` runs plans through the framework's real `PlanResourceChange` (`providerserver` + a test provider that serves only this resource), covering defaults, computed-nil marking, attribute plan modifiers, and `ModifyPlan` together. **Any plan-shape change must be tested here.** Calling `ModifyPlan` with a hand-built plan is what let the `UseStateForUnknown` bug through.
- An `httptest` fake of the exceptions endpoint that serves **one exception per page** to exercise pagination. `TestWAFRuleSetsRollsBackConsoleChanges` reproduces the FR-993 scenario end to end and checks that the second plan is empty.

`terraform-plugin-testing` is not a dependency, and adding it drags in `terraform-plugin-go` bumps (see the lockstep note). The provider's V4 URL is also hard-coded. That is why the tests call the resource methods directly instead of running acceptance tests.

## Files

- `internal/resource_waf_rule_sets.go`
- `internal/resource_waf_rule_sets_test.go`
- `internal/provider.go` (`WafRuleSetsResource`)
- `docs/resources/waf_rule_sets.md`
- `examples/resources/azion_waf_rule_sets/{resource.tf,import.sh}`

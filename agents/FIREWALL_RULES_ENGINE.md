# Firewall Rules Engine - Data Sources and Resource Generation Guide

This document provides comprehensive guidance for AI agents generating Terraform provider data sources and resources for Firewall Rules Engine from the Azion OpenAPI specification. It documents the patterns, conventions, and implementation details used in the Firewall Rules Engine data sources and resource.

## Overview

The Firewall Rules Engine is a feature within Azion Firewalls that allows you to create conditional rules with behaviors. Unlike the Application Rules Engine, the Firewall Rules Engine:

- Is associated with a **Firewall** (not an Application)
- Has only **one phase** (request phase) - there's no response phase
- Supports specific firewall behaviors like `run_function`, `set_custom_response`, `set_waf`, `set_rate_limit`, and `drop`

## API Structure

### SDK Import Path

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### API Client Access

```go
// Access the Firewall Rules Engine API
client.api.FirewallsRulesEngineAPI
```

### Key API Methods

| Method | Description |
|--------|-------------|
| `RetrieveFirewallRule(ctx, firewallId, ruleId)` | Get a specific rule by ID |
| `ListFirewallRules(ctx, firewallId)` | List all rules for a firewall |
| `CreateFirewallRule(ctx, firewallId)` | Create a new rule |
| `UpdateFirewallRule(ctx, firewallId, ruleId)` | Update an existing rule (PUT) |
| `PartialUpdateFirewallRule(ctx, firewallId, ruleId)` | Partial update (PATCH) |
| `DeleteFirewallRule(ctx, firewallId, ruleId)` | Delete a rule |
| `OrderFirewallRules(ctx, firewallId)` | Reorder rules |

---

## Data Sources

### Singular Data Source (`azion_firewall_rule_engine`)

```go
type FirewallRuleEngineDataSourceModel struct {
    ID         types.String                       `tfsdk:"id"`
    FirewallID types.Int64                        `tfsdk:"firewall_id"`
    Results    *FirewallRuleEngineResultDataModel `tfsdk:"results"`
}

type FirewallRuleEngineResultDataModel struct {
    ID           types.Int64                 `tfsdk:"id"`
    Name         types.String                `tfsdk:"name"`
    Active       types.Bool                  `tfsdk:"active"`
    Criteria     []FirewallCriteriaDataModel `tfsdk:"criteria"`
    Behaviors    []FirewallBehaviorDataModel `tfsdk:"behaviors"`
    Description  types.String                `tfsdk:"description"`
    Order        types.Int64                 `tfsdk:"order"`
    LastEditor   types.String                `tfsdk:"last_editor"`
    LastModified types.String                `tfsdk:"last_modified"`
    CreatedAt    types.String                `tfsdk:"created_at"`
}

type FirewallCriteriaDataModel struct {
    Entries []FirewallCriteriaEntryDataModel `tfsdk:"entries"`
}

type FirewallCriteriaEntryDataModel struct {
    Conditional types.String `tfsdk:"conditional"`
    Variable    types.String `tfsdk:"variable"`
    Operator    types.String `tfsdk:"operator"`
    Argument    types.String `tfsdk:"argument"`
}

type FirewallBehaviorDataModel struct {
    Type       types.String                    `tfsdk:"type"`
    Attributes *FirewallBehaviorAttrsDataModel `tfsdk:"attributes"`
}

type FirewallBehaviorAttrsDataModel struct {
    // For run_function behavior
    Value types.Int64 `tfsdk:"value"`
    // For set_custom_response behavior
    StatusCode  types.Int64  `tfsdk:"status_code"`
    ContentType types.String `tfsdk:"content_type"`
    ContentBody types.String `tfsdk:"content_body"`
    // For set_waf behavior
    WafId types.Int64  `tfsdk:"waf_id"`
    Mode  types.String `tfsdk:"mode"`
    // For set_rate_limit behavior
    Type             types.String `tfsdk:"type"`
    LimitBy          types.String `tfsdk:"limit_by"`
    AverageRateLimit types.Int64  `tfsdk:"average_rate_limit"`
    MaximumBurstSize types.Int64  `tfsdk:"maximum_burst_size"`
}
```

### Plural Data Source (`azion_firewall_rules_engine`)

```go
type FirewallRulesEngineDataSourceModel struct {
    ID         types.String                     `tfsdk:"id"`
    FirewallID types.Int64                      `tfsdk:"firewall_id"`
    Counter    types.Int64                      `tfsdk:"counter"`
    TotalPages types.Int64                      `tfsdk:"total_pages"`
    Page       types.Int64                      `tfsdk:"page"`
    PageSize   types.Int64                      `tfsdk:"page_size"`
    Links      *LinksModel                      `tfsdk:"links"`
    Results    []FirewallRulesEngineResultModel `tfsdk:"results"`
}

type FirewallRulesEngineResultModel struct {
    ID           types.Int64                 `tfsdk:"id"`
    Name         types.String                `tfsdk:"name"`
    Active       types.Bool                  `tfsdk:"active"`
    Criteria     []FirewallCriteriaDataModel `tfsdk:"criteria"`
    Behaviors    []FirewallBehaviorDataModel `tfsdk:"behaviors"`
    Description  types.String                `tfsdk:"description"`
    Order        types.Int64                 `tfsdk:"order"`
    LastEditor   types.String                `tfsdk:"last_editor"`
    LastModified types.String                `tfsdk:"last_modified"`
    CreatedAt    types.String                `tfsdk:"created_at"`
}
```

Note: The plural data source reuses the same `FirewallCriteriaDataModel`, `FirewallCriteriaEntryDataModel`, `FirewallBehaviorDataModel`, and `FirewallBehaviorAttrsDataModel` types defined in the singular data source section.

---

## Resource Implementation

### Resource File: `internal/resource_firewall_rule_engine.go`

#### Resource Model

```go
type FirewallRuleEngineResourceModel struct {
    ID          types.String                      `tfsdk:"id"`
    FirewallID  types.Int64                       `tfsdk:"firewall_id"`
    LastUpdated types.String                      `tfsdk:"last_updated"`
    Results     *FirewallRuleEngineResultResource `tfsdk:"results"`
}

type FirewallRuleEngineResultResource struct {
    ID           types.Int64                         `tfsdk:"id"`
    Name         types.String                        `tfsdk:"name"`
    Active       types.Bool                          `tfsdk:"active"`
    Criteria     []FirewallCriteriaResourceModel    `tfsdk:"criteria"`
    Behaviors    []FirewallBehaviorResourceModel    `tfsdk:"behaviors"`
    Description  types.String                        `tfsdk:"description"`
    Order        types.Int64                         `tfsdk:"order"`
    LastEditor   types.String                        `tfsdk:"last_editor"`
    LastModified types.String                        `tfsdk:"last_modified"`
    CreatedAt    types.String                        `tfsdk:"created_at"`
}
```

#### Criteria Model

```go
type FirewallCriteriaResourceModel struct {
    Entries []FirewallCriteriaEntryResourceModel `tfsdk:"entries"`
}

type FirewallCriteriaEntryResourceModel struct {
    Conditional types.String `tfsdk:"conditional"`
    Variable    types.String `tfsdk:"variable"`
    Operator    types.String `tfsdk:"operator"`
    Argument    types.String `tfsdk:"argument"`
}
```

#### Behavior Model

```go
type FirewallBehaviorResourceModel struct {
    Type       types.String                        `tfsdk:"type"`
    Attributes *FirewallBehaviorAttrsResourceModel `tfsdk:"attributes"`
}

type FirewallBehaviorAttrsResourceModel struct {
    // For run_function behavior
    Value types.Int64 `tfsdk:"value"`
    // For set_custom_response behavior
    StatusCode  types.Int64  `tfsdk:"status_code"`
    ContentType types.String `tfsdk:"content_type"`
    ContentBody types.String `tfsdk:"content_body"`
    // For set_waf behavior
    WafId types.Int64  `tfsdk:"waf_id"`
    Mode  types.String `tfsdk:"mode"`
    // For set_rate_limit behavior
    Type             types.String `tfsdk:"type"`
    LimitBy          types.String `tfsdk:"limit_by"`
    AverageRateLimit types.Int64  `tfsdk:"average_rate_limit"`
    MaximumBurstSize types.Int64  `tfsdk:"maximum_burst_size"`
}
```

#### Resource Registration

```go
func NewFirewallRuleEngineResource() resource.Resource {
    return &firewallRuleEngineResource{}
}
```

#### Metadata

```go
func (r *firewallRuleEngineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_firewall_rule_engine"
}
```

#### Key CRUD Operations

##### Create

```go
func (r *firewallRuleEngineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // 1. Extract firewall_id from config
    // 2. Build criteria using buildFirewallCriteriaRequest()
    // 3. Build behaviors using buildFirewallBehaviorsRequest()
    // 4. Create rule request with azionapi.NewFirewallRuleRequest()
    // 5. Call API: r.client.api.FirewallsRulesEngineAPI.CreateFirewallRule(ctx, firewallID)
    // 6. Handle 429 rate limiting with utils.RetryOn429
    // 7. Build state from response
}
```

##### Read

```go
func (r *firewallRuleEngineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    // 1. Parse ID format: {firewallID}/{ruleID}
    // 2. Call API: r.client.api.FirewallsRulesEngineAPI.RetrieveFirewallRule(ctx, firewallID, ruleID)
    // 3. Handle 404 by removing resource from state
    // 4. Handle 429 rate limiting
    // 5. Transform response to state
}
```

##### Update

```go
func (r *firewallRuleEngineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // 1. Get firewall_id and rule_id from state/plan
    // 2. Build criteria and behaviors
    // 3. Call API: r.client.api.FirewallsRulesEngineAPI.UpdateFirewallRule(ctx, firewallID, ruleID)
    // 4. Handle rate limiting
    // 5. Update state
}
```

##### Delete

```go
func (r *firewallRuleEngineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // 1. Get firewall_id and rule_id from state
    // 2. Call API: r.client.api.FirewallsRulesEngineAPI.DeleteFirewallRule(ctx, firewallID, ruleID)
    // 3. Handle 404 gracefully (resource already deleted)
    // 4. Handle rate limiting
}
```

##### Import

```go
func (r *firewallRuleEngineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Format: {firewall_id}/{rule_id}
    // Example: 1234567890/987654
}
```

---

## Documentation and Examples

### MANDATORY: Parent Resource Documentation

**IMPORTANT**: Firewall Rules Engine is a child resource of `azion_firewall_main_setting`. Documentation and examples MUST include the parent resource creation to show complete context.

When updating documentation, always include:

1. **Parent Firewall Example** - Show creation of the parent firewall first
2. **Reference Using Terraform Interpolation** - Use `azion_firewall_main_setting.example.data.id` to reference the parent ID

### MANDATORY: Supported Behaviors Documentation

**IMPORTANT**: The resource documentation at `docs/resources/firewall_rule_engine.md` MUST always include a "Supported Behaviors" section listing every behavior `type` accepted by the API. Whenever a behavior is added, removed, or renamed in the SDK, this section MUST be updated in the same change so users on the Terraform Registry can discover valid values without reading the SDK.

The section must:

1. **Cover every behavior** - One row per behavior type with a short description and whether it requires `attributes`.
2. **Link from the schema** - The `type` field description under `results.behaviors` must list valid values and point users to the Supported Behaviors section.
3. **Include at least one example per attribute shape** - The Example Usage section must demonstrate: a no-args behavior (`drop` or `deny`), a behavior with a scalar `value` (`run_function`), and behaviors with structured attributes (`set_custom_response`, `set_waf`, `set_rate_limit`).

#### Current Supported Behaviors (keep in sync with the SDK)

`deny`, `drop`, `set_rate_limit`, `set_waf`, `run_function`, `set_custom_response`

### Documentation Files

Documentation is auto-generated by `terraform-plugin-docs` and located in:

| Type | Location |
|------|----------|
| Singular Data Source Doc | `docs/data-sources/firewall_rule_engine.md` |
| Plural Data Source Doc | `docs/data-sources/firewall_rules_engine.md` |
| Resource Doc | `docs/resources/firewall_rule_engine.md` |

### Example Files

Example Terraform configurations are located in:

| Type | Location |
|------|----------|
| Singular Data Source Example | `examples/data-sources/azion_firewall_rule_engine/data-source.tf` |
| Plural Data Source Example | `examples/data-sources/azion_firewall_rules_engine/data-source.tf` |
| Resource Example | `examples/resources/azion_firewall_rule_engine/resource.tf` |

### Example: Complete Resource Usage with Parent Firewall

```terraform
# First, create the parent firewall
resource "azion_firewall_main_setting" "example" {
  data = {
    name   = "My Firewall"
    active = true
  }
}

# Then create the rule engine for that firewall
resource "azion_firewall_rule_engine" "example" {
  firewall_id = azion_firewall_main_setting.example.data.id
  results = {
    name        = "Block Specific Path"
    description = "Block requests to specific path"
    active      = true
    behaviors = [
      {
        type = "drop"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "${request_uri}"
            operator    = "matches"
            conditional = "if"
            argument    = "/admin.*"
          }
        ]
      }
    ]
  }
}
```

---

## SDK Types

### FirewallRule

```go
type FirewallRule struct {
    Id           int64                       `json:"id"`
    Name         string                      `json:"name"`
    LastEditor   string                      `json:"last_editor"`
    LastModified time.Time                   `json:"last_modified"`
    Active       *bool                       `json:"active,omitempty"`
    Criteria     [][]FirewallCriterionField  `json:"criteria"`
    Behaviors    []FirewallBehavior          `json:"behaviors"`
    Description  *string                     `json:"description,omitempty"`
    Order        int64                       `json:"order"`
}
```

### FirewallCriterionField

```go
type FirewallCriterionField struct {
    Conditional string                              `json:"conditional"`
    Variable    string                              `json:"variable"`
    Operator    string                              `json:"operator"`
    Argument    NullableFirewallCriterionArgument   `json:"argument,omitempty"`
}
```

### FirewallBehavior (Polymorphic)

The `FirewallBehavior` is a polymorphic type that can be one of:

1. **FirewallBehaviorArgs** - For behaviors with a simple value argument (e.g., `run_function`)
2. **FirewallBehaviorNoArgs** - For behaviors without arguments (e.g., `drop`)
3. **FirewallBehaviorObjectArgs** - For behaviors with complex object attributes

```go
type FirewallBehavior struct {
    FirewallBehaviorArgs        *FirewallBehaviorArgs
    FirewallBehaviorNoArgs      *FirewallBehaviorNoArgs
    FirewallBehaviorObjectArgs  *FirewallBehaviorObjectArgs
}
```

### FirewallBehaviorArgs

Used for `run_function` behavior:

```go
type FirewallBehaviorArgs struct {
    Type       string                                    `json:"type"`
    Attributes FirewallBehaviorRunFunctionAttributes     `json:"attributes"`
}

type FirewallBehaviorRunFunctionAttributes struct {
    Value int64 `json:"value"`  // Function instance ID
}
```

### FirewallBehaviorNoArgs

Used for behaviors like `drop`:

```go
type FirewallBehaviorNoArgs struct {
    Type string `json:"type"`
}
```

### FirewallBehaviorObjectArgs

Used for behaviors with complex attributes:

```go
type FirewallBehaviorObjectArgs struct {
    Type       string                                `json:"type"`
    Attributes FirewallBehaviorObjectArgsAttributes  `json:"attributes"`
}
```

### FirewallBehaviorObjectArgsAttributes (Polymorphic)

```go
type FirewallBehaviorObjectArgsAttributes struct {
    FirewallBehaviorSetCustomResponseAttributes *FirewallBehaviorSetCustomResponseAttributes
    FirewallBehaviorSetRateLimitAttributes      *FirewallBehaviorSetRateLimitAttributes
    FirewallBehaviorSetWafAttributes            *FirewallBehaviorSetWafAttributes
}
```

### FirewallBehaviorSetCustomResponseAttributes

For `set_custom_response` behavior:

```go
type FirewallBehaviorSetCustomResponseAttributes struct {
    StatusCode  int64    `json:"status_code"`
    ContentType *string  `json:"content_type,omitempty"`
    ContentBody *string  `json:"content_body,omitempty"`
}
```

### FirewallBehaviorSetWafAttributes

For `set_waf` behavior:

```go
type FirewallBehaviorSetWafAttributes struct {
    WafId int64  `json:"waf_id"`
    Mode  string `json:"mode"`  // "logging" or "blocking"
}
```

### FirewallBehaviorSetRateLimitAttributes

For `set_rate_limit` behavior:

```go
type FirewallBehaviorSetRateLimitAttributes struct {
    Type              *string        `json:"type,omitempty"`       // "second" or "minute"
    LimitBy           string         `json:"limit_by"`             // "client_ip" or "global"
    AverageRateLimit  int64          `json:"average_rate_limit"`
    MaximumBurstSize  NullableInt64  `json:"maximum_burst_size,omitempty"`
}
```

---

## Transformation Functions

### buildFirewallCriteriaRequest

Transforms criteria from Terraform model to SDK request:

```go
func buildFirewallCriteriaRequest(criteria []FirewallCriteriaResourceModel) [][]azionapi.FirewallCriterionFieldRequest {
    var result [][]azionapi.FirewallCriterionFieldRequest
    for _, criterion := range criteria {
        var criterionGroup []azionapi.FirewallCriterionFieldRequest
        for _, c := range criterion.Entries {
            criterionField := azionapi.NewFirewallCriterionFieldRequest(
                c.Conditional.ValueString(),
                c.Variable.ValueString(),
                c.Operator.ValueString(),
            )
            if !c.Argument.IsNull() && !c.Argument.IsUnknown() {
                arg := azionapi.FirewallCriterionArgumentRequest{
                    String: c.Argument.ValueStringPointer(),
                }
                criterionField.SetArgument(arg)
            }
            criterionGroup = append(criterionGroup, *criterionField)
        }
        result = append(result, criterionGroup)
    }
    return result
}
```

### buildFirewallBehaviorsRequest

Transforms behaviors from Terraform model to SDK request:

```go
func buildFirewallBehaviorsRequest(behaviors []FirewallBehaviorResourceModel) []azionapi.FirewallBehaviorRequest {
    var result []azionapi.FirewallBehaviorRequest
    for _, b := range behaviors {
        behaviorType := b.Type.ValueString()

        // Check if it's a behavior without arguments (like "drop")
        if b.Attributes == nil {
            noArgsBehavior := azionapi.NewFirewallBehaviorNoArgsRequest(behaviorType)
            behaviorRequest := azionapi.FirewallBehaviorNoArgsRequestAsFirewallBehaviorRequest(noArgsBehavior)
            result = append(result, behaviorRequest)
            continue
        }

        // Handle behaviors with arguments based on type
        switch behaviorType {
        case "run_function":
            attrs := azionapi.NewFirewallBehaviorRunFunctionAttributesRequest(b.Attributes.Value.ValueInt64())
            argsBehavior := azionapi.NewFirewallBehaviorArgsRequest(behaviorType, *attrs)
            behaviorRequest := azionapi.FirewallBehaviorArgsRequestAsFirewallBehaviorRequest(argsBehavior)
            result = append(result, behaviorRequest)

        case "set_custom_response":
            attrs := azionapi.NewFirewallBehaviorSetCustomResponseAttributesRequest(
                b.Attributes.StatusCode.ValueInt64(),
            )
            if !b.Attributes.ContentType.IsNull() && !b.Attributes.ContentType.IsUnknown() {
                attrs.SetContentType(b.Attributes.ContentType.ValueString())
            }
            if !b.Attributes.ContentBody.IsNull() && !b.Attributes.ContentBody.IsUnknown() {
                attrs.SetContentBody(b.Attributes.ContentBody.ValueString())
            }
            objAttrs := azionapi.FirewallBehaviorSetCustomResponseAttributesRequestAsFirewallBehaviorObjectArgsRequestAttributes(attrs)
            objectArgsBehavior := azionapi.NewFirewallBehaviorObjectArgsRequest(behaviorType, objAttrs)
            behaviorRequest := azionapi.FirewallBehaviorObjectArgsRequestAsFirewallBehaviorRequest(objectArgsBehavior)
            result = append(result, behaviorRequest)

        case "set_waf":
            attrs := azionapi.NewFirewallBehaviorSetWafAttributesRequest(
                b.Attributes.WafId.ValueInt64(),
                b.Attributes.Mode.ValueString(),
            )
            objAttrs := azionapi.FirewallBehaviorSetWafAttributesRequestAsFirewallBehaviorObjectArgsRequestAttributes(attrs)
            objectArgsBehavior := azionapi.NewFirewallBehaviorObjectArgsRequest(behaviorType, objAttrs)
            behaviorRequest := azionapi.FirewallBehaviorObjectArgsRequestAsFirewallBehaviorRequest(objectArgsBehavior)
            result = append(result, behaviorRequest)

        case "set_rate_limit":
            attrs := azionapi.NewFirewallBehaviorSetRateLimitAttributesRequest(
                b.Attributes.LimitBy.ValueString(),
                b.Attributes.AverageRateLimit.ValueInt64(),
            )
            if !b.Attributes.Type.IsNull() && !b.Attributes.Type.IsUnknown() {
                attrs.SetType(b.Attributes.Type.ValueString())
            }
            if !b.Attributes.MaximumBurstSize.IsNull() && !b.Attributes.MaximumBurstSize.IsUnknown() {
                maxBurstVal := b.Attributes.MaximumBurstSize.ValueInt64()
                attrs.SetMaximumBurstSize(maxBurstVal)
            }
            objAttrs := azionapi.FirewallBehaviorSetRateLimitAttributesRequestAsFirewallBehaviorObjectArgsRequestAttributes(attrs)
            objectArgsBehavior := azionapi.NewFirewallBehaviorObjectArgsRequest(behaviorType, objAttrs)
            behaviorRequest := azionapi.FirewallBehaviorObjectArgsRequestAsFirewallBehaviorRequest(objectArgsBehavior)
            result = append(result, behaviorRequest)

        default:
            // For unknown behavior types, treat as no-args behavior
            noArgsBehavior := azionapi.NewFirewallBehaviorNoArgsRequest(behaviorType)
            behaviorRequest := azionapi.FirewallBehaviorNoArgsRequestAsFirewallBehaviorRequest(noArgsBehavior)
            result = append(result, behaviorRequest)
        }
    }
    return result
}
```

### transformFirewallRuleToResultModel

Transforms API response to Terraform state:

```go
func transformFirewallRuleToResultModel(rule azionapi.FirewallRule) *FirewallRuleEngineResultResource {
    result := &FirewallRuleEngineResultResource{
        ID:    types.Int64Value(rule.GetId()),
        Name:  types.StringValue(rule.GetName()),
        Order: types.Int64Value(rule.GetOrder()),
    }

    if rule.Active != nil {
        result.Active = types.BoolValue(*rule.Active)
    }
    if rule.Description != nil {
        result.Description = types.StringValue(*rule.Description)
    }
    result.LastEditor = types.StringValue(rule.GetLastEditor())
    result.LastModified = types.StringValue(rule.GetLastModified().Format(time.RFC3339))
    result.CreatedAt = types.StringValue(rule.GetCreatedAt().Format(time.RFC3339))

    // Transform criteria
    for _, criterionGroup := range rule.Criteria {
        var criterionSet []FirewallCriteriaEntryResourceModel
        for _, c := range criterionGroup {
            arg := getFirewallCriterionArgumentValue(c.Argument)
            var argValue types.String
            if arg == "" {
                argValue = types.StringNull()
            } else {
                argValue = types.StringValue(arg)
            }
            criterionSet = append(criterionSet, FirewallCriteriaEntryResourceModel{
                Conditional: types.StringValue(c.GetConditional()),
                Variable:    types.StringValue(c.GetVariable()),
                Operator:    types.StringValue(c.GetOperator()),
                Argument:    argValue,
            })
        }
        result.Criteria = append(result.Criteria, FirewallCriteriaResourceModel{
            Entries: criterionSet,
        })
    }

    // Transform behaviors
    for _, b := range rule.Behaviors {
        behavior := FirewallBehaviorResourceModel{}

        if b.FirewallBehaviorArgs != nil {
            behavior.Type = types.StringValue(b.FirewallBehaviorArgs.GetType())
            attrs := transformFirewallBehaviorArgsAttrs(b.FirewallBehaviorArgs.Attributes)
            behavior.Attributes = &attrs
        } else if b.FirewallBehaviorNoArgs != nil {
            behavior.Type = types.StringValue(b.FirewallBehaviorNoArgs.GetType())
        } else if b.FirewallBehaviorObjectArgs != nil {
            behavior.Type = types.StringValue(b.FirewallBehaviorObjectArgs.GetType())
            attrs := transformFirewallBehaviorObjectAttrs(b.FirewallBehaviorObjectArgs.Attributes)
            behavior.Attributes = &attrs
        }
        result.Behaviors = append(result.Behaviors, behavior)
    }

    return result
}
```

---

## Key Differences from Application Rules Engine

| Feature | Application Rules Engine | Firewall Rules Engine |
|---------|-------------------------|----------------------|
| Parent Resource | Application | Firewall |
| Phases | Request and Response | Request only |
| API Endpoint | `/workspace/applications/{id}/request_rules` or `/response_rules` | `/workspace/firewalls/{id}/request_rules` |
| Phase Parameter Required | Yes | No |
| Behavior Types | Application-specific behaviors | Firewall-specific behaviors (run_function, set_custom_response, set_waf, set_rate_limit, drop) |

---

## Criterion Variables

The Firewall Rules Engine supports the following criterion variables:

| Variable | Description | Operators |
|----------|-------------|-----------|
| `${header_accept}` | Accept header | matches, does_not_match |
| `${header_accept_encoding}` | Accept-Encoding header | matches, does_not_match |
| `${header_accept_language}` | Accept-Language header | matches, does_not_match |
| `${header_cookie}` | Cookie header | matches, does_not_match |
| `${header_origin}` | Origin header | matches, does_not_match |
| `${header_referer}` | Referer header | matches, does_not_match |
| `${header_user_agent}` | User-Agent header | matches, does_not_match |
| `${host}` | Host | is_equal, is_not_equal, matches, does_not_match |
| `${network}` | Network | is_in_list, is_not_in_list |
| `${request_args}` | Request arguments | is_equal, is_not_equal, matches, does_not_match, exists, does_not_exist |
| `${request_method}` | Request method | is_equal, is_not_equal |
| `${request_uri}` | Request URI | starts_with, does_not_starts_with, is_equal, is_not_equal, matches, does_not_match |
| `${scheme}` | Scheme | is_equal, is_not_equal |
| `${ssl_verification_status}` | SSL verification status | is_equal, is_not_equal |
| `${client_certificate_validation}` | Client certificate validation | is_equal, is_not_equal |

---

## Error Handling

Follow the standard error handling pattern with rate limiting (429) retry:

```go
if err != nil {
    if response != nil && response.StatusCode == 429 {
        // Retry with utils.RetryOn429
        result, response, err = r.readRuleWithRetry(ctx, firewallID, ruleID)
        
        if response != nil {
            defer response.Body.Close()
        }
        
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else if response != nil {
        bodyBytes, _ := io.ReadAll(response.Body)
        resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
        response.Body.Close()
        return
    }
}
```

---

## File Naming Convention

| Type | File Name | Terraform Name |
|------|-----------|----------------|
| Singular Data Source | `data_source_firewall_rule_engine.go` | `azion_firewall_rule_engine` |
| Plural Data Source | `data_source_firewall_rules_engine.go` | `azion_firewall_rules_engine` |
| Resource | `resource_firewall_rule_engine.go` | `azion_firewall_rule_engine` |

**Note**: The Terraform resource names still use `edge_firewall` for backwards compatibility, but the internal Go code does not use the `edge` prefix per the V4 SDK naming convention.

---

## Registration in provider.go

### Data Sources

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // ... other data sources
        dataSourceAzionFirewallRulesEngine,
        dataSourceAzionFirewallRuleEngine,
        // ... other data sources
    }
}
```

### Resources

```go
func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... other resources
        NewFirewallRuleEngineResource,
        // ... other resources
    }
}
```

---

## Documentation Files

- Singular Data Source: `docs/data-sources/firewall_rule_engine.md`
- Plural Data Source: `docs/data-sources/firewall_rules_engine.md`
- Resource: `docs/resources/firewall_rule_engine.md`

---

## Example Files

- Singular Data Source: `examples/data-sources/azion_firewall_rule_engine/data-source.tf`
- Plural Data Source: `examples/data-sources/azion_firewall_rules_engine/data-source.tf`
- Resource: `examples/resources/azion_firewall_rule_engine/resource.tf`
- Import Script: `examples/resources/azion_firewall_rule_engine/import.sh`

---

## Summary Checklist

When implementing firewall rules engine data sources and resources:

### Data Sources
1. ✅ Use `azion-api` SDK import path
2. ✅ Access API via `client.api.FirewallsRulesEngineAPI`
3. ✅ Use `firewall_id` instead of `edge_application_id` (no `edge` prefix internally)
4. ✅ No phase parameter needed (only request phase exists)
5. ✅ Handle polymorphic behavior types correctly
6. ✅ Transform criterion arguments using `c.Argument.Get()`
7. ✅ Handle 429 errors with `utils.RetryOn429`
8. ✅ Register data sources in `provider.go`
9. ✅ Create documentation and example files

### Resource
1. ✅ Use `azion-api` SDK import path
2. ✅ Access API via `client.api.FirewallsRulesEngineAPI`
3. ✅ Implement Create, Read, Update, Delete, and ImportState methods
4. ✅ Build criteria using `buildFirewallCriteriaRequest()` helper
5. ✅ Build behaviors using `buildFirewallBehaviorsRequest()` helper
6. ✅ Handle all behavior types: `run_function`, `set_custom_response`, `set_waf`, `set_rate_limit`, `drop`
7. ✅ Transform responses using `transformFirewallRuleToResultModel()` helper
8. ✅ Handle 404 in Read by calling `resp.State.RemoveResource(ctx)`
9. ✅ Handle 429 errors with `utils.RetryOn429`
10. ✅ Use ID format: `{firewall_id}/{rule_id}` for import and state
11. ✅ Register resource in `provider.go` as `NewFirewallRuleEngineResource`
12. ✅ Create documentation and example files
13. ✅ Create import.sh with format documentation

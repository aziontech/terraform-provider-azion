# WAF Rule Sets (Exceptions) - Code Generation Guide

This document provides comprehensive guidance for AI agents generating Terraform provider code for WAF Rule Sets (called "WAF Exceptions" in the V4 SDK) from the Azion API.

## Overview

In the Azion API V4 SDK, WAF Rule Sets are referred to as **WAF Exceptions**. These are rules that define exceptions to the WAF's normal behavior, allowing specific traffic patterns to bypass certain WAF rules.

### Naming Convention

| Terraform Resource | V4 SDK Name | API Endpoint |
|-------------------|-------------|--------------|
| `azion_waf_rule_set` | `WAFRule` | `/v4/wafs/{waf_id}/exceptions/{exception_id}` |
| `azion_waf_rule_sets` | `PaginatedWAFRuleList` | `/v4/wafs/{waf_id}/exceptions` |

**Important**: The "edge" prefix is no longer used in V4 SDK naming. Use `api` instead of `edgeApi` for the client.

---

## SDK Import

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

## API Client Access

The API client is accessed via the `api` field of the `apiClient` struct:

```go
// Create exception
o.client.api.WAFsExceptionsAPI.CreateWafException(ctx, wafId).WAFRuleRequest(request).Execute()

// Retrieve exception
o.client.api.WAFsExceptionsAPI.RetrieveWafException(ctx, exceptionId, wafId).Execute()

// Update exception
o.client.api.WAFsExceptionsAPI.UpdateWafException(ctx, exceptionId, wafId).WAFRuleRequest(request).Execute()

// Delete exception
o.client.api.WAFsExceptionsAPI.DeleteWafException(ctx, exceptionId, wafId).Execute()

// List exceptions
o.client.api.WAFsExceptionsAPI.ListWafExceptions(ctx, wafId).Page(page).PageSize(pageSize).Execute()
```

---

## SDK Structures

### WAFRule (WAF Exception)

```go
type WAFRule struct {
    Id           int64                       `json:"id"`
    RuleId       *int64                      `json:"rule_id,omitempty"`       // 0 = all rules
    Name         string                      `json:"name"`
    Path         NullableString              `json:"path,omitempty"`
    Conditions   []WAFExceptionCondition     `json:"conditions"`
    Operator     *string                     `json:"operator,omitempty"`      // "regex" or "contains"
    Active       *bool                       `json:"active,omitempty"`
    LastEditor   string                      `json:"last_editor"`
    LastModified time.Time                   `json:"last_modified"`
}
```

### WAFRuleRequest (for Create/Update)

```go
type WAFRuleRequest struct {
    RuleId     *int64                             `json:"rule_id,omitempty"`
    Name       string                             `json:"name"`
    Path       NullableString                     `json:"path,omitempty"`
    Conditions []WAFExceptionConditionRequest     `json:"conditions"`
    Operator   *string                            `json:"operator,omitempty"`
    Active     *bool                              `json:"active,omitempty"`
}
```

### WAFRuleResponse

```go
type WAFRuleResponse struct {
    State *string   `json:"state,omitempty"`
    Data  WAFRule   `json:"data"`
}
```

### WAFExceptionCondition (Polymorphic - for Read)

The `WAFExceptionCondition` is a polymorphic type that can be one of three types:

```go
type WAFExceptionCondition struct {
    WAFExceptionGenericCondition            *WAFExceptionGenericCondition
    WAFExceptionSpecificConditionOnName     *WAFExceptionSpecificConditionOnName
    WAFExceptionSpecificConditionOnValue    *WAFExceptionSpecificConditionOnValue
}
```

### WAFExceptionConditionRequest (Polymorphic - for Create/Update)

```go
type WAFExceptionConditionRequest struct {
    WAFExceptionGenericConditionRequest            *WAFExceptionGenericConditionRequest
    WAFExceptionSpecificConditionOnNameRequest     *WAFExceptionSpecificConditionOnNameRequest
    WAFExceptionSpecificConditionOnValueRequest    *WAFExceptionSpecificConditionOnValueRequest
}
```

### Condition Types

#### 1. WAFExceptionGenericCondition

Used for generic match conditions:

```go
type WAFExceptionGenericCondition struct {
    Match string `json:"match"`  // Match type (see values below)
}
```

**Valid Match Values:**
- `any_http_header_name` - Any HTTP header name
- `any_http_header_value` - Any HTTP header value
- `any_query_string_name` - Any query string parameter name
- `any_query_string_value` - Any query string parameter value
- `any_url` - Any URL
- `body_form_field_name` - Body form field name
- `body_form_field_value` - Body form field value
- `file_extension` - File extension
- `raw_body` - Raw request body

#### 2. WAFExceptionSpecificConditionOnName

Used for specific name-based conditions:

```go
type WAFExceptionSpecificConditionOnName struct {
    Match string `json:"match"`  // Match type
    Name  string `json:"name"`   // Specific name to match
}
```

**Valid Match Values:**
- `specific_body_form_field_name` - Specific body form field name
- `specific_http_header_name` - Specific HTTP header name
- `specific_query_string_name` - Specific query string name

#### 3. WAFExceptionSpecificConditionOnValue

Used for specific value-based conditions:

```go
type WAFExceptionSpecificConditionOnValue struct {
    Match string `json:"match"`  // Match type
    Value string `json:"value"`  // Specific value to match
}
```

**Valid Match Values:**
- `specific_body_form_field_value` - Specific body form field value
- `specific_http_header_value` - Specific HTTP header value
- `specific_query_string_value` - Specific query string value

---

## Terraform Model Structure

### Resource Model

```go
type WafRuleSetResourceModel struct {
    ID          types.String               `tfsdk:"id"`
    WafID       types.Int64                `tfsdk:"waf_id"`
    LastUpdated types.String               `tfsdk:"last_updated"`
    Result      *WafRuleSetResourceResults `tfsdk:"result"`
}

type WafRuleSetResourceResults struct {
    ID           types.Int64                  `tfsdk:"exception_id"`
    RuleID       types.Int64                  `tfsdk:"rule_id"`
    Name         types.String                 `tfsdk:"name"`
    Path         types.String                 `tfsdk:"path"`
    Conditions   []WafExceptionConditionModel `tfsdk:"conditions"`
    Operator     types.String                 `tfsdk:"operator"`
    Active       types.Bool                   `tfsdk:"active"`
    LastEditor   types.String                 `tfsdk:"last_editor"`
    LastModified types.String                 `tfsdk:"last_modified"`
}
```

### Condition Model (Flattened for Terraform)

```go
type WafExceptionConditionModel struct {
    Match         types.String `tfsdk:"match"`
    Name          types.String `tfsdk:"name"`           // Only for specific_on_name
    Value         types.String `tfsdk:"value"`          // Only for specific_on_value
    ConditionType types.String `tfsdk:"condition_type"` // "generic", "specific_on_name", "specific_on_value"
}
```

### Data Source Model (Single)

```go
type WafRuleSetDataSourceModel struct {
    ID          types.String               `tfsdk:"id"`
    WafID       types.Int64                `tfsdk:"waf_id"`
    ExceptionID types.Int64                `tfsdk:"exception_id"`
    Results     *WafRuleSetResultDataModel `tfsdk:"results"`
}

type WafRuleSetResultDataModel struct {
    ID           types.Int64                  `tfsdk:"id"`
    RuleID       types.Int64                  `tfsdk:"rule_id"`
    Name         types.String                 `tfsdk:"name"`
    Path         types.String                 `tfsdk:"path"`
    Conditions   []WafExceptionConditionModel `tfsdk:"conditions"`
    Operator     types.String                 `tfsdk:"operator"`
    Active       types.Bool                   `tfsdk:"active"`
    LastEditor   types.String                 `tfsdk:"last_editor"`
    LastModified types.String                 `tfsdk:"last_modified"`
}
```

---

## Transforming Polymorphic Conditions

### From SDK to Terraform (for Resources)

```go
// transformWAFExceptionConditionsForResource transforms SDK conditions to Terraform models for resources.
func transformWAFExceptionConditionsForResource(conditions []azionapi.WAFExceptionCondition) []WafExceptionConditionModel {
    var result []WafExceptionConditionModel

    for _, cond := range conditions {
        actualInstance := cond.GetActualInstance()
        if actualInstance == nil {
            continue
        }

        model := WafExceptionConditionModel{}

        switch c := actualInstance.(type) {
        case *azionapi.WAFExceptionGenericCondition:
            model.Match = types.StringValue(c.GetMatch())
            model.Name = types.StringNull()
            model.Value = types.StringNull()
            model.ConditionType = types.StringValue("generic")

        case *azionapi.WAFExceptionSpecificConditionOnName:
            model.Match = types.StringValue(c.GetMatch())
            model.Name = types.StringValue(c.GetName())
            model.Value = types.StringNull()
            model.ConditionType = types.StringValue("specific_on_name")

        case *azionapi.WAFExceptionSpecificConditionOnValue:
            model.Match = types.StringValue(c.GetMatch())
            model.Name = types.StringNull()
            model.Value = types.StringValue(c.GetValue())
            model.ConditionType = types.StringValue("specific_on_value")
        }

        result = append(result, model)
    }

    return result
}
```

### From SDK to Terraform (for Data Sources)

```go
// transformWAFExceptionConditions transforms SDK conditions to Terraform models.
func transformWAFExceptionConditions(conditions []azionapi.WAFExceptionCondition) []WafExceptionConditionModel {
    var result []WafExceptionConditionModel

    for _, cond := range conditions {
        actualInstance := cond.GetActualInstance()
        if actualInstance == nil {
            continue
        }

        model := WafExceptionConditionModel{}

        switch c := actualInstance.(type) {
        case *azionapi.WAFExceptionGenericCondition:
            model.Match = types.StringValue(c.GetMatch())
            model.Name = types.StringNull()
            model.Value = types.StringNull()
            model.ConditionType = types.StringValue("generic")

        case *azionapi.WAFExceptionSpecificConditionOnName:
            model.Match = types.StringValue(c.GetMatch())
            model.Name = types.StringValue(c.GetName())
            model.Value = types.StringNull()
            model.ConditionType = types.StringValue("specific_on_name")

        case *azionapi.WAFExceptionSpecificConditionOnValue:
            model.Match = types.StringValue(c.GetMatch())
            model.Name = types.StringNull()
            model.Value = types.StringValue(c.GetValue())
            model.ConditionType = types.StringValue("specific_on_value")
        }

        result = append(result, model)
    }

    return result
}
```

### From Terraform to SDK (For Create/Update)

```go
// buildWAFExceptionConditionsRequest builds SDK conditions from Terraform models.
func buildWAFExceptionConditionsRequest(conditions []WafExceptionConditionModel) []azionapi.WAFExceptionConditionRequest {
    var result []azionapi.WAFExceptionConditionRequest

    for _, c := range conditions {
        switch c.ConditionType.ValueString() {
        case "generic":
            generic := azionapi.NewWAFExceptionGenericConditionRequest(c.Match.ValueString())
            result = append(result, azionapi.WAFExceptionGenericConditionRequestAsWAFExceptionConditionRequest(generic))

        case "specific_on_name":
            specificName := azionapi.NewWAFExceptionSpecificConditionOnNameRequest(
                c.Match.ValueString(),
                c.Name.ValueString(),
            )
            result = append(result, azionapi.WAFExceptionSpecificConditionOnNameRequestAsWAFExceptionConditionRequest(specificName))

        case "specific_on_value":
            specificValue := azionapi.NewWAFExceptionSpecificConditionOnValueRequest(
                c.Match.ValueString(),
                c.Value.ValueString(),
            )
            result = append(result, azionapi.WAFExceptionSpecificConditionOnValueRequestAsWAFExceptionConditionRequest(specificValue))
        }
    }

    return result
}
```

---

## Schema Definition

### Resource Schema

```go
resp.Schema = schema.Schema{
    Attributes: map[string]schema.Attribute{
        "id": schema.StringAttribute{
            Description: "Identifier of the resource.",
            Computed:    true,
            PlanModifiers: []planmodifier.String{
                stringplanmodifier.UseStateForUnknown(),
            },
        },
        "waf_id": schema.Int64Attribute{
            Description: "The WAF identifier.",
            Required:    true,
        },
        "last_updated": schema.StringAttribute{
            Description: "Timestamp of the last Terraform update of the resource.",
            Computed:    true,
        },
        "result": schema.SingleNestedAttribute{
            Required: true,
            Attributes: map[string]schema.Attribute{
                "exception_id": schema.Int64Attribute{
                    Description: "The ID of the WAF exception.",
                    Computed:    true,
                },
                "rule_id": schema.Int64Attribute{
                    Description: "The rule ID that this exception applies to. 0 means all rules.",
                    Optional:    true,
                },
                "name": schema.StringAttribute{
                    Description: "Name of the WAF exception.",
                    Required:    true,
                },
                "path": schema.StringAttribute{
                    Description: "Path pattern for the exception.",
                    Optional:    true,
                },
                "conditions": schema.ListNestedAttribute{
                    Description: "Conditions for the WAF exception.",
                    Required:    true,
                    NestedObject: schema.NestedAttributeObject{
                        Attributes: map[string]schema.Attribute{
                            "match": schema.StringAttribute{
                                Description: "The match type for the condition.",
                                Required:    true,
                            },
                            "name": schema.StringAttribute{
                                Description: "The name for specific condition on name.",
                                Optional:    true,
                            },
                            "value": schema.StringAttribute{
                                Description: "The value for specific condition on value.",
                                Optional:    true,
                            },
                            "condition_type": schema.StringAttribute{
                                Description: "Type of condition: generic, specific_on_name, or specific_on_value.",
                                Required:    true,
                            },
                        },
                    },
                },
                "operator": schema.StringAttribute{
                    Description: "The operator for the exception (regex or contains).",
                    Optional:    true,
                },
                "active": schema.BoolAttribute{
                    Description: "Whether the exception is active.",
                    Optional:    true,
                },
                "last_editor": schema.StringAttribute{
                    Description: "Last editor of the exception.",
                    Computed:    true,
                },
                "last_modified": schema.StringAttribute{
                    Description: "Last modified timestamp.",
                    Computed:    true,
                },
            },
        },
    },
}
```

### Single Data Source Schema

```go
resp.Schema = schema.Schema{
    Attributes: map[string]schema.Attribute{
        "id": schema.StringAttribute{
            Description: "Identifier of the data source.",
            Computed:    true,
        },
        "waf_id": schema.Int64Attribute{
            Description: "The WAF identifier.",
            Required:    true,
        },
        "exception_id": schema.Int64Attribute{
            Description: "The WAF exception (rule set) identifier.",
            Required:    true,
        },
        "results": schema.SingleNestedAttribute{
            Computed: true,
            Attributes: map[string]schema.Attribute{
                "id": schema.Int64Attribute{
                    Description: "The ID of the WAF exception.",
                    Computed:    true,
                },
                "rule_id": schema.Int64Attribute{
                    Description: "The rule ID that this exception applies to. 0 means all rules.",
                    Computed:    true,
                },
                "name": schema.StringAttribute{
                    Description: "Name of the WAF exception.",
                    Computed:    true,
                },
                "path": schema.StringAttribute{
                    Description: "Path pattern for the exception.",
                    Computed:    true,
                },
                "conditions": schema.ListNestedAttribute{
                    Description: "Conditions for the WAF exception.",
                    Computed:    true,
                    NestedObject: schema.NestedAttributeObject{
                        Attributes: map[string]schema.Attribute{
                            "match": schema.StringAttribute{
                                Description: "The match type for the condition.",
                                Computed:    true,
                            },
                            "name": schema.StringAttribute{
                                Description: "The name for specific condition on name.",
                                Computed:    true,
                            },
                            "value": schema.StringAttribute{
                                Description: "The value for specific condition on value.",
                                Computed:    true,
                            },
                            "condition_type": schema.StringAttribute{
                                Description: "Type of condition: generic, specific_on_name, or specific_on_value.",
                                Computed:    true,
                            },
                        },
                    },
                },
                "operator": schema.StringAttribute{
                    Description: "The operator for the exception (regex or contains).",
                    Computed:    true,
                },
                "active": schema.BoolAttribute{
                    Description: "Whether the exception is active.",
                    Computed:    true,
                },
                "last_editor": schema.StringAttribute{
                    Description: "Last editor of the exception.",
                    Computed:    true,
                },
                "last_modified": schema.StringAttribute{
                    Description: "Last modified timestamp.",
                    Computed:    true,
                },
            },
        },
    },
}
```

### List Data Source Schema

```go
resp.Schema = schema.Schema{
    Attributes: map[string]schema.Attribute{
        "id": schema.StringAttribute{
            Description: "Identifier of the data source.",
            Computed:    true,
        },
        "waf_id": schema.Int64Attribute{
            Description: "The WAF identifier.",
            Required:    true,
        },
        "counter": schema.Int64Attribute{
            Description: "The total number of WAF exceptions.",
            Computed:    true,
        },
        "page": schema.Int64Attribute{
            Description: "The page number.",
            Optional:    true,
        },
        "page_size": schema.Int64Attribute{
            Description: "The page size number.",
            Optional:    true,
        },
        "total_pages": schema.Int64Attribute{
            Description: "The total number of pages.",
            Computed:    true,
        },
        "links": schema.SingleNestedAttribute{
            Computed: true,
            Attributes: map[string]schema.Attribute{
                "previous": schema.StringAttribute{Computed: true},
                "next":     schema.StringAttribute{Computed: true},
            },
        },
        "results": schema.ListNestedAttribute{
            Computed: true,
            NestedObject: schema.NestedAttributeObject{
                // Same as single data source results
            },
        },
    },
}
```

---

## Create Operation Pattern

```go
func (r *wafRuleSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan WafRuleSetResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the conditions request.
    conditions := buildWAFExceptionConditionsRequest(plan.Result.Conditions)

    // Build the WAF exception request.
    wafRuleRequest := azionapi.NewWAFRuleRequest(plan.Result.Name.ValueString(), conditions)

    // Set optional fields.
    if !plan.Result.RuleID.IsNull() && !plan.Result.RuleID.IsUnknown() {
        wafRuleRequest.SetRuleId(plan.Result.RuleID.ValueInt64())
    }

    if !plan.Result.Path.IsNull() && !plan.Result.Path.IsUnknown() {
        wafRuleRequest.SetPath(plan.Result.Path.ValueString())
    }

    if !plan.Result.Operator.IsNull() && !plan.Result.Operator.IsUnknown() {
        wafRuleRequest.SetOperator(plan.Result.Operator.ValueString())
    }

    if !plan.Result.Active.IsNull() && !plan.Result.Active.IsUnknown() {
        wafRuleRequest.SetActive(plan.Result.Active.ValueBool())
    }

    // Create the WAF exception.
    exceptionResponse, response, err := r.client.api.WAFsExceptionsAPI.CreateWafException(ctx, plan.WafID.ValueInt64()).WAFRuleRequest(*wafRuleRequest).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            exceptionResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFRuleResponse, *http.Response, error) {
                return r.client.api.WAFsExceptionsAPI.CreateWafException(ctx, plan.WafID.ValueInt64()).WAFRuleRequest(*wafRuleRequest).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Transform the response to the model.
    data := exceptionResponse.GetData()
    plan.Result = transformWAFRuleToResourceModel(data)
    plan.ID = types.StringValue(strconv.FormatInt(data.GetId(), 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

---

## Read Operation Pattern

### Single Data Source

```go
func (o *WafRuleSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var wafID, exceptionID types.Int64

    diagsWafID := req.Config.GetAttribute(ctx, path.Root("waf_id"), &wafID)
    resp.Diagnostics.Append(diagsWafID...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsExceptionID := req.Config.GetAttribute(ctx, path.Root("exception_id"), &exceptionID)
    resp.Diagnostics.Append(diagsExceptionID...)
    if resp.Diagnostics.HasError() {
        return
    }

    exceptionResponse, response, err := o.client.api.WAFsExceptionsAPI.RetrieveWafException(ctx, exceptionID.ValueInt64(), wafID.ValueInt64()).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            exceptionResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFRuleResponse, *http.Response, error) {
                return o.client.api.WAFsExceptionsAPI.RetrieveWafException(ctx, exceptionID.ValueInt64(), wafID.ValueInt64()).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    results := transformWAFRuleToResultModel(exceptionResponse.GetData())

    state := WafRuleSetDataSourceModel{
        ID:          types.StringValue("Get WAF Rule Set By ID"),
        WafID:       wafID,
        ExceptionID: exceptionID,
        Results:     results,
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Resource Read

```go
func (r *wafRuleSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state WafRuleSetResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var exceptionID int64
    var err error
    if state.ID.IsNull() {
        exceptionID = state.Result.ID.ValueInt64()
    } else {
        exceptionID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error", "Could not convert WAF Rule Set ID")
            return
        }
    }

    exceptionResponse, response, err := r.client.api.WAFsExceptionsAPI.RetrieveWafException(ctx, exceptionID, state.WafID.ValueInt64()).Execute()
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // Handle 429 and other errors...
    }

    if response != nil {
        defer response.Body.Close()
    }

    data := exceptionResponse.GetData()
    state.Result = transformWAFRuleToResourceModel(data)
    state.ID = types.StringValue(strconv.FormatInt(exceptionID, 10))

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### List Data Source

```go
func (o *WafRuleSetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var wafID, page, pageSize types.Int64

    // Get attributes...

    if page.IsNull() || page.IsUnknown() || page.ValueInt64() == 0 {
        page = types.Int64Value(1)
    }
    if pageSize.IsNull() || pageSize.IsUnknown() || pageSize.ValueInt64() == 0 {
        pageSize = types.Int64Value(10)
    }

    listResponse, response, err := o.client.api.WAFsExceptionsAPI.ListWafExceptions(ctx, wafID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
    // Handle errors similar to single data source...

    // Transform results
    var results []WafRuleSetListItemDataModel
    for _, rule := range listResponse.GetResults() {
        results = append(results, transformWAFRuleToListItemModel(rule))
    }

    // Build state with pagination info...
}
```

---

## Update Operation Pattern

```go
func (r *wafRuleSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan WafRuleSetResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state WafRuleSetResourceModel
    diagsState := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diagsState...)
    if resp.Diagnostics.HasError() {
        return
    }

    var exceptionID int64
    var err error
    if state.ID.IsNull() {
        exceptionID = state.Result.ID.ValueInt64()
    } else {
        exceptionID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error", "Could not convert WAF Rule Set ID")
            return
        }
    }

    // Build the conditions request.
    conditions := buildWAFExceptionConditionsRequest(plan.Result.Conditions)

    // Build the WAF exception request.
    wafRuleRequest := azionapi.NewWAFRuleRequest(plan.Result.Name.ValueString(), conditions)

    // Set optional fields (same as Create)...

    // Update the WAF exception.
    exceptionResponse, response, err := r.client.api.WAFsExceptionsAPI.UpdateWafException(ctx, exceptionID, plan.WafID.ValueInt64()).WAFRuleRequest(*wafRuleRequest).Execute()
    if err != nil {
        // Handle 429 and other errors...
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Transform the response to the model.
    data := exceptionResponse.GetData()
    plan.Result = transformWAFRuleToResourceModel(data)
    plan.ID = types.StringValue(strconv.FormatInt(exceptionID, 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

---

## Delete Operation Pattern

```go
func (r *wafRuleSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state WafRuleSetResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var exceptionID int64
    var err error
    if state.ID.IsNull() {
        exceptionID = state.Result.ID.ValueInt64()
    } else {
        exceptionID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error", "Could not convert WAF Rule Set ID")
            return
        }
    }

    deleteResponse, response, err := r.client.api.WAFsExceptionsAPI.DeleteWafException(ctx, exceptionID, state.WafID.ValueInt64()).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
                delResp, resp, err := r.client.api.WAFsExceptionsAPI.DeleteWafException(ctx, exceptionID, state.WafID.ValueInt64()).Execute()
                _ = delResp // Ignore the delete response in retry.
                return resp, err
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    _ = deleteResponse // Avoid unused variable error.
}
```

---

## Rule ID Meanings

The `rule_id` field in WAFRule indicates which WAF rule this exception applies to:

| Rule ID | Meaning |
|---------|---------|
| 0 | Applies to **all rules** |
| 1-18 | Protocol validation rules |
| 1000-1017 | SQL Injection rules |
| 1100-1110 | Remote File Inclusion (RFI) rules |
| 1198-1199 | RCE/Log4Shell rules |
| 1200-1210 | Directory Traversal rules |
| 1302-1315 | Cross-Site Scripting (XSS) rules |
| 1400-1402 | Evasion tricks rules |
| 1500 | File Upload rules |
| 2001 | CVE-2022-22965 (Spring4Shell) rule |

---

## Error Handling

Always handle:
1. **429 Rate Limiting** - Use `utils.RetryOn429` with max 5 retries
2. **Response Body Closure** - Always `defer response.Body.Close()` after successful response
3. **Error Body Reading** - Read error details from response body
4. **404 Not Found** - For Read operations, remove resource from state

```go
if response != nil {
    defer response.Body.Close()
}
```

---

## Import State

```go
func (r *wafRuleSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Note**: The import uses just the `exception_id`. The `waf_id` must be provided in the Terraform configuration for the imported resource to be properly managed.

---

## Documentation and Examples

### MANDATORY: Parent Resource Documentation

**IMPORTANT**: WAF Rule Sets is a child resource of `azion_waf`. Documentation and examples MUST include the parent resource creation to show complete context.

When updating documentation, always include:

1. **Parent WAF Example** - Show creation of the parent WAF first
2. **Reference Using Terraform Interpolation** - Use `azion_waf.example.id` to reference the parent ID

### Documentation Files

Documentation is auto-generated by `terraform-plugin-docs` and located in:

| Type | Location |
|------|----------|
| Singular Data Source Doc | `docs/data-sources/waf_rule_set.md` |
| Plural Data Source Doc | `docs/data-sources/waf_rule_sets.md` |
| Resource Doc | `docs/resources/waf_rule_set.md` |

### Example Files

Example Terraform configurations are located in:

| Type | Location |
|------|----------|
| Singular Data Source Example | `examples/data-sources/azion_waf_rule_set/data-source.tf` |
| Plural Data Source Example | `examples/data-sources/azion_waf_rule_sets/data-source.tf` |
| Resource Example | `examples/resources/azion_waf_rule_set/resource.tf` |

### Example: Complete Resource Usage with Parent WAF

```terraform
# First, create the parent WAF
resource "azion_waf" "example" {
  result = {
    name   = "My WAF"
    active = true
  }
}

# Then create the rule set for that WAF
resource "azion_waf_rule_set" "example" {
  waf_id = azion_waf.example.id
  result = {
    name     = "My WAF Exception"
    path     = "/api/*"
    active   = true
    operator = "regex"
    rule_id  = 0
    conditions = [
      {
        condition = {
          match          = "any_url"
          condition_type = "generic"
        }
      }
    ]
  }
}
```

---

## Summary Checklist

When implementing WAF Rule Sets (Exceptions):

- [ ] Use `azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"` import
- [ ] Access API via `client.api.WAFsExceptionsAPI`
- [ ] Handle polymorphic `WAFExceptionCondition` using `GetActualInstance()`
- [ ] Use `WAFExceptionConditionRequest` types for Create/Update operations
- [ ] Use helper functions `WAFExceptionGenericConditionRequestAsWAFExceptionConditionRequest`, etc.
- [ ] Flatten condition types with `condition_type` field
- [ ] Handle optional fields (`rule_id`, `path`, `operator`, `active`) with null checks
- [ ] Implement 429 retry logic using `utils.RetryOn429`
- [ ] Close response body with `defer response.Body.Close()`
- [ ] Format timestamps as RFC3339
- [ ] Register data sources and resources in `internal/provider.go`

---

## Related Files

- [`internal/data_source_waf_rule_set.go`](../internal/data_source_waf_rule_set.go) - Single exception data source
- [`internal/data_source_waf_rule_sets.go`](../internal/data_source_waf_rule_sets.go) - List exceptions data source
- [`internal/resource_waf_rule_set.go`](../internal/resource_waf_rule_set.go) - Exception resource (CRUD)
- [`docs/data-sources/waf_rule_set.md`](../docs/data-sources/waf_rule_set.md) - Documentation
- [`docs/data-sources/waf_rule_sets.md`](../docs/data-sources/waf_rule_sets.md) - Documentation
- [`docs/resources/waf_rule_set.md`](../docs/resources/waf_rule_set.md) - Resource documentation

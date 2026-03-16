# Edge Application Rules Engine - Code Generation Guide

This document provides comprehensive guidance for AI agents generating Terraform provider code for Edge Application Rules Engine from the OpenAPI specification (V4 SDK).

---

## Table of Contents

1. [Overview](#overview)
2. [API Structure](#api-structure)
3. [Data Structures](#data-structures)
4. [Data Source - Single Rule](#data-source---single-rule)
5. [Data Source - List Rules](#data-source---list-rules)
6. [Resource - CRUD Operations](#resource---crud-operations)
7. [Error Handling](#error-handling)
8. [Type Conversions](#type-conversions)
9. [Provider Registration](#provider-registration)

---

## Overview

The Rules Engine in V4 API is split into two separate APIs:
- **ApplicationsRequestRulesAPI** - for rules in the request phase
- **ApplicationsResponseRulesAPI** - for rules in the response phase

Each API provides full CRUD operations for managing rules within an Edge Application.

### Key Differences from V3

| V3 (Legacy) | V4 (Current) |
|-------------|--------------|
| Single `EdgeApplicationsRulesEngineAPI` | Separate `ApplicationsRequestRulesAPI` and `ApplicationsResponseRulesAPI` |
| Phase passed as URL parameter | Phase determined by API choice |
| `RulesEngineIdResponse` | `RequestPhaseRuleResponse` / `ResponsePhaseRuleResponse` |
| `RulesEngineBehaviorString` / `RulesEngineBehaviorObject` | Polymorphic `RequestPhaseBehavior` |

---

## API Structure

### Base Paths

```
# Request Phase Rules
/workspace/applications/{application_id}/request_rules

# Response Phase Rules
/workspace/applications/{application_id}/response_rules
```

### API Methods

```go
// V4 SDK Client Access
client.azionapi.ApplicationsRequestRulesAPI
client.azionapi.ApplicationsResponseRulesAPI

// Request Phase Rules Methods
CreateApplicationRequestRule(ctx, applicationId)
DeleteApplicationRequestRule(ctx, applicationId, requestRuleId)
ListApplicationRequestRules(ctx, applicationId)
PartialUpdateApplicationRequestRule(ctx, applicationId, requestRuleId)
RetrieveApplicationRequestRule(ctx, applicationId, requestRuleId)
UpdateApplicationRequestRule(ctx, applicationId, requestRuleId)
UpdateApplicationRequestRulesOrder(ctx, applicationId)

// Response Phase Rules Methods
CreateApplicationResponseRule(ctx, applicationId)
DeleteApplicationResponseRule(ctx, applicationId, responseRuleId)
ListApplicationResponseRules(ctx, applicationId)
PartialUpdateApplicationResponseRule(ctx, applicationId, responseRuleId)
RetrieveApplicationResponseRule(ctx, applicationId, responseRuleId)
UpdateApplicationResponseRule(ctx, applicationId, responseRuleId)
UpdateApplicationResponseRulesOrder(ctx, applicationId)
```

---

## Data Structures

### Request Phase Rule

```go
// Request Phase Rule Request (for Create/Update)
type RequestPhaseRuleRequest struct {
    Name        string                                  `json:"name"`
    Active      *bool                                   `json:"active,omitempty"`
    Criteria    [][]EdgeApplicationCriterionFieldRequest `json:"criteria"`
    Behaviors   []RequestPhaseBehaviorRequest           `json:"behaviors"`
    Description *string                                 `json:"description,omitempty"`
}

// Request Phase Rule Response
type RequestPhaseRule struct {
    Id           int64                              `json:"id"`
    Name         string                             `json:"name"`
    Active       *bool                              `json:"active,omitempty"`
    Criteria     [][]EdgeApplicationCriterionField  `json:"criteria"`
    Behaviors    []RequestPhaseBehavior             `json:"behaviors"`
    Description  *string                            `json:"description,omitempty"`
    Order        int64                              `json:"order"`
    LastEditor   NullableString                     `json:"last_editor"`
    LastModified NullableTime                       `json:"last_modified"`
}
```

### Criterion Field

```go
// For Requests
type ApplicationCriterionFieldRequest struct {
    Conditional string `json:"conditional"` // "if", "and", "or"
    Variable    string `json:"variable"`
    Operator    string `json:"operator"`
    Argument    ApplicationCriterionArgumentRequest `json:"argument,omitempty"`
}

// For Responses
type ApplicationCriterionField struct {
    Conditional string `json:"conditional"`
    Variable    string `json:"variable"`
    Operator    string `json:"operator"`
    Argument    NullableApplicationCriterionArgument `json:"argument,omitempty"`
}
```

### Criterion Argument (Polymorphic)

The `ApplicationCriterionArgument` is a polymorphic type that can hold either a string or an integer:

```go
type ApplicationCriterionArgument struct {
    String *string
    Int64  *int64
}
```

**IMPORTANT**: When extracting the argument value from API responses, you MUST check both fields. Using `fmt.Sprintf("%v", arg.Get())` will print the struct's pointer address instead of the actual value.

**Correct pattern for extracting argument values:**

```go
func getCriterionArgumentValue(arg azionapi.NullableApplicationCriterionArgument) string {
    if !arg.IsSet() {
        return ""
    }
    argValue := arg.Get()
    if argValue == nil {
        return ""
    }
    if argValue.String != nil {
        return *argValue.String
    }
    if argValue.Int64 != nil {
        return fmt.Sprintf("%d", *argValue.Int64)
    }
    return ""
}
```

### Behavior Types (Polymorphic)

The `RequestPhaseBehavior` is a polymorphic type that can be one of:

#### 1. BehaviorArgs - Behaviors with string/integer argument

```go
type BehaviorArgs struct {
    Type       string               `json:"type"`
    Attributes BehaviorArgsAttributes `json:"attributes"`
}

type BehaviorArgsAttributes struct {
    Value BehaviorArgsAttributesValue `json:"value"` // Can be string or integer
}
```

Common behavior types using args:
- `add_request_header` - Add request header (e.g., "X-Custom: value")
- `add_request_cookie` - Add request cookie
- `filter_request_header` - Filter request header
- `filter_request_cookie` - Filter request cookie
- `redirect_to` - Redirect URL
- `rewrite_request` - Rewrite request
- `run_function` - Execute edge function
- `set_cache_policy` - Set cache policy
- `set_origin` - Set origin (value is origin name)

**NOTE:** Behavior types differ between Request and Response phases. Always verify the correct behavior type for the phase being used.

#### 2. BehaviorCapture - Capture match groups

```go
type BehaviorCapture struct {
    Type       string `json:"type"`
    Attributes BehaviorCaptureMatchGroupsAttributes `json:"attributes"`
}

type BehaviorCaptureMatchGroupsAttributes struct {
    Subject       string `json:"subject"`
    Regex         string `json:"regex"`
    CapturedArray string `json:"captured_array"`
}
```

Behavior type: `capture_match_groups`

#### 3. BehaviorNoArgs - Behaviors without arguments

```go
type BehaviorNoArgs struct {
    Type string `json:"type"`
}
```

Common behavior types without args:
- `deny` - Deny request
- `no_content` - Return no content
- `deliver` - Deliver response
- `finish_request_phase` - Finish request phase
- `forward_cookies` - Forward cookies
- `optimize_images` - Optimize images
- `bypass_cache` - Bypass cache
- `enable_gzip` - Enable gzip
- `redirect_http_to_https` - Redirect HTTP to HTTPS

### Available Variables for Criteria

| Variable | Description | Phase |
|----------|-------------|-------|
| `${arg_<name>}` | Query param | default, request, response |
| `${args}` | All query params | default, request, response |
| `${cookie_<name>}` | Cookie value | default, request, response |
| `${device_group}` | Device group | default, request, response |
| `${geoip_city_continent_code}` | GeoIP continent | default, request, response |
| `${geoip_city_country_code}` | GeoIP country code | default, request, response |
| `${geoip_city}` | GeoIP city | default, request, response |
| `${host}` | Host header | default, request, response |
| `${domain}` | Domain | default, request, response |
| `${http_<header_name>}` | HTTP header | default, request, response |
| `${remote_addr}` | Client IP | default, request, response |
| `${request_method}` | HTTP method | default, request, response |
| `${request_uri}` | Full request URI | default, request, response |
| `${scheme}` | URL scheme | default, request, response |
| `${status}` | Response status | response only |
| `${upstream_addr}` | Origin address | response only |
| `${uri}` | Normalized URI | default, request, response |

### Available Operators for Criteria

| Operator | Description | Requires Argument |
|----------|-------------|-------------------|
| `is_equal` | Exact match | Yes |
| `is_not_equal` | Not equal | Yes |
| `starts_with` | Starts with | Yes |
| `does_not_start_with` | Does not start with | Yes |
| `matches` | Regex match | Yes |
| `does_not_match` | Does not match regex | Yes |
| `exists` | Variable exists | **No** (use `types.StringNull()`) |
| `does_not_exist` | Variable does not exist | **No** (use `types.StringNull()`) |
| `is_in_list` | Value is in list | Yes |
| `is_not_in_list` | Value is not in list | Yes |

**IMPORTANT:** When using `exists` or `does_not_exist` operators, the argument field must be `null` (not empty string). Use `types.StringNull()` when transforming responses to state.

---

## Data Source - Single Rule

### File: `internal/data_source_edge_application_request_rule.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ datasource.DataSource              = &RequestRuleDataSource{}
    _ datasource.DataSourceWithConfigure = &RequestRuleDataSource{}
)

func dataSourceAzionEdgeApplicationRequestRule() datasource.DataSource {
    return &RequestRuleDataSource{}
}

type RequestRuleDataSource struct {
    client *apiClient
}

type RequestRuleDataSourceModel struct {
    ID            types.String          `tfsdk:"id"`
    ApplicationID types.Int64           `tfsdk:"edge_application_id"`
    Results       RequestRuleResultModel `tfsdk:"results"`
}

type RequestRuleResultModel struct {
    ID          types.Int64                    `tfsdk:"id"`
    Name        types.String                   `tfsdk:"name"`
    Active      types.Bool                     `tfsdk:"active"`
    Criteria    [][]CriterionModel             `tfsdk:"criteria"`
    Behaviors   []BehaviorModel                `tfsdk:"behaviors"`
    Description types.String                   `tfsdk:"description"`
    Order       types.Int64                    `tfsdk:"order"`
    LastEditor  types.String                   `tfsdk:"last_editor"`
    LastModified types.String                  `tfsdk:"last_modified"`
}

type CriterionModel struct {
    Conditional types.String `tfsdk:"conditional"`
    Variable    types.String `tfsdk:"variable"`
    Operator    types.String `tfsdk:"operator"`
    Argument    types.String `tfsdk:"argument"`
}

type BehaviorModel struct {
    Type          types.String                `tfsdk:"type"`
    Attributes    *BehaviorAttributesModel    `tfsdk:"attributes"`
    CaptureAttrs  *CaptureAttributesModel     `tfsdk:"capture_attributes"`
}

type BehaviorAttributesModel struct {
    Value types.String `tfsdk:"value"`
}

type CaptureAttributesModel struct {
    Subject       types.String `tfsdk:"subject"`
    Regex         types.String `tfsdk:"regex"`
    CapturedArray types.String `tfsdk:"captured_array"`
}

func (d *RequestRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_application_request_rule"
}

func (d *RequestRuleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *RequestRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "edge_application_id": schema.Int64Attribute{
                Description: "The edge application identifier.",
                Required:    true,
            },
            "results": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The ID of the request rule.",
                        Required:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "The name of the request rule.",
                        Computed:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Whether the rule is active.",
                        Computed:    true,
                    },
                    "criteria": schema.ListNestedAttribute{
                        Description: "Criteria for the rule.",
                        Computed:    true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "conditional": schema.StringAttribute{
                                    Description: "Conditional operator (if, and, or).",
                                    Computed:    true,
                                },
                                "variable": schema.StringAttribute{
                                    Description: "Variable to evaluate.",
                                    Computed:    true,
                                },
                                "operator": schema.StringAttribute{
                                    Description: "Comparison operator.",
                                    Computed:    true,
                                },
                                "argument": schema.StringAttribute{
                                    Description: "Argument for comparison.",
                                    Computed:    true,
                                },
                            },
                        },
                    },
                    "behaviors": schema.ListNestedAttribute{
                        Description: "Behaviors for the rule.",
                        Computed:    true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "type": schema.StringAttribute{
                                    Description: "Type of behavior.",
                                    Computed:    true,
                                },
                                "attributes": schema.SingleNestedAttribute{
                                    Description: "Behavior attributes (for behaviors with args).",
                                    Computed:    true,
                                    Attributes: map[string]schema.Attribute{
                                        "value": schema.StringAttribute{
                                            Description: "Value for the behavior.",
                                            Computed:    true,
                                        },
                                    },
                                },
                                "capture_attributes": schema.SingleNestedAttribute{
                                    Description: "Capture attributes (for capture_match_groups).",
                                    Computed:    true,
                                    Attributes: map[string]schema.Attribute{
                                        "subject": schema.StringAttribute{
                                            Description: "Subject for capture.",
                                            Computed:    true,
                                        },
                                        "regex": schema.StringAttribute{
                                            Description: "Regex pattern.",
                                            Computed:    true,
                                        },
                                        "captured_array": schema.StringAttribute{
                                            Description: "Captured array name.",
                                            Computed:    true,
                                        },
                                    },
                                },
                            },
                        },
                    },
                    "description": schema.StringAttribute{
                        Description: "Description of the rule.",
                        Computed:    true,
                    },
                    "order": schema.Int64Attribute{
                        Description: "Order of the rule.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "Last editor of the rule.",
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
}

func (d *RequestRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var ruleID types.Int64

    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsRuleID := req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &ruleID)
    resp.Diagnostics.Append(diagsRuleID...)
    if resp.Diagnostics.HasError() {
        return
    }

    ruleResponse, response, err := d.client.azionapi.ApplicationsRequestRulesAPI.
        RetrieveApplicationRequestRule(ctx, applicationID.ValueInt64(), ruleID.ValueInt64()).
        Execute()
    if err != nil {
        if response.StatusCode == 429 {
            ruleResponse, response, err = utils.RetryOn429(func() (*azionapi.ResponsePhaseRuleResponse, *http.Response, error) {
                return d.client.azionapi.ApplicationsRequestRulesAPI.
                    RetrieveApplicationRequestRule(ctx, applicationID.ValueInt64(), ruleID.ValueInt64()).
                    Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    // Transform criteria
    // IMPORTANT: Use types.StringNull() for empty arguments (e.g., "exists" operator)
    // to avoid "Provider produced inconsistent result" errors.
    var criteria [][]CriterionModel
    for _, criterionGroup := range ruleResponse.Data.Criteria {
        var group []CriterionModel
        for _, c := range criterionGroup {
            arg := getCriterionArgumentValue(c.Argument)
            var argValue types.String
            if arg == "" {
                argValue = types.StringNull()
            } else {
                argValue = types.StringValue(arg)
            }
            group = append(group, CriterionModel{
                Conditional: types.StringValue(c.GetConditional()),
                Variable:    types.StringValue(c.GetVariable()),
                Operator:    types.StringValue(c.GetOperator()),
                Argument:    argValue,
            })
        }
        criteria = append(criteria, group)
    }

    // Transform behaviors
    var behaviors []BehaviorModel
    for _, b := range ruleResponse.Data.Behaviors {
        behavior := BehaviorModel{
            Type: types.StringValue(b.GetType()),
        }
        
        // Check which type of behavior it is
        if b.BehaviorArgs != nil {
            val := getBehaviorArgsValue(b.BehaviorArgs.Attributes.Value)
            behavior.Attributes = &BehaviorAttributesModel{
                Value: types.StringValue(val),
            }
        } else if b.BehaviorCapture != nil {
            behavior.CaptureAttrs = &CaptureAttributesModel{
                Subject:       types.StringValue(b.BehaviorCapture.Attributes.GetSubject()),
                Regex:         types.StringValue(b.BehaviorCapture.Attributes.GetRegex()),
                CapturedArray: types.StringValue(b.BehaviorCapture.Attributes.GetCapturedArray()),
            }
        }
        behaviors = append(behaviors, behavior)
    }

    // Build result
    result := RequestRuleResultModel{
        ID:         types.Int64Value(ruleResponse.Data.GetId()),
        Name:       types.StringValue(ruleResponse.Data.GetName()),
        Order:      types.Int64Value(ruleResponse.Data.GetOrder()),
        Criteria:   criteria,
        Behaviors:  behaviors,
    }

    if ruleResponse.Data.Active != nil {
        result.Active = types.BoolValue(*ruleResponse.Data.Active)
    }
    if ruleResponse.Data.Description != nil {
        result.Description = types.StringValue(*ruleResponse.Data.Description)
    }
    if ruleResponse.Data.LastEditor.Get() != nil {
        result.LastEditor = types.StringValue(*ruleResponse.Data.LastEditor.Get())
    }
    if ruleResponse.Data.LastModified.Get() != nil {
        result.LastModified = types.StringValue(ruleResponse.Data.LastModified.Get().Format(time.RFC3339))
    }

    state := RequestRuleDataSourceModel{
        ApplicationID: applicationID,
        Results:       result,
    }
    state.ID = types.StringValue("Get By ID Edge Application Request Rule")

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

## Data Source - List Rules

### File: `internal/data_source_edge_application_request_rules.go`

```go
package provider

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ datasource.DataSource              = &RequestRulesDataSource{}
    _ datasource.DataSourceWithConfigure = &RequestRulesDataSource{}
)

func dataSourceAzionEdgeApplicationRequestRules() datasource.DataSource {
    return &RequestRulesDataSource{}
}

type RequestRulesDataSource struct {
    client *apiClient
}

type RequestRulesDataSourceModel struct {
    ID             types.String               `tfsdk:"id"`
    ApplicationID  types.Int64                `tfsdk:"edge_application_id"`
    Page           types.Int64                `tfsdk:"page"`
    PageSize       types.Int64                `tfsdk:"page_size"`
    TotalPages     types.Int64                `tfsdk:"total_pages"`
    TotalCount     types.Int64                `tfsdk:"total_count"`
    Results        []RequestRuleSummaryModel  `tfsdk:"results"`
}

type RequestRuleSummaryModel struct {
    ID          types.Int64  `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Active      types.Bool   `tfsdk:"active"`
    Description types.String `tfsdk:"description"`
    Order       types.Int64  `tfsdk:"order"`
    LastEditor  types.String `tfsdk:"last_editor"`
    LastModified types.String `tfsdk:"last_modified"`
}

func (d *RequestRulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_application_request_rules"
}

func (d *RequestRulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *RequestRulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "edge_application_id": schema.Int64Attribute{
                Description: "The edge application identifier.",
                Required:    true,
            },
            "page": schema.Int64Attribute{
                Description: "Page number for pagination.",
                Optional:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "Number of items per page.",
                Optional:    true,
            },
            "total_pages": schema.Int64Attribute{
                Description: "Total number of pages.",
                Computed:    true,
            },
            "total_count": schema.Int64Attribute{
                Description: "Total number of items.",
                Computed:    true,
            },
            "results": schema.ListNestedAttribute{
                Description: "List of request rules.",
                Computed:    true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.Int64Attribute{
                            Description: "The ID of the request rule.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Description: "The name of the request rule.",
                            Computed:    true,
                        },
                        "active": schema.BoolAttribute{
                            Description: "Whether the rule is active.",
                            Computed:    true,
                        },
                        "description": schema.StringAttribute{
                            Description: "Description of the rule.",
                            Computed:    true,
                        },
                        "order": schema.Int64Attribute{
                            Description: "Order of the rule.",
                            Computed:    true,
                        },
                        "last_editor": schema.StringAttribute{
                            Description: "Last editor of the rule.",
                            Computed:    true,
                        },
                        "last_modified": schema.StringAttribute{
                            Description: "Last modified timestamp.",
                            Computed:    true,
                        },
                    },
                },
            },
        },
    }
}

func (d *RequestRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var page types.Int64
    var pageSize types.Int64

    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
    resp.Diagnostics.Append(diagsPage...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
    resp.Diagnostics.Append(diagsPageSize...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build request
    listReq := d.client.azionapi.ApplicationsRequestRulesAPI.
        ListApplicationRequestRules(ctx, applicationID.ValueInt64())

    if !page.IsNull() && !page.IsUnknown() {
        listReq = listReq.Page(page.ValueInt64())
    }
    if !pageSize.IsNull() && !pageSize.IsUnknown() {
        listReq = listReq.PageSize(pageSize.ValueInt64())
    }

    listResponse, response, err := listReq.Execute()
    if err != nil {
        if response.StatusCode == 429 {
            listResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedRequestPhaseRuleList, *http.Response, error) {
                return listReq.Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    // Transform results
    var results []RequestRuleSummaryModel
    for _, rule := range listResponse.Results {
        summary := RequestRuleSummaryModel{
            ID:    types.Int64Value(rule.GetId()),
            Name:  types.StringValue(rule.GetName()),
            Order: types.Int64Value(rule.GetOrder()),
        }
        if rule.Active != nil {
            summary.Active = types.BoolValue(*rule.Active)
        }
        if rule.Description != nil {
            summary.Description = types.StringValue(*rule.Description)
        }
        if rule.LastEditor.Get() != nil {
            summary.LastEditor = types.StringValue(*rule.LastEditor.Get())
        }
        if rule.LastModified.Get() != nil {
            summary.LastModified = types.StringValue(rule.LastModified.Get().Format(time.RFC3339))
        }
        results = append(results, summary)
    }

    state := RequestRulesDataSourceModel{
        ApplicationID: applicationID,
        Results:       results,
    }
    state.ID = types.StringValue(fmt.Sprintf("%d", applicationID.ValueInt64()))

    if listResponse.Pagination != nil {
        state.TotalPages = types.Int64Value(int64(listResponse.Pagination.GetTotalPages()))
        state.TotalCount = types.Int64Value(int64(listResponse.Pagination.GetCount()))
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

## Resource - CRUD Operations

### File: `internal/resource_edge_application_request_rule.go`

```go
package provider

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ resource.Resource                = &requestRuleResource{}
    _ resource.ResourceWithConfigure   = &requestRuleResource{}
    _ resource.ResourceWithImportState = &requestRuleResource{}
)

func NewRequestRuleResource() resource.Resource {
    return &requestRuleResource{}
}

type requestRuleResource struct {
    client *apiClient
}

type RequestRuleResourceModel struct {
    ID            types.String           `tfsdk:"id"`
    ApplicationID types.Int64            `tfsdk:"edge_application_id"`
    LastUpdated   types.String           `tfsdk:"last_updated"`
    Results       *RequestRuleResultsModel `tfsdk:"results"`
}

type RequestRuleResultsModel struct {
    ID          types.Int64                `tfsdk:"id"`
    Name        types.String               `tfsdk:"name"`
    Active      types.Bool                 `tfsdk:"active"`
    Criteria    [][]CriterionModel         `tfsdk:"criteria"`
    Behaviors   []BehaviorModel            `tfsdk:"behaviors"`
    Description types.String               `tfsdk:"description"`
    Order       types.Int64                `tfsdk:"order"`
    LastEditor  types.String               `tfsdk:"last_editor"`
    LastModified types.String              `tfsdk:"last_modified"`
}

func (r *requestRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_application_request_rule"
}

func (r *requestRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "edge_application_id": schema.Int64Attribute{
                Description: "Numeric identifier of the Edge Application.",
                Required:    true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update.",
                Computed:    true,
            },
            "results": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The ID of the request rule.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "The name of the request rule.",
                        Required:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Whether the rule is active.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "criteria": schema.ListNestedAttribute{
                        Description: "Criteria for the rule.",
                        Required:    true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "conditional": schema.StringAttribute{
                                    Description: "Conditional operator (if, and, or).",
                                    Required:    true,
                                },
                                "variable": schema.StringAttribute{
                                    Description: "Variable to evaluate.",
                                    Required:    true,
                                },
                                "operator": schema.StringAttribute{
                                    Description: "Comparison operator.",
                                    Required:    true,
                                },
                                "argument": schema.StringAttribute{
                                    Description: "Argument for comparison.",
                                    Optional:    true,
                                },
                            },
                        },
                    },
                    "behaviors": schema.ListNestedAttribute{
                        Description: "Behaviors for the rule.",
                        Required:    true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "type": schema.StringAttribute{
                                    Description: "Type of behavior.",
                                    Required:    true,
                                },
                                "attributes": schema.SingleNestedAttribute{
                                    Description: "Behavior attributes (for behaviors with args).",
                                    Optional:    true,
                                    Attributes: map[string]schema.Attribute{
                                        "value": schema.StringAttribute{
                                            Description: "Value for the behavior.",
                                            Required:    true,
                                        },
                                    },
                                },
                                "capture_attributes": schema.SingleNestedAttribute{
                                    Description: "Capture attributes (for capture_match_groups).",
                                    Optional:    true,
                                    Attributes: map[string]schema.Attribute{
                                        "subject": schema.StringAttribute{
                                            Description: "Subject for capture.",
                                            Required:    true,
                                        },
                                        "regex": schema.StringAttribute{
                                            Description: "Regex pattern.",
                                            Required:    true,
                                        },
                                        "captured_array": schema.StringAttribute{
                                            Description: "Captured array name.",
                                            Required:    true,
                                        },
                                    },
                                },
                            },
                        },
                    },
                    "description": schema.StringAttribute{
                        Description: "Description of the rule.",
                        Optional:    true,
                    },
                    "order": schema.Int64Attribute{
                        Description: "Order of the rule.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "Last editor of the rule.",
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
}

func (r *requestRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    r.client = req.ProviderData.(*apiClient)
}

func (r *requestRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan RequestRuleResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var applicationID types.Int64
    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build criteria
    criteria := buildCriteriaRequest(plan.Results.Criteria)

    // Build behaviors
    behaviors := buildBehaviorsRequest(plan.Results.Behaviors)

    // Build request
    ruleRequest := azionapi.NewRequestPhaseRuleRequest(
        plan.Results.Name.ValueString(),
        criteria,
        behaviors,
    )

    if !plan.Results.Active.IsNull() && !plan.Results.Active.IsUnknown() {
        ruleRequest.SetActive(plan.Results.Active.ValueBool())
    }
    if !plan.Results.Description.IsNull() && !plan.Results.Description.IsUnknown() {
        ruleRequest.SetDescription(plan.Results.Description.ValueString())
    }

    createResponse, response, err := r.client.azionapi.ApplicationsRequestRulesAPI.
        CreateApplicationRequestRule(ctx, applicationID.ValueInt64()).
        RequestPhaseRuleRequest(*ruleRequest).
        Execute()
    if err != nil {
        handleAPIError(resp, response, err)
        return
    }

    // Update state from response
    plan.ID = types.StringValue(strconv.FormatInt(createResponse.Data.GetId(), 10))
    plan.Results.ID = types.Int64Value(createResponse.Data.GetId())
    plan.Results.Order = types.Int64Value(createResponse.Data.GetOrder())
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, &plan)
    resp.Diagnostics.Append(diags...)
}

func (r *requestRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state RequestRuleResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var applicationID int64
    var ruleID int64

    valueFromCmd := strings.Split(state.ID.ValueString(), "/")
    if len(valueFromCmd) > 1 {
        applicationID = int64(utils.AtoiNoError(valueFromCmd[0], resp))
        ruleID = int64(utils.AtoiNoError(valueFromCmd[1], resp))
    } else {
        applicationID = state.ApplicationID.ValueInt64()
        ruleID = state.Results.ID.ValueInt64()
    }

    ruleResponse, response, err := r.client.azionapi.ApplicationsRequestRulesAPI.
        RetrieveApplicationRequestRule(ctx, applicationID, ruleID).
        Execute()
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        handleAPIError(resp, response, err)
        return
    }

    // Update state from response
    state.Results = transformRuleToModel(&ruleResponse.Data)
    state.ApplicationID = types.Int64Value(applicationID)

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}

func (r *requestRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan RequestRuleResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    applicationID := plan.ApplicationID.ValueInt64()
    ruleID := plan.Results.ID.ValueInt64()

    // Build criteria
    criteria := buildCriteriaRequest(plan.Results.Criteria)

    // Build behaviors
    behaviors := buildBehaviorsRequest(plan.Results.Behaviors)

    // Build request
    ruleRequest := azionapi.NewRequestPhaseRuleRequest(
        plan.Results.Name.ValueString(),
        criteria,
        behaviors,
    )

    if !plan.Results.Active.IsNull() {
        ruleRequest.SetActive(plan.Results.Active.ValueBool())
    }
    if !plan.Results.Description.IsNull() && !plan.Results.Description.IsUnknown() {
        ruleRequest.SetDescription(plan.Results.Description.ValueString())
    }

    updateResponse, response, err := r.client.azionapi.ApplicationsRequestRulesAPI.
        UpdateApplicationRequestRule(ctx, applicationID, ruleID).
        RequestPhaseRuleRequest(*ruleRequest).
        Execute()
    if err != nil {
        handleAPIError(resp, response, err)
        return
    }

    plan.Results.ID = types.Int64Value(updateResponse.Data.GetId())
    plan.Results.Order = types.Int64Value(updateResponse.Data.GetOrder())
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, &plan)
    resp.Diagnostics.Append(diags...)
}

func (r *requestRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state RequestRuleResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    applicationID := state.ApplicationID.ValueInt64()
    ruleID := state.Results.ID.ValueInt64()

    _, response, err := r.client.azionapi.ApplicationsRequestRulesAPI.
        DeleteApplicationRequestRule(ctx, applicationID, ruleID).
        Execute()
    if err != nil {
        if response.StatusCode != http.StatusNotFound {
            handleAPIError(resp, response, err)
            return
        }
    }
}

func (r *requestRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    parts := strings.Split(req.ID, "/")
    if len(parts) != 2 {
        resp.Diagnostics.AddError(
            "Invalid import format",
            "Expected format: {edge_application_id}/{rule_id}",
        )
        return
    }

    applicationID, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Invalid application ID",
            "Could not parse application ID",
        )
        return
    }

    ruleID, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Invalid rule ID",
            "Could not parse rule ID",
        )
        return
    }

    var state RequestRuleResourceModel
    state.ApplicationID = types.Int64Value(applicationID)
    state.Results = &RequestRuleResultsModel{
        ID: types.Int64Value(ruleID),
    }
    state.ID = types.StringValue(fmt.Sprintf("%d/%d", applicationID, ruleID))

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}

// Helper functions

// getCriterionArgumentValue extracts the value from a polymorphic criterion argument.
// IMPORTANT: Never use fmt.Sprintf("%v", arg.Get()) - it will print the struct's pointer address.
// Returns empty string if no value is set, which should be converted to types.StringNull()
// for operators like "exists" and "does_not_exist" that don't require arguments.
func getCriterionArgumentValue(arg azionapi.NullableApplicationCriterionArgument) string {
    if !arg.IsSet() {
        return ""
    }
    argValue := arg.Get()
    if argValue == nil {
        return ""
    }
    if argValue.String != nil {
        return *argValue.String
    }
    if argValue.Int64 != nil {
        return fmt.Sprintf("%d", *argValue.Int64)
    }
    return ""
}

// When transforming criteria in response handling, convert empty string to null:
// IMPORTANT: The argument field should be types.StringNull() (not empty string) when
// operators like "exists" or "does_not_exist" are used, to match the Terraform plan.
func transformCriteriaArgument(arg string) types.String {
    if arg == "" {
        return types.StringNull()
    }
    return types.StringValue(arg)
}

// getBehaviorArgsValue extracts the value from a polymorphic behavior args value.
func getBehaviorArgsValue(value azionapi.BehaviorArgsAttributesValue) string {
    if value.String != nil {
        return *value.String
    }
    if value.Int64 != nil {
        return fmt.Sprintf("%d", *value.Int64)
    }
    return ""
}

func buildCriteriaRequest(criteria [][]CriterionModel) [][]azionapi.ApplicationCriterionFieldRequest {
    var result [][]azionapi.ApplicationCriterionFieldRequest
    for _, group := range criteria {
        var criterionGroup []azionapi.ApplicationCriterionFieldRequest
        for _, c := range group {
            criterion := azionapi.NewApplicationCriterionFieldRequest(
                c.Conditional.ValueString(),
                c.Variable.ValueString(),
                c.Operator.ValueString(),
            )
            if !c.Argument.IsNull() && !c.Argument.IsUnknown() {
                // Create polymorphic argument
                arg := azionapi.ApplicationCriterionArgumentRequest{
                    String: c.Argument.ValueStringPointer(),
                }
                criterion.SetArgument(arg)
            }
            criterionGroup = append(criterionGroup, *criterion)
        }
        result = append(result, criterionGroup)
    }
    return result
}

func buildBehaviorsRequest(behaviors []BehaviorModel) []azionapi.RequestPhaseBehaviorRequest {
    var result []azionapi.RequestPhaseBehaviorRequest
    for _, b := range behaviors {
        behavior := azionapi.NewRequestPhaseBehaviorRequest()
        
        if b.Attributes != nil && !b.Attributes.Value.IsNull() {
            // Behavior with args
            attrs := azionapi.NewBehaviorArgsAttributesWithDefaults()
            attrs.SetValueFromString(b.Attributes.Value.ValueString())
            argsBehavior := azionapi.NewBehaviorArgs(b.Type.ValueString(), *attrs)
            behavior = azionapi.NewRequestPhaseBehaviorRequest()
            behavior.SetBehaviorArgs(*argsBehavior)
        } else if b.CaptureAttrs != nil {
            // Capture behavior
            captureAttrs := azionapi.NewBehaviorCaptureMatchGroupsAttributes(
                b.CaptureAttrs.Subject.ValueString(),
                b.CaptureAttrs.Regex.ValueString(),
                b.CaptureAttrs.CapturedArray.ValueString(),
            )
            captureBehavior := azionapi.NewBehaviorCapture(b.Type.ValueString(), *captureAttrs)
            behavior.SetBehaviorCapture(*captureBehavior)
        } else {
            // No args behavior
            noArgsBehavior := azionapi.NewBehaviorNoArgs(b.Type.ValueString())
            behavior.SetBehaviorNoArgs(*noArgsBehavior)
        }
        
        result = append(result, *behavior)
    }
    return result
}

func transformRuleToModel(rule *azionapi.RequestPhaseRule) *RequestRuleResultsModel {
    result := &RequestRuleResultsModel{
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
    if rule.LastEditor.Get() != nil {
        result.LastEditor = types.StringValue(*rule.LastEditor.Get())
    }
    if rule.LastModified.Get() != nil {
        result.LastModified = types.StringValue(rule.LastModified.Get().Format(time.RFC3339))
    }

    // Transform criteria
    // IMPORTANT: Use types.StringNull() for empty arguments (e.g., "exists" operator)
    // to avoid "Provider produced inconsistent result" errors.
    for _, group := range rule.Criteria {
        var criterionGroup []CriterionModel
        for _, c := range group {
            arg := getCriterionArgumentValue(c.Argument)
            var argValue types.String
            if arg == "" {
                argValue = types.StringNull()
            } else {
                argValue = types.StringValue(arg)
            }
            criterionGroup = append(criterionGroup, CriterionModel{
                Conditional: types.StringValue(c.GetConditional()),
                Variable:    types.StringValue(c.GetVariable()),
                Operator:    types.StringValue(c.GetOperator()),
                Argument:    argValue,
            })
        }
        result.Criteria = append(result.Criteria, criterionGroup)
    }

    // Transform behaviors
    for _, b := range rule.Behaviors {
        behavior := BehaviorModel{
            Type: types.StringValue(b.GetType()),
        }
        
        if b.BehaviorArgs != nil {
            val := getBehaviorArgsValue(b.BehaviorArgs.Attributes.Value)
            behavior.Attributes = &BehaviorAttributesModel{
                Value: types.StringValue(val),
            }
        } else if b.BehaviorCapture != nil {
            behavior.CaptureAttrs = &CaptureAttributesModel{
                Subject:       types.StringValue(b.BehaviorCapture.Attributes.GetSubject()),
                Regex:         types.StringValue(b.BehaviorCapture.Attributes.GetRegex()),
                CapturedArray: types.StringValue(b.BehaviorCapture.Attributes.GetCapturedArray()),
            }
        }
        result.Behaviors = append(result.Behaviors, behavior)
    }

    return result
}

func handleAPIError(resp interface{}, response *http.Response, err error) {
    switch r := resp.(type) {
    case *resource.CreateResponse:
        if response.StatusCode == 429 {
            r.Diagnostics.AddError(err.Error(), "Rate limited. Please retry.")
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            r.Diagnostics.AddError(err.Error(), string(bodyBytes))
        }
    case *resource.ReadResponse:
        if response.StatusCode == 429 {
            r.Diagnostics.AddError(err.Error(), "Rate limited. Please retry.")
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            r.Diagnostics.AddError(err.Error(), string(bodyBytes))
        }
    case *resource.UpdateResponse:
        if response.StatusCode == 429 {
            r.Diagnostics.AddError(err.Error(), "Rate limited. Please retry.")
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            r.Diagnostics.AddError(err.Error(), string(bodyBytes))
        }
    case *resource.DeleteResponse:
        if response.StatusCode == 429 {
            r.Diagnostics.AddError(err.Error(), "Rate limited. Please retry.")
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            r.Diagnostics.AddError(err.Error(), string(bodyBytes))
        }
    }
}
```

---

## Error Handling

### Standard Error Pattern

```go
if err != nil {
    if response.StatusCode == 429 {
        // Retry with exponential backoff
        result, response, err = utils.RetryOn429(func() (*ResponseType, *http.Response, error) {
            return client.API.Method(ctx, params).Execute()
        }, 5) // Max 5 retries
        
        if response != nil {
            defer response.Body.Close()
        }
        
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else if response.StatusCode == http.StatusNotFound {
        // For Read operations - mark resource as deleted
        resp.State.RemoveResource(ctx)
        return
    } else {
        // Read error body for details
        bodyBytes, errReadAll := io.ReadAll(response.Body)
        if errReadAll != nil {
            resp.Diagnostics.AddError(errReadAll.Error(), "err")
        }
        bodyString := string(bodyBytes)
        resp.Diagnostics.AddError(err.Error(), bodyString)
        return
    }
}
```

### Error Code Handling

| Code | Description | Action |
|------|-------------|--------|
| 400 | Bad Request | Check request body format |
| 401 | Unauthorized | Check API token |
| 403 | Forbidden | Check permissions |
| 404 | Not Found | Resource doesn't exist |
| 405 | Method Not Allowed | Check HTTP method |
| 406 | Not Acceptable | Check Accept header |
| 429 | Too Many Requests | Retry with backoff |

---

## Type Conversions

### Terraform to Go

```go
// String
name := plan.Name.ValueString()
namePtr := plan.Name.ValueStringPointer()

// Int64
id := plan.ID.ValueInt64()

// Bool
active := plan.Active.ValueBool()
activePtr := plan.Active.ValueBoolPointer()

// Check if set
if !plan.OptionalField.IsNull() && !plan.OptionalField.IsUnknown() {
    // Field is set
}
```

### Go to Terraform

```go
// String
types.StringValue(str)
types.StringPointerValue(&str)

// Int64
types.Int64Value(int64Val)
types.Int64PointerValue(&int64Val)

// Bool
types.BoolValue(boolVal)
types.BoolPointerValue(&boolVal)

// Time
types.StringValue(time.Now().Format(time.RFC850))
types.StringValue(timestamp.Format(time.RFC3339))
```

---

## Provider Registration

All data sources and resources must be registered in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionEdgeApplicationRequestRule,
        dataSourceAzionEdgeApplicationRequestRules,
        dataSourceAzionEdgeApplicationResponseRule,
        dataSourceAzionEdgeApplicationResponseRules,
        // ... other data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewRequestRuleResource,
        NewResponseRuleResource,
        // ... other resources
    }
}
```

---

## Summary Checklist

When implementing rules engine resources:

1. **Choose correct API**: Request rules vs Response rules
2. **Handle polymorphic behaviors**: BehaviorArgs, BehaviorCapture, BehaviorNoArgs
3. **Build criteria correctly**: Nested arrays with conditional operators
4. **Use V4 SDK types**: `azionapi.NewRequestPhaseRuleRequest`, etc.
5. **Handle 429 errors**: Use `utils.RetryOn429`
6. **Handle optional fields**: Check `IsNull()` and `IsUnknown()`
7. **Transform nested objects**: Create helper functions
8. **Support import**: Use `application_id/rule_id` format
9. **Register in provider.go**: Add to DataSources() and Resources()

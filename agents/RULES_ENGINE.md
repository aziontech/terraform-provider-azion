# Application Rules Engine - Code Generation Guide

This document provides comprehensive guidance for AI agents generating Terraform provider code for Application Rules Engine from the OpenAPI specification (V4 SDK).

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

Each API provides full CRUD operations for managing rules within an Application.

### Key Differences from V3

| V3 (Legacy) | V4 (Current) |
|-------------|--------------|
| Single `EdgeApplicationsRulesEngineAPI` | Separate `ApplicationsRequestRulesAPI` and `ApplicationsResponseRulesAPI` |
| Phase passed as URL parameter | Phase determined by API choice |
| `RulesEngineIdResponse` | `RequestPhaseRuleResponse` / `ResponsePhaseRuleResponse` |
| `RulesEngineBehaviorString` / `RulesEngineBehaviorObject` | Polymorphic `RequestPhaseBehavior` |

### Naming Convention

**IMPORTANT**: The "edge" prefix is no longer used in the V4 SDK and Terraform provider code.

- Terraform resource name: `azion_application_rule_engine` (not `azion_edge_application_rule_engine`)
- Go function: `NewApplicationRulesEngineResource` (not `NewEdgeApplicationRulesEngineResource`)
- Go struct: `rulesEngineResource` (no edge prefix needed)
- Variable names: `applicationID` (not `edgeApplicationID`)

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
client.api.ApplicationsRequestRulesAPI
client.api.ApplicationsResponseRulesAPI

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
    Criteria    [][]ApplicationCriterionFieldRequest    `json:"criteria"`
    Behaviors   []RequestPhaseBehaviorRequest           `json:"behaviors"`
    Description *string                                 `json:"description,omitempty"`
}

// Request Phase Rule Response
type RequestPhaseRule struct {
    Id           int64                              `json:"id"`
    Name         string                             `json:"name"`
    Active       *bool                              `json:"active,omitempty"`
    Criteria     [][]ApplicationCriterionField      `json:"criteria"`
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
    Type       string                `json:"type"`
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

### File: `internal/data_source_application_rule_engine.go`

**Note:** The file name retains `edge_application` for backwards compatibility, but the Terraform resource name uses `application_rule_engine`.

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
    _ datasource.DataSource              = &RuleEngineDataSource{}
    _ datasource.DataSourceWithConfigure = &RuleEngineDataSource{}
)

func dataSourceAzionApplicationRuleEngine() datasource.DataSource {
    return &RuleEngineDataSource{}
}

type RuleEngineDataSource struct {
    client *apiClient
}

type RuleEngineDataSourceModel struct {
    ID            types.String          `tfsdk:"id"`
    ApplicationID types.Int64           `tfsdk:"application_id"`
    Results       RuleEngineResultModel `tfsdk:"results"`
}

type RuleEngineResultModel struct {
    ID           types.Int64        `tfsdk:"id"`
    Name         types.String       `tfsdk:"name"`
    Phase        types.String       `tfsdk:"phase"`
    Active       types.Bool         `tfsdk:"active"`
    Criteria     [][]CriterionModel `tfsdk:"criteria"`
    Behaviors    []BehaviorModel    `tfsdk:"behaviors"`
    Description  types.String       `tfsdk:"description"`
    Order        types.Int64        `tfsdk:"order"`
    LastEditor   types.String       `tfsdk:"last_editor"`
    LastModified types.String       `tfsdk:"last_modified"`
}

type CriterionModel struct {
    Conditional types.String `tfsdk:"conditional"`
    Variable    types.String `tfsdk:"variable"`
    Operator    types.String `tfsdk:"operator"`
    Argument    types.String `tfsdk:"argument"`
}

type BehaviorModel struct {
    Type         types.String             `tfsdk:"type"`
    Attributes   *BehaviorAttributesModel `tfsdk:"attributes"`
    CaptureAttrs *CaptureAttributesModel  `tfsdk:"capture_attributes"`
}

type BehaviorAttributesModel struct {
    Value types.String `tfsdk:"value"`
}

type CaptureAttributesModel struct {
    Subject       types.String `tfsdk:"subject"`
    Regex         types.String `tfsdk:"regex"`
    CapturedArray types.String `tfsdk:"captured_array"`
}

func (r *RuleEngineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_rule_engine"
}

func (r *RuleEngineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    r.client = req.ProviderData.(*apiClient)
}

func (r *RuleEngineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "application_id": schema.Int64Attribute{
                Description: "The application identifier.",
                Required:    true,
            },
            "results": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The ID of the rules engine rule.",
                        Required:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "The name of the rules engine rule.",
                        Computed:    true,
                    },
                    "phase": schema.StringAttribute{
                        Description: "The phase in which the rule is executed (request or response).",
                        Required:    true,
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

func (r *RuleEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var ruleID types.Int64
    var phase types.String

    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsPhase := req.Config.GetAttribute(ctx, path.Root("results").AtName("phase"), &phase)
    resp.Diagnostics.Append(diagsPhase...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsRuleID := req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &ruleID)
    resp.Diagnostics.Append(diagsRuleID...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Determine which API to use based on phase
    phaseStr := phase.ValueString()
    var result RuleEngineResultModel
    var response *http.Response
    var err error

    switch phaseStr {
    case "request":
        result, response, err = r.readRequestRule(ctx, applicationID.ValueInt64(), ruleID.ValueInt64(), phaseStr)
    case "response":
        result, response, err = r.readResponseRule(ctx, applicationID.ValueInt64(), ruleID.ValueInt64(), phaseStr)
    default:
        resp.Diagnostics.AddError(
            "Invalid phase value",
            fmt.Sprintf("Phase must be 'request' or 'response', got: %s", phaseStr),
        )
        return
    }

    if err != nil {
        if response != nil && response.StatusCode == 429 {
            switch phaseStr {
            case "request":
                result, response, err = r.readRequestRuleWithRetry(ctx, applicationID.ValueInt64(), ruleID.ValueInt64(), phaseStr)
            case "response":
                result, response, err = r.readResponseRuleWithRetry(ctx, applicationID.ValueInt64(), ruleID.ValueInt64(), phaseStr)
            }

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else if response != nil {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            response.Body.Close()
            return
        } else {
            resp.Diagnostics.AddError(err.Error(), "API request failed")
            return
        }
    }
    if response != nil {
        defer response.Body.Close()
    }

    state := RuleEngineDataSourceModel{
        ApplicationID: applicationID,
        Results:       result,
    }
    state.ID = types.StringValue("Get By ID Application Rule Engine")

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}

func (r *RuleEngineDataSource) readRequestRule(ctx context.Context, applicationID, ruleID int64, phase string) (RuleEngineResultModel, *http.Response, error) {
    ruleResponse, response, err := r.client.api.ApplicationsRequestRulesAPI.
        RetrieveApplicationRequestRule(ctx, applicationID, ruleID).
        Execute()
    if err != nil {
        return RuleEngineResultModel{}, response, err
    }
    return transformResponsePhaseRule(ruleResponse.Data, phase), response, nil
}

func (r *RuleEngineDataSource) readRequestRuleWithRetry(ctx context.Context, applicationID, ruleID int64, phase string) (RuleEngineResultModel, *http.Response, error) {
    ruleResponse, response, err := utils.RetryOn429(func() (*azionapi.ResponsePhaseRuleResponse, *http.Response, error) {
        return r.client.api.ApplicationsRequestRulesAPI.
            RetrieveApplicationRequestRule(ctx, applicationID, ruleID).
            Execute()
    }, 5)
    if err != nil {
        return RuleEngineResultModel{}, response, err
    }
    return transformResponsePhaseRule(ruleResponse.Data, phase), response, nil
}

func (r *RuleEngineDataSource) readResponseRule(ctx context.Context, applicationID, ruleID int64, phase string) (RuleEngineResultModel, *http.Response, error) {
    ruleResponse, response, err := r.client.api.ApplicationsResponseRulesAPI.
        RetrieveApplicationResponseRule(ctx, applicationID, ruleID).
        Execute()
    if err != nil {
        return RuleEngineResultModel{}, response, err
    }
    return transformResponsePhaseRule(ruleResponse.Data, phase), response, nil
}

func (r *RuleEngineDataSource) readResponseRuleWithRetry(ctx context.Context, applicationID, ruleID int64, phase string) (RuleEngineResultModel, *http.Response, error) {
    ruleResponse, response, err := utils.RetryOn429(func() (*azionapi.ResponsePhaseRuleResponse, *http.Response, error) {
        return r.client.api.ApplicationsResponseRulesAPI.
            RetrieveApplicationResponseRule(ctx, applicationID, ruleID).
            Execute()
    }, 5)
    if err != nil {
        return RuleEngineResultModel{}, response, err
    }
    return transformResponsePhaseRule(ruleResponse.Data, phase), response, nil
}

func transformResponsePhaseRule(rule azionapi.ResponsePhaseRule, phase string) RuleEngineResultModel {
    result := RuleEngineResultModel{
        ID:    types.Int64Value(rule.GetId()),
        Name:  types.StringValue(rule.GetName()),
        Phase: types.StringValue(phase),
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
    for _, criterionGroup := range rule.Criteria {
        var group []CriterionModel
        for _, c := range criterionGroup {
            arg := ""
            if c.Argument.Get() != nil {
                arg = fmt.Sprintf("%v", c.Argument.Get())
            }
            group = append(group, CriterionModel{
                Conditional: types.StringValue(c.GetConditional()),
                Variable:    types.StringValue(c.GetVariable()),
                Operator:    types.StringValue(c.GetOperator()),
                Argument:    types.StringValue(arg),
            })
        }
        result.Criteria = append(result.Criteria, group)
    }

    // Transform behaviors
    for _, b := range rule.Behaviors {
        behavior := BehaviorModel{}

        if b.BehaviorArgs != nil {
            behavior.Type = types.StringValue(b.BehaviorArgs.GetType())
            val := getBehaviorArgsValue(b.BehaviorArgs.Attributes.Value)
            behavior.Attributes = &BehaviorAttributesModel{
                Value: types.StringValue(val),
            }
        } else if b.BehaviorCapture != nil {
            behavior.Type = types.StringValue(b.BehaviorCapture.GetType())
            behavior.CaptureAttrs = &CaptureAttributesModel{
                Subject:       types.StringValue(b.BehaviorCapture.Attributes.GetSubject()),
                Regex:         types.StringValue(b.BehaviorCapture.Attributes.GetRegex()),
                CapturedArray: types.StringValue(b.BehaviorCapture.Attributes.GetCapturedArray()),
            }
        } else if b.BehaviorNoArgs != nil {
            behavior.Type = types.StringValue(b.BehaviorNoArgs.GetType())
        }
        result.Behaviors = append(result.Behaviors, behavior)
    }

    return result
}

func getBehaviorArgsValue(value azionapi.BehaviorArgsAttributesValue) string {
    if value.String != nil {
        return *value.String
    }
    if value.Int64 != nil {
        return fmt.Sprintf("%d", *value.Int64)
    }
    return ""
}
```

---

## Data Source - List Rules

### File: `internal/data_source_application_rules_engine.go`

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
    _ datasource.DataSource              = &RulesEngineDataSource{}
    _ datasource.DataSourceWithConfigure = &RulesEngineDataSource{}
)

func dataSourceAzionApplicationRulesEngine() datasource.DataSource {
    return &RulesEngineDataSource{}
}

type RulesEngineDataSource struct {
    client *apiClient
}

type RulesEngineDataSourceModel struct {
    ID            types.String             `tfsdk:"id"`
    ApplicationID types.Int64              `tfsdk:"application_id"`
    Counter       types.Int64              `tfsdk:"counter"`
    TotalPages    types.Int64              `tfsdk:"total_pages"`
    Page          types.Int64              `tfsdk:"page"`
    PageSize      types.Int64              `tfsdk:"page_size"`
    Links         *LinksModel              `tfsdk:"links"`
    Results       []RulesEngineResultModel `tfsdk:"results"`
}

type LinksModel struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

type RulesEngineResultModel struct {
    ID           types.Int64        `tfsdk:"id"`
    Name         types.String       `tfsdk:"name"`
    Phase        types.String       `tfsdk:"phase"`
    Active       types.Bool         `tfsdk:"active"`
    Criteria     [][]CriterionModel `tfsdk:"criteria"`
    Behaviors    []BehaviorModel    `tfsdk:"behaviors"`
    Description  types.String       `tfsdk:"description"`
    Order        types.Int64        `tfsdk:"order"`
    LastEditor   types.String       `tfsdk:"last_editor"`
    LastModified types.String       `tfsdk:"last_modified"`
}

func (r *RulesEngineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_rules_engine"
}

func (r *RulesEngineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    r.client = req.ProviderData.(*apiClient)
}

func (r *RulesEngineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "application_id": schema.Int64Attribute{
                Description: "The application identifier.",
                Required:    true,
            },
            "counter": schema.Int64Attribute{
                Description: "The total number of rules.",
                Computed:    true,
            },
            "page": schema.Int64Attribute{
                Description: "The page number.",
                Optional:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "The page size.",
                Optional:    true,
            },
            "total_pages": schema.Int64Attribute{
                Description: "The total number of pages.",
                Computed:    true,
            },
            "links": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "previous": schema.StringAttribute{
                        Computed: true,
                    },
                    "next": schema.StringAttribute{
                        Computed: true,
                    },
                },
            },
            "results": schema.ListNestedAttribute{
                Required: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.Int64Attribute{
                            Description: "The ID of the rules engine rule.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Description: "The name of the rules engine rule.",
                            Computed:    true,
                        },
                        "phase": schema.StringAttribute{
                            Description: "The phase in which the rule is executed (request or response).",
                            Required:    true,
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
        },
    }
}

func (r *RulesEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var phase types.String
    var page types.Int64
    var pageSize types.Int64

    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get phase from the first result element
    diagsPhase := req.Config.GetAttribute(ctx, path.Root("results").AtListIndex(0).AtName("phase"), &phase)
    resp.Diagnostics.Append(diagsPhase...)
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

    // Set defaults
    if page.IsNull() || page.IsUnknown() {
        page = types.Int64Value(1)
    }
    if pageSize.IsNull() || pageSize.IsUnknown() {
        pageSize = types.Int64Value(10)
    }

    phaseStr := phase.ValueString()
    var result RulesEngineDataSourceModel
    var response *http.Response
    var err error

    switch phaseStr {
    case "request":
        result, response, err = r.listRequestRules(ctx, applicationID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64(), phaseStr)
    case "response":
        result, response, err = r.listResponseRules(ctx, applicationID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64(), phaseStr)
    default:
        resp.Diagnostics.AddError(
            "Invalid phase value",
            fmt.Sprintf("Phase must be 'request' or 'response', got: %s", phaseStr),
        )
        return
    }

    if err != nil {
        // Handle errors with retry logic for 429
        if response != nil && response.StatusCode == 429 {
            switch phaseStr {
            case "request":
                result, response, err = r.listRequestRulesWithRetry(ctx, applicationID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64(), phaseStr)
            case "response":
                result, response, err = r.listResponseRulesWithRetry(ctx, applicationID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64(), phaseStr)
            }

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else if response != nil {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            response.Body.Close()
            return
        } else {
            resp.Diagnostics.AddError(err.Error(), "API request failed")
            return
        }
    }
    if response != nil {
        defer response.Body.Close()
    }

    result.ApplicationID = applicationID
    result.ID = types.StringValue("Get All Application Rules Engine")

    diags := resp.State.Set(ctx, &result)
    resp.Diagnostics.Append(diags...)
}
```

---

## Resource - CRUD Operations

### File: `internal/resource_application_rule_engine.go`

The resource handles both request and response phase rules, as well as the special "default" rule.

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
    _ resource.Resource                = &rulesEngineResource{}
    _ resource.ResourceWithConfigure   = &rulesEngineResource{}
    _ resource.ResourceWithImportState = &rulesEngineResource{}
)

func NewApplicationRulesEngineResource() resource.Resource {
    return &rulesEngineResource{}
}

type rulesEngineResource struct {
    client *apiClient
}

type RulesEngineResourceModel struct {
    SchemaVersion types.Int64                 `tfsdk:"schema_version"`
    RulesEngine   *RulesEngineResourceResults `tfsdk:"results"`
    ID            types.String                `tfsdk:"id"`
    ApplicationID types.Int64                 `tfsdk:"application_id"`
    LastUpdated   types.String                `tfsdk:"last_updated"`
}

type RulesEngineResourceResults struct {
    ID           types.Int64                        `tfsdk:"id"`
    Name         types.String                       `tfsdk:"name"`
    Phase        types.String                       `tfsdk:"phase"`
    Active       types.Bool                         `tfsdk:"active"`
    Behaviors    []RulesEngineBehaviorResourceModel `tfsdk:"behaviors"`
    Criteria     []CriteriaResourceModel            `tfsdk:"criteria"`
    Description  types.String                       `tfsdk:"description"`
    Order        types.Int64                        `tfsdk:"order"`
    LastEditor   types.String                       `tfsdk:"last_editor"`
    LastModified types.String                       `tfsdk:"last_modified"`
    CreatedAt    types.String                       `tfsdk:"created_at"`
}

type RulesEngineBehaviorResourceModel struct {
    Type         types.String                     `tfsdk:"type"`
    Attributes   *BehaviorAttributesResourceModel `tfsdk:"attributes"`
    CaptureAttrs *CaptureAttributesResourceModel  `tfsdk:"capture_attributes"`
}

type BehaviorAttributesResourceModel struct {
    Value types.String `tfsdk:"value"`
}

type CaptureAttributesResourceModel struct {
    Subject       types.String `tfsdk:"subject"`
    Regex         types.String `tfsdk:"regex"`
    CapturedArray types.String `tfsdk:"captured_array"`
}

type CriteriaResourceModel struct {
    Entries []RulesEngineResourceCriteria `tfsdk:"entries"`
}

type RulesEngineResourceCriteria struct {
    Conditional types.String `tfsdk:"conditional"`
    Variable    types.String `tfsdk:"variable"`
    Operator    types.String `tfsdk:"operator"`
    Argument    types.String `tfsdk:"argument"`
}

func (r *rulesEngineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_rule_engine"
}

func (r *rulesEngineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "application_id": schema.Int64Attribute{
                Description: "The application identifier.",
                Required:    true,
            },
            "schema_version": schema.Int64Attribute{
                Computed: true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the resource.",
                Computed:    true,
            },
            "results": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The ID of the rules engine rule.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "The name of the rules engine rule.",
                        Required:    true,
                    },
                    "phase": schema.StringAttribute{
                        Description: "The phase in which the rule is executed (request or response).",
                        Required:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Whether the rule is active.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "behaviors": schema.ListNestedAttribute{
                        Required: true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "type": schema.StringAttribute{
                                    Description: "The type of behavior.",
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
                    "criteria": schema.ListNestedAttribute{
                        Required: true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "entries": schema.ListNestedAttribute{
                                    Required: true,
                                    NestedObject: schema.NestedAttributeObject{
                                        Attributes: map[string]schema.Attribute{
                                            "conditional": schema.StringAttribute{
                                                Description: "The conditional operator used in the rule's criteria (e.g., if, and, or).",
                                                Required:    true,
                                            },
                                            "variable": schema.StringAttribute{
                                                Description: "The variable used in the rule's criteria.",
                                                Required:    true,
                                            },
                                            "operator": schema.StringAttribute{
                                                Description: "The operator used in the rule's criteria.",
                                                Required:    true,
                                            },
                                            "argument": schema.StringAttribute{
                                                Description: "The argument used in the rule's criteria.",
                                                Optional:    true,
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                    "description": schema.StringAttribute{
                        Description: "The description of the rules engine rule.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "order": schema.Int64Attribute{
                        Description: "The order of the rule in the rules engine.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "The last editor of the rule.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "The last modified timestamp.",
                        Computed:    true,
                    },
                    "created_at": schema.StringAttribute{
                        Description: "The creation timestamp.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}

func (r *rulesEngineResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    r.client = req.ProviderData.(*apiClient)
}

// Create, Read, Update, Delete, ImportState methods follow...
// (Full implementation shown in the actual file)

// Helper functions

func buildCriteriaRequestV4(criteria []CriteriaResourceModel) [][]azionapi.ApplicationCriterionFieldRequest {
    var result [][]azionapi.ApplicationCriterionFieldRequest
    for _, criterion := range criteria {
        var criterionGroup []azionapi.ApplicationCriterionFieldRequest
        for _, c := range criterion.Entries {
            criterionField := azionapi.NewApplicationCriterionFieldRequest(
                c.Conditional.ValueString(),
                c.Variable.ValueString(),
                c.Operator.ValueString(),
            )
            if !c.Argument.IsNull() && !c.Argument.IsUnknown() {
                arg := azionapi.ApplicationCriterionArgumentRequest{
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

func buildBehaviorsRequestV4(behaviors []RulesEngineBehaviorResourceModel) []azionapi.RequestPhaseBehaviorRequest {
    var result []azionapi.RequestPhaseBehaviorRequest
    for _, b := range behaviors {
        if b.Attributes != nil && !b.Attributes.Value.IsNull() {
            // Behavior with args
            value := azionapi.BehaviorArgsAttributesValue{
                String: b.Attributes.Value.ValueStringPointer(),
            }
            attrs := azionapi.NewBehaviorArgsAttributes(value)
            argsBehavior := azionapi.NewBehaviorArgs(b.Type.ValueString(), *attrs)
            behaviorRequest := azionapi.BehaviorArgsAsRequestPhaseBehaviorRequest(argsBehavior)
            result = append(result, behaviorRequest)
        } else if b.CaptureAttrs != nil {
            // Capture behavior
            captureAttrs := azionapi.NewBehaviorCaptureMatchGroupsAttributes(
                b.CaptureAttrs.Subject.ValueString(),
                b.CaptureAttrs.Regex.ValueString(),
                b.CaptureAttrs.CapturedArray.ValueString(),
            )
            captureBehavior := azionapi.NewBehaviorCapture(b.Type.ValueString(), *captureAttrs)
            behaviorRequest := azionapi.BehaviorCaptureAsRequestPhaseBehaviorRequest(captureBehavior)
            result = append(result, behaviorRequest)
        } else {
            // No args behavior
            noArgsBehavior := azionapi.NewBehaviorNoArgs(b.Type.ValueString())
            behaviorRequest := azionapi.BehaviorNoArgsAsRequestPhaseBehaviorRequest(noArgsBehavior)
            result = append(result, behaviorRequest)
        }
    }
    return result
}

// getCriterionArgumentValue extracts the value from a polymorphic criterion argument.
// IMPORTANT: Never use fmt.Sprintf("%v", arg.Get()) - it will print the struct's pointer address.
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

## Documentation and Examples

### MANDATORY: Parent Resource Documentation

**IMPORTANT**: The Rules Engine is a child resource of `azion_application_main_setting`. Documentation and examples MUST include the parent resource creation to show complete context.

When updating documentation, always include:

1. **Parent Application Example** - Show creation of the parent application first
2. **Reference Using Terraform Interpolation** - Use `azion_application_main_setting.example.application.application_id` to reference the parent ID

### Documentation Files

Documentation is auto-generated by `terraform-plugin-docs` and located in:

| Type | Location |
|------|----------|
| Singular Data Source Doc | `docs/data-sources/application_rule_engine.md` |
| Plural Data Source Doc | `docs/data-sources/application_rules_engine.md` |
| Resource Doc | `docs/resources/application_rule_engine.md` |

### Example Files

Example Terraform configurations are located in:

| Type | Location |
|------|----------|
| Singular Data Source Example | `examples/data-sources/azion_application_rule_engine/data-source.tf` |
| Plural Data Source Example | `examples/data-sources/azion_application_rules_engine/data-source.tf` |
| Resource Example | `examples/resources/azion_application_rule_engine/resource.tf` |

### Example: Complete Resource Usage with Parent Application

```terraform
# First, create the parent application
resource "azion_application_main_setting" "example" {
  application = {
    name   = "My Application"
    active = true
  }
}

# Then create the rule engine for that application
resource "azion_application_rule_engine" "example" {
  application_id = azion_application_main_setting.example.application.application_id
  results = {
    name        = "Terraform Example"
    phase       = "request"
    description = "My rule engine"
    behaviors = [
      {
        type = "deliver"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "$${uri}"
            operator    = "is_equal"
            conditional = "if"
            argument    = "/"
          }
        ]
      }
    ]
  }
}
```

---

## Provider Registration

All data sources and resources must be registered in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionApplicationRulesEngine,
        dataSourceAzionApplicationRuleEngine,
        // ... other data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewApplicationRulesEngineResource,
        // ... other resources
    }
}
```

---

## Import Format

Rules can be imported using the format: `{application_id}/{phase}/{rule_id}`

```shell
terraform import azion_application_rule_engine.example 12345/request/67890
```

For the default rule (stored in request phase):

```shell
terraform import azion_application_rule_engine.default 12345/default/1
```

---

## Summary Checklist

When implementing rules engine resources:

1. **Choose correct API**: Request rules vs Response rules (based on phase attribute)
2. **Handle polymorphic behaviors**: BehaviorArgs, BehaviorCapture, BehaviorNoArgs
3. **Build criteria correctly**: Nested arrays with conditional operators
4. **Use V4 SDK types**: `azionapi.NewRequestPhaseRuleRequest`, etc.
5. **Handle 429 errors**: Use `utils.RetryOn429`
6. **Handle optional fields**: Check `IsNull()` and `IsUnknown()`
7. **Transform nested objects**: Create helper functions
8. **Support import**: Use `application_id/phase/rule_id` format
9. **Register in provider.go**: Add to DataSources() and Resources()
10. **No "edge" prefix**: Use `applicationID` not `edgeApplicationID` in code

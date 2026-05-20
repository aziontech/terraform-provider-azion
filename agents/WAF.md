# WAF (Web Application Firewall) - Code Generation Guide

This document provides comprehensive guidance for AI agents generating Terraform provider code for WAF (Web Application Firewall) from the Azion API.

## Overview

In the Azion API V4 SDK, WAF is a top-level resource that represents a Web Application Firewall configuration. WAF Rule Sets (called "WAF Exceptions" in the SDK) are child resources that belong to a WAF.

### Naming Convention

| Terraform Resource | V4 SDK Name | API Endpoint |
|-------------------|-------------|--------------|
| `azion_waf` | `WAF` | `/v4/workspace/wafs/{waf_id}` |
| `azion_wafs` | `PaginatedWAFList` | `/v4/workspace/wafs` |
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
// Retrieve a single WAF
o.client.api.WAFsAPI.RetrieveWaf(ctx, wafId).Execute()

// List all WAFs
o.client.api.WAFsAPI.ListWafs(ctx).Page(page).PageSize(pageSize).Execute()

// Create a WAF
o.client.api.WAFsAPI.CreateWaf(ctx).WAFRequest(request).Execute()

// Update a WAF
o.client.api.WAFsAPI.UpdateWaf(ctx, wafId).WAFRequest(request).Execute()

// Delete a WAF
o.client.api.WAFsAPI.DeleteWaf(ctx, wafId).Execute()
```

---

## SDK Structures

### WAF

```go
type WAF struct {
    Id             int64                       `json:"id"`
    Active         *bool                       `json:"active,omitempty"`
    Name           string                      `json:"name"`
    LastEditor     string                      `json:"last_editor"`
    LastModified   time.Time                   `json:"last_modified"`
    ProductVersion NullableString              `json:"product_version,omitempty"`
    EngineSettings *WAFEngineSettingsField     `json:"engine_settings,omitempty"`
}
```

### WAFResponse

```go
type WAFResponse struct {
    State *string `json:"state,omitempty"`
    Data  WAF     `json:"data"`
}
```

### PaginatedWAFList

```go
type PaginatedWAFList struct {
    Count      *int64        `json:"count,omitempty"`
    TotalPages *int64        `json:"total_pages,omitempty"`
    Page       *int64        `json:"page,omitempty"`
    PageSize   *int64        `json:"page_size,omitempty"`
    Next       NullableString `json:"next,omitempty"`
    Previous   NullableString `json:"previous,omitempty"`
    Results    []WAF         `json:"results,omitempty"`
}
```

### WAFEngineSettingsField

```go
type WAFEngineSettingsField struct {
    EngineVersion *string                       `json:"engine_version,omitempty"`
    Type          *string                       `json:"type,omitempty"`
    Attributes    *WAFEngineSettingsAttributesField `json:"attributes,omitempty"`
}
```

### WAFEngineSettingsAttributesField

```go
type WAFEngineSettingsAttributesField struct {
    Rulesets   []int64                  `json:"rulesets,omitempty"`
    Thresholds []ThresholdsConfigField  `json:"thresholds,omitempty"`
}
```

### ThresholdsConfigField

```go
type ThresholdsConfigField struct {
    Threat      string   `json:"threat"`
    Sensitivity *string  `json:"sensitivity,omitempty"`
}
```

**Valid Threat Values:**
- `cross_site_scripting`
- `directory_traversal`
- `evading_tricks`
- `file_upload`
- `identified_attack`
- `remote_file_inclusion`
- `sql_injection`
- `unwanted_access`

**Valid Sensitivity Values:**
- `highest`
- `high`
- `medium`
- `low`
- `lowest`

---

## Terraform Model Structure

### Single Data Source Model

```go
type WafDataSourceModel struct {
    ID      types.String        `tfsdk:"id"`
    WafID   types.Int64         `tfsdk:"waf_id"`
    Results *WafResultDataModel `tfsdk:"results"`
}

type WafResultDataModel struct {
    ID             types.Int64                 `tfsdk:"id"`
    Name           types.String                `tfsdk:"name"`
    Active         types.Bool                  `tfsdk:"active"`
    LastEditor     types.String                `tfsdk:"last_editor"`
    LastModified   types.String                `tfsdk:"last_modified"`
    ProductVersion types.String                `tfsdk:"product_version"`
    EngineSettings *WafEngineSettingsModel     `tfsdk:"engine_settings"`
}

type WafEngineSettingsModel struct {
    EngineVersion types.String                 `tfsdk:"engine_version"`
    Type          types.String                 `tfsdk:"type"`
    Attributes    *WafEngineSettingsAttributesModel `tfsdk:"attributes"`
}

type WafEngineSettingsAttributesModel struct {
    Rulesets   []types.Int64          `tfsdk:"rulesets"`
    Thresholds []WafThresholdConfigModel `tfsdk:"thresholds"`
}

type WafThresholdConfigModel struct {
    Threat      types.String `tfsdk:"threat"`
    Sensitivity types.String `tfsdk:"sensitivity"`
}
```

### List Data Source Model

```go
type WafsDataSourceModel struct {
    ID         types.String       `tfsdk:"id"`
    Counter    types.Int64        `tfsdk:"counter"`
    TotalPages types.Int64        `tfsdk:"total_pages"`
    Page       types.Int64        `tfsdk:"page"`
    PageSize   types.Int64        `tfsdk:"page_size"`
    Links      *WafsResponseLinks `tfsdk:"links"`
    Results    []WafListItemModel `tfsdk:"results"`
}

type WafsResponseLinks struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

type WafListItemModel struct {
    ID             types.Int64                 `tfsdk:"id"`
    Name           types.String                `tfsdk:"name"`
    Active         types.Bool                  `tfsdk:"active"`
    LastEditor     types.String                `tfsdk:"last_editor"`
    LastModified   types.String                `tfsdk:"last_modified"`
    ProductVersion types.String                `tfsdk:"product_version"`
    EngineSettings *WafEngineSettingsModel     `tfsdk:"engine_settings"`
}
```

---

## Schema Definition

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
        "results": schema.SingleNestedAttribute{
            Computed: true,
            Attributes: map[string]schema.Attribute{
                "id": schema.Int64Attribute{
                    Description: "The ID of the WAF.",
                    Computed:    true,
                },
                "name": schema.StringAttribute{
                    Description: "Name of the WAF.",
                    Computed:    true,
                },
                "active": schema.BoolAttribute{
                    Description: "Whether the WAF is active.",
                    Computed:    true,
                },
                "last_editor": schema.StringAttribute{
                    Description: "Last editor of the WAF.",
                    Computed:    true,
                },
                "last_modified": schema.StringAttribute{
                    Description: "Last modified timestamp.",
                    Computed:    true,
                },
                "product_version": schema.StringAttribute{
                    Description: "Product version of the WAF.",
                    Computed:    true,
                },
                "engine_settings": schema.SingleNestedAttribute{
                    Description: "Engine settings for the WAF.",
                    Computed:    true,
                    Attributes: map[string]schema.Attribute{
                        "engine_version": schema.StringAttribute{
                            Description: "Engine version for the WAF.",
                            Computed:    true,
                        },
                        "type": schema.StringAttribute{
                            Description: "Type of the WAF engine.",
                            Computed:    true,
                        },
                        "attributes": schema.SingleNestedAttribute{
                            Description: "Attributes for the WAF engine settings.",
                            Computed:    true,
                            Attributes: map[string]schema.Attribute{
                                "rulesets": schema.ListAttribute{
                                    Description: "List of ruleset IDs.",
                                    Computed:    true,
                                    ElementType: types.Int64Type,
                                },
                                "thresholds": schema.ListNestedAttribute{
                                    Description: "Threshold configurations for the WAF.",
                                    Computed:    true,
                                    NestedObject: schema.NestedAttributeObject{
                                        Attributes: map[string]schema.Attribute{
                                            "threat": schema.StringAttribute{
                                                Description: "The threat type for the threshold.",
                                                Computed:    true,
                                            },
                                            "sensitivity": schema.StringAttribute{
                                                Description: "The sensitivity level for the threshold.",
                                                Computed:    true,
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
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
        "counter": schema.Int64Attribute{
            Description: "The total number of WAFs.",
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
                "previous": schema.StringAttribute{
                    Computed: true,
                },
                "next": schema.StringAttribute{
                    Computed: true,
                },
            },
        },
        "results": schema.ListNestedAttribute{
            Computed: true,
            NestedObject: schema.NestedAttributeObject{
                Attributes: map[string]schema.Attribute{
                    // Same as single data source results
                },
            },
        },
    },
}
```

---

## Transform Functions

### Transform WAF to Result Model

```go
func transformWAFToResultModel(waf azionapi.WAF) *WafResultDataModel {
    result := &WafResultDataModel{
        ID:           types.Int64Value(waf.GetId()),
        Name:         types.StringValue(waf.GetName()),
        LastEditor:   types.StringValue(waf.GetLastEditor()),
        LastModified: types.StringValue(waf.GetLastModified().Format(time.RFC3339)),
    }

    // Optional active
    if waf.HasActive() {
        result.Active = types.BoolValue(waf.GetActive())
    } else {
        result.Active = types.BoolNull()
    }

    // Optional product_version
    if waf.HasProductVersion() {
        result.ProductVersion = types.StringValue(waf.GetProductVersion())
    } else {
        result.ProductVersion = types.StringNull()
    }

    // Optional engine_settings
    if waf.HasEngineSettings() {
        engineSettings := waf.GetEngineSettings()
        result.EngineSettings = transformWAFEngineSettingsToModel(engineSettings)
    } else {
        result.EngineSettings = nil
    }

    return result
}
```

### Transform Engine Settings to Model

```go
func transformWAFEngineSettingsToModel(engineSettings azionapi.WAFEngineSettingsField) *WafEngineSettingsModel {
    result := &WafEngineSettingsModel{}

    // Optional engine_version
    if engineSettings.HasEngineVersion() {
        result.EngineVersion = types.StringValue(engineSettings.GetEngineVersion())
    } else {
        result.EngineVersion = types.StringNull()
    }

    // Optional type
    if engineSettings.HasType() {
        result.Type = types.StringValue(engineSettings.GetType())
    } else {
        result.Type = types.StringNull()
    }

    // Optional attributes
    if engineSettings.HasAttributes() {
        attrs := engineSettings.GetAttributes()
        result.Attributes = transformWAFEngineSettingsAttributesToModel(attrs)
    } else {
        result.Attributes = nil
    }

    return result
}
```

### Transform Engine Settings Attributes to Model

```go
func transformWAFEngineSettingsAttributesToModel(attrs azionapi.WAFEngineSettingsAttributesField) *WafEngineSettingsAttributesModel {
    result := &WafEngineSettingsAttributesModel{}

    // Optional rulesets
    if attrs.HasRulesets() {
        rulesets := attrs.GetRulesets()
        var rulesetValues []types.Int64
        for _, r := range rulesets {
            rulesetValues = append(rulesetValues, types.Int64Value(r))
        }
        result.Rulesets = rulesetValues
    } else {
        result.Rulesets = nil
    }

    // Optional thresholds
    if attrs.HasThresholds() {
        thresholds := attrs.GetThresholds()
        var thresholdValues []WafThresholdConfigModel
        for _, t := range thresholds {
            thresholdModel := WafThresholdConfigModel{
                Threat: types.StringValue(t.GetThreat()),
            }
            if t.HasSensitivity() {
                thresholdModel.Sensitivity = types.StringValue(t.GetSensitivity())
            } else {
                thresholdModel.Sensitivity = types.StringNull()
            }
            thresholdValues = append(thresholdValues, thresholdModel)
        }
        result.Thresholds = thresholdValues
    } else {
        result.Thresholds = nil
    }

    return result
}
```

---

## Read Method Implementation

### Single Data Source Read

```go
func (o *WafDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var wafID types.Int64

    diagsWafID := req.Config.GetAttribute(ctx, path.Root("waf_id"), &wafID)
    resp.Diagnostics.Append(diagsWafID...)
    if resp.Diagnostics.HasError() {
        return
    }

    wafResponse, response, err := o.client.api.WAFsAPI.RetrieveWaf(ctx, wafID.ValueInt64()).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            wafResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFResponse, *http.Response, error) {
                return o.client.api.WAFsAPI.RetrieveWaf(ctx, wafID.ValueInt64()).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(
                    err.Error(),
                    "API request failed after too many retries",
                )
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(
                    errReadAll.Error(),
                    "err",
                )
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(
                err.Error(),
                bodyString,
            )
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Transform the response to the model
    results := transformWAFToResultModel(wafResponse.GetData())

    state := WafDataSourceModel{
        ID:      types.StringValue("Get WAF By ID"),
        WafID:   wafID,
        Results: results,
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### List Data Source Read

```go
func (o *WafsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var page, pageSize types.Int64

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

    if page.IsNull() || page.IsUnknown() || page.ValueInt64() == 0 {
        page = types.Int64Value(1)
    }
    if pageSize.IsNull() || pageSize.IsUnknown() || pageSize.ValueInt64() == 0 {
        pageSize = types.Int64Value(10)
    }

    listResponse, response, err := o.client.api.WAFsAPI.ListWafs(ctx).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            listResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedWAFList, *http.Response, error) {
                return o.client.api.WAFsAPI.ListWafs(ctx).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(
                    err.Error(),
                    "API request failed after too many retries",
                )
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(
                    errReadAll.Error(),
                    "err",
                )
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(
                err.Error(),
                bodyString,
            )
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Transform links
    var previous, next string
    if listResponse.HasPrevious() {
        previous = listResponse.GetPrevious()
    }
    if listResponse.HasNext() {
        next = listResponse.GetNext()
    }

    // Transform results
    var results []WafListItemModel
    for _, waf := range listResponse.GetResults() {
        results = append(results, transformWAFToListItemModel(waf))
    }

    state := WafsDataSourceModel{
        ID:         types.StringValue("Get All WAFs"),
        Results:    results,
        TotalPages: types.Int64Value(listResponse.GetTotalPages()),
        Page:       page,
        PageSize:   pageSize,
        Counter:    types.Int64Value(listResponse.GetCount()),
        Links: &WafsResponseLinks{
            Previous: types.StringValue(previous),
            Next:     types.StringValue(next),
        },
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

## Provider Registration

Register the data sources in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // ... other data sources
        dataSourceAzionWaf,
        dataSourceAzionWafs,
        // ... other data sources
    }
}
```

---

## Documentation Files

### Data Source Documentation

Create documentation in `docs/data-sources/waf.md`:

```markdown
---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_waf"
description: |-
  Provides a data source to read a specific WAF.
---

# azion_waf

Use this data source to read a specific WAF (Web Application Firewall).

## Example Usage

```hcl
data "azion_waf" "example" {
  waf_id = 12345
}
```

## Argument Reference

* `waf_id` - (Required) The ID of the WAF.

## Attribute Reference

* `id` - Identifier of the data source.
* `results` - The WAF data.
  * `id` - The ID of the WAF.
  * `name` - Name of the WAF.
  * `active` - Whether the WAF is active.
  * `last_editor` - Last editor of the WAF.
  * `last_modified` - Last modified timestamp.
  * `product_version` - Product version of the WAF.
  * `engine_settings` - Engine settings for the WAF.
    * `engine_version` - Engine version for the WAF.
    * `type` - Type of the WAF engine.
    * `attributes` - Attributes for the WAF engine settings.
      * `rulesets` - List of ruleset IDs.
      * `thresholds` - Threshold configurations for the WAF.
        * `threat` - The threat type for the threshold.
        * `sensitivity` - The sensitivity level for the threshold.
```

### List Data Source Documentation

Create documentation in `docs/data-sources/wafs.md`:

```markdown
---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_wafs"
description: |-
  Provides a data source to list WAFs.
---

# azion_wafs

Use this data source to list all WAFs (Web Application Firewalls).

## Example Usage

```hcl
data "azion_wafs" "example" {
  page = 1
  page_size = 10
}
```

## Argument Reference

* `page` - (Optional) The page number.
* `page_size` - (Optional) The page size number.

## Attribute Reference

* `id` - Identifier of the data source.
* `counter` - The total number of WAFs.
* `total_pages` - The total number of pages.
* `links` - Pagination links.
  * `previous` - URL to the previous page.
  * `next` - URL to the next page.
* `results` - List of WAFs.
  * `id` - The ID of the WAF.
  * `name` - Name of the WAF.
  * `active` - Whether the WAF is active.
  * `last_editor` - Last editor of the WAF.
  * `last_modified` - Last modified timestamp.
  * `product_version` - Product version of the WAF.
  * `engine_settings` - Engine settings for the WAF.
    * `engine_version` - Engine version for the WAF.
    * `type` - Type of the WAF engine.
    * `attributes` - Attributes for the WAF engine settings.
      * `rulesets` - List of ruleset IDs.
      * `thresholds` - Threshold configurations for the WAF.
        * `threat` - The threat type for the threshold.
        * `sensitivity` - The sensitivity level for the threshold.
```

---

## Example Files

Create example files in `examples/data-sources/azion_waf/data-source.tf` and `examples/data-sources/azion_wafs/data-source.tf`.

---

## Relationship with WAF Rule Sets (Exceptions)

WAF is the parent resource for WAF Rule Sets (called "WAF Exceptions" in the SDK). The WAF Rule Sets are documented separately in [agents/WAF_RULE_SETS.md](agents/WAF_RULE_SETS.md).

When working with WAF resources:
1. A WAF must exist before creating WAF Rule Sets (Exceptions)
2. WAF Rule Sets are created under a specific WAF using the `waf_id` parameter
3. The WAF data sources (`azion_waf` and `azion_wafs`) are used to read WAF configurations
4. The WAF Rule Set data sources (`azion_waf_rule_set` and `azion_waf_rule_sets`) are used to read exceptions

---

## Resource Implementation

### Resource Model

```go
type WafResourceModel struct {
    ID          types.String          `tfsdk:"id"`
    LastUpdated types.String          `tfsdk:"last_updated"`
    Result      *WafResourceResults   `tfsdk:"result"`
}

type WafResourceResults struct {
    ID             types.Int64                       `tfsdk:"id"`
    Name           types.String                      `tfsdk:"name"`
    Active         types.Bool                        `tfsdk:"active"`
    LastEditor     types.String                      `tfsdk:"last_editor"`
    LastModified   types.String                      `tfsdk:"last_modified"`
    ProductVersion types.String                      `tfsdk:"product_version"`
    EngineSettings *WafEngineSettingsResourceModel   `tfsdk:"engine_settings"`
}

type WafEngineSettingsResourceModel struct {
    EngineVersion types.String                                `tfsdk:"engine_version"`
    Type          types.String                                `tfsdk:"type"`
    Attributes    *WafEngineSettingsAttributesResourceModel   `tfsdk:"attributes"`
}

type WafEngineSettingsAttributesResourceModel struct {
    Rulesets   []types.Int64                        `tfsdk:"rulesets"`
    Thresholds []WafThresholdConfigResourceModel    `tfsdk:"thresholds"`
}

type WafThresholdConfigResourceModel struct {
    Threat      types.String `tfsdk:"threat"`
    Sensitivity types.String `tfsdk:"sensitivity"`
}
```

### Resource Schema

```go
resp.Schema = schema.Schema{
    Description: "Creates a WAF (Web Application Firewall) resource.",
    Attributes: map[string]schema.Attribute{
        "id": schema.StringAttribute{
            Description: "Identifier of the resource.",
            Computed:    true,
            PlanModifiers: []planmodifier.String{
                stringplanmodifier.UseStateForUnknown(),
            },
        },
        "last_updated": schema.StringAttribute{
            Description: "Timestamp of the last Terraform update of the resource.",
            Computed:    true,
        },
        "result": schema.SingleNestedAttribute{
            Description: "The WAF configuration.",
            Required:    true,
            Attributes: map[string]schema.Attribute{
                "id": schema.Int64Attribute{
                    Description: "The ID of the WAF.",
                    Computed:    true,
                },
                "name": schema.StringAttribute{
                    Description: "Name of the WAF.",
                    Required:    true,
                },
                "active": schema.BoolAttribute{
                    Description: "Whether the WAF is active.",
                    Optional:    true,
                },
                "last_editor": schema.StringAttribute{
                    Description: "Last editor of the WAF.",
                    Computed:    true,
                },
                "last_modified": schema.StringAttribute{
                    Description: "Last modified timestamp.",
                    Computed:    true,
                },
                "product_version": schema.StringAttribute{
                    Description: "Product version of the WAF.",
                    Optional:    true,
                },
                "engine_settings": schema.SingleNestedAttribute{
                    Description: "Engine settings for the WAF.",
                    Optional:    true,
                    Attributes: map[string]schema.Attribute{
                        "engine_version": schema.StringAttribute{
                            Description: "Engine version for the WAF.",
                            Optional:    true,
                        },
                        "type": schema.StringAttribute{
                            Description: "Type of the WAF engine.",
                            Optional:    true,
                        },
                        "attributes": schema.SingleNestedAttribute{
                            Description: "Attributes for the WAF engine settings.",
                            Optional:    true,
                            Attributes: map[string]schema.Attribute{
                                "rulesets": schema.ListAttribute{
                                    Description: "List of ruleset IDs.",
                                    Optional:    true,
                                    ElementType: types.Int64Type,
                                },
                                "thresholds": schema.ListNestedAttribute{
                                    Description: "Threshold configurations for the WAF.",
                                    Optional:    true,
                                    NestedObject: schema.NestedAttributeObject{
                                        Attributes: map[string]schema.Attribute{
                                            "threat": schema.StringAttribute{
                                                Description: "The threat type for the threshold.",
                                                Required:    true,
                                            },
                                            "sensitivity": schema.StringAttribute{
                                                Description: "The sensitivity level for the threshold.",
                                                Optional:    true,
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    },
}
```

### Create Method

```go
func (r *wafResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan WafResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the WAF request.
    wafRequest := azionapi.NewWAFRequest(plan.Result.Name.ValueString())

    // Set optional fields.
    if !plan.Result.Active.IsNull() && !plan.Result.Active.IsUnknown() {
        wafRequest.SetActive(plan.Result.Active.ValueBool())
    }

    if !plan.Result.ProductVersion.IsNull() && !plan.Result.ProductVersion.IsUnknown() {
        wafRequest.SetProductVersion(plan.Result.ProductVersion.ValueString())
    }

    // Set engine settings if provided.
    if plan.Result.EngineSettings != nil {
        engineSettings := buildWAFEngineSettingsRequest(plan.Result.EngineSettings)
        wafRequest.SetEngineSettings(engineSettings)
    }

    // Create the WAF.
    wafResponse, response, err := r.client.api.WAFsAPI.CreateWaf(ctx).WAFRequest(*wafRequest).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            wafResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFResponse, *http.Response, error) {
                return r.client.api.WAFsAPI.CreateWaf(ctx).WAFRequest(*wafRequest).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(
                    err.Error(),
                    "API request failed after too many retries",
                )
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(
                    errReadAll.Error(),
                    "err",
                )
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(
                err.Error(),
                bodyString,
            )
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Transform the response to the model.
    data := wafResponse.GetData()
    plan.Result = transformWAFToResourceModel(data)
    plan.ID = types.StringValue(strconv.FormatInt(data.GetId(), 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Update Method

```go
func (r *wafResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan WafResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state WafResourceModel
    diagsState := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diagsState...)
    if resp.Diagnostics.HasError() {
        return
    }

    var wafID int64
    var err error
    if state.ID.IsNull() {
        wafID = state.Result.ID.ValueInt64()
    } else {
        wafID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError(
                "Value Conversion error ",
                "Could not convert WAF ID",
            )
            return
        }
    }

    // Build the WAF request.
    wafRequest := azionapi.NewWAFRequest(plan.Result.Name.ValueString())

    // Set optional fields.
    if !plan.Result.Active.IsNull() && !plan.Result.Active.IsUnknown() {
        wafRequest.SetActive(plan.Result.Active.ValueBool())
    }

    if !plan.Result.ProductVersion.IsNull() && !plan.Result.ProductVersion.IsUnknown() {
        wafRequest.SetProductVersion(plan.Result.ProductVersion.ValueString())
    }

    // Set engine settings if provided.
    if plan.Result.EngineSettings != nil {
        engineSettings := buildWAFEngineSettingsRequest(plan.Result.EngineSettings)
        wafRequest.SetEngineSettings(engineSettings)
    }

    // Update the WAF.
    wafResponse, response, err := r.client.api.WAFsAPI.UpdateWaf(ctx, wafID).WAFRequest(*wafRequest).Execute()
    if err != nil {
        // Error handling similar to Create...
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Transform the response to the model.
    data := wafResponse.GetData()
    plan.Result = transformWAFToResourceModel(data)
    plan.ID = types.StringValue(strconv.FormatInt(wafID, 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Delete Method

```go
func (r *wafResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state WafResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var wafID int64
    var err error
    if state.ID.IsNull() {
        wafID = state.Result.ID.ValueInt64()
    } else {
        wafID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError(
                "Value Conversion error ",
                "Could not convert WAF ID",
            )
            return
        }
    }

    deleteResponse, response, err := r.client.api.WAFsAPI.DeleteWaf(ctx, wafID).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
                delResp, resp, err := r.client.api.WAFsAPI.DeleteWaf(ctx, wafID).Execute()
                _ = delResp // Ignore the delete response in retry.
                return resp, err
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(
                    err.Error(),
                    "API request failed after too many retries",
                )
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(
                    errReadAll.Error(),
                    "err",
                )
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(
                err.Error(),
                bodyString,
            )
            return
        }
    }

    // Close response body if not nil.
    if response != nil {
        defer response.Body.Close()
    }

    // Use deleteResponse to avoid unused variable error.
    _ = deleteResponse
}
```

### Building Request Objects

```go
// buildWAFEngineSettingsRequest builds a WAFEngineSettingsFieldRequest from the Terraform model.
func buildWAFEngineSettingsRequest(model *WafEngineSettingsResourceModel) azionapi.WAFEngineSettingsFieldRequest {
    engineSettings := azionapi.NewWAFEngineSettingsFieldRequest()

    if !model.EngineVersion.IsNull() && !model.EngineVersion.IsUnknown() {
        engineSettings.SetEngineVersion(model.EngineVersion.ValueString())
    }

    if !model.Type.IsNull() && !model.Type.IsUnknown() {
        engineSettings.SetType(model.Type.ValueString())
    }

    if model.Attributes != nil {
        attrs := buildWAFEngineSettingsAttributesRequest(model.Attributes)
        engineSettings.SetAttributes(attrs)
    }

    return *engineSettings
}

// buildWAFEngineSettingsAttributesRequest builds a WAFEngineSettingsAttributesFieldRequest from the Terraform model.
func buildWAFEngineSettingsAttributesRequest(model *WafEngineSettingsAttributesResourceModel) azionapi.WAFEngineSettingsAttributesFieldRequest {
    attrs := azionapi.NewWAFEngineSettingsAttributesFieldRequest()

    if len(model.Rulesets) > 0 {
        var rulesets []int64
        for _, r := range model.Rulesets {
            rulesets = append(rulesets, r.ValueInt64())
        }
        attrs.SetRulesets(rulesets)
    }

    if len(model.Thresholds) > 0 {
        var thresholds []azionapi.ThresholdsConfigFieldRequest
        for _, t := range model.Thresholds {
            threshold := azionapi.NewThresholdsConfigFieldRequest(t.Threat.ValueString())
            if !t.Sensitivity.IsNull() && !t.Sensitivity.IsUnknown() {
                threshold.SetSensitivity(t.Sensitivity.ValueString())
            }
            thresholds = append(thresholds, *threshold)
        }
        attrs.SetThresholds(thresholds)
    }

    return *attrs
}
```

### Resource Registration

Register the resource in `internal/provider.go`:

```go
func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... other resources
        WafResource,
        WafRuleSetResource,
        // ... other resources
    }
}
```

### Resource Documentation

Create documentation in `docs/resources/waf.md`:

```markdown
---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_waf"
description: |-
  Provides a WAF (Web Application Firewall) resource.
---

# azion_waf

Creates a WAF (Web Application Firewall) resource.

## Example Usage

```hcl
resource "azion_waf" "example" {
  result = {
    name   = "My WAF"
    active = true
    
    engine_settings = {
      engine_version = "2021-Q3"
      type           = "score"
      
      attributes = {
        rulesets = [1, 2, 3]
        
        thresholds = [
          {
            threshold = {
              threat      = "sql_injection"
              sensitivity = "high"
            }
          }
        ]
      }
    }
  }
}
```

## Import

```sh
terraform import azion_waf.example 12345
```

## Argument Reference

* `result` - (Required) The WAF configuration.
  * `name` - (Required) Name of the WAF.
  * `active` - (Optional) Whether the WAF is active.
  * `product_version` - (Optional) Product version of the WAF.
  * `engine_settings` - (Optional) Engine settings for the WAF.
    * `engine_version` - (Optional) Engine version for the WAF.
    * `type` - (Optional) Type of the WAF engine.
    * `attributes` - (Optional) Attributes for the WAF engine settings.
      * `rulesets` - (Optional) List of ruleset IDs.
      * `thresholds` - (Optional) Threshold configurations for the WAF. Each item must contain a single `threshold` object.
        * `threshold` - (Required) A single threshold configuration.
          * `threat` - (Required) The threat type for the threshold.
          * `sensitivity` - (Optional) The sensitivity level for the threshold.

## Attribute Reference

* `id` - The ID of the WAF.
* `last_editor` - Last editor of the WAF.
* `last_modified` - Last modified timestamp.
* `last_updated` - Timestamp of the last Terraform update of the resource.
```

---

## Summary Checklist

When generating WAF data sources and resources:

1. **Use correct SDK**: `azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"`
2. **Access via api field**: `o.client.api.WAFsAPI`
3. **Use correct method names**: `RetrieveWaf`, `ListWafs`, `CreateWaf`, `UpdateWaf`, `DeleteWaf`
4. **Handle optional fields**: Check `Has*()` methods before accessing
5. **Handle nested structures**: EngineSettings and its children
6. **Handle 429 errors**: Use `utils.RetryOn429`
7. **Close response bodies**: Use `defer response.Body.Close()`
8. **Register in provider.go**: Add to DataSources() and Resources() functions
9. **Create documentation**: Create docs and examples for both data sources and resources
10. **Run linters**: Run `golangci-lint run --config .golintci.yml ./internal/...`

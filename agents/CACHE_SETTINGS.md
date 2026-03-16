# Cache Settings - Code Generation Guide

This document provides specific guidance for implementing Cache Settings data sources in the Terraform provider using the V4 SDK.

## Table of Contents

1. [Overview](#overview)
2. [SDK Selection](#sdk-selection)
3. [V4 API Structure](#v4-api-structure)
   - [Response Types](#response-types)
   - [Module Structure](#module-structure)
4. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Retrieve by ID)](#singular-data-source-retrieve-by-id)
   - [Plural Data Source (List Multiple)](#plural-data-source-list-multiple)
5. [Schema Definition](#schema-definition)
   - [Singular Data Source Schema](#singular-data-source-schema)
6. [Transform Functions](#transform-functions)
7. [Key Differences: V3 vs V4](#key-differences-v3-vs-v4)
8. [Common Patterns](#common-patterns)
9. [Resource Implementation](#resource-implementation)
   - [Resource Schema Definition](#resource-schema-definition)
   - [Create Method](#create-method)
   - [Read Method](#read-method)
   - [Update Method (PATCH)](#update-method-patch)
   - [Delete Method](#delete-method)
   - [ImportState Method](#importstate-method)
   - [Helper: Build Modules Request](#helper-build-modules-request)
10. [Provider Registration](#provider-registration)

---

## Overview

Cache Settings configure caching behavior for Edge Applications. The V4 SDK provides a redesigned API structure with nested modules for better organization.

**Important:** The V4 API structure is significantly different from V3. The flat field structure has been replaced with nested module objects.

---

## SDK Selection

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Cache Settings (V4) | `azion-api` | `api.ApplicationsCacheSettingsAPI` | `https://api.azion.com/v4` |

### API Endpoint Path Pattern

**CRITICAL:** The V4 SDK uses the following URL path pattern for cache settings:

```
/v4/workspace/applications/{application_id}/cache_settings/{cache_setting_id}
```

The full URL is: `https://api.azion.com/v4/workspace/applications/{application_id}/cache_settings/{cache_setting_id}`

**Note the `/workspace/` prefix in the path!** This is different from other V4 API endpoints that may not include this prefix.

### API Methods

```go
// V4 SDK Pattern
r.client.api.ApplicationsCacheSettingsAPI.RetrieveCacheSetting(ctx, applicationId, cacheSettingId).Execute()
r.client.api.ApplicationsCacheSettingsAPI.ListCacheSettings(ctx, applicationId).Page(page).PageSize(pageSize).Execute()
r.client.api.ApplicationsCacheSettingsAPI.CreateCacheSetting(ctx, applicationId).CacheSettingRequest(request).Execute()
r.client.api.ApplicationsCacheSettingsAPI.UpdateCacheSetting(ctx, applicationId, cacheSettingId).CacheSettingRequest(request).Execute()
r.client.api.ApplicationsCacheSettingsAPI.DeleteCacheSetting(ctx, applicationId, cacheSettingId).Execute()
```

### ⚠️ SDK Validation Issue - Read Method Workaround

**IMPORTANT:** The SDK's `RetrieveCacheSetting` method has a validation issue where it expects the response body to be a `CacheSetting` directly, but the actual API returns a wrapper structure: `{"data": {...}}`.

This causes validation errors like: `"no value given for required property id"` when using the SDK directly.

**Workaround:** Use a raw HTTP request that manually parses the response:

```go
// retrieveCacheSettingRaw makes a raw HTTP request and manually parses the response
// to work around SDK validation issues with the data wrapper.
func retrieveCacheSettingRaw(ctx context.Context, client *apiClient, applicationId, cacheSettingId int64) (*azionapi.CacheSetting, error) {
    // Build the request URL - match the SDK's path pattern
    // SDK uses: /workspace/applications/{application_id}/cache_settings/{cache_setting_id}
    url := fmt.Sprintf("https://api.azion.com/v4/workspace/applications/%d/cache_settings/%d", applicationId, cacheSettingId)

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers from the SDK config
    for k, v := range client.apiConfig.DefaultHeader {
        httpReq.Header.Set(k, v)
    }
    httpReq.Header.Set("User-Agent", client.apiConfig.UserAgent)
    httpReq.Header.Set("Accept", "application/json; version=3")

    // Get HTTP client from config, or use default
    httpClient := client.apiConfig.HTTPClient
    if httpClient == nil {
        httpClient = http.DefaultClient
    }

    // Execute the request
    httpResp, err := httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer httpResp.Body.Close()

    // Read the response body
    bodyBytes, err := io.ReadAll(httpResp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }

    // Check for error status codes
    if httpResp.StatusCode == http.StatusNotFound {
        return nil, fmt.Errorf("404")
    }
    if httpResp.StatusCode >= 400 {
        return nil, fmt.Errorf("API error: %s - %s", httpResp.Status, string(bodyBytes))
    }

    // Parse the response - the API returns {"data": {...}} wrapper
    var wrapper struct {
        Data azionapi.CacheSetting `json:"data"`
    }
    if err := json.Unmarshal(bodyBytes, &wrapper); err != nil {
        return nil, fmt.Errorf("failed to parse response '%s': %w", string(bodyBytes), err)
    }

    return &wrapper.Data, nil
}
```

Use this function in the Read method instead of calling the SDK's `RetrieveCacheSetting` directly.

---

## V4 API Structure

### Response Types

```go
// Single cache setting response
CacheSettingResponse {
    State *string
    Data  CacheSetting
}

// Cache setting model
CacheSetting {
    Id           int64
    Name         string
    BrowserCache *BrowserCacheModule
    Modules      *CacheSettingsModules
}

// List response
PaginatedCacheSettingList {
    Count      *int64
    TotalPages *int64
    Page       *int64
    PageSize   *int64
    Next       NullableString
    Previous   NullableString
    Results    []CacheSetting
}
```

### Module Structure

```go
// Browser Cache Module
BrowserCacheModule {
    Behavior *string  // "override", "honor", "no-cache"
    MaxAge   *int64
}

// Cache Settings Modules (container)
CacheSettingsModules {
    Cache                  *CacheSettingsEdgeCacheModule
    ApplicationAccelerator *CacheSettingsApplicationAcceleratorModule
}

// Edge Cache Module
CacheSettingsEdgeCacheModule {
    Behavior       *string
    MaxAge         *int64
    StaleCache     *StateCacheModule
    LargeFileCache *LargeFileCacheModule
    TieredCache    NullableCacheSettingsTieredCacheModule
}

// Application Accelerator Module
CacheSettingsApplicationAcceleratorModule {
    CacheVaryByMethod       []string
    CacheVaryByQuerystring  *CacheVaryByQuerystringModule
    CacheVaryByCookies      *CacheVaryByCookiesModule
    CacheVaryByDevices      *CacheVaryByDevicesModule
}

// Query String Module
CacheVaryByQuerystringModule {
    Behavior     *string     // "ignore", "all", "allowlist", "denylist"
    Fields       []string
    SortEnabled  *bool
}

// Cookies Module
CacheVaryByCookiesModule {
    Behavior    *string     // "ignore", "all", "allowlist", "denylist"
    CookieNames []string
}

// Devices Module
CacheVaryByDevicesModule {
    Behavior    *string     // "ignore", "allowlist"
    DeviceGroup []int64
}

// Stale Cache Module
StateCacheModule {
    Enabled *bool
}

// Tiered Cache Module
CacheSettingsTieredCacheModule {
    Topology *string  // "nearest-region", "br-east-1", "us-east-1"
    Enabled  *bool
}
```

---

## Data Source Implementation

### Singular Data Source (Retrieve by ID)

File: `internal/data_source_edge_application_cache_setting.go`

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
    _ datasource.DataSource              = &CacheSettingDataSource{}
    _ datasource.DataSourceWithConfigure = &CacheSettingDataSource{}
)

func dataSourceAzionEdgeApplicationCacheSetting() datasource.DataSource {
    return &CacheSettingDataSource{}
}

type CacheSettingDataSource struct {
    client *apiClient
}

type CacheSettingDataSourceModel struct {
    ApplicationID types.Int64         `tfsdk:"edge_application_id"`
    Results       *CacheSettingModel  `tfsdk:"results"`
    ID            types.String        `tfsdk:"id"`
}

// Model structs matching V4 API structure
type CacheSettingModel struct {
    ID           types.Int64                `tfsdk:"id"`
    Name         types.String               `tfsdk:"name"`
    BrowserCache *BrowserCacheModuleModel   `tfsdk:"browser_cache"`
    Modules      *CacheSettingsModulesModel `tfsdk:"modules"`
}

type BrowserCacheModuleModel struct {
    Behavior types.String `tfsdk:"behavior"`
    MaxAge   types.Int64  `tfsdk:"max_age"`
}

type CacheSettingsModulesModel struct {
    Cache                  *CacheSettingsEdgeCacheModuleModel          `tfsdk:"cache"`
    ApplicationAccelerator *CacheSettingsApplicationAcceleratorModel   `tfsdk:"application_accelerator"`
}

type CacheSettingsEdgeCacheModuleModel struct {
    Behavior       types.String                 `tfsdk:"behavior"`
    MaxAge         types.Int64                  `tfsdk:"max_age"`
    StaleCache     *StateCacheModuleModel       `tfsdk:"stale_cache"`
    LargeFileCache *LargeFileCacheModuleModel   `tfsdk:"large_file_cache"`
    TieredCache    *CacheSettingsTieredCacheModel `tfsdk:"tiered_cache"`
}

type CacheSettingsApplicationAcceleratorModel struct {
    CacheVaryByMethod      []types.String                     `tfsdk:"cache_vary_by_method"`
    CacheVaryByQuerystring *CacheVaryByQuerystringModuleModel  `tfsdk:"cache_vary_by_querystring"`
    CacheVaryByCookies     *CacheVaryByCookiesModuleModel      `tfsdk:"cache_vary_by_cookies"`
    CacheVaryByDevices     *CacheVaryByDevicesModuleModel      `tfsdk:"cache_vary_by_devices"`
}

type CacheVaryByQuerystringModuleModel struct {
    Behavior    types.String   `tfsdk:"behavior"`
    Fields      []types.String `tfsdk:"fields"`
    SortEnabled types.Bool     `tfsdk:"sort_enabled"`
}

type CacheVaryByCookiesModuleModel struct {
    Behavior    types.String   `tfsdk:"behavior"`
    CookieNames []types.String `tfsdk:"cookie_names"`
}

type CacheVaryByDevicesModuleModel struct {
    Behavior    types.String  `tfsdk:"behavior"`
    DeviceGroup []types.Int64 `tfsdk:"device_group"`
}

type StateCacheModuleModel struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type CacheSettingsTieredCacheModel struct {
    Topology types.String `tfsdk:"topology"`
    Enabled  types.Bool   `tfsdk:"enabled"`
}

type LargeFileCacheModuleModel struct {
    Enabled types.Bool  `tfsdk:"enabled"`
    Offset  types.Int64 `tfsdk:"offset"`
}

func (d *CacheSettingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *CacheSettingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_application_cache_setting"
}

func (d *CacheSettingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var cacheSettingID types.Int64

    diags := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &applicationID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    diags = req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &cacheSettingID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    cacheSetting, response, err := d.client.api.ApplicationsCacheSettingsAPI.RetrieveCacheSetting(ctx, applicationID.ValueInt64(), cacheSettingID).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            cacheSetting, response, err = utils.RetryOn429(func() (*azionapi.CacheSetting, *http.Response, error) {
                return d.client.api.ApplicationsCacheSettingsAPI.RetrieveCacheSetting(ctx, applicationID.ValueInt64(), cacheSettingID).Execute()
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

    // Transform API response to state model
    result := transformCacheSettingToModel(cacheSetting)

    state := CacheSettingDataSourceModel{
        ApplicationID: applicationID,
        Results:       result,
        ID:            types.StringValue("Retrieve Cache Setting"),
    }

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

### Plural Data Source (List Multiple)

File: `internal/data_source_edge_application_cache_settings.go`

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
    _ datasource.DataSource              = &CacheSettingsDataSource{}
    _ datasource.DataSourceWithConfigure = &CacheSettingsDataSource{}
)

func dataSourceAzionEdgeApplicationCacheSettings() datasource.DataSource {
    return &CacheSettingsDataSource{}
}

type CacheSettingsDataSource struct {
    client *apiClient
}

type CacheSettingsDataSourceModel struct {
    ApplicationID types.Int64          `tfsdk:"edge_application_id"`
    Counter       types.Int64          `tfsdk:"counter"`
    Page          types.Int64          `tfsdk:"page"`
    PageSize      types.Int64          `tfsdk:"page_size"`
    TotalPages    types.Int64          `tfsdk:"total_pages"`
    Links         *LinksModel          `tfsdk:"links"`
    Results       []CacheSettingModel  `tfsdk:"results"`
    ID            types.String         `tfsdk:"id"`
}

type LinksModel struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

func (d *CacheSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *CacheSettingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_application_cache_settings"
}

func (d *CacheSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var page types.Int64
    var pageSize types.Int64

    diags := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &applicationID)
    resp.Diagnostics.Append(diags...)
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

    listResponse, response, err := d.client.api.ApplicationsCacheSettingsAPI.ListCacheSettings(ctx, applicationID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            listResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedCacheSettingList, *http.Response, error) {
                return d.client.api.ApplicationsCacheSettingsAPI.ListCacheSettings(ctx, applicationID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
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

    // Build state
    state := CacheSettingsDataSourceModel{
        ApplicationID: applicationID,
        Page:          page,
        PageSize:      pageSize,
    }

    if listResponse.HasCount() {
        state.Counter = types.Int64Value(listResponse.GetCount())
    }
    if listResponse.HasTotalPages() {
        state.TotalPages = types.Int64Value(listResponse.GetTotalPages())
    }

    // Links
    var previous, next string
    if listResponse.HasPrevious() {
        prev := listResponse.GetPrevious()
        if prev != "" {
            previous = prev
        }
    }
    if listResponse.HasNext() {
        n := listResponse.GetNext()
        if n != "" {
            next = n
        }
    }
    state.Links = &LinksModel{
        Previous: types.StringValue(previous),
        Next:     types.StringValue(next),
    }

    // Results
    if listResponse.HasResults() {
        for _, cs := range listResponse.GetResults() {
            state.Results = append(state.Results, *transformCacheSettingToModel(&cs))
        }
    }

    state.ID = types.StringValue("List Cache Settings")
    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

## Schema Definition

### Singular Data Source Schema

```go
func (d *CacheSettingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "edge_application_id": schema.Int64Attribute{
                Description: "Numeric identifier of the Edge Application.",
                Required:    true,
            },
            "results": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The cache setting identifier.",
                        Required:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the cache setting.",
                        Computed:    true,
                    },
                    "browser_cache": schema.SingleNestedAttribute{
                        Description: "Browser cache settings.",
                        Computed:    true,
                        Attributes: map[string]schema.Attribute{
                            "behavior": schema.StringAttribute{
                                Description: "Browser cache behavior: override, honor, no-cache.",
                                Computed:    true,
                            },
                            "max_age": schema.Int64Attribute{
                                Description: "Maximum TTL for browser cache.",
                                Computed:    true,
                            },
                        },
                    },
                    "modules": schema.SingleNestedAttribute{
                        Description: "Cache settings modules.",
                        Computed:    true,
                        Attributes: map[string]schema.Attribute{
                            "cache": schema.SingleNestedAttribute{
                                Description: "Edge cache module settings.",
                                Computed:    true,
                                Attributes: map[string]schema.Attribute{
                                    "behavior": schema.StringAttribute{
                                        Description: "Cache behavior: honor, override.",
                                        Computed:    true,
                                    },
                                    "max_age": schema.Int64Attribute{
                                        Description: "Maximum TTL for edge cache.",
                                        Computed:    true,
                                    },
                                    "stale_cache": schema.SingleNestedAttribute{
                                        Description: "Stale cache settings.",
                                        Computed:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "enabled": schema.BoolAttribute{
                                                Computed: true,
                                            },
                                        },
                                    },
                                    "large_file_cache": schema.SingleNestedAttribute{
                                        Description: "Large file cache settings.",
                                        Computed:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "enabled": schema.BoolAttribute{
                                                Computed: true,
                                            },
                                        },
                                    },
                                    "tiered_cache": schema.SingleNestedAttribute{
                                        Description: "Tiered cache settings.",
                                        Computed:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "topology": schema.StringAttribute{
                                                Description: "Tiered cache topology.",
                                                Computed:    true,
                                            },
                                            "enabled": schema.BoolAttribute{
                                                Computed: true,
                                            },
                                        },
                                    },
                                },
                            },
                            "application_accelerator": schema.SingleNestedAttribute{
                                Description: "Application accelerator module settings.",
                                Computed:    true,
                                Attributes: map[string]schema.Attribute{
                                    "cache_vary_by_method": schema.ListAttribute{
                                        ElementType: types.StringType,
                                        Computed:    true,
                                    },
                                    "cache_vary_by_querystring": schema.SingleNestedAttribute{
                                        Computed: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Computed: true},
                                            "fields": schema.ListAttribute{
                                                ElementType: types.StringType,
                                                Computed:    true,
                                            },
                                            "sort_enabled": schema.BoolAttribute{Computed: true},
                                        },
                                    },
                                    "cache_vary_by_cookies": schema.SingleNestedAttribute{
                                        Computed: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Computed: true},
                                            "cookie_names": schema.ListAttribute{
                                                ElementType: types.StringType,
                                                Computed:    true,
                                            },
                                        },
                                    },
                                    "cache_vary_by_devices": schema.SingleNestedAttribute{
                                        Computed: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Computed: true},
                                            "device_group": schema.ListAttribute{
                                                ElementType: types.Int64Type,
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
    }
}
```

---

## Transform Functions

```go
func transformCacheSettingToModel(cs *azionapi.CacheSetting) *CacheSettingModel {
    if cs == nil {
        return nil
    }

    model := &CacheSettingModel{
        ID:   types.Int64Value(cs.GetId()),
        Name: types.StringValue(cs.GetName()),
    }

    // Browser Cache
    if cs.HasBrowserCache() {
        bc := cs.GetBrowserCache()
        model.BrowserCache = &BrowserCacheModuleModel{}
        if bc.HasBehavior() {
            model.BrowserCache.Behavior = types.StringValue(bc.GetBehavior())
        }
        if bc.HasMaxAge() {
            model.BrowserCache.MaxAge = types.Int64Value(bc.GetMaxAge())
        }
    }

    // Modules
    if cs.HasModules() {
        modules := cs.GetModules()
        model.Modules = &CacheSettingsModulesModel{}

        // Cache (Edge Cache)
        if modules.HasCache() {
            cache := modules.GetCache()
            model.Modules.Cache = &CacheSettingsEdgeCacheModuleModel{}
            
            if cache.HasBehavior() {
                model.Modules.Cache.Behavior = types.StringValue(cache.GetBehavior())
            }
            if cache.HasMaxAge() {
                model.Modules.Cache.MaxAge = types.Int64Value(cache.GetMaxAge())
            }

            // Stale Cache
            if cache.HasStaleCache() {
                sc := cache.GetStaleCache()
                model.Modules.Cache.StaleCache = &StateCacheModuleModel{}
                if sc.HasEnabled() {
                    model.Modules.Cache.StaleCache.Enabled = types.BoolValue(sc.GetEnabled())
                }
            }

            // Large File Cache
            if cache.HasLargeFileCache() {
                lfc := cache.GetLargeFileCache()
                model.Modules.Cache.LargeFileCache = &LargeFileCacheModuleModel{}
                if lfc.HasEnabled() {
                    model.Modules.Cache.LargeFileCache.Enabled = types.BoolValue(lfc.GetEnabled())
                }
                if lfc.HasOffset() {
                    model.Modules.Cache.LargeFileCache.Offset = types.Int64Value(lfc.GetOffset())
                }
            }

            // Tiered Cache
            if cache.HasTieredCache() {
                tc := cache.GetTieredCache()
                model.Modules.Cache.TieredCache = &CacheSettingsTieredCacheModel{}
                if tc.HasTopology() {
                    model.Modules.Cache.TieredCache.Topology = types.StringValue(tc.GetTopology())
                }
                if tc.HasEnabled() {
                    model.Modules.Cache.TieredCache.Enabled = types.BoolValue(tc.GetEnabled())
                }
            }
        }

        // Application Accelerator
        if modules.HasApplicationAccelerator() {
            aa := modules.GetApplicationAccelerator()
            model.Modules.ApplicationAccelerator = &CacheSettingsApplicationAcceleratorModel{}

            // Cache Vary By Method
            if aa.HasCacheVaryByMethod() {
                for _, method := range aa.GetCacheVaryByMethod() {
                    model.Modules.ApplicationAccelerator.CacheVaryByMethod = append(
                        model.Modules.ApplicationAccelerator.CacheVaryByMethod,
                        types.StringValue(method),
                    )
                }
            }

            // Cache Vary By Querystring
            if aa.HasCacheVaryByQuerystring() {
                qs := aa.GetCacheVaryByQuerystring()
                model.Modules.ApplicationAccelerator.CacheVaryByQuerystring = &CacheVaryByQuerystringModuleModel{}
                
                if qs.HasBehavior() {
                    model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Behavior = types.StringValue(qs.GetBehavior())
                }
                if qs.HasFields() {
                    for _, f := range qs.GetFields() {
                        model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields = append(
                            model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields,
                            types.StringValue(f),
                        )
                    }
                }
                if qs.HasSortEnabled() {
                    model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.SortEnabled = types.BoolValue(qs.GetSortEnabled())
                }
            }

            // Cache Vary By Cookies
            if aa.HasCacheVaryByCookies() {
                cookies := aa.GetCacheVaryByCookies()
                model.Modules.ApplicationAccelerator.CacheVaryByCookies = &CacheVaryByCookiesModuleModel{}
                
                if cookies.HasBehavior() {
                    model.Modules.ApplicationAccelerator.CacheVaryByCookies.Behavior = types.StringValue(cookies.GetBehavior())
                }
                if cookies.HasCookieNames() {
                    for _, cn := range cookies.GetCookieNames() {
                        model.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames = append(
                            model.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames,
                            types.StringValue(cn),
                        )
                    }
                }
            }

            // Cache Vary By Devices
            if aa.HasCacheVaryByDevices() {
                devices := aa.GetCacheVaryByDevices()
                model.Modules.ApplicationAccelerator.CacheVaryByDevices = &CacheVaryByDevicesModuleModel{}
                
                if devices.HasBehavior() {
                    model.Modules.ApplicationAccelerator.CacheVaryByDevices.Behavior = types.StringValue(devices.GetBehavior())
                }
                if devices.HasDeviceGroup() {
                    for _, dg := range devices.GetDeviceGroup() {
                        model.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup = append(
                            model.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup,
                            types.Int64Value(dg),
                        )
                    }
                }
            }
        }
    }

    return model
}
```

---

## Key Differences: V3 vs V4

| Feature | V3 (Legacy) | V4 (Current) |
|---------|-------------|--------------|
| Structure | Flat fields | Nested modules |
| Field Names | snake_case (browser_cache_settings) | Nested objects (browser_cache.behavior) |
| API Client | `edgeApplicationsApi` | `api.ApplicationsCacheSettingsAPI` |
| Method Names | `EdgeApplicationsEdgeApplicationIdCacheSettingsGet` | `ListCacheSettings` |
| Response | `ApplicationCacheGetResponse` | `PaginatedCacheSettingList` |
| SDK Package | `azionapi-go-sdk/edgeapplications` | `azionapi-v4-go-sdk-dev/azion-api` |

---

## Common Patterns

### Handling Nullable Fields

```go
// Check if field exists before accessing
if cache.HasTieredCache() {
    tc := cache.GetTieredCache()
    // Access nested fields
}
```

### Handling List Fields

```go
// Transform slice of strings
var fields []types.String
for _, f := range qs.GetFields() {
    fields = append(fields, types.StringValue(f))
}
```

### Handling Nested Modules

```go
// Always check parent exists before accessing children
if modules.HasCache() {
    cache := modules.GetCache()
    if cache.HasStaleCache() {
        // Access stale_cache
    }
}
```

---

## Resource Implementation

File: `internal/resource_edge_application_cache_setting.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "strconv"
    "time"

    edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ resource.Resource                = &cacheSettingResource{}
    _ resource.ResourceWithConfigure   = &cacheSettingResource{}
    _ resource.ResourceWithImportState = &cacheSettingResource{}
)

func NewCacheSettingResource() resource.Resource {
    return &cacheSettingResource{}
}

type cacheSettingResource struct {
    client *apiClient
}

// Resource Model - represents Terraform state
type CacheSettingResourceModel struct {
    ApplicationID types.Int64              `tfsdk:"edge_application_id"`
    CacheSetting  *CacheSettingModel       `tfsdk:"cache_setting"`
    ID            types.String             `tfsdk:"id"`
    LastUpdated   types.String             `tfsdk:"last_updated"`
}

// CacheSettingModel - V4 API structure
type CacheSettingModel struct {
    ID           types.Int64                `tfsdk:"id"`
    Name         types.String               `tfsdk:"name"`
    BrowserCache *BrowserCacheModuleModel   `tfsdk:"browser_cache"`
    Modules      *CacheSettingsModulesModel `tfsdk:"modules"`
}

type BrowserCacheModuleModel struct {
    Behavior types.String `tfsdk:"behavior"`
    MaxAge   types.Int64  `tfsdk:"max_age"`
}

type CacheSettingsModulesModel struct {
    Cache                  *CacheSettingsEdgeCacheModuleModel        `tfsdk:"cache"`
    ApplicationAccelerator *CacheSettingsApplicationAcceleratorModel `tfsdk:"application_accelerator"`
}

type CacheSettingsEdgeCacheModuleModel struct {
    Behavior       types.String                    `tfsdk:"behavior"`
    MaxAge         types.Int64                     `tfsdk:"max_age"`
    StaleCache     *StateCacheModuleModel          `tfsdk:"stale_cache"`
    LargeFileCache *LargeFileCacheModuleModel      `tfsdk:"large_file_cache"`
    TieredCache    *CacheSettingsTieredCacheModel  `tfsdk:"tiered_cache"`
}

type CacheSettingsApplicationAcceleratorModel struct {
    CacheVaryByMethod      []types.String                    `tfsdk:"cache_vary_by_method"`
    CacheVaryByQuerystring *CacheVaryByQuerystringModuleModel `tfsdk:"cache_vary_by_querystring"`
    CacheVaryByCookies     *CacheVaryByCookiesModuleModel     `tfsdk:"cache_vary_by_cookies"`
    CacheVaryByDevices     *CacheVaryByDevicesModuleModel     `tfsdk:"cache_vary_by_devices"`
}

type CacheVaryByQuerystringModuleModel struct {
    Behavior    types.String   `tfsdk:"behavior"`
    Fields      []types.String `tfsdk:"fields"`
    SortEnabled types.Bool     `tfsdk:"sort_enabled"`
}

type CacheVaryByCookiesModuleModel struct {
    Behavior    types.String   `tfsdk:"behavior"`
    CookieNames []types.String `tfsdk:"cookie_names"`
}

type CacheVaryByDevicesModuleModel struct {
    Behavior    types.String  `tfsdk:"behavior"`
    DeviceGroup []types.Int64 `tfsdk:"device_group"`
}

type StateCacheModuleModel struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type CacheSettingsTieredCacheModel struct {
    Topology types.String `tfsdk:"topology"`
    Enabled  types.Bool   `tfsdk:"enabled"`
}

type LargeFileCacheModuleModel struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

func (r *cacheSettingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_application_cache_setting"
}

func (r *cacheSettingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    r.client = req.ProviderData.(*apiClient)
}

// Schema definition continues below...
```

---

## Resource Schema Definition

```go
func (r *cacheSettingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Resource identifier.",
                Computed:    true,
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
            "cache_setting": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "Cache setting identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the cache setting.",
                        Required:    true,
                    },
                    "browser_cache": schema.SingleNestedAttribute{
                        Description: "Browser cache settings.",
                        Optional:    true,
                        Attributes: map[string]schema.Attribute{
                            "behavior": schema.StringAttribute{
                                Description: "Browser cache behavior: override, honor, no-cache.",
                                Optional:    true,
                            },
                            "max_age": schema.Int64Attribute{
                                Description: "Maximum TTL for browser cache.",
                                Optional:    true,
                            },
                        },
                    },
                    "modules": schema.SingleNestedAttribute{
                        Description: "Cache settings modules.",
                        Optional:    true,
                        Attributes: map[string]schema.Attribute{
                            "cache": schema.SingleNestedAttribute{
                                Description: "Edge cache module settings.",
                                Optional:    true,
                                Attributes: map[string]schema.Attribute{
                                    "behavior": schema.StringAttribute{
                                        Description: "Cache behavior: honor, override.",
                                        Optional:    true,
                                    },
                                    "max_age": schema.Int64Attribute{
                                        Description: "Maximum TTL for edge cache.",
                                        Optional:    true,
                                    },
                                    "stale_cache": schema.SingleNestedAttribute{
                                        Description: "Stale cache settings.",
                                        Optional:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "enabled": schema.BoolAttribute{
                                                Optional: true,
                                            },
                                        },
                                    },
                                    "large_file_cache": schema.SingleNestedAttribute{
                                        Description: "Large file cache settings.",
                                        Optional:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "enabled": schema.BoolAttribute{
                                                Optional: true,
                                            },
                                        },
                                    },
                                    "tiered_cache": schema.SingleNestedAttribute{
                                        Description: "Tiered cache settings.",
                                        Optional:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "topology": schema.StringAttribute{
                                                Description: "Tiered cache topology.",
                                                Optional:    true,
                                            },
                                            "enabled": schema.BoolAttribute{
                                                Optional: true,
                                            },
                                        },
                                    },
                                },
                            },
                            "application_accelerator": schema.SingleNestedAttribute{
                                Description: "Application accelerator module settings.",
                                Optional:    true,
                                Attributes: map[string]schema.Attribute{
                                    "cache_vary_by_method": schema.ListAttribute{
                                        ElementType: types.StringType,
                                        Optional:    true,
                                    },
                                    "cache_vary_by_querystring": schema.SingleNestedAttribute{
                                        Optional: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Optional: true},
                                            "fields": schema.ListAttribute{
                                                ElementType: types.StringType,
                                                Optional:    true,
                                            },
                                            "sort_enabled": schema.BoolAttribute{Optional: true},
                                        },
                                    },
                                    "cache_vary_by_cookies": schema.SingleNestedAttribute{
                                        Optional: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Optional: true},
                                            "cookie_names": schema.ListAttribute{
                                                ElementType: types.StringType,
                                                Optional:    true,
                                            },
                                        },
                                    },
                                    "cache_vary_by_devices": schema.SingleNestedAttribute{
                                        Optional: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Optional: true},
                                            "device_group": schema.ListAttribute{
                                                ElementType: types.Int64Type,
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
    }
}
```

---

## Create Method

```go
func (r *cacheSettingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan CacheSettingResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the V4 API request
    cacheSettingRequest := azionapi.NewCacheSettingRequest(plan.CacheSetting.Name.ValueString())

    // Browser Cache
    if plan.CacheSetting.BrowserCache != nil {
        browserCache := azionapi.NewBrowserCacheModuleRequest()
        if !plan.CacheSetting.BrowserCache.Behavior.IsNull() {
            browserCache.SetBehavior(plan.CacheSetting.BrowserCache.Behavior.ValueString())
        }
        if !plan.CacheSetting.BrowserCache.MaxAge.IsNull() {
            browserCache.SetMaxAge(plan.CacheSetting.BrowserCache.MaxAge.ValueInt64())
        }
        cacheSettingRequest.SetBrowserCache(*browserCache)
    }

    // Modules
    if plan.CacheSetting.Modules != nil {
        modulesRequest := azionapi.NewCacheSettingsModulesRequest()
        
        // Cache (Edge Cache)
        if plan.CacheSetting.Modules.Cache != nil {
            cacheRequest := azionapi.NewCacheSettingsEdgeCacheModuleRequest()
            
            if !plan.CacheSetting.Modules.Cache.Behavior.IsNull() {
                cacheRequest.SetBehavior(plan.CacheSetting.Modules.Cache.Behavior.ValueString())
            }
            if !plan.CacheSetting.Modules.Cache.MaxAge.IsNull() {
                cacheRequest.SetMaxAge(plan.CacheSetting.Modules.Cache.MaxAge.ValueInt64())
            }
            
            // Stale Cache
            if plan.CacheSetting.Modules.Cache.StaleCache != nil {
                staleCache := azionapi.NewStateCacheModuleRequest()
                if !plan.CacheSetting.Modules.Cache.StaleCache.Enabled.IsNull() {
                    staleCache.SetEnabled(plan.CacheSetting.Modules.Cache.StaleCache.Enabled.ValueBool())
                }
                cacheRequest.SetStaleCache(*staleCache)
            }
            
            // Tiered Cache
            if plan.CacheSetting.Modules.Cache.TieredCache != nil {
                tieredCache := azionapi.NewCacheSettingsTieredCacheModuleRequest()
                if !plan.CacheSetting.Modules.Cache.TieredCache.Topology.IsNull() {
                    tieredCache.SetTopology(plan.CacheSetting.Modules.Cache.TieredCache.Topology.ValueString())
                }
                if !plan.CacheSetting.Modules.Cache.TieredCache.Enabled.IsNull() {
                    tieredCache.SetEnabled(plan.CacheSetting.Modules.Cache.TieredCache.Enabled.ValueBool())
                }
                cacheRequest.SetTieredCache(*tieredCache)
            }
            
            // Large File Cache
            if plan.CacheSetting.Modules.Cache.LargeFileCache != nil {
                largeFileCache := azionapi.NewLargeFileCacheModuleRequest()
                if !plan.CacheSetting.Modules.Cache.LargeFileCache.Enabled.IsNull() {
                    largeFileCache.SetEnabled(plan.CacheSetting.Modules.Cache.LargeFileCache.Enabled.ValueBool())
                }
                cacheRequest.SetLargeFileCache(*largeFileCache)
            }
            
            modulesRequest.SetCache(*cacheRequest)
        }
        
        // Application Accelerator
        if plan.CacheSetting.Modules.ApplicationAccelerator != nil {
            aa := plan.CacheSetting.Modules.ApplicationAccelerator
            aaRequest := azionapi.NewCacheSettingsApplicationAcceleratorModuleRequest()
            
            // Cache Vary By Method
            if len(aa.CacheVaryByMethod) > 0 {
                var methods []string
                for _, m := range aa.CacheVaryByMethod {
                    methods = append(methods, m.ValueString())
                }
                aaRequest.SetCacheVaryByMethod(methods)
            }
            
            // Cache Vary By Querystring
            if aa.CacheVaryByQuerystring != nil {
                qs := azionapi.NewCacheVaryByQuerystringModuleRequest()
                if !aa.CacheVaryByQuerystring.Behavior.IsNull() {
                    qs.SetBehavior(aa.CacheVaryByQuerystring.Behavior.ValueString())
                }
                if len(aa.CacheVaryByQuerystring.Fields) > 0 {
                    var fields []string
                    for _, f := range aa.CacheVaryByQuerystring.Fields {
                        fields = append(fields, f.ValueString())
                    }
                    qs.SetFields(fields)
                }
                if !aa.CacheVaryByQuerystring.SortEnabled.IsNull() {
                    qs.SetSortEnabled(aa.CacheVaryByQuerystring.SortEnabled.ValueBool())
                }
                aaRequest.SetCacheVaryByQuerystring(*qs)
            }
            
            // Cache Vary By Cookies
            if aa.CacheVaryByCookies != nil {
                cookies := azionapi.NewCacheVaryByCookiesModuleRequest()
                if !aa.CacheVaryByCookies.Behavior.IsNull() {
                    cookies.SetBehavior(aa.CacheVaryByCookies.Behavior.ValueString())
                }
                if len(aa.CacheVaryByCookies.CookieNames) > 0 {
                    var names []string
                    for _, n := range aa.CacheVaryByCookies.CookieNames {
                        names = append(names, n.ValueString())
                    }
                    cookies.SetCookieNames(names)
                }
                aaRequest.SetCacheVaryByCookies(*cookies)
            }
            
            // Cache Vary By Devices
            if aa.CacheVaryByDevices != nil {
                devices := azionapi.NewCacheVaryByDevicesModuleRequest()
                if !aa.CacheVaryByDevices.Behavior.IsNull() {
                    devices.SetBehavior(aa.CacheVaryByDevices.Behavior.ValueString())
                }
                if len(aa.CacheVaryByDevices.DeviceGroup) > 0 {
                    var groups []int64
                    for _, g := range aa.CacheVaryByDevices.DeviceGroup {
                        groups = append(groups, g.ValueInt64())
                    }
                    devices.SetDeviceGroup(groups)
                }
                aaRequest.SetCacheVaryByDevices(*devices)
            }
            
            modulesRequest.SetApplicationAccelerator(*aaRequest)
        }
        
        cacheSettingRequest.SetModules(*modulesRequest)
    }

    // Call V4 API
    createdCacheSetting, response, err := r.client.api.ApplicationsCacheSettingsAPI.
        CreateCacheSetting(ctx, plan.ApplicationID.ValueInt64()).
        CacheSettingRequest(*cacheSettingRequest).
        Execute()
    if err != nil {
        if response.StatusCode == 429 {
            createdCacheSetting, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
                return r.client.api.ApplicationsCacheSettingsAPI.
                    CreateCacheSetting(ctx, plan.ApplicationID.ValueInt64()).
                    CacheSettingRequest(*cacheSettingRequest).
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

    // Transform response to state
    plan.CacheSetting = transformCacheSettingResponseToModel(createdCacheSetting.GetData())
    plan.ID = types.StringValue(strconv.FormatInt(createdCacheSetting.GetData().GetId(), 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, &plan)
    resp.Diagnostics.Append(diags...)
}
```

---

## Read Method

**⚠️ IMPORTANT:** The Read method must use the `retrieveCacheSettingRaw` function instead of the SDK's `RetrieveCacheSetting` method due to a validation issue. See the [SDK Validation Issue - Read Method Workaround](#️-sdk-validation-issue---read-method-workaround) section above.

```go
func (r *edgeApplicationCacheSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state EdgeApplicationCacheSettingsResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    applicationId := state.ApplicationID.ValueInt64()
    cacheSettingId := state.CacheSetting.ID.ValueInt64()

    // Use raw HTTP request to work around SDK validation issue with data wrapper
    // The API returns {"data": {...}} but SDK's RetrieveCacheSetting validation expects
    // the CacheSetting directly, causing "no value given for required property id" errors.
    cacheSetting, err := retrieveCacheSettingRaw(ctx, r.client, applicationId, cacheSettingId)
    if err != nil {
        if err.Error() == "404" {
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.AddError("Failed to retrieve cache setting", err.Error())
        return
    }

    // Debug: ensure we got a valid cache setting
    if cacheSetting == nil {
        resp.Diagnostics.AddError("Empty response", "cacheSetting is nil after successful API call")
        return
    }

    // Update state with response - Read should return the full API state
    state.CacheSetting = transformCacheSettingResponseToResourceModel(cacheSetting)
    // Preserve top-level ID from state if not already set
    if state.ID.IsNull() || state.ID.IsUnknown() {
        if state.CacheSetting != nil {
            state.ID = state.CacheSetting.ID
        }
    }

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

## Update Method (PATCH)

```go
func (r *cacheSettingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan CacheSettingResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    applicationId := plan.ApplicationID.ValueInt64()
    cacheSettingId := plan.CacheSetting.ID.ValueInt64()

    // Build patched request (only include fields that changed)
    patchedRequest := azionapi.NewPatchedCacheSettingRequest()
    
    if !plan.CacheSetting.Name.IsNull() {
        patchedRequest.SetName(plan.CacheSetting.Name.ValueString())
    }
    
    // Browser Cache
    if plan.CacheSetting.BrowserCache != nil {
        browserCache := azionapi.NewBrowserCacheModuleRequest()
        if !plan.CacheSetting.BrowserCache.Behavior.IsNull() {
            browserCache.SetBehavior(plan.CacheSetting.BrowserCache.Behavior.ValueString())
        }
        if !plan.CacheSetting.BrowserCache.MaxAge.IsNull() {
            browserCache.SetMaxAge(plan.CacheSetting.BrowserCache.MaxAge.ValueInt64())
        }
        patchedRequest.SetBrowserCache(*browserCache)
    }
    
    // Modules - similar to Create
    if plan.CacheSetting.Modules != nil {
        modulesRequest := buildModulesRequest(plan.CacheSetting.Modules)
        patchedRequest.SetModules(*modulesRequest)
    }

    // Call V4 API PATCH
    updatedCacheSetting, response, err := r.client.api.ApplicationsCacheSettingsAPI.
        PartialUpdateCacheSetting(ctx, applicationId, cacheSettingId).
        PatchedCacheSettingRequest(*patchedRequest).
        Execute()
    if err != nil {
        if response.StatusCode == 429 {
            updatedCacheSetting, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
                return r.client.api.ApplicationsCacheSettingsAPI.
                    PartialUpdateCacheSetting(ctx, applicationId, cacheSettingId).
                    PatchedCacheSettingRequest(*patchedRequest).
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

    // Update state
    plan.CacheSetting = transformCacheSettingResponseToModel(updatedCacheSetting.GetData())
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, &plan)
    resp.Diagnostics.Append(diags...)
}
```

---

## Delete Method

```go
func (r *cacheSettingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state CacheSettingResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    applicationId := state.ApplicationID.ValueInt64()
    cacheSettingId := state.CacheSetting.ID.ValueInt64()

    _, response, err := r.client.api.ApplicationsCacheSettingsAPI.
        DeleteCacheSetting(ctx, applicationId, cacheSettingId).
        Execute()
    if err != nil {
        if response.StatusCode == 429 {
            _, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
                return r.client.api.ApplicationsCacheSettingsAPI.
                    DeleteCacheSetting(ctx, applicationId, cacheSettingId).
                    Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else if response.StatusCode != http.StatusNotFound {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }
}
```

---

## ImportState Method

**⚠️ IMPORTANT:** The ImportState method must also use the `retrieveCacheSettingRaw` function for the same reason as the Read method.

```go
func (r *edgeApplicationCacheSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Expected format: {application_id}/{cache_setting_id}
    parts := strings.Split(req.ID, "/")
    if len(parts) != 2 {
        resp.Diagnostics.AddError(
            "Invalid import ID format",
            "Expected format: {application_id}/{cache_setting_id}",
        )
        return
    }

    applicationId, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid application ID", err.Error())
        return
    }

    cacheSettingId, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid cache setting ID", err.Error())
        return
    }

    // Set the application ID
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("edge_application_id"), applicationId)...)

    // Read the cache setting using raw HTTP to work around SDK validation issue
    cacheSetting, err := retrieveCacheSettingRaw(ctx, r.client, applicationId, cacheSettingId)
    if err != nil {
        if err.Error() == "404" {
            resp.Diagnostics.AddError("Cache setting not found", "")
            return
        }
        resp.Diagnostics.AddError("Failed to retrieve cache setting", err.Error())
        return
    }

    // Build state
    state := EdgeApplicationCacheSettingsResourceModel{
        ApplicationID: types.Int64Value(applicationId),
        CacheSetting:  transformCacheSettingResponseToResourceModel(cacheSetting),
        ID:            types.Int64Value(cacheSettingId),
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

## Helper: Build Modules Request

```go
func buildModulesRequest(modules *CacheSettingsModulesModel) *azionapi.CacheSettingsModulesRequest {
    modulesRequest := azionapi.NewCacheSettingsModulesRequest()
    
    if modules.Cache != nil {
        cacheRequest := azionapi.NewCacheSettingsEdgeCacheModuleRequest()
        
        if !modules.Cache.Behavior.IsNull() {
            cacheRequest.SetBehavior(modules.Cache.Behavior.ValueString())
        }
        if !modules.Cache.MaxAge.IsNull() {
            cacheRequest.SetMaxAge(modules.Cache.MaxAge.ValueInt64())
        }
        
        if modules.Cache.StaleCache != nil {
            staleCache := azionapi.NewStateCacheModuleRequest()
            if !modules.Cache.StaleCache.Enabled.IsNull() {
                staleCache.SetEnabled(modules.Cache.StaleCache.Enabled.ValueBool())
            }
            cacheRequest.SetStaleCache(*staleCache)
        }
        
        if modules.Cache.TieredCache != nil {
            tieredCache := azionapi.NewCacheSettingsTieredCacheModuleRequest()
            if !modules.Cache.TieredCache.Topology.IsNull() {
                tieredCache.SetTopology(modules.Cache.TieredCache.Topology.ValueString())
            }
            if !modules.Cache.TieredCache.Enabled.IsNull() {
                tieredCache.SetEnabled(modules.Cache.TieredCache.Enabled.ValueBool())
            }
            cacheRequest.SetTieredCache(*tieredCache)
        }
        
        if modules.Cache.LargeFileCache != nil {
            largeFileCache := azionapi.NewLargeFileCacheModuleRequest()
            if !modules.Cache.LargeFileCache.Enabled.IsNull() {
                largeFileCache.SetEnabled(modules.Cache.LargeFileCache.Enabled.ValueBool())
            }
            cacheRequest.SetLargeFileCache(*largeFileCache)
        }
        
        modulesRequest.SetCache(*cacheRequest)
    }
    
    if modules.ApplicationAccelerator != nil {
        aa := modules.ApplicationAccelerator
        aaRequest := azionapi.NewCacheSettingsApplicationAcceleratorModuleRequest()
        
        if len(aa.CacheVaryByMethod) > 0 {
            var methods []string
            for _, m := range aa.CacheVaryByMethod {
                methods = append(methods, m.ValueString())
            }
            aaRequest.SetCacheVaryByMethod(methods)
        }
        
        if aa.CacheVaryByQuerystring != nil {
            qs := buildQuerystringRequest(aa.CacheVaryByQuerystring)
            aaRequest.SetCacheVaryByQuerystring(*qs)
        }
        
        if aa.CacheVaryByCookies != nil {
            cookies := buildCookiesRequest(aa.CacheVaryByCookies)
            aaRequest.SetCacheVaryByCookies(*cookies)
        }
        
        if aa.CacheVaryByDevices != nil {
            devices := buildDevicesRequest(aa.CacheVaryByDevices)
            aaRequest.SetCacheVaryByDevices(*devices)
        }
        
        modulesRequest.SetApplicationAccelerator(*aaRequest)
    }
    
    return modulesRequest
}

func buildQuerystringRequest(qs *CacheVaryByQuerystringModuleModel) *azionapi.CacheVaryByQuerystringModuleRequest {
    request := azionapi.NewCacheVaryByQuerystringModuleRequest()
    
    if !qs.Behavior.IsNull() {
        request.SetBehavior(qs.Behavior.ValueString())
    }
    if len(qs.Fields) > 0 {
        var fields []string
        for _, f := range qs.Fields {
            fields = append(fields, f.ValueString())
        }
        request.SetFields(fields)
    }
    if !qs.SortEnabled.IsNull() {
        request.SetSortEnabled(qs.SortEnabled.ValueBool())
    }
    
    return request
}

func buildCookiesRequest(cookies *CacheVaryByCookiesModuleModel) *azionapi.CacheVaryByCookiesModuleRequest {
    request := azionapi.NewCacheVaryByCookiesModuleRequest()
    
    if !cookies.Behavior.IsNull() {
        request.SetBehavior(cookies.Behavior.ValueString())
    }
    if len(cookies.CookieNames) > 0 {
        var names []string
        for _, n := range cookies.CookieNames {
            names = append(names, n.ValueString())
        }
        request.SetCookieNames(names)
    }
    
    return request
}

func buildDevicesRequest(devices *CacheVaryByDevicesModuleModel) *azionapi.CacheVaryByDevicesModuleRequest {
    request := azionapi.NewCacheVaryByDevicesModuleRequest()
    
    if !devices.Behavior.IsNull() {
        request.SetBehavior(devices.Behavior.ValueString())
    }
    if len(devices.DeviceGroup) > 0 {
        var groups []int64
        for _, g := range devices.DeviceGroup {
            groups = append(groups, g.ValueInt64())
        }
        request.SetDeviceGroup(groups)
    }
    
    return request
}
```

---

## Provider Registration

Register in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionEdgeApplicationCacheSetting,
        dataSourceAzionEdgeApplicationCacheSettings,
        // ... other data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewCacheSettingResource,
        // ... other resources
    }
}
